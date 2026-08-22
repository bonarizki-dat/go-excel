package concurrency

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// stopGracePeriod bounds how long Stop() waits for workers to finish
// draining naturally before assuming nobody is reading Results() and
// forcing stuck senders to give up. A consumer that keeps draining until
// Results() closes, the normal usage pattern, always finishes well within
// this window, since each send is CPU-only work with no external I/O.
const stopGracePeriod = 250 * time.Millisecond

// WorkerPool manages a pool of worker goroutines for concurrent processing.
//
// It distributes tasks across multiple workers for parallel execution,
// utilizing all available CPU cores by default. Supports graceful shutdown,
// context cancellation, and error propagation.
//
// Usage:
//
//	pool := NewWorkerPool(runtime.NumCPU(), processFunc)
//	pool.Start(ctx)
//	pool.Submit(task)
//	pool.Stop()
type WorkerPool struct {
	// numWorkers is the number of concurrent worker goroutines.
	// Default: runtime.NumCPU() for optimal CPU utilization.
	numWorkers int

	// taskChan receives tasks to be processed by workers. Buffered to
	// reduce contention and improve throughput.
	taskChan chan Task

	// resultChan sends processing results back to caller
	// Buffered to prevent workers from blocking on send.
	// Errors are carried on Result.Err rather than a separate channel,
	// since a task's error and its (absent) data are inseparable.
	resultChan chan Result

	// wg tracks active worker goroutines for graceful shutdown
	wg sync.WaitGroup

	// ctx is the caller-supplied context for cancellation. Workers check
	// it to abort early on genuine cancellation (Submit, processTask, and
	// the receive fallback wait). It is never touched by Stop(), so a
	// graceful shutdown cannot be mistaken for caller cancellation.
	ctx context.Context

	// processFunc is the function workers execute on each task
	processFunc ProcessFunc

	// stats tracks worker pool statistics for monitoring
	stats WorkerStats

	// once ensures Stop() is only called once
	once sync.Once

	// stopping is closed by Stop() to unblock a worker stuck sending a
	// result that nobody is draining from Results(). It is deliberately
	// separate from ctx: honoring it in processTask's cancellation check
	// would wrongly fail tasks that were legitimately drained from
	// taskChan during a normal, error-free shutdown.
	stopping chan struct{}
}

// Types defined in types.go:
//   - Task: Unit of work to be processed
//   - Result: Outcome of task processing
//   - ProcessFunc: Function signature for task processing
//   - WorkerStats: Statistics tracking for worker pool
//   - WorkerError: Error type for worker failures

// NewWorkerPool creates a worker pool that will run numWorkers
// goroutines, each executing processFunc on submitted tasks. If
// numWorkers is 0 or negative, it defaults to runtime.NumCPU(). The
// pool is configured but not started; call Start to begin processing.
func NewWorkerPool(numWorkers int, processFunc ProcessFunc) *WorkerPool {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	return &WorkerPool{
		numWorkers: numWorkers,
		// 2x workers gives some slack so a burst of Submit calls does
		// not immediately block on a full channel.
		taskChan: make(chan Task, numWorkers*2),
		// One slot per worker is enough that a worker's send never
		// blocks waiting for another worker's send to be drained.
		resultChan: make(chan Result, numWorkers),
		// No cancellation available until Start() is given a real context.
		ctx:         context.Background(),
		processFunc: processFunc,
		stats: WorkerStats{
			TotalWorkers: numWorkers,
		},
		stopping: make(chan struct{}),
	}
}

// Start launches all worker goroutines and returns immediately; workers
// wait for tasks on the task channel in the background. If ctx is
// non-nil, it replaces the pool's internal background context, so
// canceling it aborts in-flight and future task processing.
func (wp *WorkerPool) Start(ctx context.Context) {
	if ctx != nil {
		wp.ctx = ctx
	}

	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// worker() and processTask() methods are defined in process.go

// Submit enqueues task for processing, blocking until a worker has
// capacity to accept it. It returns the pool's context error if the
// context is canceled before the task is accepted.
func (wp *WorkerPool) Submit(task Task) error {
	atomic.AddUint64(&wp.stats.TasksSubmitted, 1)

	select {
	case wp.taskChan <- task:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}

// Results returns the channel of processed results. The caller must
// drain it to receive results and to let workers make progress; it is
// closed once Stop has been called and all workers have exited.
func (wp *WorkerPool) Results() <-chan Result {
	return wp.resultChan
}

// Stop gracefully shuts down the worker pool.
//
// Closes the task channel and waits for all workers to drain any
// already-buffered tasks and exit. Should be called when no more tasks
// will be submitted. Safe to call multiple times (only first call has
// effect).
//
// If the caller keeps reading Results() until it closes, which is the
// expected usage, every worker finishes on its own well within
// stopGracePeriod and stopping is never closed early enough to matter.
// It exists purely so Stop() cannot hang forever if the caller stopped
// draining Results() before calling Stop(). Using a separate signal
// instead of canceling ctx keeps this from being confused with caller
// cancellation: see worker() and handleTask() for how the two are kept
// apart.
func (wp *WorkerPool) Stop() {
	wp.once.Do(func() {
		// Close task channel: workers will drain remaining buffered
		// tasks, then exit via the closed-channel branch.
		close(wp.taskChan)

		done := make(chan struct{})
		go func() {
			wp.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// All workers exited on their own; nothing was stuck.
		case <-time.After(stopGracePeriod):
			// Some worker is still blocked sending a result that
			// nobody is draining. Unblock it and wait for the rest.
			close(wp.stopping)
			<-done
		}

		// Close the result channel now that no worker can send on it.
		close(wp.resultChan)
	})
}

// Stats returns a snapshot of the pool's current statistics. It is
// safe to call concurrently with Submit and the running workers.
func (wp *WorkerPool) Stats() WorkerStats {
	return WorkerStats{
		TasksSubmitted: atomic.LoadUint64(&wp.stats.TasksSubmitted),
		TasksCompleted: atomic.LoadUint64(&wp.stats.TasksCompleted),
		TasksFailed:    atomic.LoadUint64(&wp.stats.TasksFailed),
		TotalWorkers:   wp.stats.TotalWorkers,
	}
}

// WorkerError is defined in types.go
