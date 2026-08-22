// Command struct_mapping demonstrates mapping struct fields to columns
// using excel tags, including custom names, formats, and skipped fields.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
	"github.com/bonarizki-dat/go-excel/excel/importer"
	"github.com/xuri/excelize/v2"
)

// User represents a user with Excel tags for struct mapping.
type User struct {
	ID        int       `excel:"id,required"`
	Name      string    `excel:"name,required"`
	Email     string    `excel:"email"`
	Age       int       `excel:"age"`
	Active    bool      `excel:"active"`
	Score     float64   `excel:"score"`
	CreatedAt time.Time `excel:"created_at"`
}

func main() {
	ctx := context.Background()

	// Example 1: Create sample Excel file with user data
	fmt.Println("=== Example 1: Create Sample Excel File ===")
	if err := createSampleFile(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ Sample file created: users.xlsx")

	// Example 2: Import to structs
	fmt.Println("=== Example 2: Import to Structs ===")
	users, err := importToStructs(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Imported %d users:\n", len(users))
	for i, user := range users {
		fmt.Printf("  User %d: ID=%d, Name=%s, Email=%s, Age=%d, Active=%v, Score=%.2f\n",
			i+1, user.ID, user.Name, user.Email, user.Age, user.Active, user.Score)
	}
	fmt.Println()

	// Example 3: Type conversion demonstration
	fmt.Println("=== Example 3: Type Conversion ===")
	demonstrateTypeConversion(ctx)
}

// createSampleFile creates a sample Excel file with user data.
func createSampleFile(ctx context.Context) error {
	// Prepare data with headers
	data := [][]any{
		{"id", "name", "email", "age", "active", "score", "created_at"},
		{1, "John Doe", "john@example.com", 30, true, 95.5, "2024-01-15"},
		{2, "Jane Smith", "jane@example.com", 25, true, 88.0, "2024-01-16"},
		{3, "Bob Johnson", "bob@example.com", 35, false, 75.5, "2024-01-17"},
		{4, "Alice Williams", "alice@example.com", 28, true, 92.0, "2024-01-18"},
	}

	// Create exporter with headings
	exp := exporter.NewExporter(data, excel.WithHeadings(true))

	// Export to file
	if err := exp.ExportToFile(ctx, "users.xlsx"); err != nil {
		return fmt.Errorf("failed to export: %w", err)
	}

	return nil
}

// importToStructs demonstrates importing Excel data directly to Go structs.
func importToStructs(ctx context.Context) ([]User, error) {
	// Import directly to User structs
	// The library automatically:
	// - Maps column names to struct fields using excel tags
	// - Converts string values to appropriate types (int, float, bool, time)
	// - Validates required fields
	users, err := importer.ImportToStructs[User](ctx, "users.xlsx")
	if err != nil {
		return nil, fmt.Errorf("failed to import to structs: %w", err)
	}

	return users, nil
}

// demonstrateTypeConversion shows how different types are automatically converted.
func demonstrateTypeConversion(ctx context.Context) {
	// Create a file with various data types
	f := excelize.NewFile()
	sheet := "Sheet1"

	// Headers
	headers := []string{"id", "name", "age", "active", "score", "created_at"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			log.Fatalf("failed to set header cell: %v", err)
		}
	}

	// Data with different types (all as strings in Excel)
	data := [][]any{
		{"1", "Test User", "30", "true", "95.5", "2024-01-15"},
		{"2", "Another User", "25", "false", "88.0", "2024-01-16"},
	}

	for i, row := range data {
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			if err := f.SetCellValue(sheet, cell, val); err != nil {
				log.Fatalf("failed to set cell value: %v", err)
			}
		}
	}

	tmpFile := "type_conversion_demo.xlsx"
	if err := f.SaveAs(tmpFile); err != nil {
		log.Printf("Failed to create demo file: %v", err)
		return
	}
	defer func() {
		// Clean up
		_ = os.Remove(tmpFile)
	}()

	// Import - types are automatically converted
	users, err := importer.ImportToStructs[User](ctx, tmpFile)
	if err != nil {
		log.Printf("Import failed: %v", err)
		return
	}

	fmt.Printf("✅ Type conversion successful:\n")
	for _, user := range users {
		fmt.Printf("  - ID (int): %d (type: %T)\n", user.ID, user.ID)
		fmt.Printf("  - Age (int): %d (type: %T)\n", user.Age, user.Age)
		fmt.Printf("  - Active (bool): %v (type: %T)\n", user.Active, user.Active)
		fmt.Printf("  - Score (float64): %.2f (type: %T)\n", user.Score, user.Score)
		fmt.Printf("  - CreatedAt (time.Time): %v (type: %T)\n", user.CreatedAt, user.CreatedAt)
		fmt.Println()
	}
}
