package stream

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestNewStreamExporter_InvalidSheetNameReturnsError verifies a sheet
// name excelize's NewSheet rejects (a colon) fails the constructor with
// an error instead of silently falling back to "Sheet1".
func TestNewStreamExporter_InvalidSheetNameReturnsError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf, WithExportSheetName("Bad:Name"))

	if err == nil {
		t.Error("NewStreamExporter with an invalid sheet name = nil error, want error")
	}
	if exp != nil {
		t.Errorf("NewStreamExporter with an invalid sheet name = %v, want nil", exp)
	}
}

// failWriteCloser is an io.Writer whose Write always fails, used to
// force csvio.NewWriter's BOM-write failure inside newSink.
type failWriteCloser struct{}

func (failWriteCloser) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

// TestNewSink_CSV_WriterFailure verifies a csvio.NewWriter failure (a
// writer that fails during the BOM write) surfaces as a newSink error.
func TestNewSink_CSV_WriterFailure(t *testing.T) {
	opts := config.DefaultCSVOptions()
	opts.UseBOM = true

	exp, err := NewStreamExporter(failWriteCloser{}, WithExportFormat(config.FileFormatCSV), WithExportCSVOptions(opts))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	if _, err := exp.newSink(); err == nil {
		t.Error("newSink (CSV) with a failing writer = nil error, want error")
	}
}

// TestNewSink_XLSX_InvalidSheetName verifies a sheetName that is
// invalid at NewStreamWriter time (mutated directly after construction,
// bypassing NewStreamExporter's own validation) surfaces as a newSink
// error.
func TestNewSink_XLSX_InvalidSheetName(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	exp.sheetName = "Bad:Name"

	if _, err := exp.newSink(); err == nil {
		t.Error("newSink (XLSX) with an invalid sheet name = nil error, want error")
	}
}

// TestExportStream_NewSinkFailurePropagates verifies a newSink failure
// surfaces through ExportStream and closes the exporter's file.
func TestExportStream_NewSinkFailurePropagates(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	exp.sheetName = "Bad:Name"

	rows := make(chan []any)
	close(rows)

	if err := exp.ExportStream(context.Background(), rows); err == nil {
		t.Error("ExportStream with an invalid sheet name = nil error, want error")
	}
}
