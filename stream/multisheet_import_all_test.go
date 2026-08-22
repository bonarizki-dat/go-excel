package stream

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

func TestImportAllSheets_OrderAndCounts(t *testing.T) {
	path := defaultMultiSheetFile(t)
	var sheets []string
	var rows int
	err := newSheetOnlyImporter().ImportAllSheets(context.Background(), path,
		func(name string) error {
			sheets = append(sheets, name)
			return nil
		},
		func([]any) error {
			rows++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ImportAllSheets: %v", err)
	}
	want := []string{"Users", "Orders", "Products"}
	if len(sheets) != 3 || sheets[0] != want[0] || sheets[1] != want[1] || sheets[2] != want[2] {
		t.Errorf("sheets = %v, want %v", sheets, want)
	}
	if rows != 3+4+3 {
		t.Errorf("rows = %d, want 10", rows)
	}
}

func TestImportAllSheets_NilSheetHandler(t *testing.T) {
	path := defaultMultiSheetFile(t)
	var rows int
	err := newSheetOnlyImporter().ImportAllSheets(context.Background(), path, nil, func([]any) error {
		rows++
		return nil
	})
	if err != nil {
		t.Fatalf("ImportAllSheets: %v", err)
	}
	if rows != 10 {
		t.Errorf("rows = %d, want 10", rows)
	}
}

func TestImportAllSheets_SheetHandlerError(t *testing.T) {
	path := defaultMultiSheetFile(t)
	boom := errors.New("no orders")
	var rows int
	err := newSheetOnlyImporter().ImportAllSheets(context.Background(), path,
		func(name string) error {
			if name == "Orders" {
				return boom
			}
			return nil
		},
		func([]any) error {
			rows++
			return nil
		},
	)
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("error = %v, want wrapped no orders", err)
	}
	if rows != 3 {
		t.Errorf("rows before failure = %d, want 3 (Users only)", rows)
	}
}

func TestImportAllSheets_RowHandlerErrorStops(t *testing.T) {
	path := defaultMultiSheetFile(t)
	boom := errors.New("bad row")
	var sheets []string
	err := newSheetOnlyImporter().ImportAllSheets(context.Background(), path,
		func(name string) error {
			sheets = append(sheets, name)
			return nil
		},
		func([]any) error { return boom },
	)
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("error = %v, want wrapped bad row", err)
	}
	if len(sheets) != 1 || sheets[0] != "Users" {
		t.Errorf("sheets visited = %v, want [Users]", sheets)
	}
}

func TestImportAllSheets_EmptyAndPopulated(t *testing.T) {
	path := writeSheetsXLSX(t, "mix.xlsx", []sheetSpec{
		{name: "Empty", rows: nil},
		{name: "Data", rows: [][]any{{"x"}, {"y"}}},
	})
	var seen []string
	var rows int
	err := newSheetOnlyImporter().ImportAllSheets(context.Background(), path,
		func(name string) error {
			seen = append(seen, name)
			return nil
		},
		func([]any) error {
			rows++
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ImportAllSheets: %v", err)
	}
	if len(seen) != 2 {
		t.Errorf("sheets = %v, want Empty then Data", seen)
	}
	if rows != 2 {
		t.Errorf("rows = %d, want 2", rows)
	}
}

func TestImportAllSheets_Validation(t *testing.T) {
	si := newSheetOnlyImporter()
	if err := si.ImportAllSheets(context.Background(), "", nil, func([]any) error { return nil }); err == nil {
		t.Fatal("empty filename: want error")
	}
	if err := si.ImportAllSheets(context.Background(), defaultMultiSheetFile(t), nil, nil); err == nil {
		t.Fatal("nil rowHandler: want error")
	}
}

func TestGetSheetNames(t *testing.T) {
	path := defaultMultiSheetFile(t)
	names, err := newSheetOnlyImporter().GetSheetNames(path)
	if err != nil {
		t.Fatalf("GetSheetNames: %v", err)
	}
	if len(names) != 3 || names[0] != "Users" || names[1] != "Orders" || names[2] != "Products" {
		t.Errorf("names = %v, want [Users Orders Products]", names)
	}
}

func TestGetSheetNames_Errors(t *testing.T) {
	si := newSheetOnlyImporter()
	if _, err := si.GetSheetNames(""); err == nil {
		t.Fatal("empty filename: want error")
	}
	if _, err := si.GetSheetNames(filepath.Join(t.TempDir(), "missing.xlsx")); err == nil {
		t.Fatal("missing file: want error")
	}
}

func TestMultiSheetImport_RejectsCSV(t *testing.T) {
	csvPath := filepath.Join(t.TempDir(), "data.csv")
	if err := os.WriteFile(csvPath, []byte("a,b\n1,2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	si := newSheetOnlyImporter()
	handler := func([]any) error { return nil }

	if err := si.ImportFromSheet(context.Background(), csvPath, "Sheet1", handler); err == nil {
		t.Fatal("ImportFromSheet CSV: want error")
	}
	if err := si.ImportAllSheets(context.Background(), csvPath, nil, handler); err == nil {
		t.Fatal("ImportAllSheets CSV: want error")
	}
	if _, err := si.GetSheetNames(csvPath); err == nil {
		t.Fatal("GetSheetNames CSV: want error")
	}

	xlsx := defaultMultiSheetFile(t)
	csvSI := newSheetOnlyImporter(WithImportFormat(config.FileFormatCSV))
	if err := csvSI.ImportFromSheet(context.Background(), xlsx, "Users", handler); err == nil {
		t.Fatal("ImportFromSheet with FileFormatCSV: want error")
	}
}
