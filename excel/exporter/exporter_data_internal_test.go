package exporter

import (
	"context"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestExporter_PrepareData_2DArray tests prepareData with [][]interface{}.
func TestExporter_PrepareData_2DArray(t *testing.T) {
	ctx := context.Background()

	data := [][]any{
		{"A", "B"},
		{"C", "D"},
	}

	exporter := NewExporter(data)
	rows, err := exporter.prepareData(ctx)
	if err != nil {
		t.Fatalf("prepareData failed: %v", err)
	}

	// Should return data as-is
	if len(rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(rows))
	}
	if len(rows[0]) != 2 {
		t.Errorf("Expected 2 columns in first row, got %d", len(rows[0]))
	}
	if rows[0][0] != "A" {
		t.Errorf("rows[0][0] = %v, want A", rows[0][0])
	}
}

// TestExporter_PrepareData_1DArray tests prepareData with []interface{}.
func TestExporter_PrepareData_1DArray(t *testing.T) {
	ctx := context.Background()

	data := []any{"A", "B", "C"}

	exporter := NewExporter(data)
	rows, err := exporter.prepareData(ctx)
	if err != nil {
		t.Fatalf("prepareData failed: %v", err)
	}

	// Each element should become a row with single cell
	if len(rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(rows))
	}
	if len(rows[0]) != 1 {
		t.Errorf("Expected 1 column per row, got %d", len(rows[0]))
	}
	if rows[0][0] != "A" {
		t.Errorf("rows[0][0] = %v, want A", rows[0][0])
	}
}

// TestExporter_PrepareData_BareStruct tests prepareData with a plain struct
// that does not implement excel.Exportable. prepareData falls back to
// convertUsingReflection for such types, which converts the struct's
// fields into a single row rather than returning an error.
func TestExporter_PrepareData_BareStruct(t *testing.T) {
	ctx := context.Background()

	type PlainStruct struct {
		Field string
	}
	data := PlainStruct{Field: "test"}

	exporter := NewExporter(data)
	rows, err := exporter.prepareData(ctx)
	if err != nil {
		t.Fatalf("prepareData failed: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}
	if len(rows[0]) != 1 || rows[0][0] != "test" {
		t.Errorf("rows[0] = %v, want [test]", rows[0])
	}
}

// TestExporter_ReflectStructToRow_OmitEmpty verifies that a field
// tagged `excel:"...,omitempty"` renders as a blank cell (nil) when it
// holds its type's zero value, and renders normally otherwise.
func TestExporter_ReflectStructToRow_OmitEmpty(t *testing.T) {
	ctx := context.Background()

	type Item struct {
		Name  string `excel:"name"`
		Notes string `excel:"notes,omitempty"`
		Count int    `excel:"count,omitempty"`
	}

	data := []Item{
		{Name: "with notes", Notes: "hello", Count: 5},
		{Name: "zero values", Notes: "", Count: 0},
	}

	exporter := NewExporter(data)
	rows, err := exporter.prepareData(ctx)
	if err != nil {
		t.Fatalf("prepareData failed: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(rows))
	}

	// Row 0: no zero values, so omitempty has no effect.
	if rows[0][1] != "hello" || rows[0][2] != 5 {
		t.Errorf("row 0 = %v, want [with notes, hello, 5]", rows[0])
	}

	// Row 1: Notes and Count are zero values with omitempty, so both
	// render as nil rather than "" and 0.
	if rows[1][1] != nil {
		t.Errorf("row 1 Notes = %v, want nil (blank cell)", rows[1][1])
	}
	if rows[1][2] != nil {
		t.Errorf("row 1 Count = %v, want nil (blank cell)", rows[1][2])
	}
}

// TestExporter_ReflectStructToRow_EmbeddedValueStruct verifies an
// anonymous struct field's columns are flattened into the parent row.
func TestExporter_ReflectStructToRow_EmbeddedValueStruct(t *testing.T) {
	ctx := context.Background()

	type Base struct {
		ID int `excel:"id"`
	}
	type Item struct {
		Base
		Name string `excel:"name"`
	}

	exporter := NewExporter([]Item{{Base: Base{ID: 1}, Name: "Alice"}})
	rows, err := exporter.prepareData(ctx)
	if err != nil {
		t.Fatalf("prepareData failed: %v", err)
	}
	if len(rows) != 1 || len(rows[0]) != 2 {
		t.Fatalf("rows = %v, want 1 row with 2 cells", rows)
	}
	if rows[0][0] != 1 || rows[0][1] != "Alice" {
		t.Errorf("rows[0] = %v, want [1 Alice]", rows[0])
	}
}

// TestExporter_ReflectStructToRow_EmbeddedPointerStruct verifies an
// anonymous *struct field (non-nil) is also flattened into the parent
// row, exercising the pointer branch of isEmbeddedStructField distinct
// from the plain-value branch above.
func TestExporter_ReflectStructToRow_EmbeddedPointerStruct(t *testing.T) {
	ctx := context.Background()

	type Base struct {
		ID int `excel:"id"`
	}
	type Item struct {
		*Base
		Name string `excel:"name"`
	}

	exporter := NewExporter([]Item{{Base: &Base{ID: 2}, Name: "Bob"}})
	rows, err := exporter.prepareData(ctx)
	if err != nil {
		t.Fatalf("prepareData failed: %v", err)
	}
	if len(rows) != 1 || len(rows[0]) != 2 {
		t.Fatalf("rows = %v, want 1 row with 2 cells", rows)
	}
	if rows[0][0] != 2 || rows[0][1] != "Bob" {
		t.Errorf("rows[0] = %v, want [2 Bob]", rows[0])
	}
}

// Note: reflectStructToRow's own error branches (invoked recursively
// for embedded fields, and directly for a top-level struct) are
// effectively unreachable through the public API: isEmbeddedStructField
// only ever returns true for a Struct-kind field or a non-nil pointer
// whose Elem() is already confirmed to be a Struct, so the recursive
// call can never observe a non-struct Kind. A nil embedded pointer
// instead falls through to the ordinary (non-embedded) field handling.

// TestExporter_ConvertSliceToRows tests convertSliceToRows.
func TestExporter_ConvertSliceToRows(t *testing.T) {
	exporter := NewExporter(nil)

	slice := []any{1, 2, 3, 4, 5}
	rows, err := exporter.convertSliceToRows(slice)
	if err != nil {
		t.Fatalf("convertSliceToRows failed: %v", err)
	}

	if len(rows) != 5 {
		t.Errorf("Expected 5 rows, got %d", len(rows))
	}

	for i, row := range rows {
		if len(row) != 1 {
			t.Errorf("Row %d: expected 1 cell, got %d", i, len(row))
		}
		if row[0] != i+1 {
			t.Errorf("Row %d: value = %v, want %d", i, row[0], i+1)
		}
	}
}

// TestExporter_ConvertSliceToRows_WithPreallocation tests preallocation.
func TestExporter_ConvertSliceToRows_WithPreallocation(t *testing.T) {
	exporter := NewExporter(nil, excel.WithPreallocation(true))

	slice := []any{1, 2, 3}
	rows, err := exporter.convertSliceToRows(slice)
	if err != nil {
		t.Fatalf("convertSliceToRows failed: %v", err)
	}

	// Should work the same with preallocation
	if len(rows) != 3 {
		t.Errorf("Expected 3 rows, got %d", len(rows))
	}
}

// TestExporter_GetFile tests GetFile method.
func TestExporter_GetFile(t *testing.T) {
	ctx := context.Background()
	data := [][]any{{"test"}}

	exporter := NewExporter(data)
	exporter.Export(ctx)

	file := exporter.GetFile()
	if file == nil {
		t.Error("GetFile should return non-nil file after export")
	}
}

// TestExporter_GetConfig tests GetConfig method.
func TestExporter_GetConfig(t *testing.T) {
	exporter := NewExporter(nil, excel.WithSheetName("Test"))

	config := exporter.GetConfig()
	if config == nil {
		t.Fatal("GetConfig should not return nil")
	}
	if config.SheetName != "Test" {
		t.Errorf("Config SheetName = %v, want Test", config.SheetName)
	}

	// Verify it's a clone (modifying shouldn't affect exporter)
	config.SheetName = "Modified"
	if exporter.GetConfig().SheetName == "Modified" {
		t.Error("GetConfig should return a clone, not the original")
	}
}

// TestExporter_GetErrors tests GetErrors method.
func TestExporter_GetErrors(t *testing.T) {
	exporter := NewExporter(nil)

	errors := exporter.GetErrors()
	if errors == nil {
		t.Error("GetErrors should not return nil")
	}
	if len(errors) != 0 {
		t.Error("GetErrors should return empty slice initially")
	}
}

// TestExporter_Close tests Close method.
func TestExporter_Close(t *testing.T) {
	ctx := context.Background()
	data := [][]any{{"test"}}

	exporter := NewExporter(data)
	exporter.Export(ctx)

	err := exporter.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
