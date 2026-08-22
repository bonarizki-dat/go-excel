package concurrency

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewWorkerPool tests worker pool creation.
func TestNewWorkerPool(t *testing.T) {
	// Simple process function for testing
	processFunc := func(_ context.Context, task Task) (any, error) {
		return task.Data, nil
	}

	// Test with specific worker count
	t.Run("WithSpecificWorkerCount", func(t *testing.T) {
		wp := NewWorkerPool(4, processFunc)
		if wp == nil {
			t.Fatal("NewWorkerPool returned nil")
		}
		if wp.numWorkers != 4 {
			t.Errorf("numWorkers is %d, expected 4", wp.numWorkers)
		}
		wp.Stop() // Clean up
	})

	// Test with zero workers (should default to CPU count)
	t.Run("WithZeroWorkers", func(t *testing.T) {
		wp := NewWorkerPool(0, processFunc)
		if wp == nil {
			t.Fatal("NewWorkerPool returned nil")
		}
		expected := runtime.NumCPU()
		if wp.numWorkers != expected {
			t.Errorf("numWorkers is %d, expected %d", wp.numWorkers, expected)
		}
		wp.Stop() // Clean up
	})

	// Test with negative workers (should default to CPU count)
	t.Run("WithNegativeWorkers", func(t *testing.T) {
		wp := NewWorkerPool(-1, processFunc)
		if wp == nil {
			t.Fatal("NewWorkerPool returned nil")
		}
		expected := runtime.NumCPU()
		if wp.numWorkers != expected {
			t.Errorf("numWorkers is %d, expected %d", wp.numWorkers, expected)
		}
		wp.Stop() // Clean up
	})
}

// TestWorkerPool_Start tests starting worker pool.
func TestWorkerPool_Start(t *testing.T) {
	// Create process function
	processFunc := func(_ context.Context, task Task) (any, error) {
		return task.Data, nil
	}

	// Create worker pool
	wp := NewWorkerPool(2, processFunc)
	defer wp.Stop()

	// Start workers
	wp.Start(context.TODO())

	// Give workers time to start
	time.Sleep(10 * time.Millisecond)

	// Workers should be ready to process tasks
	// We verify by submitting a task
	err := wp.Submit(Task{ID: 1, Data: "test"})
	if err != nil {
		t.Errorf("Submit failed: %v", err)
	}
}

// TestWorkerPool_Submit tests task submission.
func TestWorkerPool_Submit(t *testing.T) {
	// Create process function
	var processedCount uint64
	processFunc := func(_ context.Context, task Task) (any, error) {
		atomic.AddUint64(&processedCount, 1)
		return task.Data, nil
	}

	// Create and start worker pool
	wp := NewWorkerPool(2, processFunc)
	wp.Start(context.TODO())
	defer wp.Stop()

	// Submit tasks
	for i := range 5 {
		err := wp.Submit(Task{ID: uint64(i), Data: i})
		if err != nil {
			t.Errorf("Submit task %d failed: %v", i, err)
		}
	}

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Check stats
	stats := wp.Stats()
	if stats.TasksSubmitted != 5 {
		t.Errorf("TasksSubmitted is %d, expected 5", stats.TasksSubmitted)
	}
}

// TestWorkerPool_Results tests receiving results from workers.
func TestWorkerPool_Results(t *testing.T) {
	// Create process function that doubles the input
	processFunc := func(_ context.Context, task Task) (any, error) {
		val := task.Data.(int)
		return val * 2, nil
	}

	// Create and start worker pool
	wp := NewWorkerPool(2, processFunc)
	wp.Start(context.TODO())

	// Submit tasks
	for i := 1; i <= 3; i++ {
		wp.Submit(Task{ID: uint64(i), Data: i})
	}

	// Collect results
	results := make(map[uint64]any)
	timeout := time.After(100 * time.Millisecond)
	resultCount := 0

	for resultCount < 3 {
		select {
		case result := <-wp.Results():
			results[result.TaskID] = result.Data
			resultCount++
		case <-timeout:
			t.Fatal("Timeout waiting for results")
		}
	}

	// Stop pool
	wp.Stop()

	// Verify results
	if len(results) != 3 {
		t.Errorf("Got %d results, expected 3", len(results))
	}
	for id := uint64(1); id <= 3; id++ {
		val, ok := results[id]
		if !ok {
			t.Errorf("Missing result for task %d", id)
			continue
		}
		expected := int(id) * 2
		if val != expected {
			t.Errorf("Result for task %d is %v, expected %d", id, val, expected)
		}
	}
}

// TestWorkerPool_Errors tests error handling in workers.
func TestWorkerPool_Errors(t *testing.T) {
	// Create process function that returns error for odd numbers
	processFunc := func(_ context.Context, task Task) (any, error) {
		val := task.Data.(int)
		if val%2 != 0 {
			return nil, errors.New("odd number")
		}
		return val, nil
	}

	// Create and start worker pool
	wp := NewWorkerPool(2, processFunc)
	wp.Start(context.TODO())

	// Submit mixed tasks
	wp.Submit(Task{ID: 1, Data: 1}) // Will error
	wp.Submit(Task{ID: 2, Data: 2}) // Will succeed
	wp.Submit(Task{ID: 3, Data: 3}) // Will error

	// Collect results and errors
	timeout := time.After(100 * time.Millisecond)
	errorCount := 0
	successCount := 0

	for errorCount+successCount < 3 {
		select {
		case result := <-wp.Results():
			if result.Err != nil {
				errorCount++
			} else {
				successCount++
			}
		case <-timeout:
			t.Fatalf("Timeout: got %d errors, %d successes", errorCount, successCount)
		}
	}

	// Stop pool
	wp.Stop()

	// Verify counts
	if errorCount != 2 {
		t.Errorf("Got %d errors, expected 2", errorCount)
	}
	if successCount != 1 {
		t.Errorf("Got %d successes, expected 1", successCount)
	}
}

// TestWorkerPool_Stop tests graceful shutdown.
func TestWorkerPool_Stop(t *testing.T) {
	// Create process function
	processFunc := func(_ context.Context, task Task) (any, error) {
		return task.Data, nil
	}

	// Create and start worker pool
	wp := NewWorkerPool(2, processFunc)
	wp.Start(context.TODO())

	// Submit some tasks
	for i := range 3 {
		wp.Submit(Task{ID: uint64(i), Data: i})
	}

	// Drain results before stopping
	for range 3 {
		<-wp.Results()
	}

	// Stop pool
	wp.Stop()

	// After Stop(), channel should be closed
	select {
	case _, ok := <-wp.Results():
		if ok {
			t.Error("Results channel should be closed after Stop()")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout checking if channel is closed")
	}
}

// Additional worker tests are in worker_advanced_test.go.
