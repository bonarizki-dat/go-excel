package stream

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/bonarizki-dat/go-excel/internal/concurrency"
)

// ImportStream orchestration. Options and read helpers stay in their own files.

// ImportStream reads rows from the importer's source as they arrive,
// processing each one concurrently across a worker pool by calling
// processor. It returns once every row has been processed or once ctx
// is canceled, returning any resulting error.
func (si *StreamImporter) ImportStream(ctx context.Context, processor RowProcessor) error {
	rowChan := make(chan []any, si.numWorkers*2)

	si.workerPool = concurrency.NewWorkerPool(si.numWorkers, si.taskProcessor(processor))

	// readerWG and cancelRun must be declared in this exact order so
	// their deferred calls run in reverse: workerPool.Stop() first
	// (registered below, after Start), then cancelRun() (unblocks any
	// pending send in readRows), then readerWG.Wait() last. Reversing
	// this order would make Wait() run before cancelRun() and deadlock
	// forever on a reader stuck sending to a channel nobody drains.
	var readerWG sync.WaitGroup
	defer readerWG.Wait()
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	si.workerPool.Start(runCtx)
	defer si.workerPool.Stop()

	workerErrChan := make(chan error, 1)
	drainerDone := make(chan struct{})
	go si.drainWorkerErrors(workerErrChan, drainerDone)

	errChan := make(chan error, 1)
	readerWG.Add(1)
	go si.runReader(runCtx, rowChan, errChan, &readerWG)

	return si.dispatchRows(ctx, rowChan, workerErrChan, drainerDone, errChan)
}

// taskProcessor adapts processor into the ProcessFunc shape expected by
// the worker pool, decoding task.Data back into the row it wrapped.
func (si *StreamImporter) taskProcessor(processor RowProcessor) concurrency.ProcessFunc {
	return func(ctx context.Context, task concurrency.Task) (any, error) {
		rowData, ok := task.Data.([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected task data type %T for row task %d", task.Data, task.ID)
		}
		rowNum := int(task.ID)

		// rowData is not returned to the pool afterward: PutRow
		// immediately clears and recycles the slice, so any processor
		// that retains it (e.g. rows = append(rows, rowData)) would
		// later read a slice another task has since overwritten. The
		// row buffer pool now only ever hands out buffers that stay
		// internal to this package; see GetRow's caller in
		// importer_read.go.
		return processor(ctx, rowNum, rowData)
	}
}

// drainWorkerErrors forwards the first non-nil error observed on
// si.workerPool.Results() onto workerErrChan (subsequent errors are
// dropped; the first is enough to know the run failed), then closes
// workerErrChan once Results() closes. done is closed last, so a
// caller blocked on it knows every result has been drained and
// workerErrChan will not receive anything further.
func (si *StreamImporter) drainWorkerErrors(workerErrChan chan error, done chan struct{}) {
	defer close(done)
	for res := range si.workerPool.Results() {
		if res.Err != nil {
			select {
			case workerErrChan <- res.Err:
			default:
			}
		}
	}
	close(workerErrChan)
}

// runReader reads rows into rowChan until runCtx is done or the source
// is exhausted, reporting any read error on errChan. rowChan and
// errChan are both closed on return so dispatchRows can observe reader
// completion either way, and wg.Done unblocks ImportStream's deferred
// Wait.
func (si *StreamImporter) runReader(runCtx context.Context, rowChan chan []any, errChan chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(rowChan)
	defer close(errChan)

	if err := si.readRows(runCtx, rowChan); err != nil {
		errChan <- err
	}
}

// dispatchRows is ImportStream's main event loop: it submits rows
// arriving on rowChan to si.workerPool as tasks while watching
// workerErrChan and errChan for the first failure, and once rowChan
// closes, hands off to finishDispatch for shutdown and final error
// reporting.
func (si *StreamImporter) dispatchRows(ctx context.Context, rowChan <-chan []any, workerErrChan <-chan error, drainerDone <-chan struct{}, errChan <-chan error) error {
	var taskID uint64
	var readerErr error
	readerDone := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-workerErrChan:
			if err != nil {
				return fmt.Errorf("worker error: %w", err)
			}
			// workerErrChan closed, continue

		case err := <-errChan:
			// Reader finished (either error or success). Store the
			// error but keep draining rowChan so any buffered rows
			// are still submitted before this function returns.
			if err != nil {
				readerErr = err
			}
			readerDone = true

		case row, ok := <-rowChan:
			if !ok {
				return si.finishDispatch(workerErrChan, drainerDone, readerErr)
			}

			task := concurrency.Task{
				ID:   atomic.AddUint64(&taskID, 1),
				Data: row,
			}

			if err := si.workerPool.Submit(task); err != nil {
				return fmt.Errorf("submit row task: %w", err)
			}

			if readerDone {
				// Reader already finished, but rowChan may still hold
				// buffered rows; keep draining them.
				continue
			}
		}
	}
}

// finishDispatch runs once rowChan has closed with no more rows to
// submit. It stops the worker pool (closing taskChan so workers drain
// buffered tasks and exit, which in turn closes resultChan) and waits
// for the drainer goroutine to observe that close. This replaces a
// Stats()-polling busy loop: the drainer finishing is itself the exact
// signal that every submitted task has been accounted for, no
// invariant-checking needed. A worker error takes priority over
// readerErr in the returned error, since it means a row was actually
// processed and failed, not just that reading stopped early.
func (si *StreamImporter) finishDispatch(workerErrChan <-chan error, drainerDone <-chan struct{}, readerErr error) error {
	si.workerPool.Stop()
	<-drainerDone

	select {
	case err := <-workerErrChan:
		if err != nil {
			return fmt.Errorf("worker error: %w", err)
		}
	default:
	}

	if readerErr != nil {
		return fmt.Errorf("read rows: %w", readerErr)
	}
	return nil
}
