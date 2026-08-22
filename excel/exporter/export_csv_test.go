package exporter

import (
	"bytes"
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bonarizki-dat/go-excel/excel"
)

func csvData() [][]any {
	return [][]any{
		{"id", "name", "active"},
		{1, "Alice", true},
		{2, "Bob", false},
	}
}

func readCSVRecords(t *testing.T, data []byte) [][]string {
	t.Helper()
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV output: %v", err)
	}
	return records
}

func TestExportToFile_CSVExtension_WritesCSV(t *testing.T) {
	exp := NewExporter(csvData())
	filename := filepath.Join(t.TempDir(), "report.csv")

	if err := exp.ExportToFile(context.Background(), filename); err != nil {
		t.Fatalf("ExportToFile() error = %v", err)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}

	if !bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		t.Error("expected UTF-8 BOM at start of CSV file")
	}

	records := readCSVRecords(t, data)
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}
	if records[0][1] != "name" || records[1][1] != "Alice" {
		t.Errorf("unexpected records: %v", records)
	}

	// GetFile returns nil for a CSV export since there is no excelize
	// file behind it.
	if exp.GetFile() != nil {
		t.Error("GetFile() should be nil after a CSV export")
	}
}

func TestExportToWriter_WithFormatCSV_WritesCSV(t *testing.T) {
	exp := NewExporter(csvData(), excel.WithFormat(excel.FileFormatCSV))

	var buf bytes.Buffer
	if err := exp.ExportToWriter(context.Background(), &buf); err != nil {
		t.Fatalf("ExportToWriter() error = %v", err)
	}

	records := readCSVRecords(t, buf.Bytes())
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}
}

func TestExportToBuffer_WithFormatCSV_WritesCSV(t *testing.T) {
	exp := NewExporter(csvData(), excel.WithFormat(excel.FileFormatCSV))

	buf, err := exp.ExportToBuffer(context.Background())
	if err != nil {
		t.Fatalf("ExportToBuffer() error = %v", err)
	}

	records := readCSVRecords(t, buf)
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}
}

func TestExportToFile_CSVDelimiter_UsesConfiguredSeparator(t *testing.T) {
	exp := NewExporter(csvData(), excel.WithCSVDelimiter(';'))
	filename := filepath.Join(t.TempDir(), "report.csv")

	if err := exp.ExportToFile(context.Background(), filename); err != nil {
		t.Fatalf("ExportToFile() error = %v", err)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}

	if !bytes.Contains(data, []byte("id;name;active")) {
		t.Errorf("expected semicolon-delimited header, got: %s", data)
	}
}

func TestExportToFile_CSVBOMDisabled_OmitsBOM(t *testing.T) {
	exp := NewExporter(csvData(), excel.WithCSVBOM(false))
	filename := filepath.Join(t.TempDir(), "report.csv")

	if err := exp.ExportToFile(context.Background(), filename); err != nil {
		t.Fatalf("ExportToFile() error = %v", err)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}

	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		t.Error("expected no BOM when WithCSVBOM(false) is set")
	}
}

func TestExportToFile_XLSXExtension_StillWritesXLSX(t *testing.T) {
	// Guards against a format-routing regression: a non-.csv extension
	// must still produce a real XLSX file via excelize, unaffected by
	// the new CSV path.
	exp := NewExporter(csvData(), excel.WithFormat(excel.FileFormatCSV))
	filename := filepath.Join(t.TempDir(), "report.xlsx")

	if err := exp.ExportToFile(context.Background(), filename); err != nil {
		t.Fatalf("ExportToFile() error = %v", err)
	}

	if exp.GetFile() == nil {
		t.Fatal("GetFile() should be non-nil after an .xlsx export, even with WithFormat(CSV) set")
	}
	cell, err := exp.GetFile().GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("GetCellValue() error = %v", err)
	}
	if cell != "id" {
		t.Errorf("cell A1 = %q, want %q", cell, "id")
	}
}

func TestExportToFile_CSVTimeAndFloatFormatting(t *testing.T) {
	when, err := time.Parse(time.RFC3339, "2024-06-01T10:00:00Z")
	if err != nil {
		t.Fatalf("time.Parse() error = %v", err)
	}
	data := [][]any{
		{"amount", "when"},
		{1234567.5, when},
	}
	exp := NewExporter(data)
	filename := filepath.Join(t.TempDir(), "report.csv")

	if err := exp.ExportToFile(context.Background(), filename); err != nil {
		t.Fatalf("ExportToFile() error = %v", err)
	}

	raw, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}
	records := readCSVRecords(t, raw)

	if records[1][0] != "1234567.5" {
		t.Errorf("amount = %q, want %q (no scientific notation)", records[1][0], "1234567.5")
	}
	if records[1][1] != "2024-06-01 10:00:00" {
		t.Errorf("when = %q, want default CSV time layout", records[1][1])
	}
}
