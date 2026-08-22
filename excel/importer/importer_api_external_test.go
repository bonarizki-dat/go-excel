package importer_test

import (
	"context"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

func TestNewImporter(t *testing.T) {
	imp := importer.NewImporter()
	if imp == nil {
		t.Fatal("NewImporter should not return nil")
	}
	if imp.GetConfig() == nil {
		t.Error("Importer config should not be nil")
	}
	if imp.GetErrors() == nil {
		t.Error("Importer errors slice should be initialized")
	}

	imp = importer.NewImporter(
		excel.WithSheetName("TestSheet"),
		excel.WithHeadings(true),
	)
	if imp.GetConfig().SheetName != "TestSheet" {
		t.Errorf("SheetName = %v, want TestSheet", imp.GetConfig().SheetName)
	}
	if !imp.GetConfig().WithHeadings {
		t.Error("WithHeadings should be true")
	}
}

func TestNewImporterWithFile(t *testing.T) {
	file := excelize.NewFile()

	imp := importer.NewImporterWithFile(file)
	if imp == nil {
		t.Fatal("NewImporterWithFile should not return nil")
	}

	imp = importer.NewImporterWithFile(file, excel.WithSheetName("Data"))
	if imp.GetConfig().SheetName != "Data" {
		t.Errorf("SheetName = %v, want Data", imp.GetConfig().SheetName)
	}
}

func TestImporter_GetRows(t *testing.T) {
	ctx := context.Background()

	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "test")

	// Save and reload to ensure data is readable
	buf, _ := file.WriteToBuffer()
	file, _ = excelize.OpenReader(buf)

	imp := importer.NewImporterWithFile(file, excel.WithHeadings(false))
	err := imp.Import(ctx)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	rows := imp.GetRows()
	if rows == nil {
		t.Error("GetRows should not return nil")
	}
	if len(rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(rows))
	}
}

// TestImporter_GetHeadings tests GetHeadings method.
func TestImporter_GetHeadings(t *testing.T) {
	imp := importer.NewImporter()

	headings := imp.GetHeadings()
	if headings == nil {
		t.Error("GetHeadings should not return nil")
	}
	if len(headings) != 0 {
		t.Error("GetHeadings should return empty slice initially")
	}
}

// TestImporter_GetConfig tests GetConfig method.
func TestImporter_GetConfig(t *testing.T) {
	imp := importer.NewImporter(excel.WithSheetName("Test"))

	config := imp.GetConfig()
	if config == nil {
		t.Fatal("GetConfig should not return nil")
	}
	if config.SheetName != "Test" {
		t.Errorf("Config SheetName = %v, want Test", config.SheetName)
	}

	// Verify it's a clone (modifying shouldn't affect importer)
	config.SheetName = "Modified"
	if imp.GetConfig().SheetName == "Modified" {
		t.Error("GetConfig should return a clone, not the original")
	}
}

// TestImporter_GetErrors tests GetErrors method.
func TestImporter_GetErrors(t *testing.T) {
	imp := importer.NewImporter()

	errors := imp.GetErrors()
	if errors == nil {
		t.Error("GetErrors should not return nil")
	}
	if len(errors) != 0 {
		t.Error("GetErrors should return empty slice initially")
	}
}

// TestImporter_Close tests Close method.
func TestImporter_Close(t *testing.T) {
	file := excelize.NewFile()
	imp := importer.NewImporterWithFile(file)

	err := imp.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
