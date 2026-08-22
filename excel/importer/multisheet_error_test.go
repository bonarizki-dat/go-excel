package importer

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestImportFromSheet_EmptyFilename verifies an empty filename is
// rejected before any file I/O is attempted.
func TestImportFromSheet_EmptyFilename(t *testing.T) {
	imp := NewImporter()
	if _, err := imp.ImportFromSheet(context.Background(), "", "Sheet1"); err == nil {
		t.Error("ImportFromSheet with an empty filename = nil error, want error")
	}
}

// TestImportFromSheet_EmptySheetName verifies an empty sheet name is
// rejected before any file I/O is attempted.
func TestImportFromSheet_EmptySheetName(t *testing.T) {
	imp := NewImporter()
	_, err := imp.ImportFromSheet(context.Background(), "report.xlsx", "")
	if !errors.Is(err, excel.ErrInvalidSheetName) {
		t.Errorf("ImportFromSheet with an empty sheet name = %v, want error wrapping ErrInvalidSheetName", err)
	}
}

// TestImportFromSheet_OpenFileFailure verifies a nonexistent file
// surfaces as an error from excelize.OpenFile.
func TestImportFromSheet_OpenFileFailure(t *testing.T) {
	imp := NewImporter()
	_, err := imp.ImportFromSheet(context.Background(), "/nonexistent/path/report.xlsx", "Sheet1")
	if err == nil {
		t.Error("ImportFromSheet with a nonexistent file = nil error, want error")
	}
}

// TestImportFromSheet_ImportFailureRestoresSheetName verifies a failing
// Import() call (triggered by a StartRow/EndRow combination that fails
// config validation) restores the importer's original config.SheetName
// instead of leaving the temporary override in place.
func TestImportFromSheet_ImportFailureRestoresSheetName(t *testing.T) {
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	imp := NewImporter(excel.WithSheetName("Original"))
	imp.config.StartRow = 10
	imp.config.EndRow = 1 // EndRow < StartRow fails config.Validate()
	defer imp.Close()

	_, err := imp.ImportFromSheet(context.Background(), tempFile, "Users")
	if err == nil {
		t.Fatal("ImportFromSheet with an invalid row range = nil error, want error")
	}
	if imp.config.SheetName != "Original" {
		t.Errorf("config.SheetName = %q after failed ImportFromSheet, want it restored to %q", imp.config.SheetName, "Original")
	}
}

// TestImportAllSheets_EmptyFilename verifies an empty filename is
// rejected before any file I/O is attempted.
func TestImportAllSheets_EmptyFilename(t *testing.T) {
	imp := NewImporter()
	if _, err := imp.ImportAllSheets(context.Background(), ""); err == nil {
		t.Error("ImportAllSheets with an empty filename = nil error, want error")
	}
}

// TestImportAllSheets_OpenFileFailure verifies a nonexistent file
// surfaces as an error from excelize.OpenFile.
func TestImportAllSheets_OpenFileFailure(t *testing.T) {
	imp := NewImporter()
	_, err := imp.ImportAllSheets(context.Background(), "/nonexistent/path/report.xlsx")
	if err == nil {
		t.Error("ImportAllSheets with a nonexistent file = nil error, want error")
	}
}

// TestImportAllSheets_ContextCanceledMidway verifies the per-sheet
// ctx.Done() check in ImportAllSheets' loop stops processing once the
// context is canceled between sheets, and restores config.SheetName.
func TestImportAllSheets_ContextCanceledMidway(t *testing.T) {
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	ctx, cancel := context.WithCancel(context.Background())
	imp := NewImporter(excel.WithSheetName("Original"))
	imp.On(excel.AfterImport, func(context.Context, *excel.Event) error {
		cancel() // cancel after the first sheet's import completes
		return nil
	})
	defer imp.Close()

	_, err := imp.ImportAllSheets(ctx, tempFile)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("ImportAllSheets() = %v, want error wrapping context.Canceled", err)
	}
	if imp.config.SheetName != "Original" {
		t.Errorf("config.SheetName = %q after canceled ImportAllSheets, want it restored to %q", imp.config.SheetName, "Original")
	}
}

// TestImportAllSheets_ImportFailureRestoresSheetName verifies a failing
// per-sheet Import() call surfaces as an ImportAllSheets error and
// restores config.SheetName.
func TestImportAllSheets_ImportFailureRestoresSheetName(t *testing.T) {
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	imp := NewImporter(excel.WithSheetName("Original"))
	imp.config.StartRow = 10
	imp.config.EndRow = 1 // EndRow < StartRow fails config.Validate()
	defer imp.Close()

	_, err := imp.ImportAllSheets(context.Background(), tempFile)
	if err == nil {
		t.Fatal("ImportAllSheets with an invalid row range = nil error, want error")
	}
	if imp.config.SheetName != "Original" {
		t.Errorf("config.SheetName = %q after failed ImportAllSheets, want it restored to %q", imp.config.SheetName, "Original")
	}
}

// TestImportAllSheets_NoSheetsFound verifies a workbook with zero
// sheets (not achievable via excelize's public API for a saved file in
// practice, so this uses a minimal empty xlsx-like file... actually
// excelize always creates at least one sheet, so this path is exercised
// indirectly is not possible; this test instead documents the check by
// asserting GetSheetList's normal (non-empty) behavior is what every
// other test relies on). Retained as a lightweight sanity check rather
// than a true reproduction of the zero-sheets branch, which excelize's
// API does not allow constructing.
func TestImportAllSheets_NoSheetsFound_Sanity(t *testing.T) {
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	imp := NewImporter()
	defer imp.Close()

	sheetsData, err := imp.ImportAllSheets(context.Background(), tempFile)
	if err != nil {
		t.Fatalf("ImportAllSheets: %v", err)
	}
	if len(sheetsData) == 0 {
		t.Error("expected at least one sheet")
	}
}

// TestImportAllSheetsCSV_ImportFromFileFailure verifies a failing
// ImportFromFile call (nonexistent CSV file) surfaces as an
// ImportAllSheets error via importAllSheetsCSV.
func TestImportAllSheetsCSV_ImportFromFileFailure(t *testing.T) {
	imp := NewImporter()
	defer imp.Close()

	_, err := imp.ImportAllSheets(context.Background(), "/nonexistent/path/report.csv")
	if err == nil {
		t.Error("ImportAllSheets (CSV) with a nonexistent file = nil error, want error")
	}
}

// TestGetSheetNames_NoFileOpened verifies GetSheetNames errors when no
// file has been opened (and format is not CSV).
func TestGetSheetNames_NoFileOpened(t *testing.T) {
	imp := NewImporter()
	if _, err := imp.GetSheetNames(); err == nil {
		t.Error("GetSheetNames with no file opened = nil error, want error")
	}
}

// TestGetSheetCount_NoFileOpened verifies GetSheetCount errors when no
// file has been opened (and format is not CSV).
func TestGetSheetCount_NoFileOpened(t *testing.T) {
	imp := NewImporter()
	if _, err := imp.GetSheetCount(); err == nil {
		t.Error("GetSheetCount with no file opened = nil error, want error")
	}
}

// TestSheetExists_NoFileOpened verifies SheetExists errors when no file
// has been opened (and format is not CSV).
func TestSheetExists_NoFileOpened(t *testing.T) {
	imp := NewImporter()
	if _, err := imp.SheetExists("Sheet1"); err == nil {
		t.Error("SheetExists with no file opened = nil error, want error")
	}
}

// TestSheetExists_InvalidSheetName verifies an invalid (not merely
// absent) sheet name surfaces file.GetSheetIndex's own error instead of
// a plain false.
func TestSheetExists_InvalidSheetName(t *testing.T) {
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	imp := NewImporter()
	defer imp.Close()

	if _, err := imp.ImportFromSheet(context.Background(), tempFile, "Users"); err != nil {
		t.Fatalf("ImportFromSheet: %v", err)
	}

	// A sheet name containing a colon is syntactically invalid to
	// excelize, unlike a merely-absent name.
	if _, err := imp.SheetExists("Bad:Name"); err == nil {
		t.Error("SheetExists with an invalid sheet name = nil error, want error")
	}
}
