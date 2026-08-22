package exporter

import (
	"context"
	"testing"
	"time"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestExporter_Timeout_SetsContextDeadline verifies that WithTimeout wraps
// the context passed through Export with a deadline, since it was
// previously accepted into Config but never actually applied.
func TestExporter_Timeout_SetsContextDeadline(t *testing.T) {
	exp := NewExporter([][]any{{"a"}}, excel.WithTimeout(50*time.Millisecond))

	var sawDeadline bool
	exp.On(excel.BeforeExport, func(ctx context.Context, _ *excel.Event) error {
		_, sawDeadline = ctx.Deadline()
		return nil
	})

	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if !sawDeadline {
		t.Error("expected context to have a deadline when Timeout is set")
	}
}

// TestExporter_NoTimeout_NoContextDeadline verifies that Export does not
// impose a deadline when Timeout is left at its zero value.
func TestExporter_NoTimeout_NoContextDeadline(t *testing.T) {
	exp := NewExporter([][]any{{"a"}})

	var hasDeadline bool
	exp.On(excel.BeforeExport, func(ctx context.Context, _ *excel.Event) error {
		_, hasDeadline = ctx.Deadline()
		return nil
	})

	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if hasDeadline {
		t.Error("expected no deadline on context when Timeout is not set")
	}
}

// TestExporter_Timeout_Expires verifies that an already-expired timeout
// causes Export to fail with a context error rather than succeeding.
func TestExporter_Timeout_Expires(t *testing.T) {
	exp := NewExporter([][]any{{"a"}, {"b"}}, excel.WithTimeout(1*time.Nanosecond))

	// Give the 1ns timeout time to elapse before Export even starts.
	time.Sleep(time.Millisecond)

	if err := exp.Export(context.Background()); err == nil {
		t.Fatal("expected Export to fail once the configured timeout has elapsed")
	}
}
