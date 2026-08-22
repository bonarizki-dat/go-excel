package stream

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestImportFromSheet_Users(t *testing.T) {
	path := defaultMultiSheetFile(t)
	handler, got := collectRows()

	if err := newSheetOnlyImporter().ImportFromSheet(context.Background(), path, "Users", handler); err != nil {
		t.Fatalf("ImportFromSheet: %v", err)
	}
	if len(*got) != 3 {
		t.Fatalf("rows = %d, want 3 (heading + 2 data)", len(*got))
	}
	if (*got)[0][0] != "ID" {
		t.Errorf("first cell = %v, want ID (heading is not skipped by default)", (*got)[0][0])
	}
}

func TestImportFromSheet_EmptySheet(t *testing.T) {
	path := writeSheetsXLSX(t, "empty.xlsx", []sheetSpec{
		{name: "Empty", rows: nil},
	})
	handler, got := collectRows()

	if err := newSheetOnlyImporter().ImportFromSheet(context.Background(), path, "Empty", handler); err != nil {
		t.Fatalf("ImportFromSheet empty sheet: %v", err)
	}
	if len(*got) != 0 {
		t.Errorf("rows = %d, want 0", len(*got))
	}
}

func TestImportFromSheet_SheetNotFound(t *testing.T) {
	path := defaultMultiSheetFile(t)
	err := newSheetOnlyImporter().ImportFromSheet(context.Background(), path, "Missing", func([]any) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v, want sheet not found", err)
	}
}

func TestImportFromSheet_Validation(t *testing.T) {
	path := defaultMultiSheetFile(t)
	si := newSheetOnlyImporter()
	handler := func([]any) error { return nil }

	tests := []struct {
		name, file, sheet string
		handler           func([]any) error
		want              string
	}{
		{"empty filename", "", "Users", handler, "filename cannot be empty"},
		{"empty sheet", path, "", handler, "sheet name cannot be empty"},
		{"nil handler", path, "Users", nil, "row handler cannot be nil"},
		{"missing file", filepath.Join(t.TempDir(), "no-such.xlsx"), "Users", handler, "failed to open file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := si.ImportFromSheet(context.Background(), tt.file, tt.sheet, tt.handler)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestImportFromSheet_HandlerError(t *testing.T) {
	path := defaultMultiSheetFile(t)
	var n int
	boom := errors.New("stop")
	err := newSheetOnlyImporter().ImportFromSheet(context.Background(), path, "Users", func([]any) error {
		n++
		if n == 2 {
			return boom
		}
		return nil
	})
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("error = %v, want wrapped stop", err)
	}
	if n != 2 {
		t.Errorf("handler calls = %d, want 2", n)
	}
}

func TestImportFromSheet_ContextCanceledBeforeRows(t *testing.T) {
	path := defaultMultiSheetFile(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := newSheetOnlyImporter().ImportFromSheet(ctx, path, "Users", func([]any) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestImportFromSheet_ContextCanceledMidSheet(t *testing.T) {
	path := defaultMultiSheetFile(t)
	ctx, cancel := context.WithCancel(context.Background())
	err := newSheetOnlyImporter().ImportFromSheet(ctx, path, "Users", func([]any) error {
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestImportFromSheet_WithImportHeadings(t *testing.T) {
	path := defaultMultiSheetFile(t)
	handler, got := collectRows()
	si := newSheetOnlyImporter(WithImportHeadings())

	if err := si.ImportFromSheet(context.Background(), path, "Users", handler); err != nil {
		t.Fatalf("ImportFromSheet: %v", err)
	}
	if len(*got) != 2 {
		t.Fatalf("rows = %d, want 2 (heading skipped)", len(*got))
	}
	if (*got)[0][1] != "John Doe" {
		t.Errorf("first data name = %v, want John Doe", (*got)[0][1])
	}
	if len(si.GetHeadings()) != 0 {
		t.Errorf("GetHeadings() = %v, want empty (heading is not recorded on si)", si.GetHeadings())
	}
	if si.GetRowCount() != 2 {
		t.Errorf("GetRowCount() = %d, want 2", si.GetRowCount())
	}
}

func TestImportFromSheet_WithImportRows(t *testing.T) {
	path := defaultMultiSheetFile(t)
	handler, got := collectRows()
	si := newSheetOnlyImporter(WithImportRows(2, 2))

	if err := si.ImportFromSheet(context.Background(), path, "Users", handler); err != nil {
		t.Fatalf("ImportFromSheet: %v", err)
	}
	if len(*got) != 1 {
		t.Fatalf("rows = %d, want 1", len(*got))
	}
	if (*got)[0][1] != "John Doe" {
		t.Errorf("row = %v, want John Doe", (*got)[0])
	}
}

func TestImportFromSheet_WithImportSkipEmptyRows(t *testing.T) {
	path := writeSheetsXLSX(t, "gaps.xlsx", []sheetSpec{
		{name: "Gaps", rows: [][]any{
			{"keep"},
			{""},
			{"also"},
		}},
	})
	handler, got := collectRows()
	si := newSheetOnlyImporter(WithImportSkipEmptyRows(true))

	if err := si.ImportFromSheet(context.Background(), path, "Gaps", handler); err != nil {
		t.Fatalf("ImportFromSheet: %v", err)
	}
	if len(*got) != 2 {
		t.Fatalf("rows = %d, want 2", len(*got))
	}
}

func TestImportFromSheet_ProgressAndCount(t *testing.T) {
	path := defaultMultiSheetFile(t)
	var reports int
	si := newSheetOnlyImporter(WithImportProgress(func(uint64, string) {
		reports++
	}, 1))

	if err := si.ImportFromSheet(context.Background(), path, "Users", func([]any) error { return nil }); err != nil {
		t.Fatalf("ImportFromSheet: %v", err)
	}
	if si.GetRowCount() != 3 {
		t.Errorf("GetRowCount() = %d, want 3", si.GetRowCount())
	}
	if reports != 3 {
		t.Errorf("progress reports = %d, want 3", reports)
	}
}

func TestImportFromSheet_LeavesConstructorStateIntact(t *testing.T) {
	buf, err := createImporterTestFile([][]any{
		{"h1", "h2"},
		{1, "a"},
		{2, "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	si, err := NewStreamImporter(buf, WithImportSheetName("Sheet1"))
	if err != nil {
		t.Fatal(err)
	}

	origFile := si.file
	origSheet := si.sheetName
	path := defaultMultiSheetFile(t)
	var n int
	if err := si.ImportFromSheet(context.Background(), path, "Orders", func([]any) error {
		n++
		return nil
	}); err != nil {
		t.Fatalf("ImportFromSheet: %v", err)
	}
	if n != 4 {
		t.Errorf("Orders rows = %d, want 4", n)
	}
	if si.file != origFile {
		t.Error("ImportFromSheet replaced si.file")
	}
	if si.sheetName != origSheet {
		t.Errorf("sheetName = %q, want %q", si.sheetName, origSheet)
	}

	var streamRows atomic.Int64
	if err := si.ImportStream(context.Background(), func(_ context.Context, _ int, row []any) ([]any, error) {
		streamRows.Add(1)
		return row, nil
	}); err != nil {
		t.Fatalf("ImportStream after ImportFromSheet: %v", err)
	}
	if streamRows.Load() != 3 {
		t.Errorf("ImportStream rows = %d, want 3", streamRows.Load())
	}
	if err := si.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
