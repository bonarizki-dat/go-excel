package importer_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

// TestImporter_ImportFromFile_Success tests successful file import.
func TestImporter_ImportFromFile_Success(t *testing.T) {
	ctx := context.Background()

	// Create test file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_import.xlsx")

	file := excelize.NewFile()
	// Use Sheet1 explicitly - add 3 rows to test
	file.SetCellValue("Sheet1", "A1", "Name")
	file.SetCellValue("Sheet1", "B1", "Age")
	file.SetCellValue("Sheet1", "A2", "Alice")
	file.SetCellValue("Sheet1", "B2", "30")
	file.SetCellValue("Sheet1", "A3", "Bob")
	file.SetCellValue("Sheet1", "B3", "25")

	if err := file.SaveAs(filename); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()

	// Create importer with excel.WithHeadings(false) to import all rows as data
	importer := importer.NewImporter(excel.WithHeadings(false))
	err := importer.ImportFromFile(ctx, filename)
	if err != nil {
		t.Fatalf("ImportFromFile failed: %v", err)
	}

	// Verify all 3 rows were imported
	rows := importer.GetRows()
	if len(rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(rows))
	}
	// Verify first row data
	if len(rows) > 0 && rows[0][0] != "Name" {
		t.Errorf("Row 0 Col 0 = %v, want Name", rows[0][0])
	}
}

// TestImporter_ImportFromFile_EmptyFilename tests import with empty filename.
func TestImporter_ImportFromFile_EmptyFilename(t *testing.T) {
	ctx := context.Background()

	importer := importer.NewImporter()
	err := importer.ImportFromFile(ctx, "")
	if err == nil {
		t.Error("ImportFromFile should fail with empty filename")
	}

	// Verify it's an excel.ImportError
	var importErr *excel.ImportError
	if !errors.As(err, &importErr) {
		t.Error("Error should be *excel.ImportError")
	}
}

// TestImporter_ImportFromFile_FileNotFound tests import with non-existent file.
func TestImporter_ImportFromFile_FileNotFound(t *testing.T) {
	ctx := context.Background()

	importer := importer.NewImporter()
	err := importer.ImportFromFile(ctx, "/nonexistent/file.xlsx")
	if err == nil {
		t.Error("ImportFromFile should fail with non-existent file")
	}

	// Verify it's an excel.ImportError
	var importErr *excel.ImportError
	if !errors.As(err, &importErr) {
		t.Error("Error should be *excel.ImportError")
	}
}

// TestImporter_ImportFromFile_ContextCancellation tests context cancellation.
func TestImporter_ImportFromFile_ContextCancellation(t *testing.T) {
	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create test file
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_cancelled.xlsx")

	file := excelize.NewFile()
	for i := 1; i <= 1000; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i)
		file.SetCellValue("Sheet1", cell, i)
	}
	file.SaveAs(filename)

	importer := importer.NewImporter()
	err := importer.ImportFromFile(ctx, filename)
	if err == nil {
		t.Error("ImportFromFile should fail with canceled context")
	}

	// Error should be or wrap context.Canceled
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error should be or wrap context.Canceled, got %v", err)
	}
}

// TestImporter_ImportFromFile_WithOptions tests import with various options.
func TestImporter_ImportFromFile_WithOptions(t *testing.T) {
	ctx := context.Background()

	// Create test file with headings
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_options.xlsx")

	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "ID")
	file.SetCellValue("Sheet1", "B1", "Name")
	file.SetCellValue("Sheet1", "A2", "1")
	file.SetCellValue("Sheet1", "B2", "Alice")
	file.SaveAs(filename)

	// Create importer with options
	importer := importer.NewImporter(excel.WithHeadings(true))
	err := importer.ImportFromFile(ctx, filename)
	if err != nil {
		t.Fatalf("ImportFromFile failed: %v", err)
	}

	// Verify headings were extracted
	headings := importer.GetHeadings()
	if len(headings) != 2 {
		t.Errorf("Expected 2 headings, got %d", len(headings))
	}

	// Verify data rows
	rows := importer.GetRows()
	if len(rows) != 1 {
		t.Errorf("Expected 1 data row, got %d", len(rows))
	}
}

// TestImporter_ImportFromFile_InvalidFile tests import with invalid Excel file.
func TestImporter_ImportFromFile_InvalidFile(t *testing.T) {
	ctx := context.Background()

	// Create invalid file (not an Excel file)
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "invalid.xlsx")

	err := os.WriteFile(filename, []byte("not an excel file"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	importer := importer.NewImporter()
	err = importer.ImportFromFile(ctx, filename)
	if err == nil {
		t.Error("ImportFromFile should fail with invalid Excel file")
	}
}
