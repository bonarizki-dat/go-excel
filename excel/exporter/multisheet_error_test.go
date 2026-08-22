package exporter

import (
	"context"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestExportToSheets_InitWorkbookFailure verifies a sheet name excelize
// itself rejects at rename time (a colon, which validateSheetName does
// not check even though excelize does) surfaces as an ExportToSheets
// error instead of panicking.
func TestExportToSheets_InitWorkbookFailure(t *testing.T) {
	sheets := []excel.Sheet{
		{Name: "Sheet:1", Data: [][]any{{"a"}}},
	}
	exp := NewExporter(nil)
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets with a colon in the first sheet name = nil error, want error")
	}
}

// TestExportToSheets_SecondSheetNewSheetFailure verifies the same colon
// rejection when it is the second (not first) sheet, exercising
// exportSheet's own NewSheet error path rather than initWorkbook's.
func TestExportToSheets_SecondSheetNewSheetFailure(t *testing.T) {
	sheets := []excel.Sheet{
		{Name: "First", Data: [][]any{{"a"}}},
		{Name: "Sheet:2", Data: [][]any{{"b"}}},
	}
	exp := NewExporter(nil)
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets with a colon in the second sheet name = nil error, want error")
	}
}

// TestExportToSheets_LoopContextCanceledMidway verifies the per-sheet
// ctx.Done() check in exportSheetsLoop stops processing once the
// context is canceled between sheets.
func TestExportToSheets_LoopContextCanceledMidway(t *testing.T) {
	sheets := []excel.Sheet{
		{Name: "First", Data: [][]any{{"a"}}},
		{Name: "Second", Data: [][]any{{"b"}}},
	}
	exp := NewExporter(nil)

	ctx, cancel := context.WithCancel(context.Background())
	options := excel.DefaultMultiSheetExportOptions()
	options.ProgressCallback = func(context.Context, int, string) error {
		cancel() // cancel after the first sheet completes
		return nil
	}

	err := exp.ExportToSheets(ctx, sheets, options)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("ExportToSheets() = %v, want error wrapping context.Canceled", err)
	}
}

// TestExportToSheets_ContinueOnError verifies a failing sheet is
// collected rather than aborting the whole export when ContinueOnError
// is set, and that the collected errors are reported at the end.
func TestExportToSheets_ContinueOnError(t *testing.T) {
	badConfig := config.DefaultConfig()
	badConfig.SheetName = "" // fails Config.Validate()

	sheets := []excel.Sheet{
		{Name: "Bad", Data: [][]any{{"a"}}, Config: badConfig},
		{Name: "Good", Data: [][]any{{"b"}}},
	}
	exp := NewExporter(nil)

	options := excel.DefaultMultiSheetExportOptions()
	options.ContinueOnError = true

	err := exp.ExportToSheets(context.Background(), sheets, options)
	if err == nil {
		t.Fatal("ExportToSheets with a failing sheet = nil error, want error summarizing collected failures")
	}

	// The good sheet should still have been written.
	sheetList := exp.GetFile().GetSheetList()
	found := false
	for _, name := range sheetList {
		if name == "Good" {
			found = true
		}
	}
	if !found {
		t.Errorf("sheets = %v, want 'Good' sheet to have been written despite 'Bad' failing", sheetList)
	}
}

// TestExportToSheets_LoopProgressCallbackError verifies a
// MultiSheetExportOptions.ProgressCallback error aborts the export.
func TestExportToSheets_LoopProgressCallbackError(t *testing.T) {
	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: [][]any{{"a"}}},
	}
	exp := NewExporter(nil)

	callbackErr := errors.New("progress callback failed")
	options := excel.DefaultMultiSheetExportOptions()
	options.ProgressCallback = func(context.Context, int, string) error {
		return callbackErr
	}

	if err := exp.ExportToSheets(context.Background(), sheets, options); !errors.Is(err, callbackErr) {
		t.Errorf("ExportToSheets() = %v, want error wrapping %v", err, callbackErr)
	}
}

// TestExportToSheets_DataChannel_NormalCompletion verifies a sheet's
// DataChannel that closes normally (without cancellation) is read to
// completion, exercising readFromChannel's non-canceled closed-channel
// branch that TestExportToSheets_DataChannel_ContextCancelled does not.
func TestExportToSheets_DataChannel_NormalCompletion(t *testing.T) {
	rows := make(chan []any, 2)
	rows <- []any{"a"}
	rows <- []any{"b"}
	close(rows)

	sheets := []excel.Sheet{
		{Name: "Streamed", DataChannel: rows},
	}
	exp := NewExporter(nil)
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets: %v", err)
	}

	got, err := exp.GetFile().GetRows("Streamed")
	if err != nil {
		t.Fatalf("GetRows: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d rows, want 2", len(got))
	}
}

// TestExportToSheets_PerSheetConfigInvalid verifies a per-sheet Config
// that fails validation surfaces as an ExportToSheets error (the
// non-CSV path; exportToSheetsCSV's equivalent check is covered
// separately).
func TestExportToSheets_PerSheetConfigInvalid(t *testing.T) {
	badConfig := config.DefaultConfig()
	badConfig.SheetName = ""

	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: [][]any{{"a"}}, Config: badConfig},
	}
	exp := NewExporter(nil)
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets with an invalid per-sheet config = nil error, want error")
	}
}

// TestExportToSheets_RowWriteFailurePropagates verifies a row that
// fails to write (via an out-of-range column count) surfaces through
// exportRowsToSheet/exportSheet as an ExportToSheets error.
func TestExportToSheets_RowWriteFailurePropagates(t *testing.T) {
	hugeRow := make([]any, 16385) // exceeds excelize's 16384 column limit
	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: [][]any{hugeRow}, Config: func() *config.Config {
			c := config.DefaultConfig()
			c.WithHeadings = false
			return c
		}()},
	}
	exp := NewExporter(nil)
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets with a too-wide row = nil error, want error")
	}
}

// TestExportToSheets_HeaderRowWriteFailurePropagates verifies the same
// out-of-range failure for the header row specifically (WithHeadings
// true, the default), exercising exportRowsToSheet's header branch.
func TestExportToSheets_HeaderRowWriteFailurePropagates(t *testing.T) {
	hugeRow := make([]any, 16385)
	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: [][]any{hugeRow, {"a"}}},
	}
	exp := NewExporter(nil)
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets with a too-wide header row = nil error, want error")
	}
}

// TestExportToSheets_DataRowLoop_ContextCanceledMidway verifies
// exportRowsToSheet's own per-row ctx.Done() check. exportRowsToSheet
// does not dispatch BeforeRowExport/AfterRowExport events (unlike the
// single-sheet exportRows), so the periodic per-100-rows
// ProgressCallback is used instead to cancel partway through the loop.
func TestExportToSheets_DataRowLoop_ContextCanceledMidway(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := config.DefaultConfig()
	cfg.WithHeadings = false
	cfg.ProgressCallback = func(processed, _ int64) {
		if processed == 100 {
			cancel()
		}
	}

	rows := make([][]any, 150)
	for i := range rows {
		rows[i] = []any{i}
	}
	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: rows, Config: cfg},
	}
	exp := NewExporter(nil)

	err := exp.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())
	if !errors.Is(err, context.Canceled) {
		t.Errorf("ExportToSheets() = %v, want error wrapping context.Canceled", err)
	}
}

// TestExportToSheets_DataRowWriteFailure_Collected verifies a per-row
// write failure is collected into GetErrors when the sheet's Config has
// StopOnError false, rather than aborting immediately.
func TestExportToSheets_DataRowWriteFailure_Collected(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WithHeadings = false
	cfg.StopOnError = false

	rows := [][]any{make([]any, 16385), {1}}
	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: rows, Config: cfg},
	}
	exp := NewExporter(nil)

	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets with StopOnError=false should not itself fail: %v", err)
	}
	if len(exp.errors) == 0 {
		t.Error("exp.errors is empty, want the collected row write failure")
	}
}

// TestExportToSheets_DataRowWriteFailure_ErrorCallbackStops verifies an
// ErrorCallback returning false aborts exportRowsToSheet even with
// StopOnError false.
func TestExportToSheets_DataRowWriteFailure_ErrorCallbackStops(t *testing.T) {
	var callbackCalls int
	cfg := config.DefaultConfig()
	cfg.WithHeadings = false
	cfg.StopOnError = false
	cfg.ErrorCallback = func(int, error) bool {
		callbackCalls++
		return false
	}

	rows := [][]any{make([]any, 16385), {1}}
	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: rows, Config: cfg},
	}
	exp := NewExporter(nil)

	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets with a stopping ErrorCallback = nil error, want error")
	}
	if callbackCalls == 0 {
		t.Error("ErrorCallback was never called")
	}
}

// TestExportToSheetsCSV_DataChannel verifies exportToSheetsCSV reads a
// sheet's DataChannel to completion instead of only supporting the
// Data slice field.
func TestExportToSheetsCSV_DataChannel(t *testing.T) {
	rows := make(chan []any, 2)
	rows <- []any{"a"}
	rows <- []any{"b"}
	close(rows)

	sheets := []excel.Sheet{
		{Name: "Streamed", DataChannel: rows},
	}
	exp := NewExporter(nil, excel.WithFormat(excel.FileFormatCSV), excel.WithHeadings(false))
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets: %v", err)
	}
}

// TestExportToSheetsCSV_DataChannel_ContextCanceled verifies a canceled
// context while reading a CSV sheet's DataChannel surfaces as an error.
func TestExportToSheetsCSV_DataChannel_ContextCanceled(t *testing.T) {
	rows := make(chan []any)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		rows <- []any{"a"}
		cancel()
	}()

	sheets := []excel.Sheet{
		{Name: "Streamed", DataChannel: rows},
	}
	exp := NewExporter(nil, excel.WithFormat(excel.FileFormatCSV))
	err := exp.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())
	if !errors.Is(err, context.Canceled) {
		t.Errorf("ExportToSheets() = %v, want error wrapping context.Canceled", err)
	}
}

// TestExportToSheetsCSV_PerSheetConfig verifies a valid per-sheet
// Config override is applied (and restored afterward) for the CSV path.
func TestExportToSheetsCSV_PerSheetConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WithHeadings = false

	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: [][]any{{"a"}}, Config: cfg},
	}
	exp := NewExporter(nil, excel.WithFormat(excel.FileFormatCSV))
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets: %v", err)
	}
	if exp.GetConfig().WithHeadings != true {
		t.Error("exporter's own config should be restored to WithHeadings=true after the sheet-scoped override")
	}
}

// TestExportToSheetsCSV_PerSheetConfigInvalid verifies an invalid
// per-sheet Config surfaces as an error for the CSV path too.
func TestExportToSheetsCSV_PerSheetConfigInvalid(t *testing.T) {
	badConfig := config.DefaultConfig()
	badConfig.SheetName = ""

	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: [][]any{{"a"}}, Config: badConfig},
	}
	exp := NewExporter(nil, excel.WithFormat(excel.FileFormatCSV))
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err == nil {
		t.Error("ExportToSheets with an invalid per-sheet config = nil error, want error")
	}
}

// TestExportToSheetsCSV_ExportFailurePropagates verifies a failure from
// the underlying Export() call (empty row data) surfaces as an
// ExportToSheets error.
func TestExportToSheetsCSV_ExportFailurePropagates(t *testing.T) {
	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: [][]any{}},
	}
	options := excel.DefaultMultiSheetExportOptions()
	options.SkipEmptySheets = false
	exp := NewExporter(nil, excel.WithFormat(excel.FileFormatCSV))
	if err := exp.ExportToSheets(context.Background(), sheets, options); err == nil {
		t.Error("ExportToSheets with empty sheet data = nil error, want error")
	}
}

// TestExportToSheetsCSV_ProgressCallback verifies
// MultiSheetExportOptions.ProgressCallback fires for the CSV path.
func TestExportToSheetsCSV_ProgressCallback(t *testing.T) {
	var called bool
	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: [][]any{{"a"}}},
	}
	options := excel.DefaultMultiSheetExportOptions()
	options.ProgressCallback = func(context.Context, int, string) error {
		called = true
		return nil
	}
	exp := NewExporter(nil, excel.WithFormat(excel.FileFormatCSV))
	if err := exp.ExportToSheets(context.Background(), sheets, options); err != nil {
		t.Fatalf("ExportToSheets: %v", err)
	}
	if !called {
		t.Error("ProgressCallback was never called")
	}
}

// TestExportToSheetsCSV_ProgressCallbackError verifies a
// ProgressCallback error surfaces as an ExportToSheets error for the
// CSV path.
func TestExportToSheetsCSV_ProgressCallbackError(t *testing.T) {
	callbackErr := errors.New("progress callback failed")
	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: [][]any{{"a"}}},
	}
	options := excel.DefaultMultiSheetExportOptions()
	options.ProgressCallback = func(context.Context, int, string) error {
		return callbackErr
	}
	exp := NewExporter(nil, excel.WithFormat(excel.FileFormatCSV))
	if err := exp.ExportToSheets(context.Background(), sheets, options); !errors.Is(err, callbackErr) {
		t.Errorf("ExportToSheets() = %v, want error wrapping %v", err, callbackErr)
	}
}

// TestBasicExporter_ExportRowsToSheet_ContextCanceled exercises
// exportRowsToSheet's propagation of a canceled context through the
// sink's own ctx.Done() check on the header row write. Reaching it
// through ExportToSheets would require canceling the context in the
// narrow window between exportSheetsLoop's per-sheet check and the
// header row's write, which every other test's own ctx.Done() checks
// intercept first; calling exportRowsToSheet directly avoids that race.
func TestBasicExporter_ExportRowsToSheet_ContextCanceled(t *testing.T) {
	exp := NewExporter(nil)
	if err := exp.initWorkbook("Sheet1"); err != nil {
		t.Fatalf("initWorkbook: %v", err)
	}
	sink, ok := exp.sink.(*xlsxSink)
	if !ok {
		t.Fatalf("exp.sink = %T, want *xlsxSink", exp.sink)
	}
	if err := sink.setSheet("Sheet1"); err != nil {
		t.Fatalf("setSheet: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := exp.exportRowsToSheet(ctx, sink, [][]any{{"a"}}, config.DefaultConfig()); !errors.Is(err, context.Canceled) {
		t.Errorf("exportRowsToSheet() = %v, want error wrapping context.Canceled", err)
	}
}

// TestExportToSheets_PerSheetProgressCallback verifies a sheet Config's
// ProgressCallback (as opposed to MultiSheetExportOptions'
// ProgressCallback) fires during exportRowsToSheet.
func TestExportToSheets_PerSheetProgressCallback(t *testing.T) {
	var calls int
	cfg := config.DefaultConfig()
	cfg.WithHeadings = false
	cfg.ProgressCallback = func(int64, int64) {
		calls++
	}

	sheets := []excel.Sheet{
		{Name: "Sheet1", Data: [][]any{{1}, {2}}, Config: cfg},
	}
	exp := NewExporter(nil)
	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets: %v", err)
	}
	if calls == 0 {
		t.Error("per-sheet ProgressCallback was never called")
	}
}
