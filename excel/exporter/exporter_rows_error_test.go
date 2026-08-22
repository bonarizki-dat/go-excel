package exporter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// badColumnStyle returns a style excelize rejects at NewStyle time (a
// "gradient" fill needs exactly two colors), used to force a specific
// data row's sink.WriteRow to fail deterministically without needing an
// oversized row.
func badColumnStyle() *excel.Style {
	s := excel.NewStyle()
	s.Fill.Type = "gradient"
	s.Fill.Color = []string{excel.ColorBlack}
	return s
}

// TestExporter_Export_HeaderRowBeforeHookError verifies a BeforeRowExport
// hook failing on the header row aborts the export before any data row
// is written.
func TestExporter_Export_HeaderRowBeforeHookError(t *testing.T) {
	data := [][]any{{"id"}, {1}}
	exp := exporter.NewExporter(data)
	hookErr := errors.New("before-row hook failed")
	exp.On(excel.BeforeRowExport, func(context.Context, *excel.Event) error {
		return hookErr
	})

	if err := exp.Export(context.Background()); !errors.Is(err, hookErr) {
		t.Errorf("Export() = %v, want error wrapping %v", err, hookErr)
	}
}

// TestExporter_Export_HeaderRowAfterHookError verifies an AfterRowExport
// hook failing on the header row aborts the export before any data row
// is written.
func TestExporter_Export_HeaderRowAfterHookError(t *testing.T) {
	data := [][]any{{"id"}, {1}}
	exp := exporter.NewExporter(data)
	hookErr := errors.New("after-row hook failed")
	exp.On(excel.AfterRowExport, func(context.Context, *excel.Event) error {
		return hookErr
	})

	if err := exp.Export(context.Background()); !errors.Is(err, hookErr) {
		t.Errorf("Export() = %v, want error wrapping %v", err, hookErr)
	}
}

// TestExporter_Export_HeaderRowWriteFailure verifies a header row that
// fails to write (via an invalid header style) surfaces as an Export
// error.
func TestExporter_Export_HeaderRowWriteFailure(t *testing.T) {
	data := [][]any{{"id"}, {1}}
	exp := exporter.NewExporter(data)
	exp.SetHeaderStyle(badColumnStyle())

	if err := exp.Export(context.Background()); err == nil {
		t.Error("Export with an invalid header style = nil error, want error")
	}
}

// TestExporter_Export_DataRowBeforeHookError verifies a BeforeRowExport
// hook failing on a data row (headings disabled, so the only
// BeforeRowExport events are for data rows) surfaces as an Export error.
func TestExporter_Export_DataRowBeforeHookError(t *testing.T) {
	data := [][]any{{1}}
	exp := exporter.NewExporter(data, excel.WithHeadings(false))
	hookErr := errors.New("before-row hook failed")
	exp.On(excel.BeforeRowExport, func(context.Context, *excel.Event) error {
		return hookErr
	})

	if err := exp.Export(context.Background()); !errors.Is(err, hookErr) {
		t.Errorf("Export() = %v, want error wrapping %v", err, hookErr)
	}
}

// TestExporter_Export_DataRowLoop_ContextCanceledMidway verifies the
// per-iteration ctx.Done() check in the data-row loop stops processing
// once the context is canceled between rows, rather than only being
// checked once up front.
func TestExporter_Export_DataRowLoop_ContextCanceledMidway(t *testing.T) {
	data := [][]any{{1}, {2}}
	exp := exporter.NewExporter(data, excel.WithHeadings(false))

	ctx, cancel := context.WithCancel(context.Background())
	exp.On(excel.AfterRowExport, func(context.Context, *excel.Event) error {
		cancel() // cancel after the first data row completes
		return nil
	})

	err := exp.Export(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Export() = %v, want error wrapping context.Canceled", err)
	}
}

// TestExporter_Export_DataRowWriteFailure_StopOnError verifies a data
// row write failure (via an invalid per-column style) aborts the export
// immediately when StopOnError is left at its default (true).
func TestExporter_Export_DataRowWriteFailure_StopOnError(t *testing.T) {
	data := [][]any{{"id"}, {1}}
	exp := exporter.NewExporter(data)
	exp.SetColumnStyle("A", badColumnStyle())

	if err := exp.Export(context.Background()); err == nil {
		t.Error("Export with an invalid column style = nil error, want error")
	}
}

// TestExporter_Export_DataRowWriteFailure_Collected verifies a data row
// write failure is collected into GetErrors and reported at the end of
// Export when StopOnError is false and no ErrorCallback is set.
func TestExporter_Export_DataRowWriteFailure_Collected(t *testing.T) {
	data := [][]any{{"id"}, {1}, {2}}
	exp := exporter.NewExporter(data, excel.WithStopOnError(false))
	exp.SetColumnStyle("A", badColumnStyle())

	err := exp.Export(context.Background())
	if err == nil {
		t.Fatal("Export with an invalid column style = nil error, want error")
	}
	if len(exp.GetErrors()) == 0 {
		t.Error("GetErrors() is empty, want the collected per-row write failures")
	}
}

// TestExporter_Export_DataRowWriteFailure_ErrorCallbackStops verifies an
// ErrorCallback returning false aborts the export immediately even
// though StopOnError is false.
func TestExporter_Export_DataRowWriteFailure_ErrorCallbackStops(t *testing.T) {
	data := [][]any{{"id"}, {1}, {2}}
	var callbackCalls int
	exp := exporter.NewExporter(data,
		excel.WithStopOnError(false),
		excel.WithErrorHandler(func(int, error) bool {
			callbackCalls++
			return false
		}),
	)
	exp.SetColumnStyle("A", badColumnStyle())

	if err := exp.Export(context.Background()); err == nil {
		t.Error("Export with a stopping ErrorCallback = nil error, want error")
	}
	if callbackCalls == 0 {
		t.Error("ErrorCallback was never called")
	}
}
