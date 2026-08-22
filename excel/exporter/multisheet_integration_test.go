package exporter

import (
	"context"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

type invoiceRow struct {
	ID     int
	Amount float64
}

// TestExportToSheets_StructRowsWithHeaderStyle verifies that rows
// produced by the exporter's struct-reflection path (the same one
// Export uses for its own data field) combine correctly with a header
// style on the multi-sheet path. ExportToSheets' Sheet.Data takes
// [][]any directly rather than running its own reflection pass, so the
// struct slice is converted through prepareData first, the same
// conversion Export() would apply if this data were passed to it
// directly.
func TestExportToSheets_StructRowsWithHeaderStyle(t *testing.T) {
	data := []invoiceRow{{ID: 1, Amount: 10.5}, {ID: 2, Amount: 20.25}}

	rows, err := NewExporter(data).prepareData(context.Background())
	if err != nil {
		t.Fatalf("prepareData: %v", err)
	}
	rows = append([][]any{{"ID", "Amount"}}, rows...)

	exp := NewExporter(nil)
	exp.SetHeaderStyle(excel.NewStyle().Bold(true))

	sheets := []excel.Sheet{{Name: "Invoices", Data: rows}}
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets: %v", err)
	}

	got, err := exp.GetFile().GetRows("Invoices")
	if err != nil {
		t.Fatalf("GetRows: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("rows = %d, want 3 (header + 2 struct rows)", len(got))
	}
	if got[1][0] != "1" || got[2][0] != "2" {
		t.Errorf("data rows = %v, want ID column 1 then 2", got)
	}

	styleID, err := exp.GetFile().GetCellStyle("Invoices", "A1")
	if err != nil {
		t.Fatalf("GetCellStyle: %v", err)
	}
	if styleID == 0 {
		t.Error("header style ID = 0, want the registered bold style to be applied")
	}
}

// TestExportIntegration_EventsAndMultiSheet verifies BeforeExport/
// AfterExport/BeforeRowExport/AfterRowExport fire the expected number
// of times for a normal Export() call, and that ExportToSheets on the
// same exporter and hooks dispatches none of them: ExportToSheets does
// not go through Export()'s dispatcher calls at any level (operation,
// sheet, or row), a contract documented on ExportToSheets' godoc.
func TestExportIntegration_EventsAndMultiSheet(t *testing.T) {
	exp := NewExporter([][]any{{"Name"}, {"Alice"}})

	counts := make(map[excel.EventType]int)
	for _, et := range []excel.EventType{excel.BeforeExport, excel.AfterExport, excel.BeforeRowExport, excel.AfterRowExport} {
		eventType := et
		exp.On(eventType, func(context.Context, *excel.Event) error {
			counts[eventType]++
			return nil
		})
	}

	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if counts[excel.BeforeExport] != 1 || counts[excel.AfterExport] != 1 {
		t.Errorf("operation-level event counts = %v, want BeforeExport=1 AfterExport=1", counts)
	}
	if counts[excel.BeforeRowExport] != 2 || counts[excel.AfterRowExport] != 2 {
		t.Errorf("row-level event counts = %v, want 2 each (header row + one data row)", counts)
	}

	for et := range counts {
		delete(counts, et)
	}

	sheets := []excel.Sheet{{Name: "Sheet1", Data: [][]any{{"a"}}}}
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets: %v", err)
	}
	for et, n := range counts {
		if n != 0 {
			t.Errorf("ExportToSheets dispatched %s %d time(s), want 0", et, n)
		}
	}
}
