package pipeline

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// TestPipeline_ContextCancellation tests pipeline behavior when context is canceled.
func TestPipeline_ContextCancellation(t *testing.T) {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	p := New(ctx)

	// Track how many items were processed
	var processed int32

	// Add slow stage
	p.AddStage("slow", func(_ context.Context, data any) (any, error) {
		// Increment processed counter
		atomic.AddInt32(&processed, 1)

		// Simulate slow processing
		time.Sleep(100 * time.Millisecond)
		return data, nil
	}, 1)

	// Start pipeline
	p.Start()

	// Send data in goroutine
	go func() {
		for i := range 100 {
			select {
			case p.Input() <- i:
				// Data sent
			case <-ctx.Done():
				// Context canceled
				return
			}
		}
	}()

	// Let some processing happen
	time.Sleep(250 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait a bit for cancellation to propagate
	time.Sleep(200 * time.Millisecond)

	// Verify not all items were processed (due to cancellation)
	processedCount := atomic.LoadInt32(&processed)
	if processedCount >= 100 {
		t.Errorf("Expected < 100 items processed due to cancellation, got %d", processedCount)
	}

	// Verify some items were processed before cancellation
	if processedCount == 0 {
		t.Error("Expected some items to be processed before cancellation")
	}
}

// TestPipeline_ConcurrentProcessing tests concurrent stage execution.
func TestPipeline_ConcurrentProcessing(t *testing.T) {
	// Create pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := New(ctx)

	// Track concurrent executions
	var activeCount int32
	var maxConcurrent int32

	// Add stage with concurrency = 5
	p.AddStage("concurrent", func(_ context.Context, data any) (any, error) {
		// Increment active count
		current := atomic.AddInt32(&activeCount, 1)

		// Update max concurrent if needed
		for {
			curMax := atomic.LoadInt32(&maxConcurrent)
			if current <= curMax || atomic.CompareAndSwapInt32(&maxConcurrent, curMax, current) {
				break
			}
		}

		// Simulate work
		time.Sleep(50 * time.Millisecond)

		// Decrement active count
		atomic.AddInt32(&activeCount, -1)

		return data, nil
	}, 5) // 5 concurrent processors

	// Start pipeline
	p.Start()

	// Send 20 items
	go func() {
		for i := range 20 {
			p.Input() <- i
		}
		// Use Stop() to properly close all channels
		p.Stop()
	}()

	// Drain output
	for range p.Output() { //nolint:revive // intentionally empty: just draining the channel
	}

	// Verify we achieved concurrency > 1
	maxReached := atomic.LoadInt32(&maxConcurrent)
	if maxReached < 2 {
		t.Errorf("Expected concurrent processing (max >= 2), got max = %d", maxReached)
	}

	// Verify we didn't exceed concurrency limit of 5
	if maxReached > 5 {
		t.Errorf("Expected max concurrency <= 5, got %d", maxReached)
	}
}
