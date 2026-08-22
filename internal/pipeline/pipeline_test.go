package pipeline

import (
	"context"
	"testing"
)

// TestNew tests pipeline creation.
func TestNew(t *testing.T) {
	// Create context for pipeline
	ctx := context.Background()

	// Create new pipeline
	p := New(ctx)

	// Verify pipeline was created
	if p == nil {
		t.Fatal("New returned nil")
	}

	// Verify context is set
	if p.ctx != ctx {
		t.Error("Pipeline context not set correctly")
	}

	// Verify stages are initialized
	if p.stages == nil {
		t.Error("Pipeline stages not initialized")
	}

	// Verify stages are empty initially
	if len(p.stages) != 0 {
		t.Errorf("Expected 0 stages, got %d", len(p.stages))
	}

	// Verify input channel is created
	if p.inputChan == nil {
		t.Error("Pipeline input channel not created")
	}

	// Verify output channel is created
	if p.outputChan == nil {
		t.Error("Pipeline output channel not created")
	}

	// Verify error channel is created
	if p.errorChan == nil {
		t.Error("Pipeline error channel not created")
	}

	// Verify pool manager is created
	if p.poolManager == nil {
		t.Error("Pipeline pool manager not created")
	}

	// Verify backpressure size is set
	expectedBackpressure := 1000
	if p.backpressureSize != expectedBackpressure {
		t.Errorf("Expected backpressure size %d, got %d", expectedBackpressure, p.backpressureSize)
	}
}

// TestPipeline_AddStage tests adding stages to pipeline.
func TestPipeline_AddStage(t *testing.T) {
	// Create pipeline
	ctx := context.Background()
	p := New(ctx)

	// Define simple stage function
	stageFunc := func(_ context.Context, data any) (any, error) {
		return data, nil
	}

	// Test adding first stage
	p.AddStage("stage1", stageFunc, 1)

	// Verify stage was added
	if len(p.stages) != 1 {
		t.Fatalf("Expected 1 stage, got %d", len(p.stages))
	}

	// Verify first stage properties
	stage1 := p.stages[0]
	if stage1.name != "stage1" {
		t.Errorf("Expected stage name 'stage1', got '%s'", stage1.name)
	}
	if stage1.concurrency != 1 {
		t.Errorf("Expected concurrency 1, got %d", stage1.concurrency)
	}

	// Verify first stage uses pipeline input
	if stage1.inputChan != p.inputChan {
		t.Error("First stage should use pipeline input channel")
	}

	// Test adding second stage
	p.AddStage("stage2", stageFunc, 2)

	// Verify both stages exist
	if len(p.stages) != 2 {
		t.Fatalf("Expected 2 stages, got %d", len(p.stages))
	}

	// Verify second stage uses first stage's output
	stage2 := p.stages[1]
	if stage2.inputChan != stage1.outputChan {
		t.Error("Second stage should use first stage's output channel")
	}

	// Verify second stage concurrency
	if stage2.concurrency != 2 {
		t.Errorf("Expected concurrency 2, got %d", stage2.concurrency)
	}
}

// TestPipeline_AddStageWithZeroConcurrency tests adding stage with zero concurrency.
func TestPipeline_AddStageWithZeroConcurrency(t *testing.T) {
	// Create pipeline
	ctx := context.Background()
	p := New(ctx)

	// Define stage function
	stageFunc := func(_ context.Context, data any) (any, error) {
		return data, nil
	}

	// Add stage with zero concurrency (should default to 1)
	p.AddStage("stage", stageFunc, 0)

	// Verify concurrency was set to 1
	if p.stages[0].concurrency != 1 {
		t.Errorf("Expected default concurrency 1, got %d", p.stages[0].concurrency)
	}
}

// TestPipeline_AddStageWithNegativeConcurrency tests adding stage with negative concurrency.
func TestPipeline_AddStageWithNegativeConcurrency(t *testing.T) {
	// Create pipeline
	ctx := context.Background()
	p := New(ctx)

	// Define stage function
	stageFunc := func(_ context.Context, data any) (any, error) {
		return data, nil
	}

	// Add stage with negative concurrency (should default to 1)
	p.AddStage("stage", stageFunc, -5)

	// Verify concurrency was set to 1
	if p.stages[0].concurrency != 1 {
		t.Errorf("Expected default concurrency 1, got %d", p.stages[0].concurrency)
	}
}

// TestPipeline_ChannelAccessors tests Input, Output, Errors channel accessors.
func TestPipeline_ChannelAccessors(t *testing.T) {
	// Create pipeline
	ctx := context.Background()
	p := New(ctx)

	// Test Input() accessor
	inputChan := p.Input()
	if inputChan == nil {
		t.Error("Input() returned nil")
	}

	// Test Output() accessor
	outputChan := p.Output()
	if outputChan == nil {
		t.Error("Output() returned nil")
	}

	// Test Errors() accessor
	errorChan := p.Errors()
	if errorChan == nil {
		t.Error("Errors() returned nil")
	}
}

// TestPipeline_PoolManager tests PoolManager accessor.
func TestPipeline_PoolManager(t *testing.T) {
	// Create pipeline
	ctx := context.Background()
	p := New(ctx)

	// Test PoolManager() accessor
	pm := p.PoolManager()
	if pm == nil {
		t.Error("PoolManager() returned nil")
	}

	// Verify it's the same instance
	if pm != p.poolManager {
		t.Error("PoolManager() returned different instance")
	}
}
