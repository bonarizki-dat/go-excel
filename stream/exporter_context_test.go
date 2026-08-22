package stream

import (
	"bytes"
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

func TestExportStream_ContextCancellation(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter
	exporter, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Create data channel
	rows := make(chan []any)

	// Create context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Send rows in background
	go func() {
		defer close(rows)
		for i := range 1000 {
			// Cancel after 10 rows
			if i == 10 {
				cancel()
			}
			// Try to send row (may block if canceled)
			select {
			case rows <- []any{i, "data"}:
			case <-time.After(100 * time.Millisecond):
				// Timeout sending, context likely canceled
				return
			}
		}
	}()

	// Export (should be canceled)
	err = exporter.ExportStream(ctx, rows)

	// Verify error is context.Canceled
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}

	// Verify some rows were processed (less than 1000)
	rowCount := exporter.GetRowCount()
	if rowCount >= 1000 {
		t.Errorf("Expected fewer than 1000 rows due to cancellation, got %d", rowCount)
	}
}

// TestExportStream_LargeDataset tests exporting larger dataset.
func TestExportStream_LargeDataset(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping large dataset test in short mode")
	}

	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter with multiple workers
	exporter, err := NewStreamExporter(&buf, WithExportWorkers(4))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Create data channel with buffer
	rows := make(chan []any, 100)

	// Send 1000 rows
	go func() {
		defer close(rows)
		for i := range 1000 {
			rows <- []any{i, "user" + string(rune(i%26+65)), "test@example.com"}
		}
	}()

	// Export with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = exporter.ExportStream(ctx, rows)

	// Verify no error
	if err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	// Verify the output file has all 1000 rows in input order, exactly
	// the property that a race between workers writing SetRow directly
	// used to corrupt.
	got := readBackXLSX(t, &buf, "Sheet1")
	if len(got) != 1000 {
		t.Fatalf("Expected 1000 rows in output file, got %d", len(got))
	}
	for i, row := range got {
		want := strconv.Itoa(i)
		if len(row) == 0 || row[0] != want {
			t.Fatalf("row %d: expected first cell %q, got %v", i+1, want, row)
		}
	}
}

func TestExportStream_ContextTimeout(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter
	exporter, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Create data channel
	rows := make(chan []any)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// Send rows slowly
	go func() {
		defer close(rows)
		for i := range 100 {
			time.Sleep(5 * time.Millisecond) // Slow sending
			select {
			case rows <- []any{i}:
			case <-time.After(50 * time.Millisecond):
				return
			}
		}
	}()

	// Export (should timeout)
	err = exporter.ExportStream(ctx, rows)

	// Verify error is timeout
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
	}
}
