package stream

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/bonarizki-dat/go-excel/internal/concurrency"
)

// ExportStream and the counters it updates.

// ExportStream reads rows from the rows channel, optionally transforms
// them concurrently across a worker pool, and writes them to the
// output file from a single writer goroutine in the order they were
// received, regardless of worker count or completion order. It returns
// once rows is closed and all rows have been written, or once ctx is
// canceled.
//
// It returns an error if export fails. If the context is canceled, the
// returned error satisfies errors.Is(err, context.Canceled) or
// errors.Is(err, context.DeadlineExceeded).
func (se *StreamExporter) ExportStream(ctx context.Context, rows <-chan []any) (retErr error) {
	// On any error return below, close the underlying file rather than
	// leaving it to the caller, since a caller that only checks the
	// returned error and moves on would otherwise never see this
	// exporter again to call Close() themselves.
	defer func() {
		if retErr != nil {
			_ = se.Close()
		}
	}()

	sink, err := se.newSink()
	if err != nil {
		return err
	}
	se.sink = sink

	// Write headings before any concurrent access to streamWriter begins;
	// only this goroutine touches it until the writer goroutine starts.
	if se.hasHeadingRow() {
		if err := se.writeHeadings(); err != nil {
			return fmt.Errorf("write headings: %w", err)
		}
	}

	// runCtx is canceled either by the caller's ctx or by the first
	// error encountered by a worker or the writer, so the feeder can
	// unblock promptly instead of continuing to feed a doomed export.
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	// inFlight bounds how many rows may be submitted before being
	// written. A row occupies a slot from submission until the writer
	// dequeues it in order, so the writer's reorder buffer can never
	// hold more than cap(inFlight) entries: memory stays proportional to
	// worker count, not dataset size.
	maxInFlight := se.numWorkers * 2
	if maxInFlight < 2 {
		maxInFlight = 2
	}
	inFlight := make(chan struct{}, maxInFlight)

	se.workerPool = concurrency.NewWorkerPool(se.numWorkers, se.processRow)
	se.workerPool.Start(runCtx)

	writerDone := make(chan error, 1)
	go se.runOrderedWriter(inFlight, cancelRun, writerDone)

	feedErr := se.feedRows(runCtx, rows, inFlight)

	// Closing taskChan lets workers drain any buffered tasks and exit,
	// which closes resultChan, which lets the writer goroutine finish
	// draining pending results and return.
	se.workerPool.Stop()
	writerErr := <-writerDone

	// A canceled ctx takes priority over writerErr/feedErr: both can
	// surface as a downstream symptom of cancellation (for example the
	// writer's "export incomplete" error, when a worker abandons a task
	// mid-processing instead of returning a result for it), and the
	// documented contract is that a canceled run's error always
	// satisfies errors.Is(err, context.Canceled) or
	// errors.Is(err, context.DeadlineExceeded).
	if err := ctx.Err(); err != nil {
		return err
	}
	if writerErr != nil {
		return writerErr
	}
	if feedErr != nil {
		return fmt.Errorf("submit row task: %w", feedErr)
	}

	if se.progressCallback != nil {
		finalCount := atomic.LoadUint64(&se.rowCounter)
		se.progressCallback(finalCount, fmt.Sprintf("Exported %d rows (complete)", finalCount))
	}

	return se.writeToOutput()
}

// feedRows reads rows from the input channel and submits each as a Task
// to se.workerPool, reserving a slot in inFlight first so the writer's
// reorder buffer stays bounded. Returns nil when rows closes or runCtx
// is done; returns the error from workerPool.Submit if submission
// itself fails (which only happens once runCtx is already done, but the
// caller still deserves to know submission was the failure point, not
// just that the run ended).
func (se *StreamExporter) feedRows(runCtx context.Context, rows <-chan []any, inFlight chan struct{}) error {
	var taskID uint64

	for {
		select {
		case <-runCtx.Done():
			return nil

		case row, ok := <-rows:
			if !ok {
				return nil
			}

			select {
			case inFlight <- struct{}{}:
			case <-runCtx.Done():
				return nil
			}

			task := concurrency.Task{ID: atomic.AddUint64(&taskID, 1), Data: row}
			if err := se.workerPool.Submit(task); err != nil {
				return err
			}
		}
	}
}

// GetRowCount returns the number of rows written so far. It is safe to
// call concurrently with a running ExportStream.
func (se *StreamExporter) GetRowCount() uint64 {
	return atomic.LoadUint64(&se.rowCounter)
}

// GetStats returns a snapshot of the underlying worker pool's
// statistics, for monitoring export progress and performance. It
// returns the zero WorkerStats if ExportStream has not yet started.
func (se *StreamExporter) GetStats() WorkerStats {
	if se.workerPool != nil {
		return se.workerPool.Stats()
	}
	return WorkerStats{}
}
