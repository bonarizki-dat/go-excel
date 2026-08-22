package importer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

// TestNewImporter tests the NewImporter constructor
// TestImporter_Import_NoFile tests import with no file set.
func TestImporter_Import_NoFile(t *testing.T) {
	ctx := context.Background()

	imp := importer.NewImporter()
	err := imp.Import(ctx)
	if !errors.Is(err, excel.ErrInvalidFile) {
		t.Errorf("Import with no file should return excel.ErrInvalidFile, got %v", err)
	}
}

// TestImporter_Import_InvalidConfig tests import with invalid config.
func TestImporter_Import_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	// Create test file
	file := excelize.NewFile()

	// Create importer with invalid config
	imp := importer.NewImporterWithFile(file, excel.WithSheetName("")) // Empty sheet name

	err := imp.Import(ctx)
	if err == nil {
		t.Error("Import with invalid config should return error")
	}
}

// TestImporter_Import_SimpleData tests basic import functionality.
func TestImporter_Import_SimpleData(t *testing.T) {
	ctx := context.Background()

	// Create test file with data and save/reload to ensure data is readable
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Alice")
	file.SetCellValue("Sheet1", "B1", "30")
	file.SetCellValue("Sheet1", "A2", "Bob")
	file.SetCellValue("Sheet1", "B2", "25")

	// Save to buffer and reload
	buf, err := file.WriteToBuffer()
	if err != nil {
		t.Fatalf("Failed to write buffer: %v", err)
	}
	file, err = excelize.OpenReader(buf)
	if err != nil {
		t.Fatalf("Failed to open from buffer: %v", err)
	}

	// Create importer with excel.WithHeadings(false)
	imp := importer.NewImporterWithFile(file, excel.WithHeadings(false))

	// Perform import
	err = imp.Import(ctx)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify rows were imported
	rows := imp.GetRows()
	if len(rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(rows))
	}

	// Verify first row data
	if rows[0][0] != "Alice" {
		t.Errorf("Row 0 Col 0 = %v, want Alice", rows[0][0])
	}
}

// TestImporter_Import_excel.WithHeadings tests import with headings enabled.
func TestImporter_Import_WithHeadings(t *testing.T) {
	ctx := context.Background()

	// Create test file with headings
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "ID")
	file.SetCellValue("Sheet1", "B1", "Name")
	file.SetCellValue("Sheet1", "C1", "Email")
	file.SetCellValue("Sheet1", "A2", "1")
	file.SetCellValue("Sheet1", "B2", "Alice")
	file.SetCellValue("Sheet1", "C2", "alice@example.com")

	// Create importer with headings enabled
	imp := importer.NewImporterWithFile(file, excel.WithHeadings(true))

	err := imp.Import(ctx)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify headings were extracted
	headings := imp.GetHeadings()
	if len(headings) != 3 {
		t.Errorf("Expected 3 headings, got %d", len(headings))
	}
	if headings[0] != "ID" {
		t.Errorf("Heading 0 = %v, want ID", headings[0])
	}

	// Verify data rows don't include heading
	rows := imp.GetRows()
	if len(rows) != 1 {
		t.Errorf("Expected 1 data row, got %d", len(rows))
	}
	if rows[0][0] != "1" {
		t.Errorf("Row 0 Col 0 = %v, want 1", rows[0][0])
	}
}

// TestImporter_Import_EmptySheet tests import with empty sheet.
func TestImporter_Import_EmptySheet(t *testing.T) {
	ctx := context.Background()

	// Create empty file
	file := excelize.NewFile()

	imp := importer.NewImporterWithFile(file)
	err := imp.Import(ctx)
	if !errors.Is(err, excel.ErrEmptyData) {
		t.Errorf("Import empty sheet should return excel.ErrEmptyData, got %v", err)
	}
}

// TestImporter_Import_ContextCancellation tests context cancellation.
func TestImporter_Import_ContextCancellation(t *testing.T) {
	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create large test file
	file := excelize.NewFile()
	for i := 1; i <= 1000; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i)
		file.SetCellValue("Sheet1", cell, i)
	}

	imp := importer.NewImporterWithFile(file)
	err := imp.Import(ctx)
	if err == nil {
		t.Error("Import should fail with canceled context")
	}

	// Error might be wrapped
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error should be or wrap context.Canceled, got %v", err)
	}
}

// TestImporter_Import_WithTimeout tests import with timeout.
func TestImporter_Import_WithTimeout(t *testing.T) {
	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait to ensure timeout
	time.Sleep(10 * time.Millisecond)

	// Create large test file to ensure timeout is hit
	file := excelize.NewFile()
	for i := 1; i <= 1000; i++ {
		cell, _ := excelize.CoordinatesToCellName(1, i)
		file.SetCellValue("Sheet1", cell, i)
	}

	imp := importer.NewImporterWithFile(file)
	err := imp.Import(ctx)
	// Should fail due to timeout
	if err == nil {
		t.Error("Import should fail with timeout")
	}
}

// TestImporter_GetRows tests GetRows method
