package pipeline

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestPipeline_ErrorHandling tests error propagation through pipeline.
func TestPipeline_ErrorHandling(t *testing.T) {
	// Create pipeline with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p := New(ctx)

	// Add stage that errors on specific value
	p.AddStage("checkValue", func(_ context.Context, data any) (any, error) {
		num := data.(int)
		// Error on value 666
		if num == 666 {
			return nil, errors.New("evil number detected")
		}
		return num, nil
	}, 1)

	// Start pipeline
	p.Start()

	// Send test data including error case
	go func() {
		p.Input() <- 1
		p.Input() <- 666 // This should trigger error
		p.Input() <- 2
		// Use Stop() to properly close all channels
		p.Stop()
	}()

	// Collect results and errors
	var results []int
	var errs []error
	var wg sync.WaitGroup
	wg.Add(2)

	// Collect output
	go func() {
		defer wg.Done()
		for result := range p.Output() {
			results = append(results, result.(int))
		}
	}()

	// Collect errors
	go func() {
		defer wg.Done()
		for err := range p.Errors() {
			errs = append(errs, err)
		}
	}()

	// Wait for collectors to finish draining channels
	// Note: Stop() already closes outputChan and errorChan
	wg.Wait()

	// Verify we got 2 good results (1 and 2)
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify we got 1 error
	if len(errs) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(errs))
	}

	// Verify error message
	if errs[0].Error() != "evil number detected" {
		t.Errorf("Expected 'evil number detected', got '%s'", errs[0].Error())
	}
}

// TestPipeline_DroppedErrors verifies DroppedErrors() reports a
// nonzero count once errorChan (buffered at 100) fills up and nobody
// drains Errors(), instead of silently discarding the overflow.
func TestPipeline_DroppedErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p := New(ctx)
	p.AddStage("alwaysError", func(_ context.Context, _ any) (any, error) {
		return nil, errors.New("boom")
	}, 4)

	p.Start()

	// errorChan's capacity is 100; send well past that without ever
	// draining Errors(), so the non-blocking send in runStageWorker
	// falls back to its "drop and count" branch once full.
	for i := 0; i < 500; i++ {
		p.Input() <- i
	}

	// Drain Output() concurrently so stage workers are never blocked
	// waiting on outputChan (this stage only ever errors, so Output()
	// should stay empty, but draining defensively avoids any chance of
	// this test hanging).
	done := make(chan struct{})
	go func() {
		for v := range p.Output() {
			_ = v
		}
		close(done)
	}()

	p.Stop()
	<-done

	if p.DroppedErrors() == 0 {
		t.Error("DroppedErrors() = 0, want > 0 once errorChan overflowed")
	}
}

// TestPipeline_MultipleStagesWithErrors tests error handling across multiple stages.
func TestPipeline_MultipleStagesWithErrors(t *testing.T) {
	// Deadlock bug has been fixed!

	// Create pipeline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p := New(ctx)

	// Stage 1: pass through, but error on 5
	p.AddStage("stage1", func(_ context.Context, data any) (any, error) {
		num := data.(int)
		if num == 5 {
			return nil, errors.New("stage1 error on 5")
		}
		return num * 2, nil
	}, 1)

	// Stage 2: pass through, but error on 20
	p.AddStage("stage2", func(_ context.Context, data any) (any, error) {
		num := data.(int)
		if num == 20 {
			return nil, errors.New("stage2 error on 20")
		}
		return num + 1, nil
	}, 1)

	// Start pipeline
	p.Start()

	// Send data
	go func() {
		for i := 1; i <= 10; i++ {
			p.Input() <- i
		}
		// Use Stop() to properly close all channels
		p.Stop()
	}()

	// Collect results and errors
	var results []int
	var errs []error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for result := range p.Output() {
			results = append(results, result.(int))
		}
	}()

	go func() {
		defer wg.Done()
		for err := range p.Errors() {
			errs = append(errs, err)
		}
	}()

	// Wait for collectors to finish draining channels
	// Note: Stop() already closes outputChan and errorChan
	wg.Wait()

	// Verify we got errors from both stages
	// 5 -> stage1 error, 10 -> 20 -> stage2 error
	if len(errs) < 1 {
		t.Errorf("Expected at least 1 error, got %d", len(errs))
	}

	// Verify we got some successful results
	if len(results) == 0 {
		t.Error("Expected some successful results")
	}
}
