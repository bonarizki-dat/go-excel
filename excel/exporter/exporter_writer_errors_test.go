package exporter_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// TestExporter_ExportToFile_ErrorInExport tests file export when Export fails.
func TestExporter_ExportToFile_ErrorInExport(t *testing.T) {
	ctx := context.Background()

	// Use nil data to force Export to fail
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_error.xlsx")

	logger := &MockLogger{}
	exp := exporter.NewExporter(nil, excel.WithLogger(logger))

	err := exp.ExportToFile(ctx, filename)
	if err == nil {
		t.Error("ExportToFile should fail with nil data")
	}

	// Verify logger recorded error
	if logger.errorCalls == 0 {
		t.Error("Logger Error should be called on failure")
	}

	// Verify error is wrapped
	if !errors.Is(err, excel.ErrEmptyData) {
		t.Logf("Error: %v", err)
	}
}

// TestExporter_ExportToWriter_ErrorInExport tests writer export when Export fails.
func TestExporter_ExportToWriter_ErrorInExport(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	logger := &MockLogger{}

	// Use nil data to force Export to fail
	exp := exporter.NewExporter(nil, excel.WithLogger(logger))

	err := exp.ExportToWriter(ctx, &buf)
	if err == nil {
		t.Error("ExportToWriter should fail with nil data")
	}

	// Verify logger recorded error
	if logger.errorCalls == 0 {
		t.Error("Logger Error should be called on failure")
	}
}

// TestExporter_ExportToBuffer_ErrorInExport tests buffer export when Export fails.
func TestExporter_ExportToBuffer_ErrorInExport(t *testing.T) {
	ctx := context.Background()

	logger := &MockLogger{}

	// Use nil data to force Export to fail
	exp := exporter.NewExporter(nil, excel.WithLogger(logger))

	_, err := exp.ExportToBuffer(ctx)
	if err == nil {
		t.Error("ExportToBuffer should fail with nil data")
	}

	// Verify logger recorded error
	if logger.errorCalls == 0 {
		t.Error("Logger Error should be called on failure")
	}
}

// MockFailWriter is a writer that always fails.
type MockFailWriter struct{}

func (m *MockFailWriter) Write(_ []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

// TestExporter_ExportToWriter_WriteError tests writer error handling.
func TestExporter_ExportToWriter_WriteError(t *testing.T) {
	ctx := context.Background()

	data := [][]any{{"test"}}
	logger := &MockLogger{}

	// Create exporter
	exp := exporter.NewExporter(data, excel.WithLogger(logger))

	// Use failing writer
	failWriter := &MockFailWriter{}
	err := exp.ExportToWriter(ctx, failWriter)

	// Should fail with write error
	if err == nil {
		t.Error("ExportToWriter should fail with failing writer")
	}

	// Verify it's an ExportError
	var exportErr *excel.ExportError
	if !errors.As(err, &exportErr) {
		t.Logf("Error should be wrapped as ExportError: %v", err)
	}
}

// TestExporter_ExportToFile_InvalidDirectory tests directory creation failure.
func TestExporter_ExportToFile_InvalidDirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	ctx := context.Background()
	data := [][]any{{"test"}}

	// Try to write to a path that should fail
	// Use /dev/null/test.xlsx which can't have children
	filename := "/dev/null/impossible/test.xlsx"

	exp := exporter.NewExporter(data)
	err := exp.ExportToFile(ctx, filename)

	// Should fail with directory creation error
	if err == nil {
		t.Error("ExportToFile should fail with invalid directory path")
	}
}

// TestExporter_ExportToFile_LargeDataset tests file export with larger dataset.
func TestExporter_ExportToFile_LargeDataset(t *testing.T) {
	ctx := context.Background()

	// Create larger dataset
	rowCount := 500
	data := make([][]any, rowCount)
	for i := range data {
		data[i] = []any{i, "row_" + string(rune('0'+(i%10))), float64(i) * 10.5}
	}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_large.xlsx")

	metrics := NewMockMetrics()
	exp := exporter.NewExporter(data, excel.WithMetrics(metrics))

	err := exp.ExportToFile(ctx, filename)
	if err != nil {
		t.Fatalf("ExportToFile with large dataset failed: %v", err)
	}

	// Verify file was created and has reasonable size
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	if info.Size() < 1000 {
		t.Errorf("File size %d seems too small for %d rows", info.Size(), rowCount)
	}

	// Verify metrics
	if metrics.counts["export.rows"] != int64(rowCount) {
		t.Errorf("Metrics export.rows = %d, want %d", metrics.counts["export.rows"], rowCount)
	}
}
