// Command basic_import demonstrates importing rows from an xlsx file using
// the excel/importer package.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

func main() {
	ctx := context.Background()

	// Create importer with headings enabled
	imp := importer.NewImporter(excel.WithHeadings(true))

	// Import from file
	err := imp.ImportFromFile(ctx, "input.xlsx")
	if err != nil {
		log.Fatal(err)
	}

	// Get imported data
	data := imp.GetRows()
	headings := imp.GetHeadings()

	// Display headings
	fmt.Printf("Columns: %v\n", headings)

	// Process data
	for i, row := range data {
		fmt.Printf("Row %d: %v\n", i+1, row)
	}
}
