package exporter_test

import (
	"context"
	"slices"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// TestExporter_Export_DefaultSheet tests default sheet handling.
func TestExporter_Export_DefaultSheet(t *testing.T) {
	ctx := context.Background()

	// Create simple data
	data := [][]any{
		{"Column1"},
		{"Value1"},
	}

	// Create exporter with default sheet name
	exp := exporter.NewExporter(data)

	// Export should create default sheet
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export should succeed: %v", err)
	}

	// Verify sheet exists
	sheets := exp.GetFile().GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("Should have at least one sheet")
	}
}

// TestExporter_Export_CustomSheetName tests custom sheet name.
func TestExporter_Export_CustomSheetName(t *testing.T) {
	ctx := context.Background()

	data := [][]any{{"test"}}

	// Create exporter with custom sheet name
	customName := "MyCustomSheet"
	exp := exporter.NewExporter(data, excel.WithSheetName(customName))

	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify custom sheet name is used
	sheets := exp.GetFile().GetSheetList()
	found := slices.Contains(sheets, customName)
	if !found {
		t.Errorf("Sheet %s not found, available sheets: %v", customName, sheets)
	}
}

// TestExporter_Export_MultipleRows tests export with many rows.
func TestExporter_Export_MultipleRows(t *testing.T) {
	ctx := context.Background()

	// Create dataset with multiple rows
	rowCount := 50
	data := make([][]any, rowCount)
	for i := range data {
		data[i] = []any{i, "name_" + string(rune('A'+(i%26))), i * 100}
	}

	exp := exporter.NewExporter(data)

	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export with multiple rows failed: %v", err)
	}

	// Verify row count by checking last row
	file := exp.GetFile()
	// Try to read last row (row 50)
	lastCell, err := file.GetCellValue("Sheet1", "A50")
	if err != nil {
		t.Logf("Note: Could not verify last cell: %v", err)
	} else if lastCell == "" {
		t.Error("Last row should have data")
	}
}

// TestExporter_Export_MultipleColumns tests export with many columns.
func TestExporter_Export_MultipleColumns(t *testing.T) {
	ctx := context.Background()

	// Create data with multiple columns
	colCount := 15
	row := make([]any, colCount)
	for i := range row {
		row[i] = "col_" + string(rune('A'+i))
	}
	data := [][]any{row}

	exp := exporter.NewExporter(data)

	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export with multiple columns failed: %v", err)
	}

	// Verify columns were written
	file := exp.GetFile()
	firstCell, _ := file.GetCellValue("Sheet1", "A1")
	if firstCell != "col_A" {
		t.Errorf("First cell = %v, want col_A", firstCell)
	}
}

// TestExporter_Export_MixedTypes tests export with mixed data types.
func TestExporter_Export_MixedTypes(t *testing.T) {
	ctx := context.Background()

	// Create data with mixed types
	data := [][]any{
		{"String", "Int", "Float", "Bool", "Nil"},
		{"text", 42, 3.14, true, nil},
		{"more", -100, -2.5, false, nil},
	}

	exp := exporter.NewExporter(data)

	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export with mixed types failed: %v", err)
	}

	// Verify data was written
	file := exp.GetFile()
	cell, _ := file.GetCellValue("Sheet1", "A1")
	if cell != "String" {
		t.Errorf("Cell A1 = %v, want String", cell)
	}
}
