package concurrency

import (
	"context"
	"testing"
	"time"
)

// TestWorker_BlockingSelect_HandleTaskAbandoned exercises worker()'s
// second (blocking) select branch specifically: task 1 arrives only
// after the first non-blocking check already fell through to
// default, and resultChan is pre-filled so handleTask's own send for
// it blocks. While it is blocked, task 2 is buffered into taskChan;
// once stopping is closed, handleTask gives up on task 1 (counted as
// TasksCompleted, since processFunc itself succeeded) and worker()
// calls drainAbandoned, which drains task 2 (counted as TasksFailed,
// since it is abandoned without ever being processed) before hitting
// its own empty-channel default branch and returning.
func TestWorker_BlockingSelect_HandleTaskAbandoned(t *testing.T) {
	wp := NewWorkerPool(1, func(context.Context, Task) (any, error) {
		return "ok", nil
	})
	wp.ctx = context.Background()

	// Fill resultChan (capacity 1) so handleTask's sends never succeed.
	wp.resultChan <- Result{TaskID: 0}

	wp.wg.Add(1)
	workerDone := make(chan struct{})
	go func() {
		wp.worker()
		close(workerDone)
	}()

	// Give worker() time to reach its first select and fall through to
	// default (taskChan is still empty at this point), so task 1 is
	// only ever visible to the second, blocking select.
	time.Sleep(20 * time.Millisecond)
	wp.taskChan <- Task{ID: 1}

	// Give handleTask time to try (and fail) its non-blocking send for
	// task 1 and block on the fallback, then buffer task 2 for
	// drainAbandoned to find once stopping unblocks task 1's sender.
	time.Sleep(20 * time.Millisecond)
	wp.taskChan <- Task{ID: 2}
	close(wp.stopping)

	select {
	case <-workerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("worker() did not return after stopping was closed while blocked sending a result")
	}

	stats := wp.Stats()
	if stats.TasksCompleted != 1 {
		t.Errorf("TasksCompleted = %d, want 1 (task 1, processed but its result could not be sent)", stats.TasksCompleted)
	}
	if stats.TasksFailed != 1 {
		t.Errorf("TasksFailed = %d, want 1 (task 2, abandoned by drainAbandoned)", stats.TasksFailed)
	}
}

// TestDrainAbandoned_EmptyChannel verifies drainAbandoned returns
// immediately via its default branch when taskChan has nothing
// buffered, rather than only being exercised via the loop's other
// branches.
func TestDrainAbandoned_EmptyChannel(t *testing.T) {
	wp := NewWorkerPool(1, nil)

	done := make(chan struct{})
	go func() {
		wp.drainAbandoned()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("drainAbandoned did not return for an empty taskChan")
	}
}
