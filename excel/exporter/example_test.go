package exporter_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

// Example_basicExport exports a slice of rows, with a header row, to an
// xlsx file on disk.
func Example_basicExport() {
	dir, err := os.MkdirTemp("", "go-excel-example")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer os.RemoveAll(dir)

	data := [][]any{
		{"Name", "Age", "Email"},
		{"John Doe", 30, "john@example.com"},
		{"Jane Smith", 25, "jane@example.com"},
	}

	exp := exporter.NewExporter(data, excel.WithHeadings(true))
	if err := exp.ExportToFile(context.Background(), filepath.Join(dir, "output.xlsx")); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("exported 2 data rows")

	// Output: exported 2 data rows
}

// Example_csvExport exports the same rows as Example_basicExport, but as
// CSV with a custom delimiter and no BOM, into an in-memory buffer.
func Example_csvExport() {
	data := [][]any{
		{"Name", "Age", "Email"},
		{"John Doe", 30, "john@example.com"},
	}

	exp := exporter.NewExporter(data,
		excel.WithHeadings(true),
		excel.WithFormat(excel.FileFormatCSV),
		excel.WithCSVDelimiter(';'),
		excel.WithCSVBOM(false),
		excel.WithCSVLineEnding("\n"),
	)

	buf, err := exp.ExportToBuffer(context.Background())
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Print(string(buf))

	// Output:
	// Name;Age;Email
	// John Doe;30;john@example.com
}

// Example_styling applies header, data, and column-specific styles, plus
// explicit column widths, before exporting.
func Example_styling() {
	dir, err := os.MkdirTemp("", "go-excel-example")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer os.RemoveAll(dir)

	data := [][]any{
		{"Product", "Price"},
		{"Laptop", 1200.50},
		{"Mouse", 25.99},
	}

	exp := exporter.NewExporter(data, excel.WithHeadings(true))

	exp.SetHeaderStyle(excel.NewStyle().
		Bold(true).
		BackgroundColor("#4472C4").
		FontColor(excel.ColorWhite).
		AlignHorizontal(excel.AlignCenter))

	exp.SetDataStyle(excel.NewStyle().BorderAll(excel.BorderThin))

	exp.SetColumnStyle("B", excel.NewStyle().
		Format(excel.FormatCurrency).
		FontColor("#00AA00"))

	exp.SetColumnWidth("A", 20.0)

	if err := exp.Export(context.Background()); err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := exp.GetFile().SaveAs(filepath.Join(dir, "styled.xlsx")); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("styled export complete")

	// Output: styled export complete
}

// Example_multiSheet exports several sheets to one workbook, then reads
// a specific sheet and all sheets back.
func Example_multiSheet() {
	dir, err := os.MkdirTemp("", "go-excel-example")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "report.xlsx")

	ctx := context.Background()

	sheets := []excel.Sheet{
		{Name: "Users", Data: [][]any{
			{"ID", "Name"},
			{1, "John Doe"},
			{2, "Jane Smith"},
		}},
		{Name: "Orders", Data: [][]any{
			{"OrderID", "UserID"},
			{101, 1},
		}},
	}

	exp := exporter.NewExporter(nil, excel.WithHeadings(true))
	if err := exp.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := exp.GetFile().SaveAs(path); err != nil {
		fmt.Println("error:", err)
		return
	}

	imp := importer.NewImporter(excel.WithHeadings(true))
	usersRows, err := imp.ImportFromSheet(ctx, path, "Users")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("Users sheet: %d data rows\n", len(usersRows))

	allSheets, err := imp.ImportAllSheets(ctx, path)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("workbook has %d sheets\n", len(allSheets))

	// Output:
	// Users sheet: 2 data rows
	// workbook has 2 sheets
}
