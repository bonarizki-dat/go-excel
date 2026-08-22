package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bonarizki-dat/go-excel/internal/concurrency"
)

// stopGracePeriod is how long the pipeline may sit completely idle
// during Stop(), with no item moving anywhere, before Stop() concludes
// nobody is reading Output() (or Errors(), though that channel never
// blocks a sender) and forces stuck senders to give up. It is an
// inactivity window, not a deadline on the whole shutdown, so a
// consumer that keeps draining has as long as it needs. Mirrors
// WorkerPool's stopGracePeriod in internal/concurrency/worker.go.
const stopGracePeriod = 250 * time.Millisecond

// Pipeline is a generic multi-stage concurrent processing primitive:
// data flows through a sequence of named stages, each with its own
// concurrency level, connected by backpressured channels.
//
// Pipeline is a standalone building block. Nothing in this module calls
// New internally: StreamExporter and StreamImporter use
// internal/concurrency.WorkerPool directly instead, and do not depend
// on or exercise Pipeline's own concurrency, shutdown, or backpressure
// handling. Pipeline is
// validated only by its own tests, not by production call paths
// elsewhere in this module, so treat it as an independently useful
// component rather than something already battle-tested via the rest
// of this library.
//
// Usage:
//
//	pipeline := pipeline.New(ctx)
//	pipeline.AddStage("parse", parseStage, 1)
//	pipeline.AddStage("validate", validateStage, 1)
//	pipeline.Start()
type Pipeline struct {
	// ctx is the context for cancellation and timeout control
	ctx context.Context

	// stages holds all pipeline stages in execution order; each
	// stage's outputChan feeds the next stage's inputChan.
	stages []Stage

	// inputChan is where external code feeds data into the first stage.
	inputChan chan any

	// outputChan is where external code reads the last stage's results.
	outputChan chan any

	// errorChan collects errors from all stages. Buffered so a stage
	// reporting an error does not block on a slow error consumer.
	errorChan chan error

	// wg tracks active stage goroutines
	wg sync.WaitGroup

	// poolManager manages memory pools for efficiency
	poolManager *concurrency.PoolManager

	// backpressureSize sets inputChan/outputChan/per-stage buffer
	// capacity; larger values trade memory for throughput.
	backpressureSize int

	// stopping is closed by Stop() to unblock a stage worker or the
	// output forwarder stuck sending a value that nobody is draining
	// (in particular, forwardOutput sending to outputChan when the
	// caller never reads Output()). Deliberately separate from ctx,
	// same reasoning as WorkerPool.stopping in
	// internal/concurrency/worker.go: honoring it inside the per-item
	// processing loop would wrongly abandon items that were
	// legitimately received during a normal shutdown.
	stopping chan struct{}

	// stopOnce ensures Stop() only runs its shutdown sequence once.
	stopOnce sync.Once

	// droppedErrors counts errors discarded by runStageWorker because
	// errorChan was full at send time. Exposed via DroppedErrors() so
	// callers who do not drain Errors() promptly can still detect that
	// errors were lost, instead of dropping them in total silence.
	droppedErrors uint64

	// progress counts items handed off to a stage's output channel or
	// to outputChan. Stop() watches it to tell a pipeline that is
	// still draining, however slowly, from one that has wedged.
	progress uint64
}

// Stage and StageFunc are defined in stage.go

// New creates a pipeline governed by ctx, with default channel
// buffering. Add stages with AddStage before calling Start.
func New(ctx context.Context) *Pipeline {
	// 1000 items per channel balances memory usage against throughput.
	backpressureSize := 1000

	return &Pipeline{
		ctx:              ctx,
		stages:           make([]Stage, 0),
		inputChan:        make(chan any, backpressureSize),
		outputChan:       make(chan any, backpressureSize),
		errorChan:        make(chan error, 100),
		poolManager:      concurrency.NewPoolManager(),
		backpressureSize: backpressureSize,
		stopping:         make(chan struct{}),
	}
}

// AddStage appends a stage named name to the pipeline, to be run after
// any previously added stages. workers sets how many goroutines
// process this stage's input concurrently; a value of 0 or less
// defaults to 1.
func (p *Pipeline) AddStage(name string, process StageFunc, workers int) {
	if workers <= 0 {
		workers = 1
	}

	stage := Stage{
		name:        name,
		process:     process,
		concurrency: workers,
		inputChan:   p.getInputChannel(),
		outputChan:  make(chan any, p.backpressureSize),
	}

	p.stages = append(p.stages, stage)
}

// getInputChannel returns the channel the next stage being added should
// read from: the pipeline's input channel if no stages exist yet, or
// the most recently added stage's output channel otherwise.
func (p *Pipeline) getInputChannel() chan any {
	if len(p.stages) == 0 {
		return p.inputChan
	}
	return p.stages[len(p.stages)-1].outputChan
}

// Start launches each stage's worker goroutines, at the concurrency
// given to AddStage, and returns immediately without waiting for any
// data to be processed.
func (p *Pipeline) Start() {
	for i := range p.stages {
		stage := &p.stages[i]

		// Per-stage WaitGroup lets the closer goroutine below know when
		// this stage's own workers (not the whole pipeline) are done.
		var stageWg sync.WaitGroup

		for j := 0; j < stage.concurrency; j++ {
			p.wg.Add(1)
			stageWg.Add(1)

			go func(s *Stage) {
				defer p.wg.Done()
				defer stageWg.Done()
				p.runStageWorker(s)
			}(stage)
		}

		// Closing outputChan once every worker for this stage has
		// exited is what lets the next stage (or forwardOutput) detect
		// end-of-input via its own range/receive loop.
		go func(s *Stage, wg *sync.WaitGroup) {
			wg.Wait()
			close(s.outputChan)
		}(stage, &stageWg)
	}

	if len(p.stages) > 0 {
		lastStage := &p.stages[len(p.stages)-1]
		p.wg.Add(1)
		go p.forwardOutput(lastStage.outputChan)
	}
}

// runStageWorker() and forwardOutput() methods are defined in stage.go

// Input returns the channel callers feed pipeline data into. Close it
// once no more data will be sent.
func (p *Pipeline) Input() chan<- any {
	return p.inputChan
}

// Output returns the channel of results produced by the last stage. It
// is closed once the pipeline has finished processing everything sent
// on Input, after Stop is called.
func (p *Pipeline) Output() <-chan any {
	return p.outputChan
}

// Errors returns the channel of errors reported by stages. It is
// closed when Stop is called.
func (p *Pipeline) Errors() <-chan error {
	return p.errorChan
}

// Stop gracefully shuts down the pipeline.
//
// Closes input channel and waits for all stages to complete. Stage
// output channels are closed automatically by their respective
// workers. Should be called when no more data will be sent.
//
// If the caller keeps reading Output() until it closes, which is the
// expected usage, Stop() waits for every buffered item to reach the
// caller no matter how long that takes. The stopGracePeriod escape
// hatch exists purely so Stop() cannot hang forever if the caller
// stopped draining Output() (or a stage's downstream neighbor stopped
// draining its input) before calling Stop(); without it, forwardOutput
// or runStageWorker blocked on a full, undrained channel would keep
// p.wg from ever reaching zero. A StageFunc that takes longer than
// stopGracePeriod to process a single item looks the same as a wedge
// from the outside, so pipelines built from stages that slow can still
// lose in-flight items on Stop(). Safe to call multiple times; only
// the first call has effect.
func (p *Pipeline) Stop() {
	p.stopOnce.Do(func() {
		// Close input channel to signal first stage to exit
		close(p.inputChan)

		// Stage output channels are closed automatically by per-stage
		// goroutines when all workers for that stage complete. This
		// cascade triggers subsequent stages to exit.
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		if !p.awaitDrain(done) {
			// Something is still blocked sending to a channel nobody
			// is draining. Unblock it and wait for the rest.
			close(p.stopping)
			<-done
		}

		// Stage output channels are already closed by now via the
		// goroutines started in Start(); only the pipeline-level
		// channels remain.
		close(p.outputChan)
		close(p.errorChan)
	})
}

// awaitDrain waits for done, reporting whether the pipeline drained on
// its own. It gives up once a full stopGracePeriod passes without a
// single item moving anywhere, which is what a caller who stopped
// reading Output() looks like from here.
//
// The window has to be measured against that progress rather than
// against the moment Stop() was called: a consumer that drains
// steadily but takes longer than stopGracePeriod in total is doing
// exactly what backpressure asks of it, and a fixed deadline would
// expire mid-drain and discard every item still in flight.
func (p *Pipeline) awaitDrain(done <-chan struct{}) bool {
	timer := time.NewTimer(stopGracePeriod)
	defer timer.Stop()

	last := atomic.LoadUint64(&p.progress)

	for {
		select {
		case <-done:
			return true

		case <-timer.C:
			current := atomic.LoadUint64(&p.progress)
			if current == last {
				return false
			}
			last = current
			timer.Reset(stopGracePeriod)
		}
	}
}

// DroppedErrors returns the number of stage errors discarded because
// errorChan was full when a stage tried to report them. It is safe to
// call concurrently with a running pipeline. A non-zero value means
// Errors() is not being drained quickly enough relative to the rate at
// which stages fail; increase errorChan's consumption rate or
// investigate why stages are failing so often.
func (p *Pipeline) DroppedErrors() uint64 {
	return atomic.LoadUint64(&p.droppedErrors)
}

// PoolManager returns the pipeline's pool manager, which stages can
// use to get and put pooled objects for reduced allocation overhead.
func (p *Pipeline) PoolManager() *concurrency.PoolManager {
	return p.poolManager
}
