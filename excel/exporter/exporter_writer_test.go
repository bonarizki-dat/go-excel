package exporter_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// TestExporter_ExportToFile_SaveAsFailure verifies a filename excelize's
// SaveAs itself rejects (a null byte, which the OS filesystem calls
// reject but no earlier validation in this package catches) surfaces
// as an ExportToFile error via saveSinkAs's FileSaver branch.
func TestExporter_ExportToFile_SaveAsFailure(t *testing.T) {
	ctx := context.Background()
	data := [][]any{{"a"}}

	logger := &MockLogger{}
	exp := exporter.NewExporter(data, excel.WithLogger(logger))

	if err := exp.ExportToFile(ctx, "/tmp/bad\x00name.xlsx"); err == nil {
		t.Error("ExportToFile with a null byte in the path = nil error, want error")
	}
	if logger.errorCalls == 0 {
		t.Error("Logger.Error should be called when saveSinkAs fails")
	}
}

// TestExporter_ExportToFile_CSV_CreateFailure verifies the same failure
// mode for the CSV path, which falls back to os.Create instead of
// FileSaver since csvSink implements neither.
func TestExporter_ExportToFile_CSV_CreateFailure(t *testing.T) {
	ctx := context.Background()
	data := [][]any{{"a"}}
	exp := exporter.NewExporter(data)

	if err := exp.ExportToFile(ctx, "/tmp/bad\x00name.csv"); err == nil {
		t.Error("ExportToFile (CSV) with a null byte in the path = nil error, want error")
	}
}

// TestExporter_ExportToWriter_Success tests successful writer export.
func TestExporter_ExportToWriter_Success(t *testing.T) {
	ctx := context.Background()

	// Prepare test data
	data := [][]any{
		{"Product", "Price", "Quantity"},
		{"Widget", 9.99, 100},
		{"Gadget", 19.99, 50},
	}

	// Create buffer as writer
	var buf bytes.Buffer

	// Create exporter and export to writer
	exporter := exporter.NewExporter(data)
	err := exporter.ExportToWriter(ctx, &buf)
	if err != nil {
		t.Fatalf("ExportToWriter failed: %v", err)
	}

	// Verify buffer is not empty
	if buf.Len() == 0 {
		t.Error("Writer buffer should not be empty")
	}

	// Verify it looks like an Excel file (starts with PK zip signature)
	bytes := buf.Bytes()
	if len(bytes) < 4 {
		t.Fatal("Buffer too small to be valid Excel file")
	}
	// Excel files are ZIP archives, should start with PK signature
	if bytes[0] != 0x50 || bytes[1] != 0x4B {
		t.Error("Buffer should start with PK signature (ZIP format)")
	}
}

// TestExporter_ExportToWriter_NilWriter tests export with nil writer.
func TestExporter_ExportToWriter_NilWriter(t *testing.T) {
	ctx := context.Background()
	data := [][]any{{"test"}}

	exporter := exporter.NewExporter(data)
	err := exporter.ExportToWriter(ctx, nil)
	if err == nil {
		t.Error("ExportToWriter should fail with nil writer")
	}

	// Verify it's an excel.ExportError
	var exportErr *excel.ExportError
	if !errors.As(err, &exportErr) {
		t.Error("Error should be *excel.ExportError")
	}
}

// TestExporter_ExportToWriter_ContextCancellation tests context cancellation.
func TestExporter_ExportToWriter_ContextCancellation(t *testing.T) {
	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create large dataset
	data := make([][]any, 1000)
	for i := range data {
		data[i] = []any{i, "test", "data"}
	}

	var buf bytes.Buffer
	exporter := exporter.NewExporter(data)
	err := exporter.ExportToWriter(ctx, &buf)
	if err == nil {
		t.Error("ExportToWriter should fail with canceled context")
	}

	// Error should be or wrap context.Canceled
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error should be or wrap context.Canceled, got %v", err)
	}
}

// TestExporter_ExportToWriter_excel.WithHeadings tests writer export with headings.
func TestExporter_ExportToWriter_WithHeadings(t *testing.T) {
	ctx := context.Background()

	data := [][]any{
		{"Header1", "Header2"},
		{"Data1", "Data2"},
	}

	var buf bytes.Buffer
	exporter := exporter.NewExporter(data, excel.WithHeadings(true))
	err := exporter.ExportToWriter(ctx, &buf)
	if err != nil {
		t.Fatalf("ExportToWriter failed: %v", err)
	}

	// Verify buffer is not empty
	if buf.Len() == 0 {
		t.Error("Writer buffer should not be empty")
	}
}

// TestExporter_ExportToBuffer_Success tests successful buffer export.
func TestExporter_ExportToBuffer_Success(t *testing.T) {
	ctx := context.Background()

	// Prepare test data
	data := [][]any{
		{"Country", "Capital", "Population"},
		{"USA", "Washington", 331000000},
		{"Canada", "Ottawa", 38000000},
	}

	// Create exporter and export to buffer
	exporter := exporter.NewExporter(data)
	buf, err := exporter.ExportToBuffer(ctx)
	if err != nil {
		t.Fatalf("ExportToBuffer failed: %v", err)
	}

	// Verify buffer is not empty
	if len(buf) == 0 {
		t.Error("Buffer should not be empty")
	}

	// Verify it looks like an Excel file (starts with PK zip signature)
	if len(buf) < 4 {
		t.Fatal("Buffer too small to be valid Excel file")
	}
	// Excel files are ZIP archives, should start with PK signature
	if buf[0] != 0x50 || buf[1] != 0x4B {
		t.Error("Buffer should start with PK signature (ZIP format)")
	}
}

// TestExporter_ExportToBuffer_ContextCancellation tests context cancellation.
func TestExporter_ExportToBuffer_ContextCancellation(t *testing.T) {
	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create large dataset
	data := make([][]any, 1000)
	for i := range data {
		data[i] = []any{i, "test", "data"}
	}

	exporter := exporter.NewExporter(data)
	_, err := exporter.ExportToBuffer(ctx)
	if err == nil {
		t.Error("ExportToBuffer should fail with canceled context")
	}

	// Error should be or wrap context.Canceled
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error should be or wrap context.Canceled, got %v", err)
	}
}

// TestExporter_ExportToBuffer_EmptyData tests buffer export with empty data.
func TestExporter_ExportToBuffer_EmptyData(t *testing.T) {
	ctx := context.Background()

	exporter := exporter.NewExporter(nil)
	_, err := exporter.ExportToBuffer(ctx)
	if !errors.Is(err, excel.ErrEmptyData) {
		t.Errorf("ExportToBuffer with nil data should return excel.ErrEmptyData, got %v", err)
	}
}

// TestExporter_ExportToBuffer_excel.WithOptions tests buffer export with options.
func TestExporter_ExportToBuffer_WithOptions(t *testing.T) {
	ctx := context.Background()

	data := [][]any{
		{"Column1", "Column2"},
		{"Value1", "Value2"},
	}

	// Create exporter with options
	exporter := exporter.NewExporter(data,
		excel.WithSheetName("TestSheet"),
		excel.WithHeadings(true),
	)

	buf, err := exporter.ExportToBuffer(ctx)
	if err != nil {
		t.Fatalf("ExportToBuffer failed: %v", err)
	}

	// Verify buffer is not empty
	if len(buf) == 0 {
		t.Error("Buffer should not be empty")
	}
}
