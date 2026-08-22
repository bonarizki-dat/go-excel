// Command basic_export demonstrates exporting a slice of structs to an
// xlsx file using the excel/exporter package.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

func main() {
	ctx := context.Background()

	// Prepare data
	data := [][]any{
		{"Name", "Age", "Email"},
		{"John Doe", 30, "john@example.com"},
		{"Jane Smith", 25, "jane@example.com"},
		{"Bob Johnson", 35, "bob@example.com"},
	}

	// Create exporter with headings
	exp := exporter.NewExporter(data, excel.WithHeadings(true))

	// Export to file
	if err := exp.ExportToFile(ctx, "output.xlsx"); err != nil {
		log.Fatal(err)
	}

	fmt.Println("✅ Excel file created: output.xlsx")
}
