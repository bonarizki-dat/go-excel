package concurrency

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestWorkerError_Error_NoCause verifies Error() falls back to just
// Message when Cause is unset (nil), instead of always formatting a
// "%s: %v" pair.
func TestWorkerError_Error_NoCause(t *testing.T) {
	err := &WorkerError{TaskID: 1, Message: "boom"}
	if got, want := err.Error(), "boom"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// TestWorkerPool_Submit_ContextCanceled verifies Submit returns the
// context's error once taskChan is full and ctx is canceled, instead
// of blocking forever.
func TestWorkerPool_Submit_ContextCanceled(t *testing.T) {
	pool := NewWorkerPool(1, func(context.Context, Task) (any, error) {
		return "unused", nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	pool.ctx = ctx // Submit reads wp.ctx directly; Start() is never called so no worker drains taskChan.

	for i := 0; i < cap(pool.taskChan); i++ {
		pool.taskChan <- Task{ID: uint64(i)}
	}

	cancel()

	done := make(chan error, 1)
	go func() {
		done <- pool.Submit(Task{ID: 999})
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Submit() = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Submit did not return after ctx was canceled with a full taskChan")
	}
}
