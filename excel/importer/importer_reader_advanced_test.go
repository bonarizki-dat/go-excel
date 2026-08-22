package importer

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// TestImportFromReader_WithLogger tests import with logger.
func TestImportFromReader_WithLogger(t *testing.T) {
	// Create test Excel file in memory
	file := excelize.NewFile()
	defer file.Close()

	// Write test data
	file.SetCellValue("Sheet1", "A1", "Test")

	// Write to buffer
	buf := new(bytes.Buffer)
	if err := file.Write(buf); err != nil {
		t.Fatalf("Failed to write file to buffer: %v", err)
	}

	// Create mock logger
	logger := &mockLogger{logs: make([]string, 0)}

	// Create importer with logger
	importer := NewImporter(excel.WithLogger(logger))
	defer importer.Close()

	// Import from reader
	ctx := context.Background()
	err := importer.ImportFromReader(ctx, buf)
	if err != nil {
		t.Fatalf("ImportFromReader failed: %v", err)
	}

	// Verify logger was called
	if len(logger.logs) == 0 {
		t.Error("Expected logger to be called, but no logs recorded")
	}
}

// TestImportFromReader_WithMetrics tests import with metrics.
func TestImportFromReader_WithMetrics(t *testing.T) {
	// Create test Excel file in memory
	file := excelize.NewFile()
	defer file.Close()

	// Write test data
	file.SetCellValue("Sheet1", "A1", "Test")

	// Write to buffer
	buf := new(bytes.Buffer)
	if err := file.Write(buf); err != nil {
		t.Fatalf("Failed to write file to buffer: %v", err)
	}

	// Create mock metrics
	metrics := &mockMetrics{
		timings: make(map[string]int64),
		counts:  make(map[string]int64),
	}

	// Create importer with metrics
	importer := NewImporter(excel.WithMetrics(metrics))
	defer importer.Close()

	// Import from reader
	ctx := context.Background()
	err := importer.ImportFromReader(ctx, buf)
	if err != nil {
		t.Fatalf("ImportFromReader failed: %v", err)
	}

	// Verify metrics were recorded
	if _, ok := metrics.timings["import.duration"]; !ok {
		t.Error("Expected import.duration metric to be recorded")
	}
	if _, ok := metrics.counts["import.rows"]; !ok {
		t.Error("Expected import.rows metric to be recorded")
	}
	if _, ok := metrics.counts["import.success"]; !ok {
		t.Error("Expected import.success metric to be recorded")
	}
}

// TestImportFromReader_EmptySheet tests import from empty sheet.
func TestImportFromReader_EmptySheet(t *testing.T) {
	// Create test Excel file with empty sheet
	file := excelize.NewFile()
	defer file.Close()

	// Don't write any data (empty sheet)

	// Write to buffer
	buf := new(bytes.Buffer)
	if err := file.Write(buf); err != nil {
		t.Fatalf("Failed to write file to buffer: %v", err)
	}

	// Create importer
	importer := NewImporter()
	defer importer.Close()

	// Import from reader
	ctx := context.Background()
	err := importer.ImportFromReader(ctx, buf)

	// Should return error (wrapped ErrEmptyData)
	if err == nil {
		t.Error("Expected error for empty sheet, got nil")
	}
}

// TestImportFromReader_LargeData tests import with large dataset.
func TestImportFromReader_LargeData(t *testing.T) {
	// Create test Excel file with many rows
	file := excelize.NewFile()
	defer file.Close()

	// Write 1000 rows
	sheetName := "Sheet1"
	for i := 1; i <= 1000; i++ {
		file.SetCellValue(sheetName, cellName("A", i), i)
		file.SetCellValue(sheetName, cellName("B", i), fmt.Sprintf("Row %d", i))
	}

	// Write to buffer
	buf := new(bytes.Buffer)
	if err := file.Write(buf); err != nil {
		t.Fatalf("Failed to write file to buffer: %v", err)
	}

	// Create importer with progress callback
	progressCalled := 0
	importer := NewImporter(
		excel.WithProgress(func(_, _ int64) {
			progressCalled++
		}),
	)
	defer importer.Close()

	// Import from reader
	ctx := context.Background()
	err := importer.ImportFromReader(ctx, buf)
	if err != nil {
		t.Fatalf("ImportFromReader failed: %v", err)
	}

	// Verify rows imported (excelize may skip empty rows, so we check >= 999)
	rows := importer.GetRows()
	if len(rows) < 999 {
		t.Errorf("Expected at least 999 rows, got %d", len(rows))
	}

	// Verify progress callback was called
	if progressCalled == 0 {
		t.Error("Expected progress callback to be called")
	}
}

// cellName generates Excel cell name from column letter and row number.
func cellName(col string, row int) string {
	// Simple string formatting for cell address
	return fmt.Sprintf("%s%d", col, row)
}
