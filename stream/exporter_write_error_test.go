package stream

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/internal/concurrency"
)

// TestExportStream_MaxInFlightFloor verifies ExportStream clamps
// maxInFlight to at least 2 when se.numWorkers is set low enough that
// numWorkers*2 would otherwise fall below it. Unreachable through the
// public WithExportWorkers option, which ignores non-positive counts,
// so numWorkers is set directly.
func TestExportStream_MaxInFlightFloor(t *testing.T) {
	var buf bytes.Buffer
	se, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	se.numWorkers = 0

	rows := make(chan []any, 1)
	rows <- []any{"a"}
	close(rows)

	if err := se.ExportStream(context.Background(), rows); err != nil {
		t.Fatalf("ExportStream: %v", err)
	}
}

// TestFeedRows_InFlightReservation_ContextCanceled exercises feedRows'
// inner select specifically (blocked reserving an inFlight slot),
// distinct from the outer select's runCtx.Done() case: inFlight is
// pre-filled to capacity so the reservation attempt blocks, then
// runCtx is canceled while it is blocked.
func TestFeedRows_InFlightReservation_ContextCanceled(t *testing.T) {
	var buf bytes.Buffer
	se, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	se.workerPool = concurrency.NewWorkerPool(1, se.processRow)
	se.workerPool.SetContextForTest(runCtx)

	inFlight := make(chan struct{}, 1)
	inFlight <- struct{}{} // full: the reservation attempt below must block

	rows := make(chan []any, 1)
	rows <- []any{"a"}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	done := make(chan error, 1)
	go func() { done <- se.feedRows(runCtx, rows, inFlight) }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("feedRows() = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("feedRows did not return after runCtx was canceled while blocked reserving a slot")
	}
}

// TestProcessRow_InvalidDataType verifies processRow reports an error
// instead of panicking when task.Data is not []any. Unreachable
// through the public ExportStream API (feedRows only ever submits rows
// read from a <-chan []any), so this calls the unexported method
// directly.
func TestProcessRow_InvalidDataType(t *testing.T) {
	var buf bytes.Buffer
	se, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	if _, err := se.processRow(context.Background(), concurrency.Task{Data: "not a row"}); err == nil {
		t.Error("processRow with non-[]any Data = nil error, want error")
	}
}

// TestDecodeRowResult_ErrPresent verifies decodeRowResult passes
// through a Result's existing error without inspecting Data.
func TestDecodeRowResult_ErrPresent(t *testing.T) {
	wantErr := errors.New("task failed")
	got := decodeRowResult(concurrency.Result{TaskID: 1, Err: wantErr})
	if !errors.Is(got.err, wantErr) {
		t.Errorf("decodeRowResult().err = %v, want %v", got.err, wantErr)
	}
}

// TestDecodeRowResult_UnexpectedDataType verifies decodeRowResult
// turns a non-[]any Data value into an error rather than panicking on
// a failed type assertion.
func TestDecodeRowResult_UnexpectedDataType(t *testing.T) {
	got := decodeRowResult(concurrency.Result{TaskID: 2, Data: "not a row"})
	if got.err == nil {
		t.Error("decodeRowResult with non-[]any Data = nil error, want error")
	}
}

// TestRunOrderedWriter_ExportIncomplete verifies runOrderedWriter
// reports "export incomplete" when the results channel closes with an
// unresolved gap in the sequence: task 1's slot is reserved but never
// resolved (simulating a worker that abandoned it), while task 2
// completes and lands in the reorder buffer waiting for task 1 that
// will never arrive.
func TestRunOrderedWriter_ExportIncomplete(t *testing.T) {
	var buf bytes.Buffer
	se, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	sw, err := se.file.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	se.streamWriter = sw
	se.sink = newXLSXStreamSink(sw)

	pool := concurrency.NewWorkerPool(1, nil)
	se.workerPool = pool

	sem := make(chan struct{}, 4)
	sem <- struct{}{} // task 1's reserved slot, never released
	sem <- struct{}{} // task 2's reserved slot

	_, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go se.runOrderedWriter(sem, cancel, done)

	resultChan := pool.ResultChanForTest()
	resultChan <- concurrency.Result{TaskID: 2, Data: []any{"b"}}
	close(resultChan)

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "export incomplete") {
			t.Errorf("runOrderedWriter error = %v, want an 'export incomplete' error", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runOrderedWriter did not return after resultChan closed")
	}
}

// failFlushSink is a streamRowSink whose flush always fails, used to
// exercise runOrderedWriter's flush-error branch without needing a
// genuine excelize or csvio failure.
type failFlushSink struct{}

func (failFlushSink) writeRow(int, []any) error { return nil }
func (failFlushSink) flush() error              { return errors.New("flush failed") }

// TestRunOrderedWriter_FlushError verifies a sink.flush() failure
// surfaces as runOrderedWriter's error when there are no row errors and
// nothing left pending.
func TestRunOrderedWriter_FlushError(t *testing.T) {
	var buf bytes.Buffer
	se, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	se.sink = failFlushSink{}

	pool := concurrency.NewWorkerPool(1, nil)
	se.workerPool = pool
	close(pool.ResultChanForTest()) // no results at all; pending stays empty

	sem := make(chan struct{}, 1)
	_, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go se.runOrderedWriter(sem, cancel, done)

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "flush stream writer") {
			t.Errorf("runOrderedWriter error = %v, want a flush error", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runOrderedWriter did not return")
	}
}

// TestWriteDataRow_SinkError verifies a sink.writeRow failure (writing
// to a row the stream writer already considers written) surfaces
// wrapped from writeDataRow.
func TestWriteDataRow_SinkError(t *testing.T) {
	var buf bytes.Buffer
	se, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	sw, err := se.file.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	if err := sw.SetRow("A1", []any{"pre-written"}); err != nil {
		t.Fatalf("SetRow: %v", err)
	}
	se.sink = newXLSXStreamSink(sw)

	if err := se.writeDataRow(1, []any{"a"}); err == nil {
		t.Error("writeDataRow writing to an already-written row = nil error, want error")
	}
}

// TestWriteDataRow_ProgressCallback verifies the progress callback
// fires when the sequence number lands on the configured interval.
func TestWriteDataRow_ProgressCallback(t *testing.T) {
	var buf bytes.Buffer
	var calls int
	se, err := NewStreamExporter(&buf, WithExportProgress(func(uint64, string) {
		calls++
	}, 1))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	sw, err := se.file.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	se.sink = newXLSXStreamSink(sw)

	if err := se.writeDataRow(1, []any{"a"}); err != nil {
		t.Fatalf("writeDataRow: %v", err)
	}
	if calls == 0 {
		t.Error("progressCallback was never called")
	}
}

// TestWriteHeadings_SinkError verifies a sink.writeRow failure (row 1
// already written on the stream writer) surfaces wrapped from
// writeHeadings.
func TestWriteHeadings_SinkError(t *testing.T) {
	var buf bytes.Buffer
	se, err := NewStreamExporter(&buf, WithExportHeadings([]string{"a", "b"}))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}
	sw, err := se.file.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	if err := sw.SetRow("A1", []any{"pre-written"}); err != nil {
		t.Fatalf("SetRow: %v", err)
	}
	se.sink = newXLSXStreamSink(sw)

	if err := se.writeHeadings(); err == nil {
		t.Error("writeHeadings writing to an already-written row = nil error, want error")
	}
}

// TestExportStream_WriterErrorPropagates verifies a runOrderedWriter
// failure (a data row long enough to overflow encoding/csv's internal
// buffered writer, forcing a synchronous write to a failing
// io.Writer) surfaces as ExportStream's error.
//
// writerErr and feedErr (ExportStream's two post-loop error checks
// after ctx.Err()) are not independently reachable: feedErr can only
// come from workerPool.Submit observing runCtx.Done(), and the only
// thing that cancels runCtx without also canceling the caller's ctx is
// drainReadyRows on an error, which always leaves writerErr non-nil
// too. So whenever feedErr would be set, writerErr already is, and
// this branch takes priority; feedErr's own check is effectively
// unreachable dead code guarding against a future change in how
// cancellation works, not a gap in these tests.
func TestExportStream_WriterErrorPropagates(t *testing.T) {
	longCell := strings.Repeat("x", 8192)
	opts := config.DefaultCSVOptions()
	opts.UseBOM = false

	se, err := NewStreamExporter(failWriteCloser{},
		WithExportFormat(config.FileFormatCSV),
		WithExportCSVOptions(opts),
	)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	rows := make(chan []any, 1)
	rows <- []any{longCell}
	close(rows)

	if err := se.ExportStream(context.Background(), rows); err == nil {
		t.Error("ExportStream with a writer that fails mid-row = nil error, want error")
	}
}

// TestExportStream_WriteHeadingsErrorPropagates verifies a
// writeHeadings failure inside ExportStream itself (a CSV writer whose
// underlying io.Writer fails, forced to surface synchronously by using
// a long-enough row to overflow encoding/csv's internal buffered
// writer) aborts the export before any worker or writer goroutine
// starts.
func TestExportStream_WriteHeadingsErrorPropagates(t *testing.T) {
	longHeading := strings.Repeat("x", 8192)
	opts := config.DefaultCSVOptions()
	opts.UseBOM = false

	se, err := NewStreamExporter(failWriteCloser{},
		WithExportFormat(config.FileFormatCSV),
		WithExportCSVOptions(opts),
		WithExportHeadings([]string{longHeading}),
	)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	rows := make(chan []any)
	close(rows)

	if err := se.ExportStream(context.Background(), rows); err == nil {
		t.Error("ExportStream with a failing writer during headings = nil error, want error")
	}
}
