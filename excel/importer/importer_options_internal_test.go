package importer

import (
	"context"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/bonarizki-dat/go-excel/excel"
)

func TestImporter_Import_WithSkipEmptyRows(t *testing.T) {
	ctx := context.Background()

	// Create test file with empty rows
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Data1")
	file.SetCellValue("Sheet1", "A2", "") // Empty row
	file.SetCellValue("Sheet1", "A3", "Data2")
	file.SetCellValue("Sheet1", "A4", "") // Empty row
	file.SetCellValue("Sheet1", "A5", "Data3")

	// Save and reload
	buf, _ := file.WriteToBuffer()
	file, _ = excelize.OpenReader(buf)

	// Import with skip empty rows and excel.WithHeadings(false)
	importer := NewImporterWithFile(file, excel.WithSkipEmptyRows(true), excel.WithHeadings(false))
	err := importer.Import(ctx)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should only have 3 rows (empty rows skipped)
	rows := importer.GetRows()
	if len(rows) != 3 {
		t.Errorf("Expected 3 rows (empty rows skipped), got %d", len(rows))
	}
}

// TestImporter_Import_WithRows tests row range filtering.
func TestImporter_Import_WithRows(t *testing.T) {
	ctx := context.Background()

	// Create test file with multiple rows
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Row1")
	file.SetCellValue("Sheet1", "A2", "Row2")
	file.SetCellValue("Sheet1", "A3", "Row3")
	file.SetCellValue("Sheet1", "A4", "Row4")
	file.SetCellValue("Sheet1", "A5", "Row5")

	// Save and reload
	buf, _ := file.WriteToBuffer()
	file, _ = excelize.OpenReader(buf)

	// Import only rows 2-4 with excel.WithHeadings(false). WithRows is
	// 1-based and inclusive, so first=2, last=4 selects Row2, Row3, Row4.
	importer := NewImporterWithFile(file, excel.WithRows(2, 4), excel.WithHeadings(false))
	err := importer.Import(ctx)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should have 3 rows: Row2, Row3, Row4 (Row5 is excluded since EndRow=4 is exclusive)
	rows := importer.GetRows()
	if len(rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(rows))
	}
	if len(rows) > 0 && rows[0][0] != "Row2" {
		t.Errorf("First row = %v, want Row2", rows[0][0])
	}
	if len(rows) > 0 && rows[len(rows)-1][0] != "Row4" {
		t.Errorf("Last row = %v, want Row4", rows[len(rows)-1][0])
	}
}

// TestImporter_Import_RowRange_AbsoluteEndRow verifies that the upper
// bound is absolute (relative to the full sheet), not relative to the
// lower bound. Before a since-fixed bug, the upper bound was applied
// after the lower bound had already re-sliced the rows, so a
// first=3/last=4 range would incorrectly keep more rows than it
// should starting from the post-lower-bound slice instead of exactly
// rows 3 and 4 from the original sheet.
func TestImporter_Import_RowRange_AbsoluteEndRow(t *testing.T) {
	ctx := context.Background()

	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Row1")
	file.SetCellValue("Sheet1", "A2", "Row2")
	file.SetCellValue("Sheet1", "A3", "Row3")
	file.SetCellValue("Sheet1", "A4", "Row4")
	file.SetCellValue("Sheet1", "A5", "Row5")

	buf, _ := file.WriteToBuffer()
	file, _ = excelize.OpenReader(buf)

	// first=3, last=4 (1-based, inclusive) -> Row3, Row4
	importer := NewImporterWithFile(file, excel.WithRows(3, 4), excel.WithHeadings(false))
	if err := importer.Import(ctx); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	rows := importer.GetRows()
	if len(rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(rows))
	}
	if rows[0][0] != "Row3" || rows[1][0] != "Row4" {
		t.Errorf("rows = [%v, %v], want [Row3, Row4]", rows[0][0], rows[1][0])
	}
}

// TestImporter_Import_WithPreallocation tests preallocation feature.
func TestImporter_Import_WithPreallocation(t *testing.T) {
	ctx := context.Background()

	// Create test file
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Data1")
	file.SetCellValue("Sheet1", "A2", "Data2")
	file.SetCellValue("Sheet1", "A3", "Data3")

	// Save and reload
	buf, _ := file.WriteToBuffer()
	file, _ = excelize.OpenReader(buf)

	// Import with preallocation and excel.WithHeadings(false)
	importer := NewImporterWithFile(file, excel.WithPreallocation(true), excel.WithHeadings(false))
	err := importer.Import(ctx)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Should work the same with preallocation
	rows := importer.GetRows()
	if len(rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(rows))
	}
}
