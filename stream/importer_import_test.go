package stream

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// TestImportStream_SmallDataset tests importing small dataset.
func TestImportStream_SmallDataset(t *testing.T) {
	// Create test Excel file
	testData := [][]any{
		{1, "Alice", "alice@test.com"},
		{2, "Bob", "bob@test.com"},
		{3, "Charlie", "charlie@test.com"},
	}
	buf, err := createImporterTestFile(testData)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create importer
	importer, err := NewStreamImporter(buf)
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// Track processed rows
	var processedCount atomic.Uint64

	// Process rows
	processor := func(_ context.Context, _ int, row []any) ([]any, error) {
		processedCount.Add(1)
		return row, nil
	}

	// Import
	ctx := context.Background()
	err = importer.ImportStream(ctx, processor)

	// Verify no error
	if err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	// Verify all rows processed
	if processedCount.Load() != 3 {
		t.Errorf("Expected 3 rows processed, got %d", processedCount.Load())
	}

	// Verify row count
	if importer.GetRowCount() != 3 {
		t.Errorf("Expected row count 3, got %d", importer.GetRowCount())
	}
}

// TestImportStream_RetainedRowsSurviveAfterImport verifies that rows
// handed to the processor remain valid after ImportStream returns, even
// when the processor retains a reference to the row slice instead of
// copying it (e.g. rows = append(rows, row), a natural pattern for
// callers that want to collect results). Before this fix, the pool
// buffer backing each row was recycled and cleared as soon as the
// processor returned, so a retained row could later be overwritten by an
// unrelated row read from the same recycled buffer.
func TestImportStream_RetainedRowsSurviveAfterImport(t *testing.T) {
	const rowCount = 500

	testData := make([][]any, rowCount)
	for i := range rowCount {
		testData[i] = []any{i, fmt.Sprintf("value-%d", i)}
	}
	buf, err := createImporterTestFile(testData)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	importer, err := NewStreamImporter(buf, WithImportWorkers(8))
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	var mu sync.Mutex
	retained := make(map[int][]any, rowCount)

	processor := func(_ context.Context, rowNum int, row []any) ([]any, error) {
		// Retain the row itself, not a copy, mirroring a caller that
		// naively collects rows into a slice.
		mu.Lock()
		retained[rowNum] = row
		mu.Unlock()
		return row, nil
	}

	if err := importer.ImportStream(context.Background(), processor); err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	if len(retained) != rowCount {
		t.Fatalf("expected %d retained rows, got %d", rowCount, len(retained))
	}

	for i := 0; i < rowCount; i++ {
		row, ok := retained[i+1] // rowNum is 1-indexed
		if !ok {
			t.Fatalf("missing retained row for rowNum %d", i+1)
		}
		// Cell values round-trip through Excel as strings.
		wantID, wantVal := fmt.Sprintf("%d", i), fmt.Sprintf("value-%d", i)
		gotID, ok1 := row[0].(string)
		gotVal, ok2 := row[1].(string)
		if !ok1 || !ok2 || gotID != wantID || gotVal != wantVal {
			t.Errorf("row %d corrupted after import: got %v, want [%v %v]", i+1, row, wantID, wantVal)
		}
	}
}

// TestImportStream_WithHeadings tests importing with headings.
func TestImportStream_EmptyFile(t *testing.T) {
	// Create empty file (no data rows)
	testData := [][]any{}
	buf, _ := createImporterTestFile(testData)

	// Create importer
	importer, err := NewStreamImporter(buf)
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

	// Verify no error
	if err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	// Verify no rows processed
	if processedCount.Load() != 0 {
		t.Errorf("Expected 0 rows processed, got %d", processedCount.Load())
	}
}
