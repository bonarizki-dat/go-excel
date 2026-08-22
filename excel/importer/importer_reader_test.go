package importer

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// TestImportFromReader_Success tests successful import from reader.
func TestImportFromReader_Success(t *testing.T) {
	// Create test Excel file in memory
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

	// Write to buffer
	buf := new(bytes.Buffer)
	if err := file.Write(buf); err != nil {
		t.Fatalf("Failed to write file to buffer: %v", err)
	}

	// Create importer with headings
	importer := NewImporter(
		excel.WithHeadings(true),
		excel.WithSheetName(sheetName),
	)
	defer importer.Close()

	// Import from reader
	ctx := context.Background()
	err := importer.ImportFromReader(ctx, buf)
	if err != nil {
		t.Fatalf("ImportFromReader failed: %v", err)
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

// TestImportFromReader_NilReader tests import with nil reader.
func TestImportFromReader_NilReader(t *testing.T) {
	// Create importer
	importer := NewImporter()
	defer importer.Close()

	// Import with nil reader
	ctx := context.Background()
	err := importer.ImportFromReader(ctx, nil)

	// Should return error
	if err == nil {
		t.Fatal("Expected error for nil reader, got nil")
	}

	// Check error type
	importError := &excel.ImportError{}
	if !errors.As(err, &importError) {
		t.Errorf("Expected ImportError, got %T", err)
	}
}

// TestImportFromReader_InvalidData tests import with invalid Excel data.
func TestImportFromReader_InvalidData(t *testing.T) {
	// Create importer
	importer := NewImporter()
	defer importer.Close()

	// Create reader with invalid data
	buf := bytes.NewBufferString("not an excel file")

	// Import invalid data
	ctx := context.Background()
	err := importer.ImportFromReader(ctx, buf)

	// Should return error
	if err == nil {
		t.Fatal("Expected error for invalid data, got nil")
	}

	// Check error type
	importError := &excel.ImportError{}
	if !errors.As(err, &importError) {
		t.Errorf("Expected ImportError, got %T", err)
	}
}

// TestImportFromReader_ContextCancellation tests context cancellation.
func TestImportFromReader_ContextCancellation(t *testing.T) {
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

	// Create importer
	importer := NewImporter()
	defer importer.Close()

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Import should detect cancellation
	err := importer.ImportFromReader(ctx, buf)

	// Should return context error
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

// Additional tests in importer_reader_advanced_test.go
