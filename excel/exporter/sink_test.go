package exporter

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/excel/csvio"
)

// failWriter is an io.Writer that always errors, used to force Flush and
// downstream io.Writer error branches without needing a real broken sink.
type failWriter struct{}

func (failWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestXlsxSink_Close(t *testing.T) {
	exp := NewExporter([][]any{{"a"}})
	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}
	sink := exp.GetSink()
	if sink == nil {
		t.Fatal("GetSink returned nil")
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("xlsxSink.Close: %v", err)
	}
}

func TestCSVSink_Close(t *testing.T) {
	exp := NewExporter([][]any{{"a"}}, excel.WithFormat(excel.FileFormatCSV))
	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}
	sink := exp.GetSink()
	if sink == nil {
		t.Fatal("GetSink returned nil")
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("csvSink.Close: %v", err)
	}
}

// badGradientStyle returns a style whose Fill excelize rejects at
// NewStyle time (a "gradient" fill needs exactly two colors), used to
// force resolveHeaderStyleID/resolveDataStyleID/resolveColumnStyleID
// down their error paths without needing to fake excelize itself.
func badGradientStyle() *excel.Style {
	s := excel.NewStyle()
	s.Fill.Type = "gradient"
	s.Fill.Color = []string{excel.ColorBlack}
	return s
}

func TestNewXLSXSink_InvalidSheetName(t *testing.T) {
	_, err := newXLSXSink("a/b", nil, nil, nil, nil, "", false, excel.DocProperties{})
	if err == nil {
		t.Fatal("newXLSXSink with invalid sheet name = nil error, want error")
	}
}

func TestNewXLSXSink_InvalidColumnWidthKey(t *testing.T) {
	_, err := newXLSXSink("Sheet1", nil, nil, nil, map[string]float64{"1": 10}, "", false, excel.DocProperties{})
	if err == nil {
		t.Fatal("newXLSXSink with non-letter column key = nil error, want error")
	}
}

func TestXLSXSink_WriteRow_ContextCanceled(t *testing.T) {
	sink, err := newXLSXSink("Sheet1", nil, nil, nil, nil, "", false, excel.DocProperties{})
	if err != nil {
		t.Fatalf("newXLSXSink: %v", err)
	}
	defer sink.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := sink.WriteRow(ctx, 1, []any{"a"}, false); err == nil {
		t.Error("WriteRow with canceled context = nil error, want context error")
	}
}

func TestXLSXSink_WriteRow_EmptyRow(t *testing.T) {
	sink, err := newXLSXSink("Sheet1", nil, nil, nil, nil, "", false, excel.DocProperties{})
	if err != nil {
		t.Fatalf("newXLSXSink: %v", err)
	}
	defer sink.Close()

	if err := sink.WriteRow(context.Background(), 1, []any{}, false); err != nil {
		t.Errorf("WriteRow with empty row: %v", err)
	}
}

func TestXLSXSink_WriteRow_HeaderStyleFailure(t *testing.T) {
	sink, err := newXLSXSink("Sheet1", badGradientStyle(), nil, nil, nil, "", false, excel.DocProperties{})
	if err != nil {
		t.Fatalf("newXLSXSink: %v", err)
	}
	defer sink.Close()

	if err := sink.WriteRow(context.Background(), 1, []any{"a"}, true); err == nil {
		t.Error("WriteRow with an invalid header style = nil error, want error")
	}
}

func TestXLSXSink_WriteRow_DataStyleFailure(t *testing.T) {
	sink, err := newXLSXSink("Sheet1", nil, badGradientStyle(), nil, nil, "", false, excel.DocProperties{})
	if err != nil {
		t.Fatalf("newXLSXSink: %v", err)
	}
	defer sink.Close()

	if err := sink.WriteRow(context.Background(), 2, []any{"a"}, false); err == nil {
		t.Error("WriteRow with an invalid data style = nil error, want error")
	}
}

func TestXLSXSink_WriteRow_ColumnStyleFailure(t *testing.T) {
	sink, err := newXLSXSink("Sheet1", nil, nil, map[string]*excel.Style{"A": badGradientStyle()}, nil, "", false, excel.DocProperties{})
	if err != nil {
		t.Fatalf("newXLSXSink: %v", err)
	}
	defer sink.Close()

	// A non-header row with a per-column style configured routes
	// through applyPerCellStyles, which is what resolves column A's
	// (invalid) style.
	if err := sink.WriteRow(context.Background(), 2, []any{"a", "b"}, false); err == nil {
		t.Error("WriteRow with an invalid column style = nil error, want error")
	}
}

func TestCSVSink_WriteRow_ContextCanceled(t *testing.T) {
	sink, err := newCSVSink(config.DefaultCSVOptions())
	if err != nil {
		t.Fatalf("newCSVSink: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := sink.WriteRow(ctx, 1, []any{"a"}, false); err == nil {
		t.Error("WriteRow with canceled context = nil error, want context error")
	}
}

// TestCSVSink_WriteRow_InvalidDelimiter bypasses Config.Validate (which
// would normally reject a newline delimiter before this point) by
// calling newCSVSink directly, to exercise WriteRecord's error path.
func TestCSVSink_WriteRow_InvalidDelimiter(t *testing.T) {
	opts := config.DefaultCSVOptions()
	opts.Delimiter = '\r'
	sink, err := newCSVSink(opts)
	if err != nil {
		t.Fatalf("newCSVSink: %v", err)
	}

	if err := sink.WriteRow(context.Background(), 1, []any{"a", "b"}, false); err == nil {
		t.Error("WriteRow with an invalid CSV delimiter = nil error, want error")
	}
}

// TestCSVSink_Finalize_FlushError constructs a csvSink whose csvio.Writer
// wraps a failing io.Writer directly (bypassing the sink's normal
// in-memory buffer, which never fails), so Finalize's Flush call
// surfaces a write error.
func TestCSVSink_Finalize_FlushError(t *testing.T) {
	opts := config.DefaultCSVOptions()
	opts.UseBOM = false // avoid failing inside NewWriter itself, before Flush is even reached
	w, err := csvio.NewWriter(failWriter{}, opts)
	if err != nil {
		t.Fatalf("csvio.NewWriter: %v", err)
	}
	sink := &csvSink{writer: w, layout: opts.TimeLayout}

	if err := sink.WriteRow(context.Background(), 1, []any{"a"}, false); err != nil {
		t.Fatalf("WriteRow: %v", err)
	}

	var out bytes.Buffer
	if err := sink.Finalize(context.Background(), &out); err == nil {
		t.Error("Finalize with a failing underlying writer = nil error, want error")
	}
}

// TestCSVSink_Finalize_WriteError exercises Finalize's own w.Write call
// (copying the buffered CSV bytes to the caller-supplied writer), as
// distinct from the internal Flush error covered above.
func TestCSVSink_Finalize_WriteError(t *testing.T) {
	sink, err := newCSVSink(config.DefaultCSVOptions())
	if err != nil {
		t.Fatalf("newCSVSink: %v", err)
	}
	if err := sink.WriteRow(context.Background(), 1, []any{"a"}, false); err != nil {
		t.Fatalf("WriteRow: %v", err)
	}

	if err := sink.Finalize(context.Background(), failWriter{}); err == nil {
		t.Error("Finalize writing to a failing writer = nil error, want error")
	}
}

func TestXLSXSink_WriteRow_TooManyColumns(t *testing.T) {
	sink, err := newXLSXSink("Sheet1", nil, nil, nil, nil, "", false, excel.DocProperties{})
	if err != nil {
		t.Fatalf("newXLSXSink: %v", err)
	}
	defer sink.Close()

	// excelize supports at most 16384 columns; one more overflows
	// CoordinatesToCellName inside writeCellValues.
	row := make([]any, 16385)
	if err := sink.WriteRow(context.Background(), 1, row, false); err == nil {
		t.Error("WriteRow with 16385 columns = nil error, want error")
	}
}
