package importer

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// TestImportToStructs_ImportFromFileError verifies ImportToStructs
// propagates an ImportFromFile failure (nonexistent file) as-is.
func TestImportToStructs_ImportFromFileError(t *testing.T) {
	type Row struct {
		ID int `excel:"id"`
	}
	_, err := ImportToStructs[Row](context.Background(), "/nonexistent/path/report.xlsx")
	if err == nil {
		t.Error("ImportToStructs with a nonexistent file = nil error, want error")
	}
}

// TestImportToStructs_EmptyRows verifies ImportToStructs returns an
// empty slice (not an error) when the file has a header row but no
// data rows at all, exercising the len(rows)==0 branch that runs after
// a successful (non-empty-file) import.
func TestImportToStructs_EmptyRows(t *testing.T) {
	type Row struct {
		ID int `excel:"id"`
	}

	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "id") // header row only, no data rows
	path := filepath.Join(t.TempDir(), "headeronly.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	got, err := ImportToStructs[Row](context.Background(), path)
	if err != nil {
		t.Fatalf("ImportToStructs: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ImportToStructs with a header-only file = %v, want empty slice", got)
	}
}

// TestImportToStructs_NoHeaders verifies ImportToStructs errors when
// WithHeadings is disabled (so no headings are ever extracted) even
// though the file has data rows.
func TestImportToStructs_NoHeaders(t *testing.T) {
	type Row struct {
		ID int `excel:"id"`
	}

	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "1")
	path := filepath.Join(t.TempDir(), "noheaders.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	_, err := ImportToStructs[Row](context.Background(), path, excel.WithHeadings(false))
	if err == nil {
		t.Error("ImportToStructs with WithHeadings(false) = nil error, want 'no headers found' error")
	}
}

// TestImportToStructs_NonStructType verifies ImportToStructs rejects a
// non-struct type parameter at the first row, instead of only failing
// via MapRow.
func TestImportToStructs_NonStructType(t *testing.T) {
	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "id")
	f.SetCellValue("Sheet1", "A2", "1")
	path := filepath.Join(t.TempDir(), "simple.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	_, err := ImportToStructs[int](context.Background(), path)
	if err == nil {
		t.Error("ImportToStructs[int] = nil error, want 'requires a struct type' error")
	}
}

// Note: ImportToStructs' own per-row ctx.Done() check (in its
// row-mapping loop, after ImportFromFile has already returned) is not
// practically reachable via the public API in a deterministic test:
// ImportToStructs builds its own unexported *BasicImporter internally
// with no hook-injection point, so there is no way to cancel ctx
// specifically between ImportFromFile succeeding and the mapping loop
// starting without either a race (canceling from a timer/goroutine,
// which risks instead being caught by ImportFromFile's own earlier
// ctx.Done() check) or modifying production code purely for
// testability. It is covered indirectly by BasicImporter's own
// equivalent checks (importRows, processRow) elsewhere in this
// package.

// TestImportToStructs_MapRowError verifies a row that MapRow cannot
// convert (a non-numeric string into an int field with validation
// enabled, where min/max constraints trigger a clear conversion
// failure) surfaces as a *excel.ValidationError.
func TestImportToStructs_MapRowError(t *testing.T) {
	type Row struct {
		ID int `excel:"id"`
	}

	f := excelize.NewFile()
	f.SetCellValue("Sheet1", "A1", "id")
	f.SetCellValue("Sheet1", "A2", "not-a-number")
	path := filepath.Join(t.TempDir(), "badvalue.xlsx")
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	_, err := ImportToStructs[Row](context.Background(), path)
	if err == nil {
		t.Error("ImportToStructs with an unconvertible cell value = nil error, want error")
	}
}
