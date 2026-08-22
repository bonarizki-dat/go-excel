package stream

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestImportStream_WithHeadings(t *testing.T) {
	// Create test file with headings
	testData := [][]any{
		{"ID", "Name", "Email"},
		{1, "Alice", "alice@test.com"},
		{2, "Bob", "bob@test.com"},
	}
	buf, _ := createImporterTestFile(testData)

	// Create importer with headings
	importer, err := NewStreamImporter(buf, WithImportHeadings())
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// Verify headings read
	headings := importer.GetHeadings()
	if len(headings) != 3 {
		t.Errorf("Expected 3 headings, got %d", len(headings))
	}

	// Track processed rows
	var processedCount atomic.Uint64

	// Process rows (should skip heading row)
	processor := func(_ context.Context, _ int, row []any) ([]any, error) {
		processedCount.Add(1)
		return row, nil
	}

	ctx := context.Background()
	err = importer.ImportStream(ctx, processor)

	if err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	// Should process 2 data rows (skip heading row)
	if processedCount.Load() != 2 {
		t.Errorf("Expected 2 data rows, got %d", processedCount.Load())
	}
}

// TestImportStream_WithImportRows tests importing specific row range.
func TestImportStream_WithImportRows(t *testing.T) {
	// Create test file with 10 rows
	testData := make([][]any, 10)
	for i := range 10 {
		testData[i] = []any{i + 1, "user"}
	}
	buf, _ := createImporterTestFile(testData)

	// Create importer with row range (rows 3-7)
	importer, err := NewStreamImporter(buf, WithImportRows(3, 7))
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// Track processed rows
	var processedCount atomic.Uint64

	processor := func(_ context.Context, _ int, row []any) ([]any, error) {
		processedCount.Add(1)
		return row, nil
	}

	ctx := context.Background()
	err = importer.ImportStream(ctx, processor)

	if err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	// Should process 5 rows (3, 4, 5, 6, 7)
	if processedCount.Load() != 5 {
		t.Errorf("Expected 5 rows in range, got %d", processedCount.Load())
	}
}

// TestImportStream_WithImportSkipEmptyRows tests skipping empty rows.
func TestImportStream_WithImportSkipEmptyRows(t *testing.T) {
	// Create test file with empty rows
	testData := [][]any{
		{1, "Alice"},
		{"", ""}, // Empty row
		{2, "Bob"},
		{"", ""}, // Empty row
		{3, "Charlie"},
	}
	buf, _ := createImporterTestFile(testData)

	// Create importer with skip empty rows
	importer, err := NewStreamImporter(buf, WithImportSkipEmptyRows(true))
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// Track processed rows
	var processedCount atomic.Uint64

	processor := func(_ context.Context, _ int, row []any) ([]any, error) {
		processedCount.Add(1)
		return row, nil
	}

	ctx := context.Background()
	err = importer.ImportStream(ctx, processor)

	if err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	// Should process only 3 non-empty rows
	if processedCount.Load() != 3 {
		t.Errorf("Expected 3 non-empty rows, got %d", processedCount.Load())
	}
}

// TestImportStream_WithProgress tests progress reporting.
func TestImportStream_WithProgress(t *testing.T) {
	// Create test file with 20 rows
	testData := make([][]any, 20)
	for i := range 20 {
		testData[i] = []any{i + 1, "data"}
	}
	buf, _ := createImporterTestFile(testData)

	// Track progress calls
	var progressCalls atomic.Uint64
	var lastCurrent uint64

	// Create importer with progress (report every 5 rows)
	importer, err := NewStreamImporter(buf, WithImportProgress(func(current uint64, _ string) {
		progressCalls.Add(1)
		lastCurrent = current
	}, 5))
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// Process rows
	processor := func(_ context.Context, _ int, row []any) ([]any, error) {
		return row, nil
	}

	ctx := context.Background()
	err = importer.ImportStream(ctx, processor)

	if err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	// Verify progress was called (at rows 5, 10, 15, 20)
	calls := progressCalls.Load()
	if calls < 4 {
		t.Errorf("Expected at least 4 progress calls, got %d", calls)
	}

	// Verify last current value
	if lastCurrent != 20 {
		t.Errorf("Expected last progress current to be 20, got %d", lastCurrent)
	}
}

// TestImportStream_ProcessorError tests error handling in processor.
