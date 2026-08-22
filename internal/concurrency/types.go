package concurrency

import (
	"context"
	"fmt"
)

// Task represents a unit of work to be processed by workers.
//
// Tasks are submitted to the worker pool and processed concurrently.
// Each task contains the data and metadata needed for processing.
type Task struct {
	// ID uniquely identifies this task, for tracking and debugging.
	ID uint64

	// Data is the payload to be processed; its concrete type depends
	// on the use case (a row, a batch, and so on).
	Data any
}

// Result represents the outcome of processing a task.
//
// Contains both successful results and any errors encountered.
// Workers send results back through the result channel.
type Result struct {
	// TaskID links this result back to the originating Task.ID.
	TaskID uint64

	// Data is the processed result, or nil if processing failed.
	Data any

	// Err is the error encountered during processing, or nil if
	// processing succeeded.
	Err error
}

// ProcessFunc processes a single task and returns the result, or an
// error if processing fails. Workers call it for each task they
// receive, so implementations must be safe for concurrent use and
// should honor ctx cancellation.
type ProcessFunc func(ctx context.Context, task Task) (any, error)

// WorkerStats tracks worker pool statistics for monitoring.
//
// All fields use atomic operations for thread-safe updates.
// Useful for debugging and performance tuning.
type WorkerStats struct {
	// TasksSubmitted counts total tasks submitted to the pool.
	TasksSubmitted uint64

	// TasksCompleted counts tasks processed without error.
	TasksCompleted uint64

	// TasksFailed counts tasks that encountered an error.
	TasksFailed uint64

	// TotalWorkers is the number of worker goroutines in the pool.
	TotalWorkers int
}

// WorkerError represents an error from worker processing.
//
// Contains context about which task failed and why.
// Implements the error interface.
type WorkerError struct {
	// TaskID identifies which task failed.
	TaskID uint64

	// Message describes the error.
	Message string

	// Cause is the underlying error or panic value.
	Cause any
}

// Error returns the error message, including Cause when present.
//
// Implements the error interface.
func (e *WorkerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns Cause as an error for errors.Is/errors.As, or nil if
// Cause is unset or holds a non-error panic value (e.g. a string or
// other value passed to panic()).
func (e *WorkerError) Unwrap() error {
	if err, ok := e.Cause.(error); ok {
		return err
	}
	return nil
}
