package exporter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// MockExportable is a mock implementation of excel.Exportable interface.
type MockExportable struct {
	data any
}

// Data returns the underlying data.
func (m *MockExportable) Data() any {
	return m.data
}

// TestExporter_PrepareData_Exportable tests prepareData with Exportable interface.
func TestExporter_PrepareData_Exportable(t *testing.T) {
	ctx := context.Background()

	// Create mock exportable with valid data
	mockData := [][]any{
		{"ID", "Name"},
		{1, "Alice"},
		{2, "Bob"},
	}

	exportable := &MockExportable{data: mockData}

	// Create exporter with Exportable
	exp := exporter.NewExporter(exportable)

	// Perform export - should work with Exportable interface
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export with Exportable should succeed: %v", err)
	}

	// Verify data was written
	file := exp.GetFile()
	if file == nil {
		t.Fatal("File should be created")
	}

	// Check first cell
	cell, err := file.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("Failed to get cell: %v", err)
	}
	if cell != "ID" {
		t.Errorf("Cell A1 = %v, want ID", cell)
	}
}

// TestExporter_PrepareData_Exportable_Nested tests nested Exportable.
func TestExporter_PrepareData_Exportable_Nested(t *testing.T) {
	ctx := context.Background()

	// Create nested exportable (Exportable wrapping another Exportable)
	innerData := [][]any{
		{"Product", "Price"},
		{"Widget", 9.99},
	}
	innerExportable := &MockExportable{data: innerData}
	outerExportable := &MockExportable{data: innerExportable}

	// Create exporter with nested Exportable
	exp := exporter.NewExporter(outerExportable)

	// Export should recursively resolve Exportable
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export with nested Exportable should succeed: %v", err)
	}

	// Verify data was written
	cell, err := exp.GetFile().GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("Failed to get cell: %v", err)
	}
	if cell != "Product" {
		t.Errorf("Cell A1 = %v, want Product", cell)
	}
}

// TestExporter_PrepareData_Exportable_SelfCycle verifies that Export
// returns an error instead of recursing until the stack overflows when
// an Exportable's Data() returns itself (directly, or via a longer
// cycle), since prepareData previously had no bound on how many times it
// would unwrap nested Exportable values.
func TestExporter_PrepareData_Exportable_SelfCycle(t *testing.T) {
	ctx := context.Background()

	self := &MockExportable{}
	self.data = self // Data() returns the same value forever.

	exp := exporter.NewExporter(self)

	err := exp.Export(ctx)
	if err == nil {
		t.Fatal("expected Export to fail for a self-referential Exportable, got nil")
	}
}

// TestExporter_PrepareData_SingleSlice tests []interface{} conversion.
func TestExporter_PrepareData_SingleSlice(t *testing.T) {
	ctx := context.Background()

	// Create single-level slice
	data := []any{"Alice", "Bob", "Charlie"}

	// Create exporter
	exp := exporter.NewExporter(data)

	// Export should convert each element to single-cell row
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export with []interface{} should succeed: %v", err)
	}

	// Verify data was written as single-cell rows
	file := exp.GetFile()

	// Check first row
	cell1, _ := file.GetCellValue("Sheet1", "A1")
	if cell1 != "Alice" {
		t.Errorf("Cell A1 = %v, want Alice", cell1)
	}

	// Check second row
	cell2, _ := file.GetCellValue("Sheet1", "A2")
	if cell2 != "Bob" {
		t.Errorf("Cell A2 = %v, want Bob", cell2)
	}

	// Check third row
	cell3, _ := file.GetCellValue("Sheet1", "A3")
	if cell3 != "Charlie" {
		t.Errorf("Cell A3 = %v, want Charlie", cell3)
	}
}

// TestExporter_PrepareData_SingleSlice_WithPreallocate tests preallocate option.
func TestExporter_PrepareData_SingleSlice_WithPreallocate(t *testing.T) {
	ctx := context.Background()

	// Create single-level slice
	data := []any{1, 2, 3, 4, 5}

	// Create exporter with preallocation enabled
	exp := exporter.NewExporter(data, excel.WithPreallocation(true))

	// Export should use preallocated slice
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export with preallocate should succeed: %v", err)
	}

	// Verify all data was written
	file := exp.GetFile()
	for i := 1; i <= 5; i++ {
		cell, err := file.GetCellValue("Sheet1", "A"+string(rune('0'+i)))
		if err != nil {
			// Try different cell reference format
			continue
		}
		// Just verify we can read cells
		if cell == "" {
			t.Errorf("Cell should not be empty at row %d", i)
		}
	}
}

// TestExporter_PrepareData_BareStruct tests exporting a plain struct that
// does not implement excel.Exportable. The exporter falls back to
// reflection and converts the struct's fields into a single row, so this
// succeeds rather than erroring.
func TestExporter_PrepareData_BareStruct(t *testing.T) {
	ctx := context.Background()

	type CustomStruct struct {
		Field string
	}
	data := CustomStruct{Field: "test"}

	exp := exporter.NewExporter(data)
	defer exp.Close()

	if err := exp.Export(ctx); err != nil {
		t.Fatalf("Export with bare struct should succeed, got: %v", err)
	}

	cell, err := exp.GetFile().GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("GetCellValue failed: %v", err)
	}
	if cell != "test" {
		t.Errorf("A1 = %q, want %q", cell, "test")
	}
}

// TestExporter_PrepareData_EmptySlice tests that a non-nil but
// zero-length [][]any is rejected the same way as nil data: producing a
// workbook with no rows is never useful, so both are ErrEmptyData.
func TestExporter_PrepareData_EmptySlice(t *testing.T) {
	ctx := context.Background()

	// Create empty slice
	data := [][]any{}

	// Create exporter
	exp := exporter.NewExporter(data)

	err := exp.Export(ctx)
	if !errors.Is(err, excel.ErrEmptyData) {
		t.Fatalf("Export with empty slice should return ErrEmptyData, got %v", err)
	}
}

// TestExporter_PrepareData_EmptySingleSlice tests that an empty
// []any is rejected the same way, mirroring TestExporter_PrepareData_EmptySlice.
func TestExporter_PrepareData_EmptySingleSlice(t *testing.T) {
	ctx := context.Background()

	// Create empty single-level slice
	data := []any{}

	// Create exporter
	exp := exporter.NewExporter(data)

	err := exp.Export(ctx)
	if !errors.Is(err, excel.ErrEmptyData) {
		t.Fatalf("Export with empty []interface{} should return ErrEmptyData, got %v", err)
	}
}
