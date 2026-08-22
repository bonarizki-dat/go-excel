package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRunStageWorker_ErrorSend_ContextCanceled exercises the
// ctx.Done() branch of runStageWorker's error-reporting select
// directly: errorChan is filled to capacity first (so the send case
// can never be ready), and the stage's own process function cancels
// ctx just before returning its error, so ctx.Done() is the only ready
// case by the time the select runs. droppedErrors staying unchanged
// (rather than incrementing) confirms the ctx.Done() branch ran
// instead of the already-covered default/drop branch.
func TestRunStageWorker_ErrorSend_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := New(ctx)

	for i := 0; i < cap(p.errorChan); i++ {
		p.errorChan <- errors.New("filler")
	}

	stage := &Stage{
		name: "cancel-and-error",
		process: func(context.Context, any) (any, error) {
			cancel()
			return nil, errors.New("boom")
		},
		inputChan:  make(chan any, 1),
		outputChan: make(chan any, 1),
	}
	stage.inputChan <- 1

	done := make(chan struct{})
	go func() {
		p.runStageWorker(stage)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runStageWorker did not return after ctx was canceled")
	}

	// droppedErrors is only incremented by the default (drop-and-count)
	// branch, never by the ctx.Done() branch; it staying at 0 confirms
	// the ctx.Done() branch ran instead of default.
	if got := p.DroppedErrors(); got != 0 {
		t.Errorf("DroppedErrors() = %d, want 0 (the ctx.Done() branch does not increment it)", got)
	}
}

// TestRunStageWorker_OuterSelect_ContextCanceled exercises
// runStageWorker's outer ctx.Done() case directly: inputChan is left
// empty (not closed) so it is never a ready case, unlike the other
// ctx-cancellation tests here where inputChan already has a buffered
// item and cancellation happens from within process(), which
// exercises the inner selects instead.
func TestRunStageWorker_OuterSelect_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := New(ctx)
	cancel()

	stage := &Stage{
		name:       "never-runs",
		process:    func(context.Context, any) (any, error) { return "unused", nil },
		inputChan:  make(chan any), // empty and never closed
		outputChan: make(chan any),
	}

	done := make(chan struct{})
	go func() {
		p.runStageWorker(stage)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runStageWorker did not return for an already-canceled ctx with no input")
	}
}

// TestRunStageWorker_OutputSend_ContextCanceled exercises the
// ctx.Done() branch of runStageWorker's output-forwarding select: the
// output channel is unbuffered with nobody reading it, so the send can
// never be ready, and process cancels ctx before returning
// successfully.
func TestRunStageWorker_OutputSend_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := New(ctx)

	stage := &Stage{
		name: "cancel-then-succeed",
		process: func(context.Context, any) (any, error) {
			cancel()
			return "result", nil
		},
		inputChan:  make(chan any, 1),
		outputChan: make(chan any), // unbuffered, nobody reads it
	}
	stage.inputChan <- 1

	done := make(chan struct{})
	go func() {
		p.runStageWorker(stage)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runStageWorker did not return after ctx was canceled while sending output")
	}
}

// TestRunStageWorker_OutputSend_StoppingClosed exercises the
// p.stopping branch of runStageWorker's output-forwarding select: the
// output channel is unbuffered with nobody reading it, and p.stopping
// is closed (as Stop() does after its grace period) instead of ctx.
func TestRunStageWorker_OutputSend_StoppingClosed(t *testing.T) {
	ctx := context.Background()
	p := New(ctx)
	close(p.stopping)

	stage := &Stage{
		name: "succeed-while-stopping",
		process: func(context.Context, any) (any, error) {
			return "result", nil
		},
		inputChan:  make(chan any, 1),
		outputChan: make(chan any), // unbuffered, nobody reads it
	}
	stage.inputChan <- 1

	done := make(chan struct{})
	go func() {
		p.runStageWorker(stage)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runStageWorker did not return once p.stopping was closed")
	}
}

// TestForwardOutput_ContextCanceled exercises forwardOutput's inner
// ctx.Done() branch specifically (the one guarding the send to
// p.outputChan, not the outer one guarding the receive from
// stageChan): ctx is canceled only after forwardOutput has already
// picked up the value from stageChan and is blocked trying to forward
// it to an unbuffered, undrained outputChan. Canceling ctx up front
// instead would race between the outer and inner selects (both ready
// at once, since stageChan already has a buffered value), sometimes
// returning via the outer select without ever reaching the inner one.
func TestForwardOutput_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := New(ctx)
	p.outputChan = make(chan any) // unbuffered, nobody reads it

	stageChan := make(chan any, 1)
	stageChan <- "value"

	p.wg.Add(1)
	done := make(chan struct{})
	go func() {
		p.forwardOutput(stageChan)
		close(done)
	}()

	// Give forwardOutput time to receive from stageChan and block on
	// the inner send before canceling, so the outer select's race
	// between ctx.Done() and stageChan cannot short-circuit this test.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("forwardOutput did not return after ctx was canceled while blocked forwarding")
	}
}

// TestPipeline_Stop_GracePeriodTimeout exercises Stop()'s own timeout
// branch: a stage worker left blocked sending to an unbuffered,
// undrained outputChan (because the caller never reads Output())
// cannot finish within stopGracePeriod on its own, so Stop() must
// fall through to closing p.stopping to force it to give up.
func TestPipeline_Stop_GracePeriodTimeout(t *testing.T) {
	ctx := context.Background()
	p := New(ctx)

	// Block forever in process() until stopping is closed, simulating
	// a slow/stuck stage that the grace period timeout must interrupt.
	p.AddStage("blockUntilStopping", func(_ context.Context, data any) (any, error) {
		<-p.stopping
		return data, nil
	}, 1)

	p.Start()
	p.Input() <- 1

	// Give the worker a moment to actually enter process() and start
	// blocking on <-p.stopping before Stop() is called, so Stop()'s
	// internal wait genuinely times out waiting for it instead of
	// racing to close inputChan before the worker even reads.
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	p.Stop()
	elapsed := time.Since(start)

	if elapsed < stopGracePeriod {
		t.Errorf("Stop() returned after %v, want at least the grace period (%v)", elapsed, stopGracePeriod)
	}
}

// TestForwardOutput_StoppingClosed exercises forwardOutput's
// p.stopping branch: the pipeline's public output channel is
// unbuffered with nobody reading it, and p.stopping is closed instead
// of ctx.
func TestForwardOutput_StoppingClosed(t *testing.T) {
	ctx := context.Background()
	p := New(ctx)
	p.outputChan = make(chan any) // unbuffered, nobody reads it
	close(p.stopping)

	stageChan := make(chan any, 1)
	stageChan <- "value"

	p.wg.Add(1)
	done := make(chan struct{})
	go func() {
		p.forwardOutput(stageChan)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("forwardOutput did not return once p.stopping was closed")
	}
}
