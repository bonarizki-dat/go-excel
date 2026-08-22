// Command csv_format demonstrates exporting and importing CSV files and
// how CSV format limitations (styling, formulas, multiple sheets) degrade
// gracefully compared to xlsx.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

// consoleLogger prints Warn calls so the CSV degradation notices below are
// visible; Debug/Info/Error are no-ops to keep the example's output focused.
type consoleLogger struct{}

func (consoleLogger) Debug(_ string, _ ...any) {}
func (consoleLogger) Info(_ string, _ ...any)  {}
func (consoleLogger) Warn(msg string, fields ...any) {
	fmt.Printf("⚠️  %s %v\n", msg, fields)
}
func (consoleLogger) Error(_ string, _ ...any) {}

func main() {
	ctx := context.Background()

	data := [][]any{
		{"Name", "Age", "Email"},
		{"John Doe", 30, "john@example.com"},
		{"Jane Smith", 25, "jane@example.com"},
		{"Bob Johnson", 35, "bob@example.com"},
	}

	// ExportToFile derives the format from the extension, so a ".csv"
	// filename is all that's needed here.
	exp := exporter.NewExporter(data, excel.WithHeadings(true))
	if err := exp.ExportToFile(ctx, "output.csv"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("✅ CSV file created: output.csv")

	// ExportToWriter/ExportToBuffer have no filename to infer the format
	// from, so WithFormat selects it explicitly. Options here also show
	// overriding the CSV defaults (comma delimiter, BOM on, CRLF).
	exp2 := exporter.NewExporter(data,
		excel.WithHeadings(true),
		excel.WithFormat(excel.FileFormatCSV),
		excel.WithCSVDelimiter(';'),
		excel.WithCSVBOM(false),
		excel.WithCSVLineEnding("\n"),
	)
	buf, err := exp2.ExportToBuffer(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ CSV buffer with ';' delimiter, no BOM (%d bytes)\n", len(buf))

	// CSV has no concept of styles or column widths. Setting them does not
	// error; it logs one warning per export and the export still succeeds.
	exp3 := exporter.NewExporter(data,
		excel.WithFormat(excel.FileFormatCSV),
		excel.WithLogger(consoleLogger{}),
	)
	exp3.SetHeaderStyle(&excel.Style{Font: &excel.Font{Bold: true}})
	if _, err := exp3.ExportToBuffer(ctx); err != nil {
		log.Fatal(err)
	}

	// Import round-trips through the same format inference.
	imp := importer.NewImporter(excel.WithHeadings(true))
	if err := imp.ImportFromFile(ctx, "output.csv"); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Columns: %v\n", imp.GetHeadings())
	for i, row := range imp.GetRows() {
		fmt.Printf("Row %d: %v\n", i+1, row)
	}
}
