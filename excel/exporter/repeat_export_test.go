package exporter

import (
	"context"
	"testing"
)

// TestExporter_RepeatedExport_ClosesPreviousFile verifies that calling
// Export() a second time on the same exporter succeeds and produces a
// fresh, independent *excelize.File rather than reusing or erroring out
// on the one from the first call. Export() closes the previous file
// before replacing e.file; since GetFile() only ever exposes the
// current file, skipping that close would leave the previous file's
// resources unreachable and never closed.
func TestExporter_RepeatedExport_ClosesPreviousFile(t *testing.T) {
	ctx := context.Background()

	exp := NewExporter([][]any{{"a", 1}, {"b", 2}})

	if err := exp.Export(ctx); err != nil {
		t.Fatalf("first Export failed: %v", err)
	}
	firstFile := exp.GetFile()
	if firstFile == nil {
		t.Fatal("expected first export to produce a file")
	}

	exp.data = [][]any{{"c", 3}}
	if err := exp.Export(ctx); err != nil {
		t.Fatalf("second Export failed: %v", err)
	}
	secondFile := exp.GetFile()
	if secondFile == nil {
		t.Fatal("expected second export to produce a file")
	}
	if secondFile == firstFile {
		t.Fatal("expected second export to produce a distinct *excelize.File")
	}

	rows, err := secondFile.GetRows(exp.GetConfig().SheetName)
	if err != nil {
		t.Fatalf("GetRows on second file: %v", err)
	}
	if len(rows) != 1 || rows[0][0] != "c" {
		t.Errorf("second export rows = %v, want a single row starting with %q", rows, "c")
	}

	// A third call must also succeed, proving the close-then-replace
	// path is safe to repeat rather than just working once.
	exp.data = [][]any{{"d", 4}}
	if err := exp.Export(ctx); err != nil {
		t.Fatalf("third Export failed: %v", err)
	}
}
