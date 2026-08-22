// Command styling demonstrates applying cell styles (fonts, fills,
// borders, alignment) to an exported sheet.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

func main() {
	// Sample sales data
	data := [][]any{
		{"Product", "Category", "Price", "Quantity", "Total", "Date"},
		{"Laptop", "Electronics", 1200.50, 5, 6002.50, "2024-01-15"},
		{"Mouse", "Electronics", 25.99, 100, 2599.00, "2024-01-16"},
		{"Desk", "Furniture", 350.00, 10, 3500.00, "2024-01-17"},
		{"Chair", "Furniture", 150.75, 20, 3015.00, "2024-01-18"},
		{"Monitor", "Electronics", 299.99, 15, 4499.85, "2024-01-19"},
	}

	// Create exporter with headings enabled
	exp := exporter.NewExporter(data, excel.WithHeadings(true))

	// ============================================
	// HEADER STYLING
	// ============================================
	headerStyle := excel.NewStyle().
		Bold(true).                         // Bold text
		BackgroundColor("#4472C4").         // Blue background
		FontColor(excel.ColorWhite).        // White text
		FontSize(12).                       // Larger font
		AlignHorizontal(excel.AlignCenter). // Center align
		BorderAll(excel.BorderMedium)       // Medium borders

	exp.SetHeaderStyle(headerStyle)

	// ============================================
	// DATA ROW STYLING
	// ============================================
	dataStyle := excel.NewStyle().
		BorderAll(excel.BorderThin).     // Thin borders
		AlignVertical(excel.AlignMiddle) // Vertical center

	exp.SetDataStyle(dataStyle)

	// ============================================
	// COLUMN WIDTHS
	// ============================================
	exp.SetColumnWidth("A", 20.0) // Product
	exp.SetColumnWidth("B", 15.0) // Category
	exp.SetColumnWidth("C", 12.0) // Price
	exp.SetColumnWidth("D", 10.0) // Quantity
	exp.SetColumnWidth("E", 12.0) // Total
	exp.SetColumnWidth("F", 12.0) // Date

	// ============================================
	// COLUMN-SPECIFIC STYLING
	// ============================================

	// Price column (C) - Currency format with green color
	priceStyle := excel.NewStyle().
		Format(excel.FormatCurrency).
		FontColor("#00AA00").
		BorderAll(excel.BorderThin)
	exp.SetColumnStyle("C", priceStyle)

	// Total column (E) - Currency format with bold
	totalStyle := excel.NewStyle().
		Format(excel.FormatCurrency).
		Bold(true).
		FontColor("#0066CC").
		BorderAll(excel.BorderThin)
	exp.SetColumnStyle("E", totalStyle)

	// Date column (F) - Date format
	dateStyle := excel.NewStyle().
		Format(excel.FormatDate).
		BorderAll(excel.BorderThin)
	exp.SetColumnStyle("F", dateStyle)

	// ============================================
	// EXPORT
	// ============================================
	ctx := context.Background()
	if err := exp.Export(ctx); err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	// Save to file
	filename := "styled_sales_report.xlsx"
	if err := exp.GetFile().SaveAs(filename); err != nil {
		log.Fatalf("Failed to save file: %v", err)
	}

	fmt.Printf("✅ Styled Excel file created: %s\n", filename)
	fmt.Println("\nStyling applied:")
	fmt.Println("  • Header: Bold, blue background, white text, centered")
	fmt.Println("  • Data: Thin borders, vertically centered")
	fmt.Println("  • Price column: Currency format, green color")
	fmt.Println("  • Total column: Currency format, bold, blue color")
	fmt.Println("  • Date column: Date format")
	fmt.Println("  • Column widths: Optimized for readability")
}
