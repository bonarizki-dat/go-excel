package exporter_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// startCellWarnLogger captures the "features" field of the one Warn call
// warnCSVDegradation makes, so tests can assert a specific feature name
// was reported as dropped.
type startCellWarnLogger struct {
	features []string
}

func (l *startCellWarnLogger) Debug(_ string, _ ...any) {}
func (l *startCellWarnLogger) Info(_ string, _ ...any)  {}
func (l *startCellWarnLogger) Error(_ string, _ ...any) {}
func (l *startCellWarnLogger) Warn(_ string, fields ...any) {
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i] != "features" {
			continue
		}
		if fs, ok := fields[i+1].([]string); ok {
			l.features = fs
		}
	}
}

// TestExporter_WithCustomStartCell_ShiftsRowsAndColumns verifies XLSX
// export writes the header at the configured start cell and every
// subsequent row/column shifted by the same offset, leaving the cells
// before it untouched.
func TestExporter_WithCustomStartCell_ShiftsRowsAndColumns(t *testing.T) {
	data := [][]any{
		{"Name", "Score"},
		{"Alice", 10},
	}

	exp := exporter.NewExporter(data, excel.WithHeadings(true), excel.WithCustomStartCell("B3"))
	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}

	file := exp.GetFile()

	if v, _ := file.GetCellValue("Sheet1", "A1"); v != "" {
		t.Errorf("A1 = %q, want empty (before the start cell)", v)
	}
	if v, _ := file.GetCellValue("Sheet1", "B3"); v != "Name" {
		t.Errorf("B3 = %q, want %q (header at StartCell)", v, "Name")
	}
	if v, _ := file.GetCellValue("Sheet1", "C3"); v != "Score" {
		t.Errorf("C3 = %q, want %q", v, "Score")
	}
	if v, _ := file.GetCellValue("Sheet1", "B4"); v != "Alice" {
		t.Errorf("B4 = %q, want %q (data row shifted by the same offset)", v, "Alice")
	}
	if v, _ := file.GetCellValue("Sheet1", "C4"); v != "10" {
		t.Errorf("C4 = %q, want %q", v, "10")
	}
}

// TestExporter_WithCustomStartCell_ProgressCountsFromOne verifies the
// row numbers ProgressCallback observes are unaffected by StartCell:
// they still count 1..N relative to the data, not the sheet.
func TestExporter_WithCustomStartCell_ProgressCountsFromOne(t *testing.T) {
	data := [][]any{{"a"}, {"b"}, {"c"}}

	var lastProcessed int64
	exp := exporter.NewExporter(data,
		excel.WithHeadings(false),
		excel.WithCustomStartCell("D5"),
		excel.WithProgress(func(processed, _ int64) { lastProcessed = processed }),
	)
	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}
	if lastProcessed != 3 {
		t.Errorf("final processed = %d, want 3 (unshifted by StartCell)", lastProcessed)
	}
}

// TestExporter_WithCustomStartCell_InvalidCellRejected verifies a
// malformed StartCell surfaces as a Config.Validate error from Export,
// rather than a confusing failure deep inside the sink.
func TestExporter_WithCustomStartCell_InvalidCellRejected(t *testing.T) {
	exp := exporter.NewExporter([][]any{{"a"}}, excel.WithCustomStartCell("not-a-cell"))
	if err := exp.Export(context.Background()); err == nil {
		t.Error("Export with an invalid StartCell = nil error, want error")
	}
}

// TestExporter_WithCustomStartCell_IgnoredForCSV verifies StartCell has
// no effect on CSV export, which has no concept of a starting cell, and
// that it is reported by the CSV-degradation warning instead of failing
// the export.
func TestExporter_WithCustomStartCell_IgnoredForCSV(t *testing.T) {
	data := [][]any{{"Name"}, {"Alice"}}

	logger := &startCellWarnLogger{}
	exp := exporter.NewExporter(data,
		excel.WithFormat(config.FileFormatCSV),
		excel.WithCustomStartCell("B3"),
		excel.WithLogger(logger),
	)

	var buf bytes.Buffer
	if err := exp.ExportToWriter(context.Background(), &buf); err != nil {
		t.Fatalf("ExportToWriter: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("CSV output is empty")
	}

	found := false
	for _, f := range logger.features {
		if f == "start cell" {
			found = true
		}
	}
	if !found {
		t.Errorf("warned features = %v, want to include %q", logger.features, "start cell")
	}
}
