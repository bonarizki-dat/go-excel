package stream_test

import (
	"bytes"
	"context"
	"fmt"

	"github.com/bonarizki-dat/go-excel/stream"
)

// Example_streamExport writes rows to an xlsx file through a channel,
// without buffering the whole dataset in memory, then closes the
// exporter to release its resources.
func Example_streamExport() {
	var buf bytes.Buffer
	exporter, err := stream.NewStreamExporter(&buf, stream.WithExportHeadings([]string{"ID", "Name"}))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer exporter.Close()

	rows := make(chan []any, 2)
	rows <- []any{1, "Alice"}
	rows <- []any{2, "Bob"}
	close(rows)

	if err := exporter.ExportStream(context.Background(), rows); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("wrote %d rows, output non-empty: %v\n", exporter.GetRowCount(), buf.Len() > 0)

	// Output: wrote 2 rows, output non-empty: true
}

// Example_streamImport reads rows from an xlsx source through a
// RowProcessor callback, which lets each row be transformed or
// validated as it streams in rather than after the whole file loads.
func Example_streamImport() {
	var buf bytes.Buffer
	exp, err := stream.NewStreamExporter(&buf, stream.WithExportHeadings([]string{"ID", "Name"}))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	rows := make(chan []any, 2)
	rows <- []any{1, "Alice"}
	rows <- []any{2, "Bob"}
	close(rows)
	if err := exp.ExportStream(context.Background(), rows); err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := exp.Close(); err != nil {
		fmt.Println("error:", err)
		return
	}

	importer, err := stream.NewStreamImporter(bytes.NewReader(buf.Bytes()),
		stream.WithImportHeadings(),
		stream.WithImportWorkers(1), // keep this example's printed row order deterministic
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer importer.Close()

	var total int
	processor := func(_ context.Context, rowNum int, row []any) ([]any, error) {
		total++
		fmt.Printf("row %d: %v\n", rowNum, row)
		return row, nil
	}
	if err := importer.ImportStream(context.Background(), processor); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("processed %d rows\n", total)

	// Output:
	// row 1: [1 Alice]
	// row 2: [2 Bob]
	// processed 2 rows
}
