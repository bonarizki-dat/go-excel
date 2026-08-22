package importer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// TestImportFromFile_Success tests successful file import.
func TestImportFromFile_Success(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_import.xlsx")

	// Create test Excel file
	file := excelize.NewFile()
	defer file.Close()

	// Write test data
	sheetName := "Sheet1"
	file.SetCellValue(sheetName, "A1", "Name")
	file.SetCellValue(sheetName, "B1", "Age")
	file.SetCellValue(sheetName, "A2", "Alice")
	file.SetCellValue(sheetName, "B2", "25")
	file.SetCellValue(sheetName, "A3", "Bob")
	file.SetCellValue(sheetName, "B3", "30")

	// Save file
	if err := file.SaveAs(filename); err != nil {
		t.Fatalf("Failed to save test file: %v", err)
	}

	// Create importer with headings
	importer := NewImporter(
		excel.WithHeadings(true),
		excel.WithSheetName(sheetName),
	)
	defer importer.Close()

	// Import from file
	ctx := context.Background()
	err := importer.ImportFromFile(ctx, filename)
	if err != nil {
		t.Fatalf("ImportFromFile failed: %v", err)
	}

	// Verify headings
	headings := importer.GetHeadings()
	if len(headings) != 2 {
		t.Errorf("Expected 2 headings, got %d", len(headings))
	}
	if headings[0] != "Name" || headings[1] != "Age" {
		t.Errorf("Headings mismatch: got %v", headings)
	}

	// Verify rows
	rows := importer.GetRows()
	if len(rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(rows))
	}

	// Verify first row
	if rows[0][0] != "Alice" || rows[0][1] != "25" {
		t.Errorf("Row 0 mismatch: got %v", rows[0])
	}

	// Verify second row
	if rows[1][0] != "Bob" || rows[1][1] != "30" {
		t.Errorf("Row 1 mismatch: got %v", rows[1])
	}
}

// TestImportFromFile_EmptyFilename tests import with empty filename.
func TestImportFromFile_EmptyFilename(t *testing.T) {
	// Create importer
	importer := NewImporter()
	defer importer.Close()

	// Import with empty filename
	ctx := context.Background()
	err := importer.ImportFromFile(ctx, "")

	// Should return error
	if err == nil {
		t.Fatal("Expected error for empty filename, got nil")
	}

	// Check error type
	importError := &excel.ImportError{}
	if !errors.As(err, &importError) {
		t.Errorf("Expected ImportError, got %T", err)
	}
}

// TestImportFromFile_FileNotExist tests import with non-existent file.
func TestImportFromFile_FileNotExist(t *testing.T) {
	// Create importer
	importer := NewImporter()
	defer importer.Close()

	// Import non-existent file
	ctx := context.Background()
	err := importer.ImportFromFile(ctx, "/nonexistent/file.xlsx")

	// Should return error
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}

	// Check error type
	importError := &excel.ImportError{}
	if !errors.As(err, &importError) {
		t.Errorf("Expected ImportError, got %T", err)
	}
}

// TestImportFromFile_InvalidFile tests import with invalid Excel file.
func TestImportFromFile_InvalidFile(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "invalid.xlsx")

	// Write invalid content
	if err := os.WriteFile(filename, []byte("not an excel file"), 0644); err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	// Create importer
	importer := NewImporter()
	defer importer.Close()

	// Import invalid file
	ctx := context.Background()
	err := importer.ImportFromFile(ctx, filename)

	// Should return error
	if err == nil {
		t.Fatal("Expected error for invalid file, got nil")
	}

	// Check error type
	importError := &excel.ImportError{}
	if !errors.As(err, &importError) {
		t.Errorf("Expected ImportError, got %T", err)
	}
}

// TestImportFromFile_ContextCancellation tests context cancellation.
func TestImportFromFile_ContextCancellation(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_cancel.xlsx")

	// Create test Excel file
	file := excelize.NewFile()
	defer file.Close()

	// Write test data
	file.SetCellValue("Sheet1", "A1", "Test")

	// Save file
	if err := file.SaveAs(filename); err != nil {
		t.Fatalf("Failed to save test file: %v", err)
	}

	// Create importer
	importer := NewImporter()
	defer importer.Close()

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Import should detect cancellation
	err := importer.ImportFromFile(ctx, filename)

	// Should return context error
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// TestImportFromFile_WithLogger tests import with logger.
func TestImportFromFile_WithLogger(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_logger.xlsx")

	// Create test Excel file
	file := excelize.NewFile()
	defer file.Close()

	// Write test data
	file.SetCellValue("Sheet1", "A1", "Test")

	// Save file
	if err := file.SaveAs(filename); err != nil {
		t.Fatalf("Failed to save test file: %v", err)
	}

	// Create mock logger
	logger := &mockLogger{logs: make([]string, 0)}

	// Create importer with logger
	importer := NewImporter(excel.WithLogger(logger))
	defer importer.Close()

	// Import from file
	ctx := context.Background()
	err := importer.ImportFromFile(ctx, filename)
	if err != nil {
		t.Fatalf("ImportFromFile failed: %v", err)
	}

	// Verify logger was called
	if len(logger.logs) == 0 {
		t.Error("Expected logger to be called, but no logs recorded")
	}
}

// TestImportFromFile_WithMetrics tests import with metrics.
func TestImportFromFile_WithMetrics(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_metrics.xlsx")

	// Create test Excel file
	file := excelize.NewFile()
	defer file.Close()

	// Write test data
	file.SetCellValue("Sheet1", "A1", "Test")

	// Save file
	if err := file.SaveAs(filename); err != nil {
		t.Fatalf("Failed to save test file: %v", err)
	}

	// Create mock metrics
	metrics := &mockMetrics{
		timings: make(map[string]int64),
		counts:  make(map[string]int64),
	}

	// Create importer with metrics
	importer := NewImporter(excel.WithMetrics(metrics))
	defer importer.Close()

	// Import from file
	ctx := context.Background()
	err := importer.ImportFromFile(ctx, filename)
	if err != nil {
		t.Fatalf("ImportFromFile failed: %v", err)
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

// Mock types defined in importer_mocks_test.go
