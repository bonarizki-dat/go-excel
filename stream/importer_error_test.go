package stream

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/internal/concurrency"
	"github.com/xuri/excelize/v2"
)

// minimalXLSX builds a one-row, one-cell XLSX workbook in memory for
// tests that only need a valid file to open, not specific content.
func minimalXLSX(t *testing.T) *bytes.Buffer {
	t.Helper()
	f := excelize.NewFile()
	if err := f.SetCellValue("Sheet1", "A1", "a"); err != nil {
		t.Fatalf("SetCellValue: %v", err)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	return &buf
}

// TestNewStreamImporter_ReadHeadingsError verifies a malformed first
// CSV row (a bare quote after a quoted field, which encoding/csv
// rejects) surfaces as a NewStreamImporter error via readHeadings,
// instead of only being detectable once ImportStream starts reading
// data rows.
func TestNewStreamImporter_ReadHeadingsError(t *testing.T) {
	r := strings.NewReader(`a,"b"c` + "\n")

	_, err := NewStreamImporter(r, WithImportFormat(config.FileFormatCSV), WithImportHeadings())
	if err == nil {
		t.Error("NewStreamImporter with a malformed heading row = nil error, want error")
	}
}

// TestNewStreamImporter_SheetNotFound verifies an explicit
// WithImportSheetName naming a sheet that does not exist in the
// workbook surfaces initXLSXSource's newXLSXStreamSource error,
// distinct from the "no sheets found" case where no name was given at
// all.
func TestNewStreamImporter_SheetNotFound(t *testing.T) {
	buf := minimalXLSX(t)

	_, err := NewStreamImporter(buf, WithImportSheetName("DoesNotExist"))
	if err == nil {
		t.Error("NewStreamImporter with a nonexistent sheet name = nil error, want error")
	}
}

// TestStreamImporter_GetStats_BeforeImport verifies GetStats returns
// the zero WorkerStats when called before ImportStream has ever run,
// rather than panicking on a nil workerPool.
func TestStreamImporter_GetStats_BeforeImport(t *testing.T) {
	buf := minimalXLSX(t)

	si, err := NewStreamImporter(buf)
	if err != nil {
		t.Fatalf("NewStreamImporter: %v", err)
	}
	defer si.Close()

	got := si.GetStats()
	if got != (WorkerStats{}) {
		t.Errorf("GetStats() = %+v, want the zero value", got)
	}
}

// TestStreamImporter_GetStats_AfterImport verifies GetStats delegates
// to the worker pool's own Stats() once ImportStream has created one,
// rather than only ever returning the pre-import zero value.
func TestStreamImporter_GetStats_AfterImport(t *testing.T) {
	content := "a\nb\n"
	si, err := NewStreamImporter(strings.NewReader(content), WithImportFormat(config.FileFormatCSV))
	if err != nil {
		t.Fatalf("NewStreamImporter: %v", err)
	}
	defer si.Close()

	err = si.ImportStream(context.Background(), func(_ context.Context, _ int, row []any) ([]any, error) {
		return row, nil
	})
	if err != nil {
		t.Fatalf("ImportStream: %v", err)
	}

	if got := si.GetStats().TasksSubmitted; got != 2 {
		t.Errorf("GetStats().TasksSubmitted = %d, want 2", got)
	}
}

// failCloseSource is a streamRowSource whose close always fails, used
// to exercise StreamImporter.Close's propagation of a source close
// error.
type failCloseSource struct{}

func (failCloseSource) next() ([]string, error) { return nil, io.EOF }
func (failCloseSource) close() error            { return errSourceCloseFailed }

var errSourceCloseFailed = errors.New("source close failed")

// TestReadHeadings_EmptySource verifies readHeadings returns a nil
// error (leaving si.headings unset) when the source is exhausted
// immediately, rather than treating EOF on the very first read as a
// failure.
func TestReadHeadings_EmptySource(t *testing.T) {
	si, err := NewStreamImporter(strings.NewReader(""), WithImportFormat(config.FileFormatCSV), WithImportHeadings())
	if err != nil {
		t.Fatalf("NewStreamImporter: %v", err)
	}
	defer si.Close()

	if len(si.GetHeadings()) != 0 {
		t.Errorf("GetHeadings() = %v, want empty", si.GetHeadings())
	}
}

// TestSendRow_RowWiderThanPooledBuffer verifies sendRow replaces a
// pooled buffer too small for row (rowPool preallocates for ~100
// columns) instead of silently truncating or panicking on the
// out-of-bounds index assignment that would follow.
func TestSendRow_RowWiderThanPooledBuffer(t *testing.T) {
	si, err := NewStreamImporter(strings.NewReader(""), WithImportFormat(config.FileFormatCSV))
	if err != nil {
		t.Fatalf("NewStreamImporter: %v", err)
	}
	defer si.Close()

	wideRow := make([]string, 200)
	for i := range wideRow {
		wideRow[i] = "x"
	}

	rowChan := make(chan []any, 1)
	if err := si.sendRow(context.Background(), rowChan, wideRow); err != nil {
		t.Fatalf("sendRow: %v", err)
	}

	got := <-rowChan
	if len(got) != 200 {
		t.Errorf("sent row has %d cells, want 200", len(got))
	}
}

// TestImportStream_ReaderError verifies a mid-stream read failure (a
// malformed CSV row after a valid heading row) surfaces through
// dispatchRows and finishDispatch as an ImportStream error, exercising
// readRows' own error-wrapping return along with both places that
// carry readerErr to the final result.
func TestImportStream_ReaderError(t *testing.T) {
	content := "h1,h2\n" + `a,"b"c` + "\n"
	si, err := NewStreamImporter(strings.NewReader(content), WithImportFormat(config.FileFormatCSV), WithImportHeadings())
	if err != nil {
		t.Fatalf("NewStreamImporter: %v", err)
	}
	defer si.Close()

	err = si.ImportStream(context.Background(), func(_ context.Context, _ int, row []any) ([]any, error) {
		return row, nil
	})
	if err == nil {
		t.Error("ImportStream with a malformed data row = nil error, want error")
	}
}

// TestTaskProcessor_InvalidDataType verifies the ProcessFunc built by
// taskProcessor reports an error instead of panicking when task.Data
// is not []any. Unreachable through the public ImportStream API
// (dispatchRows only ever submits rows read from rowChan), so this
// calls taskProcessor's returned function directly.
func TestTaskProcessor_InvalidDataType(t *testing.T) {
	si := &StreamImporter{}
	fn := si.taskProcessor(func(_ context.Context, _ int, row []any) ([]any, error) { return row, nil })

	if _, err := fn(context.Background(), concurrency.Task{ID: 1, Data: "not a row"}); err == nil {
		t.Error("taskProcessor's ProcessFunc with non-[]any Data = nil error, want error")
	}
}

// TestDispatchRows_SubmitError exercises dispatchRows' Submit failure
// branch directly: the worker pool's own context is already canceled
// (independent of the ctx parameter passed to dispatchRows, which
// stays live), so Submit fails deterministically on the very first row
// without racing dispatchRows' own ctx.Done() case.
func TestDispatchRows_SubmitError(t *testing.T) {
	si := &StreamImporter{}
	pool := concurrency.NewWorkerPool(1, nil)
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	pool.SetContextForTest(canceledCtx)
	si.workerPool = pool

	// Fill taskChan so Submit's send can never be the ready case;
	// otherwise Submit's own select races between sending (taskChan
	// has room) and ctx.Done() (also ready), and would sometimes
	// succeed instead of deterministically failing.
	taskChan := pool.TaskChanForTest()
	for i := 0; i < cap(taskChan); i++ {
		taskChan <- concurrency.Task{ID: uint64(i)}
	}

	rowChan := make(chan []any, 1)
	rowChan <- []any{"a"}

	workerErrChan := make(chan error)
	drainerDone := make(chan struct{})
	errChan := make(chan error)

	err := si.dispatchRows(context.Background(), rowChan, workerErrChan, drainerDone, errChan)
	if err == nil {
		t.Error("dispatchRows with an already-canceled worker pool context = nil error, want error")
	}
}

// TestDispatchRows_ContextCanceled exercises dispatchRows' own
// ctx.Done() case directly: rowChan, workerErrChan, and errChan are
// all left empty and unclosed so none of them can ever be ready,
// isolating this from the workerPool's separate context used for
// Submit (see TestDispatchRows_SubmitError).
func TestDispatchRows_ContextCanceled(t *testing.T) {
	si := &StreamImporter{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rowChan := make(chan []any)
	workerErrChan := make(chan error)
	drainerDone := make(chan struct{})
	errChan := make(chan error)

	err := si.dispatchRows(ctx, rowChan, workerErrChan, drainerDone, errChan)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("dispatchRows() = %v, want error wrapping context.Canceled", err)
	}
}

// TestFinishDispatch_WorkerErrorTakesPriority verifies finishDispatch
// surfaces a buffered worker error (as drainWorkerErrors would leave
// it once Results() closes) instead of falling through to readerErr,
// confirming worker errors take priority as documented.
func TestFinishDispatch_WorkerErrorTakesPriority(t *testing.T) {
	si := &StreamImporter{workerPool: concurrency.NewWorkerPool(1, nil)}

	workerErr := errors.New("worker failed")
	workerErrChan := make(chan error, 1)
	workerErrChan <- workerErr
	close(workerErrChan)

	drainerDone := make(chan struct{})
	close(drainerDone)

	err := si.finishDispatch(workerErrChan, drainerDone, errors.New("reader also failed"))
	if !errors.Is(err, workerErr) {
		t.Errorf("finishDispatch() = %v, want error wrapping the worker error", err)
	}
}

// TestStreamImporter_Close_SourceCloseError verifies Close surfaces a
// source.close() failure instead of silently discarding it (and
// without going on to also close si.file, matching the early return).
func TestStreamImporter_Close_SourceCloseError(t *testing.T) {
	si := &StreamImporter{source: failCloseSource{}}

	if err := si.Close(); !errors.Is(err, errSourceCloseFailed) {
		t.Errorf("Close() = %v, want error wrapping %v", err, errSourceCloseFailed)
	}
}
