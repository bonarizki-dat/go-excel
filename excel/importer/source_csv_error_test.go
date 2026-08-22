package importer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestImportCSVFromReader_PrepareImportError verifies a prepareImport
// failure (a BeforeImport hook error) is returned as-is before any CSV
// decoding is attempted.
func TestImportCSVFromReader_PrepareImportError(t *testing.T) {
	imp := NewImporter()
	hookErr := errors.New("before-import hook failed")
	imp.On(excel.BeforeImport, func(context.Context, *excel.Event) error {
		return hookErr
	})

	err := imp.importCSVFromReader(context.Background(), strings.NewReader("a,b\n1,2\n"))
	if !errors.Is(err, hookErr) {
		t.Errorf("importCSVFromReader() = %v, want error wrapping %v", err, hookErr)
	}
}
