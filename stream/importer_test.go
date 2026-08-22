package stream

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/xuri/excelize/v2"
)

// createImporterTestFile creates an in-memory Excel file for importer testing.
func createImporterTestFile(rows [][]any) (*bytes.Buffer, error) {
	file := excelize.NewFile()
	sheetName := "Sheet1"

	// Write rows
	for i, row := range rows {
		rowNum := i + 1
		for j, cell := range row {
			colName, _ := excelize.ColumnNumberToName(j + 1)
			cellAddr := fmt.Sprintf("%s%d", colName, rowNum)
			file.SetCellValue(sheetName, cellAddr, cell)
		}
	}

	// Write to buffer
	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return nil, err
	}

	return &buf, nil
}

// TestNewStreamImporter tests StreamImporter creation with defaults.
func TestNewStreamImporter(t *testing.T) {
	// Create test Excel file
	testData := [][]any{
		{"ID", "Name", "Email"},
		{1, "Alice", "alice@test.com"},
		{2, "Bob", "bob@test.com"},
	}
	buf, err := createImporterTestFile(testData)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create importer with defaults
	importer, err := NewStreamImporter(buf)

	// Verify no error
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}

	// Verify importer was created
	if importer == nil {
		t.Fatal("NewStreamImporter returned nil")
	}

	// Verify default values
	if importer.numWorkers != runtime.NumCPU() {
		t.Errorf("Expected default workers %d, got %d", runtime.NumCPU(), importer.numWorkers)
	}

	if importer.startRow != 1 {
		t.Errorf("Expected default start row 1, got %d", importer.startRow)
	}

	if importer.endRow != 0 {
		t.Errorf("Expected default end row 0, got %d", importer.endRow)
	}

	if importer.progressInterval != 100 {
		t.Errorf("Expected default progress interval 100, got %d", importer.progressInterval)
	}

	if importer.poolManager == nil {
		t.Error("Pool manager not initialized")
	}

	if importer.file == nil {
		t.Error("Excelize file not initialized")
	}

	if importer.withHeadings {
		t.Error("Headings should be disabled by default")
	}

	// Clean up
	importer.Close()
}

// TestNewStreamImporter_WithOptions tests StreamImporter creation with options.
func TestNewStreamImporter_WithOptions(t *testing.T) {
	// Create test Excel file with headings
	testData := [][]any{
		{"ID", "Name", "Email"},
		{1, "Alice", "alice@test.com"},
		{2, "Bob", "bob@test.com"},
	}
	buf, err := createImporterTestFile(testData)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create importer with all options
	importer, err := NewStreamImporter(buf,
		WithImportWorkers(4),
		WithImportHeadings(),
		WithImportSkipEmptyRows(true),
		WithImportRows(2, 10),
		WithImportProgress(func(_ uint64, _ string) {
			// Progress callback
		}, 50),
	)

	// Verify no error
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}

	// Verify options applied
	if importer.numWorkers != 4 {
		t.Errorf("Expected 4 workers, got %d", importer.numWorkers)
	}

	if !importer.withHeadings {
		t.Error("Headings should be enabled")
	}

	if !importer.skipEmptyRows {
		t.Error("Skip empty rows should be enabled")
	}

	if importer.startRow != 2 {
		t.Errorf("Expected start row 2, got %d", importer.startRow)
	}

	if importer.endRow != 10 {
		t.Errorf("Expected end row 10, got %d", importer.endRow)
	}

	if importer.progressCallback == nil {
		t.Error("Progress callback not set")
	}

	if importer.progressInterval != 50 {
		t.Errorf("Expected progress interval 50, got %d", importer.progressInterval)
	}

	// Verify headings were read
	if len(importer.headings) != 3 {
		t.Errorf("Expected 3 headings, got %d", len(importer.headings))
	}

	if importer.headings[0] != "ID" {
		t.Errorf("Expected heading 'ID', got '%s'", importer.headings[0])
	}

	// Clean up
	importer.Close()
}

func TestNewStreamImporter_WithImportSheetName(t *testing.T) {
	path := defaultMultiSheetFile(t)
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	importer, err := NewStreamImporter(f, WithImportSheetName("Orders"), WithImportHeadings())
	if err != nil {
		t.Fatalf("NewStreamImporter: %v", err)
	}
	defer importer.Close()

	if importer.sheetName != "Orders" {
		t.Errorf("sheetName = %q, want Orders", importer.sheetName)
	}
	headings := importer.GetHeadings()
	if len(headings) == 0 || headings[0] != "Order ID" {
		t.Errorf("headings = %v, want Order ID...", headings)
	}
}

// TestNewStreamImporter_InvalidFile tests creating importer with invalid file.
func TestNewStreamImporter_InvalidFile(t *testing.T) {
	// Create invalid data (not an Excel file)
	buf := bytes.NewBufferString("not an excel file")

	// Try to create importer
	importer, err := NewStreamImporter(buf)

	// Verify error occurred
	if err == nil {
		t.Error("Expected error for invalid file, got nil")
		if importer != nil {
			importer.Close()
		}
	}

	// Verify importer is nil
	if importer != nil {
		t.Error("Expected nil importer for invalid file")
		importer.Close()
	}
}

// TestWithImportWorkers tests WithImportWorkers option.
