package importer

import (
	"context"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestPrepareImport_BeforeImportHookError verifies a BeforeImport hook
// returning an error aborts before config validation runs.
func TestPrepareImport_BeforeImportHookError(t *testing.T) {
	imp := NewImporter()
	hookErr := errors.New("before-import hook failed")
	imp.On(excel.BeforeImport, func(context.Context, *excel.Event) error {
		return hookErr
	})

	_, cancel, err := imp.prepareImport(context.Background())
	defer cancel()
	if !errors.Is(err, hookErr) {
		t.Errorf("prepareImport() = %v, want error wrapping %v", err, hookErr)
	}
}

// TestImportStringRows_CollectedErrorsReturnedWhenStopOnErrorFalse
// verifies importStringRows surfaces the first collected per-row error
// when StopOnError is false, instead of silently succeeding. The only
// way processRow itself fails is via ctx.Done(), so the context is
// canceled from a BeforeRowImport hook on the last row: importRows'
// own per-iteration ctx.Done() check (which runs before importRow) has
// already passed for that row, so the loop still completes normally
// afterward instead of aborting with a hard error first.
func TestImportStringRows_CollectedErrorsReturnedWhenStopOnErrorFalse(t *testing.T) {
	imp := NewImporter(excel.WithHeadings(false), excel.WithStopOnError(false))

	ctx, cancel := context.WithCancel(context.Background())
	imp.On(excel.BeforeRowImport, func(_ context.Context, e *excel.Event) error {
		row, _ := e.Data.([]string)
		if len(row) > 0 && row[0] == "last" {
			cancel()
		}
		return nil
	})

	err := imp.importStringRows(ctx, [][]string{{"first"}, {"last"}})
	if err == nil {
		t.Fatal("importStringRows with a collected row error = nil error, want error")
	}
	if len(imp.errors) != 1 {
		t.Errorf("errors = %d, want 1", len(imp.errors))
	}
}

// TestApplyRowRange_NegativeStartRowClampedToZero exercises
// applyRowRange's own defensive clamp for a negative StartRow directly
// (bypassing config.Validate, which already rejects negative StartRow
// on the public Import path, so this can only be reached by mutating
// the unexported field directly, as an internal test can).
func TestApplyRowRange_NegativeStartRowClampedToZero(t *testing.T) {
	imp := NewImporter()
	imp.config.StartRow = -3

	rows := [][]string{{"a"}, {"b"}, {"c"}}
	got := imp.applyRowRange(rows)
	if len(got) != 3 {
		t.Errorf("applyRowRange with StartRow=-3 = %d rows, want 3 (clamped to 0)", len(got))
	}
}

// TestApplyRowRange_StartRowBeyondEndClamped exercises applyRowRange's
// clamp when StartRow exceeds the (already end-clamped) range, which
// config.Validate's own StartRow<=EndRow check prevents on the public
// path, so this is reached via direct field mutation too.
func TestApplyRowRange_StartRowBeyondEndClamped(t *testing.T) {
	imp := NewImporter()
	imp.config.StartRow = 10
	imp.config.EndRow = 2

	rows := [][]string{{"a"}, {"b"}, {"c"}}
	got := imp.applyRowRange(rows)
	if len(got) != 0 {
		t.Errorf("applyRowRange with StartRow beyond EndRow = %d rows, want 0", len(got))
	}
}

// TestImportRows_ContextCanceledMidway verifies the per-row ctx.Done()
// check in importRows' loop stops processing once the context is
// canceled between rows.
func TestImportRows_ContextCanceledMidway(t *testing.T) {
	imp := NewImporter(excel.WithHeadings(false))

	ctx, cancel := context.WithCancel(context.Background())
	imp.On(excel.AfterRowImport, func(context.Context, *excel.Event) error {
		cancel() // cancel after the first row completes
		return nil
	})

	err := imp.importRows(ctx, [][]string{{"a"}, {"b"}})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("importRows() = %v, want error wrapping context.Canceled", err)
	}
}

// TestImportRow_BeforeRowImportHookError verifies a BeforeRowImport
// hook error aborts importRow before processRow runs.
func TestImportRow_BeforeRowImportHookError(t *testing.T) {
	imp := NewImporter()
	hookErr := errors.New("before-row hook failed")
	imp.On(excel.BeforeRowImport, func(context.Context, *excel.Event) error {
		return hookErr
	})

	_, err := imp.importRow(context.Background(), 0, []string{"a"})
	if !errors.Is(err, hookErr) {
		t.Errorf("importRow() = %v, want error wrapping %v", err, hookErr)
	}
}

// TestImportRow_AfterRowImportHookError verifies an AfterRowImport hook
// error surfaces after the row has already been appended.
func TestImportRow_AfterRowImportHookError(t *testing.T) {
	imp := NewImporter()
	hookErr := errors.New("after-row hook failed")
	imp.On(excel.AfterRowImport, func(context.Context, *excel.Event) error {
		return hookErr
	})

	_, err := imp.importRow(context.Background(), 0, []string{"a"})
	if !errors.Is(err, hookErr) {
		t.Errorf("importRow() = %v, want error wrapping %v", err, hookErr)
	}
	if len(imp.rows) != 1 {
		t.Errorf("rows = %d, want 1 (row appended before the AfterRowImport hook ran)", len(imp.rows))
	}
}

// TestImportRow_ProcessRowError_StopOnErrorTrue verifies a processRow
// failure aborts immediately when StopOnError is true (the default).
// A canceled context makes processRow itself fail deterministically.
func TestImportRow_ProcessRowError_StopOnErrorTrue(t *testing.T) {
	imp := NewImporter()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := imp.importRow(ctx, 0, []string{"a"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("importRow() = %v, want error wrapping context.Canceled", err)
	}
}

// TestImportRow_ProcessRowError_Collected verifies a processRow failure
// is collected into i.errors and the row skipped when StopOnError is
// false and no ErrorCallback is set.
func TestImportRow_ProcessRowError_Collected(t *testing.T) {
	imp := NewImporter(excel.WithStopOnError(false))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	skip, err := imp.importRow(ctx, 0, []string{"a"})
	if err != nil {
		t.Errorf("importRow() = %v, want nil error (collected instead)", err)
	}
	if !skip {
		t.Error("importRow() skip = false, want true")
	}
	if len(imp.errors) != 1 {
		t.Errorf("errors = %d, want 1", len(imp.errors))
	}
}

// TestImportRow_ProcessRowError_ErrorCallbackStops verifies an
// ErrorCallback returning false aborts importRow even with StopOnError
// false.
func TestImportRow_ProcessRowError_ErrorCallbackStops(t *testing.T) {
	var callbackCalls int
	imp := NewImporter(
		excel.WithStopOnError(false),
		excel.WithErrorHandler(func(int, error) bool {
			callbackCalls++
			return false
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := imp.importRow(ctx, 0, []string{"a"})
	if err == nil {
		t.Error("importRow with a stopping ErrorCallback = nil error, want error")
	}
	if callbackCalls == 0 {
		t.Error("ErrorCallback was never called")
	}
}
