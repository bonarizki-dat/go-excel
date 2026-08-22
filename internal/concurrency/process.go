package concurrency

import (
	"sync/atomic"
)

// worker is the main goroutine function for processing tasks.
//
// Each worker runs in its own goroutine, processing tasks from the task
// channel until the channel is closed or context is canceled.
//
// A graceful Stop() closes taskChan and cancels the context together, so
// a plain select between the two would race: a task already buffered in
// taskChan at that moment could be silently dropped if ctx.Done() happens
// to be chosen instead. To guarantee every buffered task is processed,
// each iteration first checks taskChan non-blockingly and only falls back
// to a cancellation-aware blocking select once no task is immediately
// available. Cancellation still interrupts a worker promptly whenever
// taskChan is empty, including while blocked sending a result that
// nobody is draining.
func (wp *WorkerPool) worker() {
	// Ensure WaitGroup is decremented when worker exits
	defer wp.wg.Done()

	for {
		select {
		case task, ok := <-wp.taskChan:
			if !ok {
				return
			}
			if !wp.handleTask(task) {
				wp.drainAbandoned()
				return
			}
			continue
		default:
			// No task immediately available; fall through to the
			// cancellation-aware wait below.
		}

		select {
		case <-wp.ctx.Done():
			return

		case task, ok := <-wp.taskChan:
			if !ok {
				return
			}
			if !wp.handleTask(task) {
				wp.drainAbandoned()
				return
			}
		}
	}
}

// drainAbandoned consumes whatever is currently buffered in taskChan
// once this worker has given up trying to deliver a result
// (handleTask returned false, meaning stopping was closed or ctx was
// canceled while sending). Those tasks were already committed to
// taskChan by Submit, which counted them in TasksSubmitted; without
// this, they would vanish with no matching TasksCompleted or
// TasksFailed, making the pool's own stats inconsistent and hiding
// the abandonment from callers who check TasksSubmitted against the
// other two. It counts them as failed rather than attempting to
// process them, since the condition that caused this worker to give
// up applies just as much to any task it would pick up next.
func (wp *WorkerPool) drainAbandoned() {
	for {
		select {
		case _, ok := <-wp.taskChan:
			if !ok {
				return
			}
			atomic.AddUint64(&wp.stats.TasksFailed, 1)
		default:
			return
		}
	}
}

// handleTask processes a single task and sends its result.
//
// Returns false if the pool is stopping or the context was canceled
// while sending the result, signaling the caller to exit; true otherwise.
//
// Sending uses a non-blocking-first priority: a result is only ever
// dropped if resultChan has no room at that moment, never merely because
// Stop() happened to run at the same time a send would otherwise have
// succeeded. The fallback wait watches stopping rather than ctx, so a
// graceful Stop() unblocks a stuck sender without that being confused
// with caller cancellation.
func (wp *WorkerPool) handleTask(task Task) bool {
	result, err := wp.processTask(task)

	res := Result{
		TaskID: task.ID,
		Data:   result,
		Err:    err,
	}

	if err != nil {
		atomic.AddUint64(&wp.stats.TasksFailed, 1)
	} else {
		atomic.AddUint64(&wp.stats.TasksCompleted, 1)
	}

	select {
	case wp.resultChan <- res:
		return true
	default:
		// No room right now; fall back to a wait that still gives up
		// once the pool is stopping or the caller cancels, so this
		// worker cannot hang forever.
	}

	select {
	case wp.resultChan <- res:
		return true
	case <-wp.stopping:
		return false
	case <-wp.ctx.Done():
		return false
	}
}

// processTask runs wp.processFunc on task, returning its result. It
// returns wp.ctx's error without calling processFunc if the context is
// already canceled, and recovers a panic from processFunc into a
// *WorkerError rather than letting it crash the worker goroutine.
func (wp *WorkerPool) processTask(task Task) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &WorkerError{
				TaskID:  task.ID,
				Message: "worker panic",
				Cause:   r,
			}
		}
	}()

	select {
	case <-wp.ctx.Done():
		return nil, wp.ctx.Err()
	default:
	}

	return wp.processFunc(wp.ctx, task)
}
