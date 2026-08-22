package exporter

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// failCloseSink is an excel.RowSink whose Close always fails, used to
// exercise closeCurrent's sink.Close() branch and Export's propagation
// of a closeCurrent error from a previous call.
type failCloseSink struct{}

var errSinkCloseFailed = errors.New("sink close failed")

func (failCloseSink) WriteRow(context.Context, int, []any, bool) error { return nil }
func (failCloseSink) Finalize(context.Context, io.Writer) error        { return nil }
func (failCloseSink) Close() error                                     { return errSinkCloseFailed }

// TestBasicExporter_CloseCurrent_ClosesSinkWhenFileNil verifies
// closeCurrent falls back to e.sink.Close() when e.file is nil (the CSV
// case, or any non-XLSX sink).
func TestBasicExporter_CloseCurrent_ClosesSinkWhenFileNil(t *testing.T) {
	exp := NewExporter(nil)
	exp.file = nil
	exp.sink = failCloseSink{}

	if err := exp.closeCurrent(); !errors.Is(err, errSinkCloseFailed) {
		t.Errorf("closeCurrent() = %v, want %v", err, errSinkCloseFailed)
	}
}

// TestBasicExporter_Export_PropagatesPriorCloseError verifies a second
// Export() call surfaces an error from closing the previous export's
// sink, instead of silently discarding it.
func TestBasicExporter_Export_PropagatesPriorCloseError(t *testing.T) {
	exp := NewExporter([][]any{{"a"}})
	exp.file = nil
	exp.sink = failCloseSink{}

	err := exp.Export(context.Background())
	if !errors.Is(err, errSinkCloseFailed) {
		t.Errorf("Export() = %v, want error wrapping %v", err, errSinkCloseFailed)
	}
}

// TestBasicExporter_Export_BeforeExportHookError verifies a BeforeExport
// hook returning an error aborts Export before any work begins.
func TestBasicExporter_Export_BeforeExportHookError(t *testing.T) {
	exp := NewExporter([][]any{{"a"}})
	hookErr := errors.New("before-export hook failed")
	exp.On(excel.BeforeExport, func(context.Context, *excel.Event) error {
		return hookErr
	})

	if err := exp.Export(context.Background()); !errors.Is(err, hookErr) {
		t.Errorf("Export() = %v, want error wrapping %v", err, hookErr)
	}
}

// TestBasicExporter_Export_PropagatesNewSinkError verifies a newSink()
// failure (an invalid sheet name excelize rejects) surfaces as an
// Export error instead of panicking or being silently swallowed.
func TestBasicExporter_Export_PropagatesNewSinkError(t *testing.T) {
	exp := NewExporter([][]any{{"a"}}, excel.WithSheetName("a/b"))

	if err := exp.Export(context.Background()); err == nil {
		t.Error("Export with an invalid sheet name = nil error, want error")
	}
}

// TestBasicExporter_Close_PropagatesCloseCurrentError verifies the
// public Close() method surfaces a closeCurrent() error rather than
// swallowing it.
func TestBasicExporter_Close_PropagatesCloseCurrentError(t *testing.T) {
	exp := NewExporter(nil)
	exp.file = nil
	exp.sink = failCloseSink{}

	if err := exp.Close(); !errors.Is(err, errSinkCloseFailed) {
		t.Errorf("Close() = %v, want error wrapping %v", err, errSinkCloseFailed)
	}
}
