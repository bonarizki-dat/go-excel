package exporter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// TestExporter_WithRowMap_TransformsDataRows verifies RowMap runs on
// every data row before it is written, and does not run on the header
// row.
func TestExporter_WithRowMap_TransformsDataRows(t *testing.T) {
	data := [][]any{
		{"Name", "Score"},
		{"alice", 10},
		{"bob", 20},
	}

	upper := func(row []any) ([]any, error) {
		name, _ := row[0].(string)
		return []any{fmt.Sprintf("UPPER:%s", name), row[1]}, nil
	}

	exp := exporter.NewExporter(data, excel.WithHeadings(true), excel.WithRowMap(upper))
	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}

	rows, err := exp.GetFile().GetRows("Sheet1")
	if err != nil {
		t.Fatalf("GetRows: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}
	if rows[0][0] != "Name" {
		t.Errorf("header row = %v, want unchanged by RowMap", rows[0])
	}
	if rows[1][0] != "UPPER:alice" || rows[2][0] != "UPPER:bob" {
		t.Errorf("data rows = %v, want RowMap applied to column 0", rows[1:])
	}
}

// TestExporter_WithRowMap_ErrorAbortsExport verifies an error from
// RowMap stops Export and surfaces that error to the caller, rather
// than being silently swallowed or subjected to StopOnError/
// ErrorCallback handling meant for sink write failures.
func TestExporter_WithRowMap_ErrorAbortsExport(t *testing.T) {
	boom := errors.New("boom")
	failing := func(_ []any) ([]any, error) {
		return nil, boom
	}

	data := [][]any{{"a"}, {"b"}}
	exp := exporter.NewExporter(data, excel.WithHeadings(false), excel.WithRowMap(failing))

	err := exp.Export(context.Background())
	if err == nil {
		t.Fatal("expected error from failing RowMap, got nil")
	}
	if !errors.Is(err, boom) {
		t.Errorf("Export() error = %v, want wrapping %v", err, boom)
	}
}
