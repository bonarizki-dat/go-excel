package stream

import (
	"context"
	"errors"
	"testing"
)

// failNextSource is a streamRowSource whose next() always fails with a
// non-EOF error, used to exercise iterateSheet's read-error branch
// without needing a genuinely corrupt XLSX file.
type failNextSource struct{}

var errSourceNextFailed = errors.New("row decode failed")

func (failNextSource) next() ([]string, error) { return nil, errSourceNextFailed }
func (failNextSource) close() error            { return nil }

// TestIterateSheet_ReadError verifies iterateSheet wraps a non-EOF
// error from the row source instead of only ever treating a failed
// next() as end-of-sheet.
func TestIterateSheet_ReadError(t *testing.T) {
	si := &StreamImporter{}

	err := si.iterateSheet(context.Background(), failNextSource{}, func([]any) error { return nil })
	if !errors.Is(err, errSourceNextFailed) {
		t.Errorf("iterateSheet() = %v, want error wrapping %v", err, errSourceNextFailed)
	}
}

// TestImportAllSheets_ContextCanceledBetweenSheets verifies the
// per-sheet ctx.Done() check in ImportAllSheets' loop itself (as
// opposed to iterateSheet's own per-row check) stops processing
// before the second sheet: ctx is canceled while handling the first
// sheet's last row, so iterateSheet's next source read hits io.EOF
// and returns cleanly (EOF short-circuits before iterateSheet's own
// ctx.Done() check), leaving cancellation to only be observed once
// ImportAllSheets' loop advances to the next sheet.
func TestImportAllSheets_ContextCanceledBetweenSheets(t *testing.T) {
	path := defaultMultiSheetFile(t)
	ctx, cancel := context.WithCancel(context.Background())

	var sheets []string
	var rowsInFirstSheet int
	err := newSheetOnlyImporter().ImportAllSheets(ctx, path,
		func(name string) error {
			sheets = append(sheets, name)
			return nil
		},
		func([]any) error {
			rowsInFirstSheet++
			if sheets[0] == "Users" && rowsInFirstSheet == 3 { // Users has 3 rows: heading + 2 data
				cancel()
			}
			return nil
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("ImportAllSheets() = %v, want error wrapping context.Canceled", err)
	}
	if len(sheets) != 1 || sheets[0] != "Users" {
		t.Errorf("sheets visited = %v, want [Users]", sheets)
	}
}
