package importer_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

// TestImporter_ImportFromReader_Success tests successful reader import.
func TestImporter_ImportFromReader_Success(t *testing.T) {
	ctx := context.Background()

	// Create test file in memory
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Product")
	file.SetCellValue("Sheet1", "B1", "Price")
	file.SetCellValue("Sheet1", "A2", "Widget")
	file.SetCellValue("Sheet1", "B2", "9.99")

	// Write to buffer
	buf, err := file.WriteToBuffer()
	if err != nil {
		t.Fatalf("Failed to write file to buffer: %v", err)
	}
	file.Close()

	// Create reader from buffer
	reader := bytes.NewReader(buf.Bytes())

	// Create importer with excel.WithHeadings(false)
	importer := importer.NewImporter(excel.WithHeadings(false))
	err = importer.ImportFromReader(ctx, reader)
	if err != nil {
		t.Fatalf("ImportFromReader failed: %v", err)
	}

	// Verify rows were imported
	rows := importer.GetRows()
	if len(rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(rows))
	}

	// Verify data
	if rows[0][0] != "Product" {
		t.Errorf("Row 0 Col 0 = %v, want Product", rows[0][0])
	}
}

// TestImporter_ImportFromReader_NilReader tests import with nil reader.
func TestImporter_ImportFromReader_NilReader(t *testing.T) {
	ctx := context.Background()

	importer := importer.NewImporter()
	err := importer.ImportFromReader(ctx, nil)
	if err == nil {
		t.Error("ImportFromReader should fail with nil reader")
	}

	// Verify it's an excel.ImportError
	var importErr *excel.ImportError
	if !errors.As(err, &importErr) {
		t.Error("Error should be *excel.ImportError")
	}
}

// TestImporter_ImportFromReader_ContextCancellation tests context cancellation.
func TestImporter_ImportFromReader_ContextCancellation(t *testing.T) {
	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create large test file
	file := excelize.NewFile()
	for i := 1; i <= 1000; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i)
		file.SetCellValue("Sheet1", cell, i)
	}
	buf, _ := file.WriteToBuffer()
	reader := bytes.NewReader(buf.Bytes())

	importer := importer.NewImporter()
	err := importer.ImportFromReader(ctx, reader)
	if err == nil {
		t.Error("ImportFromReader should fail with canceled context")
	}

	// Error should be or wrap context.Canceled
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error should be or wrap context.Canceled, got %v", err)
	}
}

// TestImporter_ImportFromReader_WithHeadings tests reader import with headings.
func TestImporter_ImportFromReader_WithHeadings(t *testing.T) {
	ctx := context.Background()

	// Create test file with headings
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Header1")
	file.SetCellValue("Sheet1", "B1", "Header2")
	file.SetCellValue("Sheet1", "A2", "Data1")
	file.SetCellValue("Sheet1", "B2", "Data2")

	buf, _ := file.WriteToBuffer()
	reader := bytes.NewReader(buf.Bytes())

	// Create importer with headings
	importer := importer.NewImporter(excel.WithHeadings(true))
	err := importer.ImportFromReader(ctx, reader)
	if err != nil {
		t.Fatalf("ImportFromReader failed: %v", err)
	}

	// Verify headings
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

// TestImporter_ImportFromReader_InvalidData tests reader with invalid data.
func TestImporter_ImportFromReader_InvalidData(t *testing.T) {
	ctx := context.Background()

	// Create reader with invalid data
	reader := bytes.NewReader([]byte("not an excel file"))

	importer := importer.NewImporter()
	err := importer.ImportFromReader(ctx, reader)
	if err == nil {
		t.Error("ImportFromReader should fail with invalid data")
	}
}

// TestImporter_ImportFromReader_EmptyData tests reader with empty Excel file.
func TestImporter_ImportFromReader_EmptyData(t *testing.T) {
	ctx := context.Background()

	// Create empty file
	file := excelize.NewFile()
	buf, _ := file.WriteToBuffer()
	reader := bytes.NewReader(buf.Bytes())

	importer := importer.NewImporter()
	err := importer.ImportFromReader(ctx, reader)
	if !errors.Is(err, excel.ErrEmptyData) {
		t.Errorf("ImportFromReader with empty data should return excel.ErrEmptyData, got %v", err)
	}
}
