package stream

import (
	"path/filepath"
	"testing"

	"github.com/bonarizki-dat/go-excel/internal/concurrency"
	"github.com/xuri/excelize/v2"
)

// sheetSpec describes one worksheet written by writeSheetsXLSX.
type sheetSpec struct {
	name string
	rows [][]any
}

// writeSheetsXLSX writes sheets to a temporary .xlsx and returns its path.
// The first spec renames the default sheet so the workbook order matches
// the spec order rather than leaving a leftover "Sheet1".
func writeSheetsXLSX(t *testing.T, name string, sheets []sheetSpec) string {
	t.Helper()

	if len(sheets) == 0 {
		t.Fatal("writeSheetsXLSX requires at least one sheet")
	}

	f := excelize.NewFile()
	if err := f.SetSheetName(f.GetSheetName(0), sheets[0].name); err != nil {
		t.Fatalf("rename default sheet: %v", err)
	}
	writeSheetRows(t, f, sheets[0].name, sheets[0].rows)

	for _, spec := range sheets[1:] {
		if _, err := f.NewSheet(spec.name); err != nil {
			t.Fatalf("create sheet %q: %v", spec.name, err)
		}
		writeSheetRows(t, f, spec.name, spec.rows)
	}

	path := filepath.Join(t.TempDir(), name)
	if err := f.SaveAs(path); err != nil {
		t.Fatalf("save %s: %v", path, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close workbook: %v", err)
	}
	return path
}

func writeSheetRows(t *testing.T, f *excelize.File, sheet string, rows [][]any) {
	t.Helper()
	for r, row := range rows {
		for c, value := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				t.Fatalf("cell name: %v", err)
			}
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				t.Fatalf("set %s!%s: %v", sheet, cell, err)
			}
		}
	}
}

// defaultMultiSheetFile writes Users / Orders / Products with heading
// rows, matching the fixture used by excel/importer multi-sheet tests.
func defaultMultiSheetFile(t *testing.T) string {
	t.Helper()
	return writeSheetsXLSX(t, "multisheet.xlsx", []sheetSpec{
		{name: "Users", rows: [][]any{
			{"ID", "Name", "Email"},
			{1, "John Doe", "john@example.com"},
			{2, "Jane Smith", "jane@example.com"},
		}},
		{name: "Orders", rows: [][]any{
			{"Order ID", "User ID", "Amount"},
			{101, 1, 250.50},
			{102, 2, 175.25},
			{103, 1, 99.99},
		}},
		{name: "Products", rows: [][]any{
			{"Product ID", "Name", "Price"},
			{1001, "Laptop", 1200.00},
			{1002, "Mouse", 25.00},
		}},
	})
}

// newSheetOnlyImporter builds an importer that has never opened a file,
// so ImportFromSheet / ImportAllSheets / GetSheetNames can be exercised
// without a constructor reader. Defaults match NewStreamImporter.
func newSheetOnlyImporter(opts ...StreamImportOption) *StreamImporter {
	si := &StreamImporter{
		startRow:         1,
		progressInterval: 100,
		poolManager:      concurrency.NewPoolManager(),
	}
	for _, opt := range opts {
		opt(si)
	}
	return si
}

func collectRows() (func([]any) error, *[][]any) {
	var got [][]any
	return func(row []any) error {
		cp := make([]any, len(row))
		copy(cp, row)
		got = append(got, cp)
		return nil
	}, &got
}
