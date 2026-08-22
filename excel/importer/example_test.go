package importer_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

// Example_basicImport reads rows and headings back from an xlsx file
// written with a header row.
func Example_basicImport() {
	dir, err := os.MkdirTemp("", "go-excel-example")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "input.xlsx")

	ctx := context.Background()

	data := [][]any{
		{"Name", "Age"},
		{"John Doe", 30},
		{"Jane Smith", 25},
	}
	if err := exporter.NewExporter(data, excel.WithHeadings(true)).ExportToFile(ctx, path); err != nil {
		fmt.Println("error:", err)
		return
	}

	imp := importer.NewImporter(excel.WithHeadings(true))
	if err := imp.ImportFromFile(ctx, path); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("headings: %v\n", imp.GetHeadings())
	for i, row := range imp.GetRows() {
		fmt.Printf("row %d: %v\n", i+1, row)
	}

	// Output:
	// headings: [Name Age]
	// row 1: [John Doe 30]
	// row 2: [Jane Smith 25]
}

// User has excel tags mapping columns to fields for Example_structMapping.
type User struct {
	ID     int     `excel:"id,required"`
	Name   string  `excel:"name,required"`
	Email  string  `excel:"email"`
	Age    int     `excel:"age"`
	Active bool    `excel:"active"`
	Score  float64 `excel:"score"`
}

// Example_structMapping imports Excel rows directly into a slice of
// structs, converting each column to its tagged field's Go type.
func Example_structMapping() {
	dir, err := os.MkdirTemp("", "go-excel-example")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "users.xlsx")

	ctx := context.Background()

	data := [][]any{
		{"id", "name", "email", "age", "active", "score"},
		{1, "John Doe", "john@example.com", 30, true, 95.5},
		{2, "Jane Smith", "jane@example.com", 25, true, 88.0},
	}
	if err := exporter.NewExporter(data, excel.WithHeadings(true)).ExportToFile(ctx, path); err != nil {
		fmt.Println("error:", err)
		return
	}

	users, err := importer.ImportToStructs[User](ctx, path)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	for _, u := range users {
		fmt.Printf("ID=%d Name=%s Email=%s Age=%d Active=%v Score=%.1f\n",
			u.ID, u.Name, u.Email, u.Age, u.Active, u.Score)
	}

	// Output:
	// ID=1 Name=John Doe Email=john@example.com Age=30 Active=true Score=95.5
	// ID=2 Name=Jane Smith Email=jane@example.com Age=25 Active=true Score=88.0
}
