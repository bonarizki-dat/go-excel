package importer

import (
	"context"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestGetSheetNames_AfterCSVImport_ReportsSingleLogicalSheet verifies
// that GetSheetNames reports a single logical sheet named after
// config.SheetName for a CSV import, instead of the "no file opened"
// error a nil excelize file would otherwise produce.
func TestGetSheetNames_AfterCSVImport_ReportsSingleLogicalSheet(t *testing.T) {
	path := writeCSVFile(t, "id,name\n1,Alice\n2,Bob\n")

	imp := NewImporter()
	defer func() { _ = imp.Close() }()

	if err := imp.ImportFromFile(context.Background(), path); err != nil {
		t.Fatalf("ImportFromFile() error = %v", err)
	}

	names, err := imp.GetSheetNames()
	if err != nil {
		t.Fatalf("GetSheetNames() error = %v", err)
	}
	if len(names) != 1 || names[0] != imp.config.SheetName {
		t.Errorf("expected a single logical sheet %q, got %v", imp.config.SheetName, names)
	}

	count, err := imp.GetSheetCount()
	if err != nil {
		t.Fatalf("GetSheetCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("expected sheet count 1, got %d", count)
	}

	exists, err := imp.SheetExists(imp.config.SheetName)
	if err != nil {
		t.Fatalf("SheetExists() error = %v", err)
	}
	if !exists {
		t.Errorf("expected SheetExists(%q) to be true", imp.config.SheetName)
	}

	exists, err = imp.SheetExists("NotTheLogicalSheet")
	if err != nil {
		t.Fatalf("SheetExists() error = %v", err)
	}
	if exists {
		t.Error("expected SheetExists for an unrelated name to be false")
	}
}

// TestImportAllSheets_CSVFile_WarnsOnceAndReturnsSingleEntry verifies
// that ImportAllSheets on a .csv path imports the file once, warns
// exactly once about the missing multi-sheet concept, and returns a map
// with a single entry keyed by config.SheetName.
func TestImportAllSheets_CSVFile_WarnsOnceAndReturnsSingleEntry(t *testing.T) {
	path := writeCSVFile(t, "id,name\n1,Alice\n2,Bob\n")

	logger := &mockLogger{}
	imp := NewImporter(excel.WithLogger(logger))
	defer func() { _ = imp.Close() }()

	result, err := imp.ImportAllSheets(context.Background(), path)
	if err != nil {
		t.Fatalf("ImportAllSheets() error = %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected a single logical sheet entry, got %d: %v", len(result), result)
	}
	rows, ok := result[imp.config.SheetName]
	if !ok {
		t.Fatalf("expected entry keyed by %q, got keys %v", imp.config.SheetName, result)
	}
	// WithHeadings defaults to true, so the header row is consumed
	// separately and only the 2 data rows remain.
	if len(rows) != 2 {
		t.Errorf("expected 2 data rows, got %d: %v", len(rows), rows)
	}

	warnCount := 0
	for _, l := range logger.logs {
		if l == "WARN: CSV has no concept of multiple sheets; importing as a single logical sheet" {
			warnCount++
		}
	}
	if warnCount != 1 {
		t.Errorf("expected exactly 1 degradation warning, got %d: %v", warnCount, logger.logs)
	}
}
