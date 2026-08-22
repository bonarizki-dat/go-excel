package pipeline

import (
	"context"
	"testing"
	"time"
)

// TestPipeline_Stop tests graceful pipeline shutdown.
func TestPipeline_Stop(t *testing.T) {
	// Note: Deadlock bug has been fixed!
	// Each stage now closes its output channel automatically when all workers complete.

	// Create pipeline
	ctx := context.Background()
	p := New(ctx)

	// Add simple stage
	p.AddStage("passthrough", func(_ context.Context, data any) (any, error) {
		return data, nil
	}, 1)

	// Start pipeline
	p.Start()

	// Drain output in background
	var processed []int
	done := make(chan bool)
	go func() {
		for result := range p.Output() {
			processed = append(processed, result.(int))
		}
		done <- true
	}()

	// Send some data
	p.Input() <- 1
	p.Input() <- 2
	p.Input() <- 3

	// Stop pipeline (closes input and waits)
	p.Stop()

	// Wait for output
	<-done

	// Verify items processed
	if len(processed) != 3 {
		t.Errorf("Expected 3 items, got %d", len(processed))
	}
}

// TestPipeline_EmptyPipeline tests pipeline with no stages.
func TestPipeline_EmptyPipeline(t *testing.T) {
	// Create pipeline with no stages
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	p := New(ctx)

	// Start empty pipeline (should not panic)
	p.Start()

	// Send data
	go func() {
		p.Input() <- 1
		p.Input() <- 2
		close(p.inputChan)
	}()

	// Try to read output (should be empty since no stages)
	timeout := time.After(1 * time.Second)
	select {
	case <-p.Output():
		t.Error("Expected no output from empty pipeline")
	case <-timeout:
		// Expected - no output
	}

	// Stop pipeline
	p.wg.Wait()
}

// TestPipeline_BackpressureHandling tests backpressure with slow consumer.
func TestPipeline_BackpressureHandling(t *testing.T) {
	// Deadlock bug has been fixed!

	// Create pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p := New(ctx)

	// Add fast stage
	p.AddStage("fast", func(_ context.Context, data any) (any, error) {
		return data, nil
	}, 1)

	// Start pipeline
	p.Start()

	// Send many items quickly
	itemCount := 5000
	go func() {
		for i := range itemCount {
			p.Input() <- i
		}
		// Use Stop() to close input and wait for completion
		// This will properly close all channels
		p.Stop()
	}()

	// Slow consumer
	time.Sleep(100 * time.Millisecond) // Let buffer fill

	// Start consuming
	count := 0
	for range p.Output() {
		count++
		// Slow consumption
		if count%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Verify all items were received despite slow consumption
	if count != itemCount {
		t.Errorf("Expected %d items, got %d", itemCount, count)
	}
}

// TestPipeline_Stop_WaitsOutConsumerSlowerThanGracePeriod pins down the
// distinction Stop() has to make between a consumer that is slow and
// one that has stopped reading: this consumer takes several times
// stopGracePeriod to finish draining, but never pauses for a whole
// grace period at once, so Stop() owes it every item.
//
// Regression test: while the grace period ran as a fixed deadline from
// the moment Stop() was called, it expired mid-drain and every item
// still buffered was discarded.
func TestPipeline_Stop_WaitsOutConsumerSlowerThanGracePeriod(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	p := New(ctx)
	p.AddStage("passthrough", func(_ context.Context, data any) (any, error) {
		return data, nil
	}, 1)
	p.Start()

	// Enough items to overflow every buffer, so a large batch is still
	// in flight by the time Stop() runs.
	itemCount := 5 * p.backpressureSize
	go func() {
		for i := range itemCount {
			p.Input() <- i
		}
		p.Stop()
	}()

	// Pause for well under a grace period every 100 items. Draining
	// all of them takes far longer than stopGracePeriod in total.
	pause := stopGracePeriod / 5
	count := 0
	for range p.Output() {
		count++
		if count%100 == 0 {
			time.Sleep(pause)
		}
	}

	if count != itemCount {
		t.Errorf("drained %d of %d items; Stop() abandoned in-flight items instead of waiting for a slow but active consumer", count, itemCount)
	}
}
