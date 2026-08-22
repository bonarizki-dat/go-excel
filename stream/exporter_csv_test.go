package stream

import (
	"bytes"
	"context"
	"encoding/csv"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// readBackCSV decodes buf as CSV, stripping a leading UTF-8 BOM if
// present, and returns the resulting records.
func readBackCSV(t *testing.T, buf *bytes.Buffer) [][]string {
	t.Helper()

	data := buf.Bytes()
	bom := []byte{0xEF, 0xBB, 0xBF}
	data = bytes.TrimPrefix(data, bom)

	records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatalf("readBackCSV: %v", err)
	}
	return records
}

// TestExportStream_CSVFormat_WritesRowsInOrder verifies that
// WithExportFormat(FileFormatCSV) makes ExportStream write plain CSV
// instead of XLSX, preserving row order.
func TestExportStream_CSVFormat_WritesRowsInOrder(t *testing.T) {
	var buf bytes.Buffer
	exporter, err := NewStreamExporter(&buf, WithExportFormat(config.FileFormatCSV))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	rows := make(chan []any, 3)
	go func() {
		defer close(rows)
		rows <- []any{"1", "Alice"}
		rows <- []any{"2", "Bob"}
		rows <- []any{"3", "Charlie"}
	}()

	if err := exporter.ExportStream(context.Background(), rows); err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	got := readBackCSV(t, &buf)
	want := [][]string{{"1", "Alice"}, {"2", "Bob"}, {"3", "Charlie"}}
	if len(got) != len(want) {
		t.Fatalf("expected %d rows, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i][0] != want[i][0] || got[i][1] != want[i][1] {
			t.Errorf("row %d: expected %v, got %v", i, want[i], got[i])
		}
	}
}

// TestExportStream_CSVFormat_WithHeadings verifies the heading row is
// written first, exactly like the XLSX path.
func TestExportStream_CSVFormat_WithHeadings(t *testing.T) {
	var buf bytes.Buffer
	headings := []string{"ID", "Name"}
	exporter, err := NewStreamExporter(&buf,
		WithExportFormat(config.FileFormatCSV),
		WithExportHeadings(headings),
	)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	rows := make(chan []any, 2)
	go func() {
		defer close(rows)
		rows <- []any{"1", "Alice"}
		rows <- []any{"2", "Bob"}
	}()

	if err := exporter.ExportStream(context.Background(), rows); err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	got := readBackCSV(t, &buf)
	if len(got) != 3 {
		t.Fatalf("expected 1 heading row + 2 data rows, got %d rows: %v", len(got), got)
	}
	if got[0][0] != "ID" || got[0][1] != "Name" {
		t.Errorf("expected heading row %v, got %v", headings, got[0])
	}
	if got[1][0] != "1" || got[2][0] != "2" {
		t.Errorf("expected data rows in order, got %v", got[1:])
	}
}

// TestExportStream_CSVFormat_CustomDelimiterAndNoBOM verifies
// WithExportCSVOptions overrides the default comma delimiter and BOM.
func TestExportStream_CSVFormat_CustomDelimiterAndNoBOM(t *testing.T) {
	var buf bytes.Buffer
	opts := config.DefaultCSVOptions()
	opts.Delimiter = ';'
	opts.UseBOM = false

	exporter, err := NewStreamExporter(&buf,
		WithExportFormat(config.FileFormatCSV),
		WithExportCSVOptions(opts),
	)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	rows := make(chan []any, 1)
	go func() {
		defer close(rows)
		rows <- []any{"1", "Alice"}
	}()

	if err := exporter.ExportStream(context.Background(), rows); err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	bom := []byte{0xEF, 0xBB, 0xBF}
	if bytes.HasPrefix(buf.Bytes(), bom) {
		t.Error("expected no BOM when UseBOM is false")
	}

	r := csv.NewReader(bytes.NewReader(buf.Bytes()))
	r.Comma = ';'
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("decode with semicolon delimiter: %v", err)
	}
	if len(records) != 1 || records[0][0] != "1" || records[0][1] != "Alice" {
		t.Errorf("expected [[1 Alice]], got %v", records)
	}
}
