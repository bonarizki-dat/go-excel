package concurrency

import "context"

// SetContextForTest overrides the pool's context directly, without
// starting any worker goroutines. It exists so white-box tests in
// other internal packages (e.g. stream) can force Submit's ctx.Done()
// branch deterministically, which Start's normal path cannot do
// without also racing a real worker draining taskChan. Since this
// package is internal, this method carries no public-API weight: it
// is unreachable from outside this module.
func (wp *WorkerPool) SetContextForTest(ctx context.Context) {
	wp.ctx = ctx
}

// TaskChanForTest exposes the pool's internal task channel so tests in
// other internal packages can fill or drain it directly, for
// scenarios (like a deterministically full channel) that the public
// Submit/Results API cannot set up on its own. See SetContextForTest.
func (wp *WorkerPool) TaskChanForTest() chan Task {
	return wp.taskChan
}

// ResultChanForTest exposes the pool's internal result channel so
// tests in other internal packages can inject or close results
// without running real workers, e.g. to test a consumer's handling of
// Results() closing with a gap in the sequence. See
// SetContextForTest.
func (wp *WorkerPool) ResultChanForTest() chan Result {
	return wp.resultChan
}
