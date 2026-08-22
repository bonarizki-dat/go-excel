package importer

import (
	"context"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// TestBasicImporter_GetHeadings tests GetHeadings method.
func TestBasicImporter_GetHeadings(t *testing.T) {
	// Create Excel file with headings
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Name")
	file.SetCellValue("Sheet1", "B1", "Age")
	file.SetCellValue("Sheet1", "C1", "Email")
	file.SetCellValue("Sheet1", "A2", "Alice")
	file.SetCellValue("Sheet1", "B2", 30)

	// Create importer with headings enabled
	imp := NewImporterWithFile(file, excel.WithHeadings(true))

	// Import data
	err := imp.Import(context.Background())
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Get headings
	headings := imp.GetHeadings()

	// Verify headings
	if len(headings) != 3 {
		t.Errorf("Got %d headings, expected 3", len(headings))
	}
	if headings[0] != "Name" {
		t.Errorf("Heading 0 is %q, expected \"Name\"", headings[0])
	}
}

// TestBasicImporter_GetHeadings_ReturnsIndependentCopy verifies that
// mutating the slice returned by GetHeadings does not affect a later
// call, mirroring the guarantee GetRows already makes.
func TestBasicImporter_GetHeadings_ReturnsIndependentCopy(t *testing.T) {
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Name")
	file.SetCellValue("Sheet1", "B1", "Age")
	file.SetCellValue("Sheet1", "A2", "Alice")
	file.SetCellValue("Sheet1", "B2", 30)

	imp := NewImporterWithFile(file, excel.WithHeadings(true))
	if err := imp.Import(context.Background()); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	first := imp.GetHeadings()
	first[0] = "Mutated"

	second := imp.GetHeadings()
	if second[0] != "Name" {
		t.Errorf("GetHeadings() second call returned %q, expected \"Name\" to be unaffected by mutating the first call's result", second[0])
	}
}

// TestBasicImporter_GetHeadings_NoHeadings tests GetHeadings without headings.
func TestBasicImporter_GetHeadings_NoHeadings(t *testing.T) {
	// Create Excel file
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Alice")
	file.SetCellValue("Sheet1", "B1", 30)

	// Create importer without headings
	imp := NewImporterWithFile(file, excel.WithHeadings(false))

	// Import data
	imp.Import(context.Background())

	// Get headings (should be empty)
	headings := imp.GetHeadings()

	if len(headings) != 0 {
		t.Errorf("Expected empty headings, got %d items", len(headings))
	}
}

// TestBasicImporter_GetErrors tests GetErrors method.
func TestBasicImporter_GetErrors(t *testing.T) {
	// Create Excel file
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Name")
	file.SetCellValue("Sheet1", "A2", "Alice")

	// Create importer
	imp := NewImporterWithFile(file)

	// Import (should succeed with no errors)
	imp.Import(context.Background())

	// Get errors
	errors := imp.GetErrors()

	// Should have no errors
	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(errors))
	}
}

// TestBasicImporter_GetConfig tests GetConfig method.
func TestBasicImporter_GetConfig(t *testing.T) {
	// Create importer with specific config
	imp := NewImporter(
		excel.WithHeadings(true),
		excel.WithSheetName("TestSheet"),
	)

	// Get config
	cfg := imp.GetConfig()

	// Verify config
	if cfg == nil {
		t.Fatal("GetConfig returned nil")
	}
	if !cfg.WithHeadings {
		t.Error("WithHeadings should be true")
	}
	if cfg.SheetName != "TestSheet" {
		t.Errorf("SheetName is %q, expected \"TestSheet\"", cfg.SheetName)
	}
}

// TestBasicImporter_Close tests Close method.
func TestBasicImporter_Close(t *testing.T) {
	// Create Excel file
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Alice")

	// Create importer
	imp := NewImporterWithFile(file)

	// Import data
	err := imp.Import(context.Background())
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Close should work without error
	err = imp.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Calling Close again should be safe
	err = imp.Close()
	if err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

// Test removed - covered by other tests

// TestBasicImporter_Import_ContextCancellation tests context cancellation.
func TestBasicImporter_Import_ContextCancellation(t *testing.T) {
	// Create Excel file
	file := excelize.NewFile()
	for i := 1; i <= 1000; i++ {
		file.SetCellValue("Sheet1", "A"+string(rune(i)), i)
	}

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create importer
	imp := NewImporterWithFile(file)

	// Import should respect cancellation
	err := imp.Import(ctx)
	if err == nil {
		t.Error("Import should fail with canceled context")
	}
}

// TestBasicImporter_Import_SkipEmptyRows tests skipping empty rows.
func TestBasicImporter_Import_SkipEmptyRows(t *testing.T) {
	// Create Excel file with empty rows
	file := excelize.NewFile()
	file.SetCellValue("Sheet1", "A1", "Name")
	file.SetCellValue("Sheet1", "A2", "Alice")
	// Row 3 is empty
	file.SetCellValue("Sheet1", "A4", "Bob")
	// Row 5 is empty

	// Create importer with skip empty rows
	imp := NewImporterWithFile(file, excel.WithSkipEmptyRows(true))

	// Import data
	err := imp.Import(context.Background())
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Get rows
	rows := imp.GetRows()

	// Should skip empty rows and get only non-empty ones
	if len(rows) < 2 {
		t.Errorf("Got %d rows, expected at least 2 non-empty rows", len(rows))
	}
}
