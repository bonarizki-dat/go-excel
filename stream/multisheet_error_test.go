package stream

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestExportToSheets_NoSheetsError verifies an empty sheets slice with
// CreateDefaultSheet left false surfaces resolveStreamSheets' error.
func TestExportToSheets_NoSheetsError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	err = exp.ExportToSheets(context.Background(), nil, excel.DefaultMultiSheetExportOptions())
	if err == nil {
		t.Error("ExportToSheets with no sheets = nil error, want error")
	}
}

// TestExportToSheets_ValidateSheetsError verifies a validateSheets
// failure (a duplicate sheet name) surfaces wrapped from ExportToSheets.
func TestExportToSheets_ValidateSheetsError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	sheets := []excel.Sheet{
		{Name: "Dup", Data: [][]any{{"a"}}},
		{Name: "Dup", Data: [][]any{{"b"}}},
	}
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets with duplicate sheet names = nil error, want error")
	}
}

// TestExportToSheets_InitWorkbookError verifies a first-sheet name that
// passes validateSheets (a colon, which it does not check) but is
// rejected by excelize's SetSheetName surfaces as an initWorkbook
// error via ExportToSheets.
func TestExportToSheets_InitWorkbookError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	sheets := []excel.Sheet{
		{Name: "Bad:Name", Data: [][]any{{"a"}}},
	}
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets with a colon in the first sheet name = nil error, want error")
	}
}

// TestExportToSheets_LoopHardErrorStopsImmediately verifies a second
// sheet's export failure (a colon in its name, rejected by excelize's
// NewSheet) aborts ExportToSheets immediately when ContinueOnError is
// left at its default (false).
func TestExportToSheets_LoopHardErrorStopsImmediately(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	sheets := []excel.Sheet{
		{Name: "First", Data: [][]any{{"a"}}},
		{Name: "Bad:Name", Data: [][]any{{"b"}}},
	}
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets with a colon in the second sheet name = nil error, want error")
	}
}

// TestExportToSheets_ContinueOnErrorCollectsAndReports verifies a
// failing sheet is collected (not fatal) when ContinueOnError is true,
// the remaining sheet still gets written, and the collected error is
// reported at the end.
func TestExportToSheets_ContinueOnErrorCollectsAndReports(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	sheets := []excel.Sheet{
		{Name: "First", Data: [][]any{{"a"}}},
		{Name: "Bad:Name", Data: [][]any{{"b"}}},
		{Name: "Good", Data: [][]any{{"c"}}},
	}
	options := excel.DefaultMultiSheetExportOptions()
	options.ContinueOnError = true

	err = exp.ExportToSheets(context.Background(), sheets, options)
	if err == nil {
		t.Fatal("ExportToSheets with a failing sheet = nil error, want error summarizing collected failures")
	}

	sheetList := exp.file.GetSheetList()
	found := false
	for _, name := range sheetList {
		if name == "Good" {
			found = true
		}
	}
	if !found {
		t.Errorf("sheets = %v, want 'Good' sheet to have been written despite 'Bad:Name' failing", sheetList)
	}
}

// TestResolveStreamSheets_NoSheetsProvided verifies the direct error
// path when sheets is empty and CreateDefaultSheet is false.
func TestResolveStreamSheets_NoSheetsProvided(t *testing.T) {
	_, err := resolveStreamSheets(nil, excel.MultiSheetExportOptions{CreateDefaultSheet: false})
	if err == nil {
		t.Error("resolveStreamSheets with no sheets and CreateDefaultSheet=false = nil error, want error")
	}
}

// TestResolveStreamSheets_CreatesDefaultSheet verifies an empty sheets
// slice with CreateDefaultSheet true synthesizes a single default
// sheet named after DefaultSheetName.
func TestResolveStreamSheets_CreatesDefaultSheet(t *testing.T) {
	got, err := resolveStreamSheets(nil, excel.MultiSheetExportOptions{
		CreateDefaultSheet: true,
		DefaultSheetName:   "MySheet",
	})
	if err != nil {
		t.Fatalf("resolveStreamSheets: %v", err)
	}
	if len(got) != 1 || got[0].Name != "MySheet" {
		t.Errorf("resolveStreamSheets = %+v, want a single sheet named MySheet", got)
	}
}

// TestInitWorkbook_SetSheetNameError verifies a firstSheetName that
// excelize's SetSheetName rejects (a colon) surfaces as an error.
func TestInitWorkbook_SetSheetNameError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	if err := exp.initWorkbook("Bad:Name"); err == nil {
		t.Error("initWorkbook with a colon in the name = nil error, want error")
	}
}

// TestExportSheetsLoop_ContextCanceled verifies the per-sheet
// ctx.Done() check stops the loop before any sheet is processed.
func TestExportSheetsLoop_ContextCanceled(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	if err := exp.initWorkbook("Sheet1"); err != nil {
		t.Fatalf("initWorkbook: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sheets := []excel.Sheet{{Name: "Sheet1", Data: [][]any{{"a"}}}}
	var exportErrors []error
	err = exp.exportSheetsLoop(ctx, sheets, excel.DefaultMultiSheetExportOptions(), &exportErrors)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("exportSheetsLoop() = %v, want error wrapping context.Canceled", err)
	}
}

// TestExportSheetsLoop_SkipEmptySheets verifies a sheet with neither
// DataChannel nor Data is skipped when SkipEmptySheets is true.
func TestExportSheetsLoop_SkipEmptySheets(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	if err := exp.initWorkbook("Sheet1"); err != nil {
		t.Fatalf("initWorkbook: %v", err)
	}

	sheets := []excel.Sheet{{Name: "Sheet1"}}
	options := excel.DefaultMultiSheetExportOptions()
	options.SkipEmptySheets = true

	var exportErrors []error
	if err := exp.exportSheetsLoop(context.Background(), sheets, options, &exportErrors); err != nil {
		t.Fatalf("exportSheetsLoop: %v", err)
	}
}

// TestExportSheetsLoop_ProgressCallbackError verifies a
// ProgressCallback error aborts the loop.
func TestExportSheetsLoop_ProgressCallbackError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	if err := exp.initWorkbook("Sheet1"); err != nil {
		t.Fatalf("initWorkbook: %v", err)
	}

	sheets := []excel.Sheet{{Name: "Sheet1", Data: [][]any{{"a"}}}}
	callbackErr := errors.New("progress callback failed")
	options := excel.DefaultMultiSheetExportOptions()
	options.ProgressCallback = func(context.Context, int, string) error {
		return callbackErr
	}

	var exportErrors []error
	err = exp.exportSheetsLoop(context.Background(), sheets, options, &exportErrors)
	if !errors.Is(err, callbackErr) {
		t.Errorf("exportSheetsLoop() = %v, want error wrapping %v", err, callbackErr)
	}
}

// TestExportToSheetsCSV_ExportStreamError verifies a newSink failure
// inside the underlying ExportStream call (a writer that fails during
// the BOM write) surfaces wrapped from exportToSheetsCSV.
func TestExportToSheetsCSV_ExportStreamError(t *testing.T) {
	opts := config.DefaultCSVOptions()
	opts.UseBOM = true

	exp, err := NewStreamExporter(failWriteCloser{}, WithExportFormat(config.FileFormatCSV), WithExportCSVOptions(opts))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	sheets := []excel.Sheet{{Name: "Sheet1", Data: [][]any{{"a"}}}}
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets (CSV) with a failing writer = nil error, want error")
	}
}

// TestExportToSheetsCSV_ProgressCallbackError verifies a
// ProgressCallback error surfaces from the CSV degradation path.
func TestExportToSheetsCSV_ProgressCallbackError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf, WithExportFormat(config.FileFormatCSV))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	sheets := []excel.Sheet{{Name: "Sheet1", Data: [][]any{{"a"}}}}
	callbackErr := errors.New("progress callback failed")
	options := excel.DefaultMultiSheetExportOptions()
	options.ProgressCallback = func(context.Context, int, string) error {
		return callbackErr
	}

	err = exp.ExportToSheets(context.Background(), sheets, options)
	if !errors.Is(err, callbackErr) {
		t.Errorf("ExportToSheets (CSV) = %v, want error wrapping %v", err, callbackErr)
	}
}

// TestExportSheetStream_NewSheetError verifies index>0 with a colon in
// the sheet name surfaces excelize's NewSheet error directly.
func TestExportSheetStream_NewSheetError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	if err := exp.initWorkbook("Sheet1"); err != nil {
		t.Fatalf("initWorkbook: %v", err)
	}

	sheet := excel.Sheet{Name: "Bad:Name", Data: [][]any{{"a"}}}
	if err := exp.exportSheetStream(context.Background(), &sheet, 1); err == nil {
		t.Error("exportSheetStream with a colon in the name (index>0) = nil error, want error")
	}
}

// TestExportSheetStream_NewStreamWriterError verifies index==0 (which
// skips NewSheet, relying on the sheet already existing) with a sheet
// name that was never actually created surfaces excelize's
// NewStreamWriter error distinctly from the NewSheet error above.
func TestExportSheetStream_NewStreamWriterError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	if err := exp.initWorkbook("Sheet1"); err != nil {
		t.Fatalf("initWorkbook: %v", err)
	}

	sheet := excel.Sheet{Name: "DoesNotExist", Data: [][]any{{"a"}}}
	if err := exp.exportSheetStream(context.Background(), &sheet, 0); err == nil {
		t.Error("exportSheetStream (index 0) with a nonexistent sheet name = nil error, want error")
	}
}

// TestExportSheetStream_StreamFromChannelError verifies a
// streamFromChannel failure (context canceled up front) surfaces
// wrapped from exportSheetStream, with a best-effort Flush attempted
// first.
func TestExportSheetStream_StreamFromChannelError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	if err := exp.initWorkbook("Sheet1"); err != nil {
		t.Fatalf("initWorkbook: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch := make(chan []any, 1)
	ch <- []any{"a"}
	sheet := excel.Sheet{Name: "Sheet1", DataChannel: ch}
	if err := exp.exportSheetStream(ctx, &sheet, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("exportSheetStream() = %v, want error wrapping context.Canceled", err)
	}
}

// TestExportSheetStream_StreamFromStaticDataError verifies a
// streamFromStaticData failure (context canceled up front) surfaces
// wrapped from exportSheetStream.
func TestExportSheetStream_StreamFromStaticDataError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	if err := exp.initWorkbook("Sheet1"); err != nil {
		t.Fatalf("initWorkbook: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sheet := excel.Sheet{Name: "Sheet1", Data: [][]any{{"a"}}}
	if err := exp.exportSheetStream(ctx, &sheet, 0); !errors.Is(err, context.Canceled) {
		t.Errorf("exportSheetStream() = %v, want error wrapping context.Canceled", err)
	}
}

// TestValidateSheets_Error verifies validateSheets surfaces
// SheetCollection.Add's error for an invalid name directly.
func TestValidateSheets_Error(t *testing.T) {
	sheets := []excel.Sheet{{Name: ""}}
	if err := validateSheets(sheets); err == nil {
		t.Error("validateSheets with an empty sheet name = nil error, want error")
	}
}

// TestStreamFromChannel_ContextCanceled verifies streamFromChannel's
// own ctx.Done() branch.
func TestStreamFromChannel_ContextCanceled(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	sw, err := exp.file.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	exp.streamWriter = sw

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch := make(chan []any, 1)
	ch <- []any{"a"}
	if err := exp.streamFromChannel(ctx, ch); !errors.Is(err, context.Canceled) {
		t.Errorf("streamFromChannel() = %v, want error wrapping context.Canceled", err)
	}
}

// TestStreamFromChannel_SetRowError verifies a SetRow failure (writing
// to a row the stream writer already considers written) surfaces
// wrapped from streamFromChannel.
func TestStreamFromChannel_SetRowError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	sw, err := exp.file.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	if err := sw.SetRow("A1", []any{"pre-written"}); err != nil {
		t.Fatalf("SetRow: %v", err)
	}
	exp.streamWriter = sw

	ch := make(chan []any, 1)
	ch <- []any{"a"}
	close(ch)

	if err := exp.streamFromChannel(context.Background(), ch); err == nil {
		t.Error("streamFromChannel writing to an already-written row = nil error, want error")
	}
}

// TestStreamFromChannel_ProgressCallback verifies the progress
// callback fires once enough rows have been streamed from a channel.
func TestStreamFromChannel_ProgressCallback(t *testing.T) {
	var buf bytes.Buffer
	var calls int
	exp, err := NewStreamExporter(&buf, WithExportProgress(func(uint64, string) {
		calls++
	}, 2))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	sw, err := exp.file.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	exp.streamWriter = sw

	ch := make(chan []any, 3)
	ch <- []any{"a"}
	ch <- []any{"b"}
	ch <- []any{"c"}
	close(ch)

	if err := exp.streamFromChannel(context.Background(), ch); err != nil {
		t.Fatalf("streamFromChannel: %v", err)
	}
	if calls == 0 {
		t.Error("progressCallback was never called")
	}
}

// TestStreamFromStaticData_ContextCanceled verifies
// streamFromStaticData's own ctx.Done() branch.
func TestStreamFromStaticData_ContextCanceled(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	sw, err := exp.file.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	exp.streamWriter = sw

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := exp.streamFromStaticData(ctx, [][]any{{"a"}}); !errors.Is(err, context.Canceled) {
		t.Errorf("streamFromStaticData() = %v, want error wrapping context.Canceled", err)
	}
}

// TestStreamFromStaticData_SetRowError verifies a SetRow failure
// (writing to a row the stream writer already considers written)
// surfaces wrapped from streamFromStaticData.
func TestStreamFromStaticData_SetRowError(t *testing.T) {
	var buf bytes.Buffer
	exp, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	sw, err := exp.file.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	if err := sw.SetRow("A1", []any{"pre-written"}); err != nil {
		t.Fatalf("SetRow: %v", err)
	}
	exp.streamWriter = sw

	if err := exp.streamFromStaticData(context.Background(), [][]any{{"a"}}); err == nil {
		t.Error("streamFromStaticData writing to an already-written row = nil error, want error")
	}
}

// TestStreamFromStaticData_ProgressCallback verifies the progress
// callback fires once enough static rows have been streamed.
func TestStreamFromStaticData_ProgressCallback(t *testing.T) {
	var buf bytes.Buffer
	var calls int
	exp, err := NewStreamExporter(&buf, WithExportProgress(func(uint64, string) {
		calls++
	}, 2))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	sw, err := exp.file.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	exp.streamWriter = sw

	if err := exp.streamFromStaticData(context.Background(), [][]any{{"a"}, {"b"}, {"c"}}); err != nil {
		t.Fatalf("streamFromStaticData: %v", err)
	}
	if calls == 0 {
		t.Error("progressCallback was never called")
	}
}
