package pipeline

import (
	"context"
	"sync/atomic"
)

// Stage represents a single processing stage in the pipeline.
//
// Each stage receives data from previous stage, processes it,
// and sends to next stage. Stages run concurrently.
type Stage struct {
	// name identifies this stage for logging and debugging
	name string

	// process transforms data; must be thread-safe since concurrency
	// concurrent workers may call it at once.
	process StageFunc

	// inputChan receives data from previous stage
	inputChan chan any

	// outputChan sends data to next stage
	outputChan chan any

	// concurrency is the number of worker goroutines running process
	// for this stage.
	concurrency int
}

// StageFunc processes a single data item within a pipeline stage,
// returning the transformed result or an error. A stage may run
// multiple StageFunc calls concurrently across its worker goroutines,
// so implementations must be safe for concurrent use.
type StageFunc func(ctx context.Context, data any) (any, error)

// runStageWorker runs one worker goroutine for stage: it reads items
// from stage.inputChan, applies stage.process, and forwards results to
// stage.outputChan until the input channel closes or the pipeline's
// context is canceled. The caller (Start) is responsible for the
// corresponding WaitGroup.Done call.
func (p *Pipeline) runStageWorker(stage *Stage) {
	// WaitGroup.Done() is called by the caller's defer in Start().
	for {
		select {
		case <-p.ctx.Done():
			return

		case data, ok := <-stage.inputChan:
			if !ok {
				return
			}

			result, err := stage.process(p.ctx, data)
			if err != nil {
				select {
				case p.errorChan <- err:
				case <-p.ctx.Done():
					return
				default:
					// errorChan is full and this send must stay
					// non-blocking (a blocking send here could
					// deadlock the whole pipeline if Errors() is
					// never drained), so the error is dropped. Count
					// it so DroppedErrors() makes the loss visible
					// instead of silent.
					atomic.AddUint64(&p.droppedErrors, 1)
				}
				continue
			}

			if !p.send(stage.outputChan, result) {
				return
			}
		}
	}
}

// send hands value to ch under normal backpressure: it blocks for as
// long as a downstream consumer keeps draining, however slowly. It
// reports false, discarding value, once the pipeline's context is
// canceled or Stop() has given up on a channel nobody is draining —
// the only two cases where blocking here would never end.
//
// Every successful hand-off bumps the progress counter Stop() uses to
// tell a draining pipeline from a wedged one; see awaitDrain.
func (p *Pipeline) send(ch chan<- any, value any) bool {
	select {
	case ch <- value:
		atomic.AddUint64(&p.progress, 1)
		return true
	case <-p.ctx.Done():
		return false
	case <-p.stopping:
		return false
	}
}

// forwardOutput copies results from the last stage's output channel,
// stageChan, to the pipeline's public Output channel. It runs in its
// own goroutine so the last stage is not directly coupled to how
// quickly the caller drains Output.
func (p *Pipeline) forwardOutput(stageChan chan any) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return

		case data, ok := <-stageChan:
			if !ok {
				return
			}

			if !p.send(p.outputChan, data) {
				return
			}
		}
	}
}
