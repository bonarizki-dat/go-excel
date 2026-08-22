package stream

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/internal/concurrency"
)

// processRow is the ProcessFunc run concurrently across the worker
// pool: it extracts task's row data and, if a RowTransformer was
// configured, applies it. It never writes to the output file, so
// concurrent invocations never touch the shared streamWriter.
func (se *StreamExporter) processRow(ctx context.Context, task concurrency.Task) (any, error) {
	row, ok := task.Data.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid row data type")
	}

	if se.transform == nil {
		return row, nil
	}

	return se.transform(ctx, task.ID, row)
}

// runOrderedWriter is the sole owner of se.streamWriter for the duration
// of an export. It reassembles worker results into their original
// sequence and writes them in order, so output order matches input order
// regardless of how workers complete.
//
// pending buffers results that have arrived out of order, waiting for the
// row that precedes them. Its size never exceeds cap(sem), because a
// task's slot is only released once this loop dequeues it, which upper
// bounds pending regardless of how large the dataset is.
//
// sem is the in-flight slot semaphore shared with the feeder in
// ExportStream. cancel is called to end the run on the first error, so
// the feeder stops submitting further rows. done receives the first
// error encountered, or nil on success.
func (se *StreamExporter) runOrderedWriter(sem <-chan struct{}, cancel context.CancelFunc, done chan<- error) {
	pending := make(map[uint64]pendingRow, cap(sem))
	next := uint64(1)
	var firstErr error

	for res := range se.workerPool.Results() {
		pending[res.TaskID] = decodeRowResult(res)
		next, firstErr = se.drainReadyRows(pending, next, sem, cancel, firstErr)
	}

	// If Results() closed with entries still in pending, some rows
	// were submitted but never written: a worker abandoned its task
	// (see worker_process.go's handleTask) or was itself abandoned
	// before returning a result. Without this check, an incomplete
	// export would return a nil error and silently drop rows.
	if firstErr == nil && len(pending) > 0 {
		firstErr = fmt.Errorf("export incomplete: %d row(s) never written, stalled at sequence %d", len(pending), next)
	}

	if firstErr == nil {
		if err := se.sink.flush(); err != nil {
			firstErr = fmt.Errorf("flush stream writer: %w", err)
		}
	}

	done <- firstErr
}

// pendingRow is a worker result waiting in runOrderedWriter's reorder
// buffer for the row that precedes it.
type pendingRow struct {
	row []any
	err error
}

// decodeRowResult converts a raw worker Result into a pendingRow, turning
// an unexpected Data type into an error rather than panicking.
func decodeRowResult(res concurrency.Result) pendingRow {
	item := pendingRow{err: res.Err}
	if item.err != nil {
		return item
	}
	row, ok := res.Data.([]any)
	if !ok {
		item.err = fmt.Errorf("unexpected result data type %T for row task %d", res.Data, res.TaskID)
		return item
	}
	item.row = row
	return item
}

// drainReadyRows writes every contiguous entry in pending starting at
// next, in order, releasing each task's semaphore slot as it is
// dequeued, until it hits a gap. Once firstErr becomes non-nil (either
// passed in or set by a row in this drain), cancel is called and
// subsequent rows are still drained and released but no longer written,
// so the writer keeps making progress draining the channel instead of
// deadlocking against a canceled run. Returns the updated next sequence
// number and the first error encountered so far.
func (se *StreamExporter) drainReadyRows(pending map[uint64]pendingRow, next uint64, sem <-chan struct{}, cancel context.CancelFunc, firstErr error) (uint64, error) {
	for {
		item, ok := pending[next]
		if !ok {
			return next, firstErr
		}
		delete(pending, next)
		<-sem // this task's in-flight slot is now resolved

		if firstErr == nil {
			if item.err != nil {
				firstErr = item.err
				cancel()
			} else if err := se.writeDataRow(next, item.row); err != nil {
				firstErr = err
				cancel()
			}
		}
		next++
	}
}

// writeDataRow writes a single data row at the given sequence position.
//
// Called only from the writer goroutine, so no synchronization is
// needed around streamWriter here.
func (se *StreamExporter) writeDataRow(seq uint64, row []any) error {
	if err := se.sink.writeRow(se.dataRowOffset(seq), row); err != nil {
		return fmt.Errorf("write row %d: %w", seq, err)
	}

	atomic.AddUint64(&se.rowCounter, 1)
	if se.progressCallback != nil && seq%se.progressInterval == 0 {
		se.progressCallback(seq, fmt.Sprintf("Exported %d rows", seq))
	}

	return nil
}

// writeHeadings writes se.headings as the sheet's first row, or
// returns an error if the write fails.
func (se *StreamExporter) writeHeadings() error {
	headingRow := make([]any, len(se.headings))
	for i, h := range se.headings {
		headingRow[i] = h
	}
	if err := se.sink.writeRow(1, headingRow); err != nil {
		return fmt.Errorf("write headings: %w", err)
	}
	return nil
}

// writeToOutput finalizes the export after all rows are written and
// the sink has been flushed. CSV has already streamed directly to
// se.writer row by row, so there is nothing left to copy; se.file is
// unused for CSV but still closed since NewStreamExporter always
// creates it. XLSX still needs to serialize se.file to se.writer.
func (se *StreamExporter) writeToOutput() error {
	if se.format == config.FileFormatCSV {
		if err := se.file.Close(); err != nil {
			return fmt.Errorf("close file: %w", err)
		}
		return nil
	}

	if err := se.file.Write(se.writer); err != nil {
		return fmt.Errorf("write to output: %w", err)
	}

	if err := se.file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}
