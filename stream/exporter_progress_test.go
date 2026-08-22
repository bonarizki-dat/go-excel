package stream

import (
	"bytes"
	"context"
	"strconv"
	"sync/atomic"
	"testing"
)

// TestExportStream_WithProgress tests progress reporting.
func TestExportStream_WithProgress(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Track progress calls
	var progressCalls atomic.Uint64
	var lastCurrent uint64

	// Create exporter with progress callback
	exporter, err := NewStreamExporter(&buf, WithExportProgress(func(current uint64, _ string) {
		progressCalls.Add(1)
		lastCurrent = current
	}, 5)) // Report every 5 rows
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Create data channel
	rows := make(chan []any, 20)

	// Send 20 rows
	go func() {
		defer close(rows)
		for i := range 20 {
			rows <- []any{i, "data"}
		}
	}()

	// Export
	ctx := context.Background()
	err = exporter.ExportStream(ctx, rows)

	// Verify no error
	if err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	// Verify progress was called (should be called at rows 5, 10, 15, 20)
	calls := progressCalls.Load()
	if calls < 4 {
		t.Errorf("Expected at least 4 progress calls, got %d", calls)
	}

	// Verify last current value
	if lastCurrent != 20 {
		t.Errorf("Expected last progress current to be 20, got %d", lastCurrent)
	}

	// Verify the output file has all 20 rows in order
	got := readBackXLSX(t, &buf, "Sheet1")
	if len(got) != 20 {
		t.Fatalf("Expected 20 rows in output file, got %d", len(got))
	}
	for i, row := range got {
		want := strconv.Itoa(i)
		if len(row) == 0 || row[0] != want {
			t.Errorf("row %d: expected first cell %q, got %v", i+1, want, row)
		}
	}
}

// TestExportStream_ContextCancellation tests canceling export.
// TestExportStream_WithHeadingsAndProgress tests combined features.
func TestExportStream_WithHeadingsAndProgress(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Track progress
	var progressCount atomic.Uint64

	// Create exporter with headings and progress
	exporter, err := NewStreamExporter(&buf,
		WithExportHeadings([]string{"ID", "Name"}),
		WithExportProgress(func(_ uint64, _ string) {
			progressCount.Add(1)
		}, 10),
		WithExportWorkers(2),
	)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Create data channel
	rows := make(chan []any, 25)

	// Send 25 rows
	go func() {
		defer close(rows)
		for i := range 25 {
			rows <- []any{i, "user" + string(rune(i%10+65))}
		}
	}()

	// Export
	ctx := context.Background()
	err = exporter.ExportStream(ctx, rows)

	// Verify no error
	if err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	// Verify progress called (at 10, 20 rows)
	if progressCount.Load() < 2 {
		t.Errorf("Expected at least 2 progress calls, got %d", progressCount.Load())
	}

	// Verify the output file has a heading row followed by 25 data rows
	// in order
	got := readBackXLSX(t, &buf, "Sheet1")
	if len(got) != 26 {
		t.Fatalf("Expected 1 heading row + 25 data rows, got %d rows", len(got))
	}
	if got[0][0] != "ID" || got[0][1] != "Name" {
		t.Errorf("Expected heading row [ID, Name], got %v", got[0])
	}
	for i, row := range got[1:] {
		want := strconv.Itoa(i)
		if len(row) == 0 || row[0] != want {
			t.Errorf("data row %d: expected first cell %q, got %v", i+1, want, row)
		}
	}
}

// TestExportStream_ContextTimeout tests export timeout.
// TestExportStream_EmptyHeadings tests with empty headings.
func TestExportStream_EmptyHeadings(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter with empty headings (should not write headings)
	exporter, err := NewStreamExporter(&buf, WithExportHeadings([]string{}))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Create data channel
	rows := make(chan []any, 3)

	// Send rows
	go func() {
		defer close(rows)
		rows <- []any{1, "data1"}
		rows <- []any{2, "data2"}
	}()

	// Export
	ctx := context.Background()
	err = exporter.ExportStream(ctx, rows)

	// Verify no error
	if err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	// Verify no blank heading row was left in place of the (empty)
	// heading: data starts at row 1.
	got := readBackXLSX(t, &buf, "Sheet1")
	if len(got) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(got))
	}
	if got[0][0] != "1" || got[1][0] != "2" {
		t.Errorf("Expected rows in order [1, 2], got %v", got)
	}
}
