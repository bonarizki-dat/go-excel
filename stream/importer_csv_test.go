package stream

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestImportStream_CSVFormat_ReadsRows verifies that
// WithImportFormat(FileFormatCSV) decodes plain CSV rows instead of
// treating the reader as an XLSX file.
func TestImportStream_CSVFormat_ReadsRows(t *testing.T) {
	data := "1,Alice,alice@test.com\n2,Bob,bob@test.com\n3,Charlie,charlie@test.com\n"
	importer, err := NewStreamImporter(strings.NewReader(data), WithImportFormat(config.FileFormatCSV))
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	// The processor runs concurrently across worker goroutines, so rows
	// must be collected keyed by rowNum under a mutex rather than
	// appended to a shared slice; see
	// TestImportStream_RetainedRowsSurviveAfterImport in
	// importer_import_test.go for the same pattern.
	var mu sync.Mutex
	got := make(map[int][]any)
	processor := func(_ context.Context, rowNum int, row []any) ([]any, error) {
		mu.Lock()
		got[rowNum] = row
		mu.Unlock()
		return row, nil
	}

	if err := importer.ImportStream(context.Background(), processor); err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 rows, got %d: %v", len(got), got)
	}
	if importer.GetRowCount() != 3 {
		t.Errorf("expected row count 3, got %d", importer.GetRowCount())
	}
}

// TestImportStream_CSVFormat_WithHeadings verifies the first row is
// consumed as headings and never reappears as a data row.
func TestImportStream_CSVFormat_WithHeadings(t *testing.T) {
	data := "ID,Name,Email\n1,Alice,alice@test.com\n2,Bob,bob@test.com\n"
	importer, err := NewStreamImporter(strings.NewReader(data),
		WithImportFormat(config.FileFormatCSV),
		WithImportHeadings(),
	)
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	headings := importer.GetHeadings()
	if len(headings) != 3 || headings[0] != "ID" || headings[1] != "Name" || headings[2] != "Email" {
		t.Fatalf("expected headings [ID Name Email], got %v", headings)
	}

	// The processor runs concurrently across worker goroutines, so rows
	// are collected keyed by rowNum under a mutex rather than tracked by
	// arrival order.
	var mu sync.Mutex
	seenRowNums := make(map[int]bool)
	processor := func(_ context.Context, rowNum int, row []any) ([]any, error) {
		mu.Lock()
		seenRowNums[rowNum] = true
		mu.Unlock()
		return row, nil
	}

	if err := importer.ImportStream(context.Background(), processor); err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	if len(seenRowNums) != 2 {
		t.Errorf("expected 2 data rows, got %d", len(seenRowNums))
	}
	// rowNum is the 1-based position among rows forwarded to the
	// processor, not the row's absolute position in the source file, so
	// with the heading row excluded the two data rows are numbered 1
	// and 2 (matches ImportStream's existing XLSX behavior).
	if !seenRowNums[1] || !seenRowNums[2] {
		t.Errorf("expected data rows numbered 1 and 2, got %v", seenRowNums)
	}
}

// TestImportStream_CSVFormat_WithImportRows verifies row-range filtering
// applies identically to CSV and XLSX inputs.
func TestImportStream_CSVFormat_WithImportRows(t *testing.T) {
	var sb strings.Builder
	for range 10 {
		sb.WriteString("user\n")
	}
	importer, err := NewStreamImporter(strings.NewReader(sb.String()),
		WithImportFormat(config.FileFormatCSV),
		WithImportRows(3, 7),
	)
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	var processedCount atomic.Uint64
	processor := func(_ context.Context, _ int, row []any) ([]any, error) {
		processedCount.Add(1)
		return row, nil
	}

	if err := importer.ImportStream(context.Background(), processor); err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	if processedCount.Load() != 5 {
		t.Errorf("expected 5 rows in range [3,7], got %d", processedCount.Load())
	}
}

// TestImportStream_CSVFormat_CustomDelimiter verifies
// WithImportCSVOptions overrides the default comma delimiter.
func TestImportStream_CSVFormat_CustomDelimiter(t *testing.T) {
	data := "1;Alice\n2;Bob\n"
	opts := config.DefaultCSVOptions()
	opts.Delimiter = ';'

	importer, err := NewStreamImporter(strings.NewReader(data),
		WithImportFormat(config.FileFormatCSV),
		WithImportCSVOptions(opts),
	)
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	var mu sync.Mutex
	got := make(map[int][]any)
	processor := func(_ context.Context, rowNum int, row []any) ([]any, error) {
		mu.Lock()
		got[rowNum] = row
		mu.Unlock()
		return row, nil
	}

	if err := importer.ImportStream(context.Background(), processor); err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	if len(got) != 2 || got[1][1] != "Alice" || got[2][1] != "Bob" {
		t.Fatalf("expected row 1 [1 Alice] and row 2 [2 Bob], got %v", got)
	}
}

// TestImportStream_CSVFormat_RoundTripsExportStreamOutput verifies that
// data written by ExportStream in CSV format reads back identically
// through StreamImporter, exercising both halves of Phase 5 together.
func TestImportStream_CSVFormat_RoundTripsExportStreamOutput(t *testing.T) {
	var buf bytes.Buffer
	exporter, err := NewStreamExporter(&buf,
		WithExportFormat(config.FileFormatCSV),
		WithExportHeadings([]string{"ID", "Name"}),
	)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	rows := make(chan []any, 3)
	go func() {
		defer close(rows)
		rows <- []any{"1", "Alice"}
		rows <- []any{"2", "Bob"}
		rows <- []any{"3", "Charlie"}
	}()

	if err := exporter.ExportStream(context.Background(), rows); err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	importer, err := NewStreamImporter(bytes.NewReader(buf.Bytes()),
		WithImportFormat(config.FileFormatCSV),
		WithImportHeadings(),
	)
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	if got := importer.GetHeadings(); len(got) != 2 || got[0] != "ID" || got[1] != "Name" {
		t.Fatalf("expected headings [ID Name], got %v", got)
	}

	var mu sync.Mutex
	got := make(map[int][]any)
	processor := func(_ context.Context, rowNum int, row []any) ([]any, error) {
		mu.Lock()
		got[rowNum] = row
		mu.Unlock()
		return row, nil
	}

	if err := importer.ImportStream(context.Background(), processor); err != nil {
		t.Fatalf("ImportStream failed: %v", err)
	}

	// rowNum is the 1-based position among forwarded rows, excluding
	// the already-consumed heading row.
	want := map[int][]any{1: {"1", "Alice"}, 2: {"2", "Bob"}, 3: {"3", "Charlie"}}
	if len(got) != len(want) {
		t.Fatalf("expected %d rows, got %d: %v", len(want), len(got), got)
	}
	for rowNum, wantRow := range want {
		gotRow, ok := got[rowNum]
		if !ok || gotRow[0] != wantRow[0] || gotRow[1] != wantRow[1] {
			t.Errorf("row %d: expected %v, got %v", rowNum, wantRow, gotRow)
		}
	}
}

func TestWithImportLogger(t *testing.T) {
	log := &degradationLogger{}
	importer, err := NewStreamImporter(strings.NewReader("a,b\n1,2\n"),
		WithImportFormat(config.FileFormatCSV),
		WithImportLogger(log),
	)
	if err != nil {
		t.Fatalf("NewStreamImporter: %v", err)
	}
	defer importer.Close()
	if importer.logger != log {
		t.Error("WithImportLogger did not set si.logger")
	}
}
