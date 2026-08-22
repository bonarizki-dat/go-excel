package events

import (
	"context"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

func TestEventDispatcher_On_Dispatch(t *testing.T) {
	d := NewEventDispatcher()
	ctx := context.Background()

	var called bool
	d.On(excel.BeforeExport, func(_ context.Context, e *excel.Event) error {
		called = true
		if e.Type != excel.BeforeExport {
			t.Errorf("Expected event type BeforeExport, got %v", e.Type)
		}
		if e.Data != "test_data" {
			t.Errorf("Expected data 'test_data', got %v", e.Data)
		}
		return nil
	})

	err := d.Dispatch(ctx, &excel.Event{Type: excel.BeforeExport, Data: "test_data"})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	if !called {
		t.Error("Hook was not called")
	}
}

func TestEventDispatcher_MultipleHooks(t *testing.T) {
	d := NewEventDispatcher()
	ctx := context.Background()

	var order []int
	d.On(excel.AfterRowExport, func(_ context.Context, _ *excel.Event) error {
		order = append(order, 1)
		return nil
	})
	d.On(excel.AfterRowExport, func(_ context.Context, _ *excel.Event) error {
		order = append(order, 2)
		return nil
	})

	err := d.Dispatch(ctx, &excel.Event{Type: excel.AfterRowExport})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	if len(order) != 2 {
		t.Fatalf("Expected 2 hooks called, got %d", len(order))
	}
	if order[0] != 1 || order[1] != 2 {
		t.Errorf("Hooks called in wrong order: %v", order)
	}
}

func TestEventDispatcher_ErrorPropagation(t *testing.T) {
	d := NewEventDispatcher()
	ctx := context.Background()

	expectedErr := errors.New("hook error")
	d.On(excel.BeforeImport, func(_ context.Context, _ *excel.Event) error {
		return expectedErr
	})

	// Second hook should not be called
	var secondCalled bool
	d.On(excel.BeforeImport, func(_ context.Context, _ *excel.Event) error {
		secondCalled = true
		return nil
	})

	err := d.Dispatch(ctx, &excel.Event{Type: excel.BeforeImport})
	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	if secondCalled {
		t.Error("Second hook should not be called after error")
	}
}

// TestEventDispatcher_Dispatch_NoListeners verifies dispatching an event
// type with no registered hooks is a no-op, not an error.
func TestEventDispatcher_Dispatch_NoListeners(t *testing.T) {
	d := NewEventDispatcher()
	if err := d.Dispatch(context.Background(), &excel.Event{Type: excel.OnError}); err != nil {
		t.Errorf("Dispatch with no listeners = %v, want nil", err)
	}
}

// TestEventDispatcher_On_LazyInitializesListeners verifies On works on a
// zero-value EventDispatcher (listeners map still nil), not just one built
// via NewEventDispatcher.
func TestEventDispatcher_On_LazyInitializesListeners(t *testing.T) {
	var d EventDispatcher

	var called bool
	d.On(excel.OnError, func(_ context.Context, _ *excel.Event) error {
		called = true
		return nil
	})

	if err := d.Dispatch(context.Background(), &excel.Event{Type: excel.OnError}); err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}
	if !called {
		t.Error("hook registered via On on a zero-value dispatcher was not called")
	}
}

func TestEventDispatcher_HasListeners(t *testing.T) {
	d := NewEventDispatcher()

	if d.HasListeners(excel.OnError) {
		t.Error("Should not have listeners initially")
	}

	d.On(excel.OnError, func(_ context.Context, _ *excel.Event) error { return nil })

	if !d.HasListeners(excel.OnError) {
		t.Error("Should have listeners after registration")
	}
}
