// Command merge_cells demonstrates merging cells in an exported sheet
// using the underlying excelize.File.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
	"github.com/xuri/excelize/v2"
)

func main() {
	ctx := context.Background()

	// Prepare data
	data := [][]any{
		{"Product", "Q1", "Q2", "Q3", "Q4", "Total"},
		{"Laptop", 120, 150, 180, 200, 650},
		{"Mouse", 50, 60, 70, 80, 260},
		{"Keyboard", 30, 40, 45, 50, 165},
	}

	// Create exporter
	exp := exporter.NewExporter(data, excel.WithHeadings(true))

	// Export data first
	if err := exp.Export(ctx); err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	// Get excelize file instance for advanced operations
	file := exp.GetFile()
	sheetName := exp.GetConfig().SheetName

	// ============================================
	// MERGE CELLS EXAMPLES
	// ============================================

	// Example 1: Merge header cells for title
	// Merge A1:F1 to create a title row
	if err := file.MergeCell(sheetName, "A1", "F1"); err != nil {
		log.Fatalf("Failed to merge cells: %v", err)
	}

	// Set title value in merged cell (use top-left cell)
	if err := file.SetCellValue(sheetName, "A1", "Sales Report 2024"); err != nil {
		log.Fatalf("Failed to set title: %v", err)
	}

	// Style the merged title cell
	titleStyle, err := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  16,
			Color: "#FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		log.Fatalf("Failed to create title style: %v", err)
	}

	if err := file.SetCellStyle(sheetName, "A1", "F1", titleStyle); err != nil {
		log.Fatalf("Failed to apply title style: %v", err)
	}

	// Example 2: Merge cells in a column (e.g., merge "Total" label)
	// This is useful for grouping related data
	// Note: We need to adjust row numbers since we added a title row
	// Original data starts at row 2 (after title), so headers are at row 2
	// Data starts at row 3

	// Example 3: Merge cells horizontally (e.g., merge category cells)
	// If you have grouped data, you can merge cells to show hierarchy

	// Example 4: Merge cells for summary section
	// Merge cells for a summary row at the bottom
	lastRow := len(data) + 2 // +2 for title and header row
	summaryStart := fmt.Sprintf("A%d", lastRow+1)
	summaryEnd := fmt.Sprintf("E%d", lastRow+1)

	if err := file.MergeCell(sheetName, summaryStart, summaryEnd); err != nil {
		log.Fatalf("Failed to merge summary cells: %v", err)
	}

	if err := file.SetCellValue(sheetName, summaryStart, "Grand Total:"); err != nil {
		log.Fatalf("Failed to set summary label: %v", err)
	}

	// Calculate and set total
	totalCell := fmt.Sprintf("F%d", lastRow+1)
	if err := file.SetCellFormula(sheetName, totalCell, fmt.Sprintf("SUM(F3:F%d)", lastRow)); err != nil {
		log.Fatalf("Failed to set formula: %v", err)
	}

	// ============================================
	// SAVE FILE
	// ============================================
	filename := "merged_cells_example.xlsx"
	if err := file.SaveAs(filename); err != nil {
		log.Fatalf("Failed to save file: %v", err)
	}

	fmt.Printf("✅ Excel file with merged cells created: %s\n", filename)
	fmt.Println("\nMerged cells:")
	fmt.Println("  • A1:F1 - Title row (merged)")
	fmt.Println("  • Summary row - Label cells merged")
	fmt.Println("\nNote: Use GetFile() to access excelize.MergeCell() for advanced merge operations")
}
