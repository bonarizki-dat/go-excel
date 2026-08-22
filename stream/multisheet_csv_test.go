package stream

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// degradationLogger records Warn calls so tests can assert a feature was
// dropped exactly once, never once per row or per sheet.
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

// TestExportToSheets_CSVFormat_WritesFirstSheetAndWarnsOnce verifies
// that ExportToSheets on a CSV-format StreamExporter writes only the
// first sheet's rows, logs exactly one degradation warning regardless
// of how many sheets were dropped, and still succeeds.
func TestExportToSheets_CSVFormat_WritesFirstSheetAndWarnsOnce(t *testing.T) {
	var buf bytes.Buffer
	logger := &degradationLogger{}
	exporter, err := NewStreamExporter(&buf,
		WithExportFormat(config.FileFormatCSV),
		WithExportLogger(logger),
	)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	sheets := []excel.Sheet{
		{Name: "Users", Data: [][]any{{"1", "Alice"}, {"2", "Bob"}}},
		{Name: "Orders", Data: [][]any{{"o1"}, {"o2"}}},
		{Name: "Products", Data: [][]any{{"p1"}}},
	}

	if err := exporter.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets() error = %v", err)
	}

	if n := logger.count("CSV export supports only one sheet; writing the first sheet and dropping the rest"); n != 1 {
		t.Errorf("expected exactly 1 multi-sheet degradation warning, got %d: %v", n, logger.warnMsgs)
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

// TestExportToSheets_CSVFormat_StrictFormatReturnsError verifies that
// WithExportStrictFormat(true) turns the multi-sheet-on-CSV degradation
// into an error wrapping excel.ErrStrictFormatUnsupported instead of a
// warning.
func TestExportToSheets_CSVFormat_StrictFormatReturnsError(t *testing.T) {
	var buf bytes.Buffer
	exporter, err := NewStreamExporter(&buf,
		WithExportFormat(config.FileFormatCSV),
		WithExportStrictFormat(true),
	)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	sheets := []excel.Sheet{
		{Name: "Users", Data: [][]any{{"1", "Alice"}}},
		{Name: "Orders", Data: [][]any{{"o1"}}},
	}

	if err := exporter.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); !errors.Is(err, excel.ErrStrictFormatUnsupported) {
		t.Errorf("ExportToSheets() error = %v, want error wrapping ErrStrictFormatUnsupported", err)
	}
}

// TestExportToSheets_CSVFormat_SingleSheet_NoWarning verifies that no
// degradation warning fires when there is only one sheet to begin with,
// since nothing was actually dropped.
func TestExportToSheets_CSVFormat_SingleSheet_NoWarning(t *testing.T) {
	var buf bytes.Buffer
	logger := &degradationLogger{}
	exporter, err := NewStreamExporter(&buf,
		WithExportFormat(config.FileFormatCSV),
		WithExportLogger(logger),
	)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	sheets := []excel.Sheet{
		{Name: "Users", Data: [][]any{{"1", "Alice"}}},
	}

	if err := exporter.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets() error = %v", err)
	}

	if len(logger.warnMsgs) != 0 {
		t.Errorf("expected no warnings for a single sheet, got %v", logger.warnMsgs)
	}
}
