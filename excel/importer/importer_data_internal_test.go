package importer

import (
	"context"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestImporter_ProcessRow_Basic tests basic row processing.
func TestImporter_ProcessRow_Basic(t *testing.T) {
	ctx := context.Background()

	importer := NewImporter()
	row := []string{"A", "B", "C"}

	result, err := importer.processRow(ctx, row)
	if err != nil {
		t.Fatalf("processRow failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 cells, got %d", len(result))
	}

	if result[0] != "A" {
		t.Errorf("Cell 0 = %v, want A", result[0])
	}
}

// TestImporter_ProcessRow_WithTrimSpaces tests row processing with trimming.
func TestImporter_ProcessRow_WithTrimSpaces(t *testing.T) {
	ctx := context.Background()

	importer := NewImporter(excel.WithTrimSpaces(true))
	row := []string{"  A  ", " B ", "C"}

	result, err := importer.processRow(ctx, row)
	if err != nil {
		t.Fatalf("processRow failed: %v", err)
	}

	// Values should be trimmed
	if result[0] != "A" {
		t.Errorf("Cell 0 = %v, want A (trimmed)", result[0])
	}
	if result[1] != "B" {
		t.Errorf("Cell 1 = %v, want B (trimmed)", result[1])
	}
}

// TestImporter_ProcessRow_ColumnRange tests column range filtering.
func TestImporter_ProcessRow_ColumnRange(t *testing.T) {
	ctx := context.Background()

	// Test with column range (1-based inclusive columns 2-3)
	importer := NewImporter(excel.WithColumns(2, 3))
	row := []string{"A", "B", "C", "D", "E"}

	result, err := importer.processRow(ctx, row)
	if err != nil {
		t.Fatalf("processRow failed: %v", err)
	}

	// Should only include columns 1 and 2 (B and C)
	if len(result) != 2 {
		t.Errorf("Expected 2 cells, got %d", len(result))
	}
	if result[0] != "B" {
		t.Errorf("Cell 0 = %v, want B", result[0])
	}
	if result[1] != "C" {
		t.Errorf("Cell 1 = %v, want C", result[1])
	}
}

// TestImporter_ProcessRow_StartColBeyondLength tests StartCol beyond row length.
func TestImporter_ProcessRow_StartColBeyondLength(t *testing.T) {
	ctx := context.Background()

	importer := NewImporter(excel.WithColumns(11, 0))
	row := []string{"A", "B", "C"}

	result, err := importer.processRow(ctx, row)
	if err != nil {
		t.Fatalf("processRow failed: %v", err)
	}

	// Should return empty row
	if len(result) != 0 {
		t.Errorf("Expected empty row, got %d cells", len(result))
	}
}

// TestImporter_ProcessRow_InvalidStartCol tests that a below-minimum
// (1-based) first column is ignored rather than applied.
func TestImporter_ProcessRow_InvalidStartCol(t *testing.T) {
	ctx := context.Background()

	importer := NewImporter(excel.WithColumns(0, 0))
	row := []string{"A", "B", "C"}

	result, err := importer.processRow(ctx, row)
	if err != nil {
		t.Fatalf("processRow failed: %v", err)
	}

	// Should treat the invalid first column as "no restriction"
	if len(result) != 3 {
		t.Errorf("Expected 3 cells, got %d", len(result))
	}
}

// TestImporter_ProcessRow_EndColBeyondLength tests EndCol beyond row length.
func TestImporter_ProcessRow_EndColBeyondLength(t *testing.T) {
	ctx := context.Background()

	importer := NewImporter(excel.WithColumns(1, 10))
	row := []string{"A", "B", "C"}

	result, err := importer.processRow(ctx, row)
	if err != nil {
		t.Fatalf("processRow failed: %v", err)
	}

	// Should include all columns
	if len(result) != 3 {
		t.Errorf("Expected 3 cells, got %d", len(result))
	}
}

// TestImporter_IsEmptyRow_EmptySlice tests empty row detection with empty slice.
func TestImporter_IsEmptyRow_EmptySlice(t *testing.T) {
	importer := NewImporter()

	row := []any{}
	if !importer.isEmptyRow(row) {
		t.Error("Empty slice should be detected as empty row")
	}
}

// TestImporter_IsEmptyRow_AllEmptyStrings tests all empty strings.
func TestImporter_IsEmptyRow_AllEmptyStrings(t *testing.T) {
	importer := NewImporter()

	row := []any{"", "", ""}
	if !importer.isEmptyRow(row) {
		t.Error("Row with all empty strings should be detected as empty")
	}
}

// TestImporter_IsEmptyRow_AllNil tests all nil values.
func TestImporter_IsEmptyRow_AllNil(t *testing.T) {
	importer := NewImporter()

	row := []any{nil, nil, nil}
	if !importer.isEmptyRow(row) {
		t.Error("Row with all nil values should be detected as empty")
	}
}

// TestImporter_IsEmptyRow_MixedEmptyAndNil tests mixed empty and nil.
func TestImporter_IsEmptyRow_MixedEmptyAndNil(t *testing.T) {
	importer := NewImporter()

	row := []any{"", nil, ""}
	if !importer.isEmptyRow(row) {
		t.Error("Row with mixed empty strings and nil should be detected as empty")
	}
}

// TestImporter_IsEmptyRow_WithData tests row with actual data.
func TestImporter_IsEmptyRow_WithData(t *testing.T) {
	importer := NewImporter()

	row := []any{"", "data", ""}
	if importer.isEmptyRow(row) {
		t.Error("Row with data should not be detected as empty")
	}
}

// TestImporter_IsEmptyRow_WithNonString tests row with non-string values.
func TestImporter_IsEmptyRow_WithNonString(t *testing.T) {
	importer := NewImporter()

	row := []any{"", 0, ""}
	if importer.isEmptyRow(row) {
		t.Error("Row with non-string value should not be detected as empty")
	}
}

// TestImporter_Import_WithSkipEmptyRows tests skip empty rows feature
