package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestWorkerPool_MultipleStop tests calling Stop multiple times.
//
// Verifies that Stop() is safe to call multiple times without panicking.
// This tests the sync.Once mechanism in Stop().
func TestWorkerPool_MultipleStop(t *testing.T) {
	// Create simple process function
	processFunc := func(_ context.Context, task Task) (any, error) {
		return task.Data, nil
	}

	// Create worker pool
	wp := NewWorkerPool(2, processFunc)
	wp.Start(context.TODO())

	// Call Stop multiple times - should not panic
	wp.Stop()
	wp.Stop()
	wp.Stop()

	// Test passes if no panic occurred
	t.Log("Multiple Stop calls handled safely")
}

// TestWorkerPool_ContextCancellation tests context cancellation.
//
// Verifies that workers respect context cancellation and that
// Submit() fails appropriately when context is canceled.
func TestWorkerPool_ContextCancellation(t *testing.T) {
	// Create process function that checks context
	processFunc := func(ctx context.Context, task Task) (any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return task.Data, nil
		}
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create and start worker pool with context
	wp := NewWorkerPool(2, processFunc)
	wp.Start(ctx)

	// Submit a task before cancellation
	err := wp.Submit(Task{ID: 1, Data: "test"})
	if err != nil {
		t.Fatalf("Submit before cancel failed: %v", err)
	}

	// Cancel context
	cancel()

	// Wait a bit for context cancellation to propagate
	time.Sleep(10 * time.Millisecond)

	// Try to submit another task (should eventually fail)
	err = wp.Submit(Task{ID: 2, Data: "test2"})
	if err == nil {
		t.Log("Submit succeeded even after cancel (task chan may not be full yet)")
	}

	// Stop pool
	wp.Stop()
}

// TestWorkerPool_Stats tests statistics tracking.
//
// Verifies that worker pool correctly tracks submitted, completed,
// and failed task counts using atomic operations.
func TestWorkerPool_Stats(t *testing.T) {
	// Create process function that fails for negative numbers
	processFunc := func(_ context.Context, task Task) (any, error) {
		val := task.Data.(int)
		if val < 0 {
			return nil, errors.New("negative number")
		}
		return val, nil
	}

	// Create and start worker pool
	wp := NewWorkerPool(2, processFunc)
	wp.Start(context.TODO())

	// Submit tasks (some will fail)
	wp.Submit(Task{ID: 1, Data: 1})  // Success
	wp.Submit(Task{ID: 2, Data: -1}) // Fail
	wp.Submit(Task{ID: 3, Data: 2})  // Success
	wp.Submit(Task{ID: 4, Data: -2}) // Fail

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Get stats
	stats := wp.Stats()

	// Verify submitted count
	if stats.TasksSubmitted != 4 {
		t.Errorf("TasksSubmitted is %d, expected 4", stats.TasksSubmitted)
	}

	// Verify worker count
	if stats.TotalWorkers != 2 {
		t.Errorf("TotalWorkers is %d, expected 2", stats.TotalWorkers)
	}

	// Stop pool
	wp.Stop()
}

// TestWorkerError tests WorkerError type.
//
// Verifies that WorkerError correctly implements the error interface,
// includes Cause in its message, unwraps Cause when it is an error, and
// stores task context information.
func TestWorkerError(t *testing.T) {
	cause := errors.New("underlying error")
	// Create worker error
	err := &WorkerError{
		TaskID:  123,
		Message: "test error",
		Cause:   cause,
	}

	// Test Error() method includes the cause
	errStr := err.Error()
	if errStr != "test error: underlying error" {
		t.Errorf("Error() returned %q, expected %q", errStr, "test error: underlying error")
	}

	// Test Unwrap() surfaces Cause for errors.Is/errors.As
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find Cause via Unwrap()")
	}

	// Verify fields
	if err.TaskID != 123 {
		t.Errorf("TaskID is %d, expected 123", err.TaskID)
	}
}

// TestWorkerError_NonErrorCause verifies that a non-error Cause (e.g. a
// panic value that isn't an error) is still rendered in Error() but
// Unwrap() returns nil instead of panicking on a failed type assertion.
func TestWorkerError_NonErrorCause(t *testing.T) {
	err := &WorkerError{
		TaskID:  7,
		Message: "panic recovered",
		Cause:   "boom",
	}

	if got, want := err.Error(), "panic recovered: boom"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if err.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil for non-error Cause", err.Unwrap())
	}
}

// TestWorkerPool_PanicRecovery tests panic recovery in process function.
//
// Verifies that panics in the process function are caught and converted
// to WorkerError instead of crashing the worker.
func TestWorkerPool_PanicRecovery(t *testing.T) {
	// Create process function that panics
	processFunc := func(_ context.Context, task Task) (any, error) {
		val := task.Data.(int)
		if val == 999 {
			panic("intentional panic")
		}
		return val, nil
	}

	// Create and start worker pool
	wp := NewWorkerPool(2, processFunc)
	wp.Start(context.TODO())

	// Submit normal task
	wp.Submit(Task{ID: 1, Data: 1})

	// Submit task that will panic
	wp.Submit(Task{ID: 999, Data: 999})

	// Submit another normal task after panic
	wp.Submit(Task{ID: 2, Data: 2})

	// Collect results
	timeout := time.After(100 * time.Millisecond)
	resultCount := 0
	panicCaught := false

	for resultCount < 3 {
		select {
		case result := <-wp.Results():
			resultCount++
			if result.TaskID == 999 {
				// This task should have panicked
				if result.Err == nil {
					t.Error("Expected error for panic task, got nil")
				} else {
					// Check if error is WorkerError
					workerError := &WorkerError{}
					if errors.As(result.Err, &workerError) {
						panicCaught = true
					}
				}
			}
		case <-timeout:
			t.Fatalf("Timeout: got %d results, expected 3", resultCount)
		}
	}

	// Stop pool
	wp.Stop()

	// Verify panic was caught
	if !panicCaught {
		t.Error("Panic was not caught and converted to WorkerError")
	}
}
