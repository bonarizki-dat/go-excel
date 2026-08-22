package stream

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestImportStream_ProcessorError(t *testing.T) {
	// Create test file
	testData := [][]any{
		{1, "Alice"},
		{2, "Bob"},
		{3, "Charlie"},
	}
	buf, _ := createImporterTestFile(testData)

	// Create importer
	importer, err := NewStreamImporter(buf)
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// Processor that errors on row 2
	processor := func(_ context.Context, rowNum int, row []any) ([]any, error) {
		if rowNum == 2 {
			return nil, errors.New("processor error")
		}
		return row, nil
	}

	ctx := context.Background()
	err = importer.ImportStream(ctx, processor)

	// Verify error occurred
	if err == nil {
		t.Error("Expected error from processor, got nil")
	}
}

// TestImportStream_ContextCancellation tests canceling import.
func TestImportStream_ContextCancellation(t *testing.T) {
	// Create test file with many rows
	testData := make([][]any, 100)
	for i := range 100 {
		testData[i] = []any{i + 1, "data"}
	}
	buf, _ := createImporterTestFile(testData)

	// Create importer
	importer, err := NewStreamImporter(buf)
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Track processed rows
	var processedCount atomic.Uint64

	// Processor that cancels after 10 rows
	processor := func(_ context.Context, _ int, row []any) ([]any, error) {
		count := processedCount.Add(1)
		if count == 10 {
			cancel()
		}
		time.Sleep(10 * time.Millisecond) // Slow processing
		return row, nil
	}

	err = importer.ImportStream(ctx, processor)

	// Verify error is context.Canceled (may be wrapped)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}

	// Verify some rows were processed (less than 100)
	if processedCount.Load() >= 100 {
		t.Errorf("Expected fewer than 100 rows, got %d", processedCount.Load())
	}
}

// TestImportStream_EmptyFile tests importing empty file.
