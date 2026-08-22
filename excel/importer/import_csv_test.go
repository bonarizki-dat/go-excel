package importer

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

func writeCSVFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "data.csv")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write CSV fixture: %v", err)
	}
	return path
}

func TestImportFromFile_CSVExtension_ReadsCSV(t *testing.T) {
	path := writeCSVFile(t, "id,name\n1,Alice\n2,Bob\n")

	imp := NewImporter(excel.WithHeadings(true))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), path); err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}

	rows := imp.GetRows()
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0][1] != "Alice" || rows[1][1] != "Bob" {
		t.Errorf("unexpected rows: %v", rows)
	}
	if got := imp.GetHeadings(); len(got) != 2 || got[0] != "id" || got[1] != "name" {
		t.Errorf("headings = %v, want [id name]", got)
	}
}

func TestImportFromFile_CSVWithoutHeadings_ReadsAllRowsAsData(t *testing.T) {
	path := writeCSVFile(t, "1,Alice\n2,Bob\n")

	imp := NewImporter(excel.WithHeadings(false))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), path); err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}

	rows := imp.GetRows()
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if len(imp.GetHeadings()) != 0 {
		t.Errorf("expected no headings, got %v", imp.GetHeadings())
	}
}

func TestImportFromFile_CSVRaggedRows_DoesNotFail(t *testing.T) {
	// The last row has fewer fields than the header; encoding/csv
	// rejects this by default, so importCSVFromReader must configure
	// FieldsPerRecord = -1 to tolerate it.
	path := writeCSVFile(t, "id,name,note\n1,Alice,hi\n2,Bob\n")

	imp := NewImporter(excel.WithHeadings(true))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), path); err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}

	rows := imp.GetRows()
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if len(rows[1]) != 2 {
		t.Errorf("short row = %v, want 2 fields", rows[1])
	}
}

func TestImportFromFile_CSVRowRange_FiltersRows(t *testing.T) {
	path := writeCSVFile(t, "id\n1\n2\n3\n4\n5\n")

	imp := NewImporter(excel.WithHeadings(true), excel.WithRows(2, 3))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), path); err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}

	rows := imp.GetRows()
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (rows 2 and 3)", len(rows))
	}
	if rows[0][0] != "2" || rows[1][0] != "3" {
		t.Errorf("unexpected rows: %v", rows)
	}
}

func TestImportFromFile_CSVColumnRange_FiltersColumns(t *testing.T) {
	path := writeCSVFile(t, "a,b,c\n1,2,3\n")

	imp := NewImporter(excel.WithHeadings(true), excel.WithColumns(2, 3))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), path); err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}

	rows := imp.GetRows()
	if len(rows) != 1 || len(rows[0]) != 2 {
		t.Fatalf("unexpected rows: %v", rows)
	}
	if rows[0][0] != "2" || rows[0][1] != "3" {
		t.Errorf("unexpected columns: %v", rows[0])
	}
}

func TestImportFromFile_CSVSkipEmptyRows_RemovesBlankLines(t *testing.T) {
	path := writeCSVFile(t, "id\n1\n\n2\n")

	imp := NewImporter(excel.WithHeadings(true), excel.WithSkipEmptyRows(true))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), path); err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}

	rows := imp.GetRows()
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (blank line skipped): %v", len(rows), rows)
	}
}

func TestImportFromFile_MalformedCSV_ReturnsError(t *testing.T) {
	// An unterminated quoted field is a hard decode error even with
	// FieldsPerRecord = -1.
	path := writeCSVFile(t, "id,name\n1,\"unterminated\n")

	imp := NewImporter(excel.WithHeadings(true))
	defer func() { _ = imp.Close() }()

	err := imp.ImportFromFile(context.Background(), path)
	if err == nil {
		t.Fatal("expected error for malformed CSV, got nil")
	}
	var importErr *excel.ImportError
	if !errors.As(err, &importErr) {
		t.Errorf("expected error to wrap *excel.ImportError, got %T", err)
	}
}

func TestImportFromReader_WithFormatCSV_ReadsCSV(t *testing.T) {
	imp := NewImporter(excel.WithFormat(excel.FileFormatCSV), excel.WithHeadings(true))
	defer func() { _ = imp.Close() }()

	r := strings.NewReader("id,name\n1,Alice\n")
	if err := imp.ImportFromReader(context.Background(), r); err != nil {
		t.Fatalf("ImportFromReader() error = %v", err)
	}

	rows := imp.GetRows()
	if len(rows) != 1 || rows[0][1] != "Alice" {
		t.Errorf("unexpected rows: %v", rows)
	}
}

func TestImportFromFile_CSVCustomDelimiter_ParsesCorrectly(t *testing.T) {
	path := writeCSVFile(t, "id;name\n1;Alice\n")

	imp := NewImporter(excel.WithHeadings(true), excel.WithCSVDelimiter(';'))
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), path); err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}

	rows := imp.GetRows()
	if len(rows) != 1 || rows[0][1] != "Alice" {
		t.Errorf("unexpected rows: %v", rows)
	}
}

type csvPerson struct {
	ID   int    `excel:"id"`
	Name string `excel:"name"`
}

func TestImportToStructs_CSVFile_MapsRowsToStructs(t *testing.T) {
	path := writeCSVFile(t, "id,name\n1,Alice\n2,Bob\n")

	people, err := ImportToStructs[csvPerson](context.Background(), path)
	if err != nil {
		t.Fatalf("ImportToStructs() error = %v", err)
	}

	if len(people) != 2 {
		t.Fatalf("got %d people, want 2", len(people))
	}
	if people[0].Name != "Alice" || people[1].Name != "Bob" {
		t.Errorf("unexpected people: %+v", people)
	}
}
