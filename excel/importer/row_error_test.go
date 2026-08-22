package importer

import (
	"context"
	"errors"
	"testing"
)

// TestProcessRow_ContextCanceled exercises processRow's own ctx.Done()
// check directly. Reaching it through Import() would require canceling
// in the narrow window between importRows' own per-row check and
// processRow's, which every other test's checks intercept first;
// calling the unexported method directly avoids that race.
func TestProcessRow_ContextCanceled(t *testing.T) {
	imp := NewImporter()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := imp.processRow(ctx, []string{"a"}); !errors.Is(err, context.Canceled) {
		t.Errorf("processRow() = %v, want error wrapping context.Canceled", err)
	}
}

// TestProcessRow_NegativeStartColClampedToZero exercises processRow's
// own defensive clamp for a negative StartCol directly (bypassing
// config.Validate, which already rejects negative StartCol on the
// public Import path, so this can only be reached by mutating the
// unexported field directly).
func TestProcessRow_NegativeStartColClampedToZero(t *testing.T) {
	imp := NewImporter()
	imp.config.StartCol = -2

	got, err := imp.processRow(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatalf("processRow: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("processRow with StartCol=-2 = %v, want both cells (clamped to 0)", got)
	}
}

// TestProcessRow_StartColBeyondRowLength verifies a StartCol at or
// beyond the row's length returns an empty row instead of panicking on
// an out-of-range slice.
func TestProcessRow_StartColBeyondRowLength(t *testing.T) {
	imp := NewImporter()
	imp.config.StartCol = 5

	got, err := imp.processRow(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatalf("processRow: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("processRow with StartCol beyond row length = %v, want empty row", got)
	}
}
