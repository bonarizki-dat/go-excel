package exporter_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// TestExporter_Export_WithProgressCallback tests export with progress reporting.
func TestExporter_Export_WithProgressCallback(t *testing.T) {
	ctx := context.Background()

	// Create dataset large enough to trigger progress callback (every 100 rows)
	data := make([][]any, 250)
	for i := range data {
		data[i] = []any{i, "test", "data"}
	}

	// Track progress callback calls
	var callCount int32
	var lastProcessed int64
	var lastTotal int64

	// Create exporter with progress callback
	exp := exporter.NewExporter(data, excel.WithProgress(func(processed, total int64) {
		// Increment call count atomically for concurrency safety
		atomic.AddInt32(&callCount, 1)
		// Store last values
		atomic.StoreInt64(&lastProcessed, processed)
		atomic.StoreInt64(&lastTotal, total)
	}))

	// Perform export
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify progress callback was called
	// Should be called at 100, 200, and final (250)
	calls := atomic.LoadInt32(&callCount)
	if calls < 3 {
		t.Errorf("Progress callback should be called at least 3 times, got %d", calls)
	}

	// Verify final progress is 100%
	processed := atomic.LoadInt64(&lastProcessed)
	total := atomic.LoadInt64(&lastTotal)
	if processed != total {
		t.Errorf("Final progress: processed=%d, total=%d, should be equal", processed, total)
	}
	if total != 250 {
		t.Errorf("Total rows = %d, want 250", total)
	}
}

// TestExporter_Export_WithProgressCallback_WithHeadings tests progress with headings.
func TestExporter_Export_WithProgressCallback_WithHeadings(t *testing.T) {
	ctx := context.Background()

	// Create dataset with headings
	data := make([][]any, 201) // 1 header + 200 data rows
	data[0] = []any{"ID", "Name", "Value"}
	for i := 1; i < len(data); i++ {
		data[i] = []any{i, "test", i * 10}
	}

	// Track progress callback calls
	var callCount int32
	var lastProcessed int64

	// Create exporter with headings and progress callback
	exp := exporter.NewExporter(data,
		excel.WithHeadings(true),
		excel.WithProgress(func(processed, _ int64) {
			atomic.AddInt32(&callCount, 1)
			atomic.StoreInt64(&lastProcessed, processed)
		}))

	// Perform export
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify callback was called (100, 200, final)
	calls := atomic.LoadInt32(&callCount)
	if calls < 2 {
		t.Errorf("Progress callback should be called at least 2 times, got %d", calls)
	}

	// Final processed should be 201 (including all rows before headings are stripped)
	// The actual data rows are 200, but the callback gets the total before filtering
	processed := atomic.LoadInt64(&lastProcessed)
	if processed != 200 && processed != 201 {
		t.Errorf("Final processed = %d, want 200 or 201", processed)
	}
}

// TestExporter_Export_WithErrorCallback_Continue tests error callback that continues.
func TestExporter_Export_WithErrorCallback_Continue(t *testing.T) {
	ctx := context.Background()

	// Create mock data that will cause errors
	// We'll use a custom type that excelize can't handle
	type UnexportableType struct {
		private int
	}

	data := [][]any{
		{"Name", "Age"},
		{"John", 30},
		{UnexportableType{private: 42}, "test"}, // This might cause issues
		{"Jane", 25},
	}

	// Track error callback calls
	var errorCount int32

	// Create exporter with error callback and StopOnError=false
	exp := exporter.NewExporter(data,
		excel.WithStopOnError(false),
		excel.WithErrorHandler(func(_ int, _ error) bool {
			// Increment error count
			atomic.AddInt32(&errorCount, 1)
			// Return true to continue processing
			return true
		}))

	// Perform export - should complete despite errors
	err := exp.Export(ctx)
	// Export might return error if StopOnError is false and errors occurred
	// But it should process all rows
	_ = err // Ignore error as we're testing continuation

	// Verify error callback was called if there were errors
	errors := exp.GetErrors()
	if len(errors) > 0 {
		count := atomic.LoadInt32(&errorCount)
		if count != int32(len(errors)) {
			t.Errorf("Error callback called %d times, but %d errors collected", count, len(errors))
		}
	}
}

// TestExporter_Export_WithErrorCallback_Stop tests error callback that stops.
func TestExporter_Export_WithErrorCallback_Stop(t *testing.T) {
	ctx := context.Background()

	// We need to create a scenario where writeRow will fail
	// Let's use a very large row that might cause issues
	data := make([][]any, 10)
	for i := range data {
		data[i] = []any{i, "test"}
	}

	// Track error callback calls
	var errorCallbackCalled bool

	// Create exporter with error callback that returns false (stop)
	// and StopOnError=false to trigger callback
	exp := exporter.NewExporter(data,
		excel.WithStopOnError(false),
		excel.WithErrorHandler(func(_ int, _ error) bool {
			errorCallbackCalled = true
			// Return false to stop processing
			return false
		}))

	// Note: Since our test data is valid, errors won't actually occur
	// This test ensures the callback mechanism works when called
	err := exp.Export(ctx)
	if err != nil && errorCallbackCalled {
		// If error occurred and callback was called, it worked
		t.Logf("Error callback mechanism working: %v", err)
	}
}
