package importer

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

var errHookFailed = errors.New("before-import hook failed")

// createSimpleXLSXFile creates a minimal one-sheet xlsx file (default
// sheet name "Sheet1") and returns its path.
func createSimpleXLSXFile(t *testing.T) string {
	t.Helper()
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "id")
	f.SetCellValue("Sheet1", "A2", "1")

	path := filepath.Join(t.TempDir(), "simple.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	return path
}

// TestImportFromFile_StatNonNotExistError verifies an os.Stat failure
// that is not "file does not exist" (here, a path with a regular file
// as a non-final component) surfaces via the non-IsNotExist branch.
func TestImportFromFile_StatNonNotExistError(t *testing.T) {
	regularFile := filepath.Join(t.TempDir(), "notadir.txt")
	if err := os.WriteFile(regularFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	imp := NewImporter()
	err := imp.ImportFromFile(context.Background(), filepath.Join(regularFile, "sub.xlsx"))
	if err == nil {
		t.Error("ImportFromFile through a non-directory path component = nil error, want error")
	}
}

// TestImportXLSXFile_OpenFileFailure verifies a malformed xlsx file
// (valid on disk, invalid as a zip/xlsx) surfaces an OpenFile error and
// logs it.
func TestImportXLSXFile_OpenFileFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.xlsx")
	if err := os.WriteFile(path, []byte("not a real xlsx file"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	logger := &mockLogger{}
	imp := NewImporter(excel.WithLogger(logger))
	if err := imp.ImportFromFile(context.Background(), path); err == nil {
		t.Error("ImportFromFile with a corrupt xlsx file = nil error, want error")
	}
	if len(logger.logs) == 0 {
		t.Error("expected the OpenFile failure to be logged")
	}
}

// TestImportXLSXFile_ImportFailure verifies a valid xlsx file whose
// configured sheet name does not exist surfaces an Import() error
// (wrapped with "import failed") and logs it.
func TestImportXLSXFile_ImportFailure(t *testing.T) {
	path := createSimpleXLSXFile(t)

	logger := &mockLogger{}
	imp := NewImporter(excel.WithSheetName("NoSuchSheet"), excel.WithLogger(logger))
	if err := imp.ImportFromFile(context.Background(), path); err == nil {
		t.Error("ImportFromFile with a nonexistent configured sheet = nil error, want error")
	}
	found := false
	for _, l := range logger.logs {
		if l == "ERROR: Import failed" {
			found = true
		}
	}
	if !found {
		t.Errorf("logs = %v, want an 'Import failed' error log", logger.logs)
	}
}

// TestImportCSVFile_OpenFailure verifies a CSV file that exists (so
// os.Stat succeeds) but cannot be opened for reading (permission
// denied) surfaces an Open error.
func TestImportCSVFile_OpenFailure(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root bypasses file permission checks")
	}

	path := filepath.Join(t.TempDir(), "noperm.csv")
	if err := os.WriteFile(path, []byte("a,b\n1,2\n"), 0o000); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	imp := NewImporter()
	if err := imp.ImportFromFile(context.Background(), path); err == nil {
		t.Error("ImportFromFile with a permission-denied CSV file = nil error, want error")
	}
}

// TestImportCSVFile_ImportFailure verifies a CSV import failure (a
// BeforeImport hook error) surfaces wrapped with "import failed" and is
// logged.
func TestImportCSVFile_ImportFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.csv")
	if err := os.WriteFile(path, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	logger := &mockLogger{}
	imp := NewImporter(excel.WithLogger(logger))
	imp.On(excel.BeforeImport, func(context.Context, *excel.Event) error {
		return errHookFailed
	})

	if err := imp.ImportFromFile(context.Background(), path); err == nil {
		t.Error("ImportFromFile (CSV) with a failing hook = nil error, want error")
	}
	found := false
	for _, l := range logger.logs {
		if l == "ERROR: Import failed" {
			found = true
		}
	}
	if !found {
		t.Errorf("logs = %v, want an 'Import failed' error log", logger.logs)
	}
}

// TestImport_GetRowsFailure verifies a configured sheet name that does
// not exist in an already-open file surfaces excelize.GetRows' error
// through Import().
func TestImport_GetRowsFailure(t *testing.T) {
	f := excelize.NewFile()
	imp := NewImporterWithFile(f, excel.WithSheetName("NoSuchSheet"))

	if err := imp.Import(context.Background()); err == nil {
		t.Error("Import with a nonexistent sheet name = nil error, want error")
	}
}

// TestImportFromReader_CSVImportFailureLogsAndWraps verifies
// ImportFromReader's CSV branch logs and wraps an importCSVFromReader
// failure.
func TestImportFromReader_CSVImportFailureLogsAndWraps(t *testing.T) {
	logger := &mockLogger{}
	imp := NewImporter(excel.WithFormat(excel.FileFormatCSV), excel.WithLogger(logger))
	imp.On(excel.BeforeImport, func(context.Context, *excel.Event) error {
		return errHookFailed
	})

	err := imp.ImportFromReader(context.Background(), bytes.NewReader([]byte("a,b\n1,2\n")))
	if err == nil {
		t.Error("ImportFromReader (CSV) with a failing hook = nil error, want error")
	}
	found := false
	for _, l := range logger.logs {
		if l == "ERROR: Import failed" {
			found = true
		}
	}
	if !found {
		t.Errorf("logs = %v, want an 'Import failed' error log", logger.logs)
	}
}

// TestImportXLSXFromReader_OpenReaderFailure verifies malformed xlsx
// bytes surface an OpenReader error and log it.
func TestImportXLSXFromReader_OpenReaderFailure(t *testing.T) {
	logger := &mockLogger{}
	imp := NewImporter(excel.WithLogger(logger))

	err := imp.ImportFromReader(context.Background(), bytes.NewReader([]byte("not a real xlsx")))
	if err == nil {
		t.Error("ImportFromReader with malformed xlsx bytes = nil error, want error")
	}
	if len(logger.logs) == 0 {
		t.Error("expected the OpenReader failure to be logged")
	}
}

// TestImportXLSXFromReader_ImportFailure verifies a valid xlsx byte
// stream whose configured sheet name does not exist surfaces an
// Import() error (wrapped with "import failed") and logs it.
func TestImportXLSXFromReader_ImportFailure(t *testing.T) {
	f := excelize.NewFile()
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	logger := &mockLogger{}
	imp := NewImporter(excel.WithSheetName("NoSuchSheet"), excel.WithLogger(logger))
	if err := imp.ImportFromReader(context.Background(), &buf); err == nil {
		t.Error("ImportFromReader with a nonexistent configured sheet = nil error, want error")
	}
	found := false
	for _, l := range logger.logs {
		if l == "ERROR: Import failed" {
			found = true
		}
	}
	if !found {
		t.Errorf("logs = %v, want an 'Import failed' error log", logger.logs)
	}
}
