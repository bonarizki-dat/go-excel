package exporter

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// degradationLogger records Warn calls so tests can assert a feature was
// dropped exactly once, never once per row.
type degradationLogger struct {
	warnMsgs []string
}

func (l *degradationLogger) Debug(_ string, _ ...any) {}
func (l *degradationLogger) Info(_ string, _ ...any)  {}
func (l *degradationLogger) Error(_ string, _ ...any) {}
func (l *degradationLogger) Warn(msg string, _ ...any) {
	l.warnMsgs = append(l.warnMsgs, msg)
}

func (l *degradationLogger) count(msg string) int {
	n := 0
	for _, m := range l.warnMsgs {
		if m == msg {
			n++
		}
	}
	return n
}

// TestExport_CSVWithStyles_WarnsOnceAndSucceeds verifies that
// configuring header/data/column styles for a CSV export logs exactly
// one warning, regardless of row count, and the export still succeeds.
func TestExport_CSVWithStyles_WarnsOnceAndSucceeds(t *testing.T) {
	logger := &degradationLogger{}
	data := [][]any{
		{"id", "name"},
		{1, "Alice"},
		{2, "Bob"},
		{3, "Charlie"},
	}

	exp := NewExporter(data,
		excel.WithFormat(excel.FileFormatCSV),
		excel.WithLogger(logger),
	)
	exp.SetHeaderStyle(&excel.Style{Font: &excel.Font{Bold: true}})
	exp.SetColumnWidth("A", 20)

	buf, err := exp.ExportToBuffer(context.Background())
	if err != nil {
		t.Fatalf("ExportToBuffer() error = %v", err)
	}
	if len(buf) == 0 {
		t.Error("expected CSV output, got empty buffer")
	}

	if n := logger.count("CSV export ignores unsupported features"); n != 1 {
		t.Errorf("expected exactly 1 degradation warning, got %d: %v", n, logger.warnMsgs)
	}
}

// TestExport_CSVWithoutStyles_NoWarning verifies that a plain CSV export
// with no styles or column widths configured logs no degradation
// warning at all.
func TestExport_CSVWithoutStyles_NoWarning(t *testing.T) {
	logger := &degradationLogger{}
	data := [][]any{{"id", "name"}, {1, "Alice"}}

	exp := NewExporter(data,
		excel.WithFormat(excel.FileFormatCSV),
		excel.WithLogger(logger),
	)

	if _, err := exp.ExportToBuffer(context.Background()); err != nil {
		t.Fatalf("ExportToBuffer() error = %v", err)
	}

	if len(logger.warnMsgs) != 0 {
		t.Errorf("expected no warnings, got %v", logger.warnMsgs)
	}
}

// TestExportToSheets_CSVFormat_WritesFirstSheetAndWarnsOnce verifies
// that ExportToSheets on a CSV-format exporter writes only the first
// sheet's data, logs exactly one degradation warning regardless of how
// many sheets were dropped, and still succeeds.
func TestExportToSheets_CSVFormat_WritesFirstSheetAndWarnsOnce(t *testing.T) {
	logger := &degradationLogger{}
	exp := NewExporter(nil,
		excel.WithFormat(excel.FileFormatCSV),
		excel.WithLogger(logger),
	)

	sheets := []excel.Sheet{
		{Name: "Users", Data: [][]any{{"1", "Alice"}, {"2", "Bob"}}},
		{Name: "Orders", Data: [][]any{{"o1"}, {"o2"}}},
		{Name: "Products", Data: [][]any{{"p1"}}},
	}

	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets() error = %v", err)
	}

	if n := logger.count("CSV export supports only one sheet; writing the first sheet and dropping the rest"); n != 1 {
		t.Errorf("expected exactly 1 multi-sheet degradation warning, got %d: %v", n, logger.warnMsgs)
	}

	sink := exp.GetSink()
	if sink == nil {
		t.Fatal("expected a non-nil sink after ExportToSheets")
	}
	var buf bytes.Buffer
	if err := sink.Finalize(context.Background(), &buf); err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}

	records, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(buf.Bytes(), []byte{0xEF, 0xBB, 0xBF}))).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV output: %v", err)
	}
	want := [][]string{{"1", "Alice"}, {"2", "Bob"}}
	if len(records) != len(want) {
		t.Fatalf("expected only the first sheet's %d rows, got %d: %v", len(want), len(records), records)
	}
	for i := range want {
		if records[i][0] != want[i][0] || records[i][1] != want[i][1] {
			t.Errorf("row %d: expected %v, got %v", i, want[i], records[i])
		}
	}
}

// TestExport_CSVWithStyles_StrictFormatReturnsError verifies that
// WithStrictFormat(true) turns the styles-on-CSV degradation into an
// error wrapping excel.ErrStrictFormatUnsupported instead of a warning,
// and the export does not produce output.
func TestExport_CSVWithStyles_StrictFormatReturnsError(t *testing.T) {
	data := [][]any{{"id", "name"}, {1, "Alice"}}

	exp := NewExporter(data,
		excel.WithFormat(excel.FileFormatCSV),
		excel.WithStrictFormat(true),
	)
	exp.SetHeaderStyle(&excel.Style{Font: &excel.Font{Bold: true}})

	if _, err := exp.ExportToBuffer(context.Background()); !errors.Is(err, excel.ErrStrictFormatUnsupported) {
		t.Errorf("ExportToBuffer() error = %v, want error wrapping ErrStrictFormatUnsupported", err)
	}
}

// TestExport_CSVWithoutStyles_StrictFormatSucceeds verifies that
// WithStrictFormat(true) has no effect when no XLSX-only feature was
// actually configured: a plain CSV export still succeeds.
func TestExport_CSVWithoutStyles_StrictFormatSucceeds(t *testing.T) {
	data := [][]any{{"id", "name"}, {1, "Alice"}}

	exp := NewExporter(data,
		excel.WithFormat(excel.FileFormatCSV),
		excel.WithStrictFormat(true),
	)

	if _, err := exp.ExportToBuffer(context.Background()); err != nil {
		t.Errorf("ExportToBuffer() error = %v, want nil", err)
	}
}

// TestExportToSheets_CSVFormat_StrictFormatReturnsError verifies that
// WithStrictFormat(true) turns the multi-sheet-on-CSV degradation into
// an error wrapping excel.ErrStrictFormatUnsupported instead of a
// warning.
func TestExportToSheets_CSVFormat_StrictFormatReturnsError(t *testing.T) {
	exp := NewExporter(nil,
		excel.WithFormat(excel.FileFormatCSV),
		excel.WithStrictFormat(true),
	)

	sheets := []excel.Sheet{
		{Name: "Users", Data: [][]any{{"1", "Alice"}}},
		{Name: "Orders", Data: [][]any{{"o1"}}},
	}

	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); !errors.Is(err, excel.ErrStrictFormatUnsupported) {
		t.Errorf("ExportToSheets() error = %v, want error wrapping ErrStrictFormatUnsupported", err)
	}
}

// TestExportToSheets_CSVFormat_SingleSheet_NoWarning verifies that no
// degradation warning fires when there is only one sheet to begin with,
// since nothing was actually dropped.
func TestExportToSheets_CSVFormat_SingleSheet_NoWarning(t *testing.T) {
	logger := &degradationLogger{}
	exp := NewExporter(nil,
		excel.WithFormat(excel.FileFormatCSV),
		excel.WithLogger(logger),
	)

	sheets := []excel.Sheet{
		{Name: "Users", Data: [][]any{{"1", "Alice"}}},
	}

	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets() error = %v", err)
	}

	if len(logger.warnMsgs) != 0 {
		t.Errorf("expected no warnings for a single sheet, got %v", logger.warnMsgs)
	}
}
