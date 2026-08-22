package pipeline

import (
	"context"
	"testing"
	"time"
)

// TestPipeline_SingleStageDataFlow tests data flowing through single stage.
func TestPipeline_SingleStageDataFlow(t *testing.T) {
	// Deadlock bug has been fixed!

	// Create pipeline with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p := New(ctx)

	// Add stage that doubles the input
	p.AddStage("double", func(_ context.Context, data any) (any, error) {
		// Convert data to int and double it
		num := data.(int)
		return num * 2, nil
	}, 1)

	// Start pipeline
	p.Start()

	// Send test data
	go func() {
		p.Input() <- 5
		p.Input() <- 10
		p.Input() <- 15
		// Use Stop() to properly close all channels
		p.Stop()
	}()

	// Collect results
	var results []int
	for result := range p.Output() {
		results = append(results, result.(int))
	}

	// Verify results
	expected := []int{10, 20, 30}
	if len(results) != len(expected) {
		t.Fatalf("Expected %d results, got %d", len(expected), len(results))
	}

	for i, exp := range expected {
		if results[i] != exp {
			t.Errorf("Result[%d]: expected %d, got %d", i, exp, results[i])
		}
	}
}

// TestPipeline_MultiStageDataFlow tests data flowing through multiple stages.
func TestPipeline_MultiStageDataFlow(t *testing.T) {
	// Deadlock bug has been fixed!

	// Create pipeline with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p := New(ctx)

	// Add stage 1: double the input
	p.AddStage("double", func(_ context.Context, data any) (any, error) {
		num := data.(int)
		return num * 2, nil
	}, 1)

	// Add stage 2: add 10
	p.AddStage("add10", func(_ context.Context, data any) (any, error) {
		num := data.(int)
		return num + 10, nil
	}, 1)

	// Add stage 3: convert to string
	p.AddStage("toString", func(_ context.Context, data any) (any, error) {
		num := data.(int)
		return string(rune(num)), nil
	}, 1)

	// Start pipeline
	p.Start()

	// Send test data
	go func() {
		p.Input() <- 5  // 5 -> 10 -> 20 -> chr(20)
		p.Input() <- 10 // 10 -> 20 -> 30 -> chr(30)
		// Use Stop() to properly close all channels
		p.Stop()
	}()

	// Collect results
	var results []any
	for result := range p.Output() {
		results = append(results, result)
	}

	// Verify results count
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
}
