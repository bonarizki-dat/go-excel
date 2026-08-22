package exporter_test

import (
	"context"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// TestExporter_ShouldAutoSize_WidensColumnToLongestCell verifies each
// column's width after export matches its longest cell's character
// count plus the fixed padding, not a fixed or unset width.
func TestExporter_ShouldAutoSize_WidensColumnToLongestCell(t *testing.T) {
	data := [][]any{
		{"Name", "Note"},
		{"Alice", "short"},
		{"Bob", "a much longer note than the others"},
	}

	exp := exporter.NewExporter(data, excel.WithHeadings(true), excel.ShouldAutoSize(true))
	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}

	file := exp.GetFile()
	widthA, err := file.GetColWidth("Sheet1", "A")
	if err != nil {
		t.Fatalf("GetColWidth(A): %v", err)
	}
	// "Name" (4 chars, header) is measured against "Alice" (5) and
	// "Bob" (3); the widest is 5, plus the fixed padding.
	if want := 5.0 + 2.0; widthA != want {
		t.Errorf("column A width = %v, want %v", widthA, want)
	}

	widthB, err := file.GetColWidth("Sheet1", "B")
	if err != nil {
		t.Fatalf("GetColWidth(B): %v", err)
	}
	longest := len("a much longer note than the others")
	if want := float64(longest) + 2.0; widthB != want {
		t.Errorf("column B width = %v, want %v", widthB, want)
	}
}

// TestExporter_ShouldAutoSize_ExplicitWidthWins verifies a column with
// an explicit SetColumnWidth keeps that width instead of the measured
// one, even when ShouldAutoSize is enabled.
func TestExporter_ShouldAutoSize_ExplicitWidthWins(t *testing.T) {
	data := [][]any{
		{"Name"},
		{"a very long value that would otherwise widen the column a lot"},
	}

	exp := exporter.NewExporter(data, excel.WithHeadings(true), excel.ShouldAutoSize(true))
	exp.SetColumnWidth("A", 12.0)

	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}

	width, err := exp.GetFile().GetColWidth("Sheet1", "A")
	if err != nil {
		t.Fatalf("GetColWidth(A): %v", err)
	}
	if width != 12.0 {
		t.Errorf("column A width = %v, want the explicit 12.0 to win over the measured width", width)
	}
}

// TestExportToSheets_ShouldAutoSize_MeasuresEachSheetIndependently
// verifies auto-size measurements from one sheet are not attributed to
// another when a single sink writes multiple sheets in sequence: each
// sheet's column A width should reflect only that sheet's own data.
func TestExportToSheets_ShouldAutoSize_MeasuresEachSheetIndependently(t *testing.T) {
	exp := exporter.NewExporter(nil, excel.ShouldAutoSize(true))

	sheets := []excel.Sheet{
		{Name: "Short", Data: [][]any{{"a"}}},
		{Name: "Long", Data: [][]any{{"a much longer value here"}}},
	}

	if err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		t.Fatalf("ExportToSheets: %v", err)
	}

	file := exp.GetFile()

	shortWidth, err := file.GetColWidth("Short", "A")
	if err != nil {
		t.Fatalf("GetColWidth(Short, A): %v", err)
	}
	if want := 1.0 + 2.0; shortWidth != want {
		t.Errorf("Short sheet column A width = %v, want %v", shortWidth, want)
	}

	longWidth, err := file.GetColWidth("Long", "A")
	if err != nil {
		t.Fatalf("GetColWidth(Long, A): %v", err)
	}
	if want := float64(len("a much longer value here")) + 2.0; longWidth != want {
		t.Errorf("Long sheet column A width = %v, want %v", longWidth, want)
	}
}

// TestExporter_ShouldAutoSize_Disabled_NoWidthApplied verifies a column
// keeps excelize's default width when ShouldAutoSize is left at its
// default (false), even with long cell values present.
func TestExporter_ShouldAutoSize_Disabled_NoWidthApplied(t *testing.T) {
	data := [][]any{{"a very long value that would widen the column if auto-sizing were enabled"}}

	exp := exporter.NewExporter(data, excel.WithHeadings(false))
	if err := exp.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}

	width, err := exp.GetFile().GetColWidth("Sheet1", "A")
	if err != nil {
		t.Fatalf("GetColWidth(A): %v", err)
	}
	// excelize's undefined-width default; asserting it is small proves
	// no auto-size width was ever applied.
	if width > 20 {
		t.Errorf("column A width = %v, want the small excelize default (auto-sizing was not enabled)", width)
	}
}
