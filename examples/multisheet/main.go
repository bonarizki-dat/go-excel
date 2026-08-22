// Command multisheet demonstrates exporting to and importing from multiple
// sheets in a single workbook.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

func main() {
	ctx := context.Background()

	// Example 1: Export multiple sheets
	fmt.Println("=== Example 1: Export Multiple Sheets ===")
	if err := exportMultipleSheets(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Multi-sheet export completed: report.xlsx")

	// Example 2: Import from specific sheet
	fmt.Println("=== Example 2: Import from Specific Sheet ===")
	users, err := importFromSheet(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Imported %d users from 'Users' sheet\n\n", len(users))

	// Example 3: Import all sheets
	fmt.Println("=== Example 3: Import All Sheets ===")
	allSheets, err := importAllSheets(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Imported %d sheets:\n", len(allSheets))
	for sheetName, data := range allSheets {
		fmt.Printf("  - %s: %d rows\n", sheetName, len(data))
	}
	fmt.Println()

	// Example 4: Get sheet information
	fmt.Println("=== Example 4: Get Sheet Information ===")
	if err := getSheetInfo(ctx); err != nil {
		log.Fatal(err)
	}
}

// exportMultipleSheets demonstrates exporting multiple sheets to a single Excel file.
func exportMultipleSheets(ctx context.Context) error {
	// Prepare data for different sheets
	usersData := [][]any{
		{"ID", "Name", "Email"},
		{1, "John Doe", "john@example.com"},
		{2, "Jane Smith", "jane@example.com"},
		{3, "Bob Johnson", "bob@example.com"},
	}

	ordersData := [][]any{
		{"OrderID", "UserID", "Product", "Amount"},
		{101, 1, "Laptop", 1200.50},
		{102, 1, "Mouse", 25.99},
		{103, 2, "Keyboard", 75.00},
		{104, 3, "Monitor", 299.99},
	}

	productsData := [][]any{
		{"ProductID", "Name", "Price", "Stock"},
		{1, "Laptop", 1200.50, 10},
		{2, "Mouse", 25.99, 50},
		{3, "Keyboard", 75.00, 20},
		{4, "Monitor", 299.99, 15},
	}

	// Create sheets
	sheets := []excel.Sheet{
		{
			Name: "Users",
			Data: usersData,
		},
		{
			Name: "Orders",
			Data: ordersData,
		},
		{
			Name: "Products",
			Data: productsData,
		},
	}

	// Create exporter
	exp := exporter.NewExporter(nil, excel.WithHeadings(true))

	// Export to multiple sheets
	options := excel.DefaultMultiSheetExportOptions()
	options.ProgressCallback = func(_ context.Context, index int, name string) error {
		fmt.Printf("  Progress: Exported sheet %d: %s\n", index+1, name)
		return nil
	}

	if err := exp.ExportToSheets(ctx, sheets, options); err != nil {
		return fmt.Errorf("failed to export sheets: %w", err)
	}

	// Save to file
	if err := exp.GetFile().SaveAs("report.xlsx"); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// importFromSheet demonstrates importing data from a specific sheet.
func importFromSheet(ctx context.Context) ([][]any, error) {
	// Create importer
	imp := importer.NewImporter(excel.WithHeadings(true))

	// Import from specific sheet
	rows, err := imp.ImportFromSheet(ctx, "report.xlsx", "Users")
	if err != nil {
		return nil, fmt.Errorf("failed to import from sheet: %w", err)
	}

	// Get headings
	headings := imp.GetHeadings()
	fmt.Printf("  Headings: %v\n", headings)
	fmt.Printf("  Rows: %d\n", len(rows))

	return rows, nil
}

// importAllSheets demonstrates importing data from all sheets.
func importAllSheets(ctx context.Context) (map[string][][]any, error) {
	// Create importer
	imp := importer.NewImporter(excel.WithHeadings(true))

	// Import all sheets
	sheetsData, err := imp.ImportAllSheets(ctx, "report.xlsx")
	if err != nil {
		return nil, fmt.Errorf("failed to import all sheets: %w", err)
	}

	return sheetsData, nil
}

// getSheetInfo demonstrates getting sheet information.
func getSheetInfo(ctx context.Context) error {
	// Create importer and open file
	imp := importer.NewImporter()
	err := imp.ImportFromFile(ctx, "report.xlsx")
	if err != nil {
		return fmt.Errorf("failed to import file: %w", err)
	}

	// Get all sheet names (works on currently opened file)
	sheetNames, err := imp.GetSheetNames()
	if err != nil {
		return fmt.Errorf("failed to get sheet names: %w", err)
	}
	fmt.Printf("  Sheet names: %v\n", sheetNames)

	// Get sheet count (works on currently opened file)
	count, err := imp.GetSheetCount()
	if err != nil {
		return fmt.Errorf("failed to get sheet count: %w", err)
	}
	fmt.Printf("  Sheet count: %d\n", count)

	// Check if sheet exists (works on currently opened file)
	exists, err := imp.SheetExists("Users")
	if err != nil {
		return fmt.Errorf("failed to check sheet existence: %w", err)
	}
	fmt.Printf("  'Users' sheet exists: %v\n", exists)

	exists, err = imp.SheetExists("NonExistent")
	if err != nil {
		return fmt.Errorf("failed to check sheet existence: %w", err)
	}
	fmt.Printf("  'NonExistent' sheet exists: %v\n", exists)

	return nil
}
