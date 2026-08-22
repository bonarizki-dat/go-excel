package stream

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/xuri/excelize/v2"
)

// createTestExcelFile builds an in-memory Excel file with numRows rows
// of numCols columns each, using a mix of data types to resemble
// realistic benchmark data, and returns its serialized bytes.
func createTestExcelFile(b *testing.B, numRows, numCols int) []byte { //nolint:unparam // numCols is always 10 today, but kept configurable for future benchmarks
	// Create new Excel file
	file := excelize.NewFile()

	// Get default sheet name
	sheetName := file.GetSheetName(0)

	// Write headings
	for col := range numCols {
		cell := fmt.Sprintf("%s1", columnName(col))
		heading := fmt.Sprintf("Column%d", col+1)
		if err := file.SetCellValue(sheetName, cell, heading); err != nil {
			b.Fatalf("write heading: %v", err)
		}
	}

	// Write data rows
	for row := range numRows {
		for col := range numCols {
			// Calculate cell reference
			cell := fmt.Sprintf("%s%d", columnName(col), row+2) // +2 for 1-based and heading

			// Generate cell value based on column type
			var value any
			switch col % 4 {
			case 0:
				// String value
				value = fmt.Sprintf("Row%d_Col%d", row, col)
			case 1:
				// Integer value
				value = row*numCols + col
			case 2:
				// Float value
				value = float64(row) + float64(col)/100.0
			case 3:
				// Boolean value
				value = row%2 == 0
			}

			// Write cell value
			if err := file.SetCellValue(sheetName, cell, value); err != nil {
				b.Fatalf("write cell %s: %v", cell, err)
			}
		}
	}

	// Write to buffer
	buf := new(bytes.Buffer)
	if err := file.Write(buf); err != nil {
		b.Fatalf("write file: %v", err)
	}

	// Close file
	if err := file.Close(); err != nil {
		b.Fatalf("close file: %v", err)
	}

	return buf.Bytes()
}

// createTestExcelFileStreaming builds a benchmark fixture the same way
// createTestExcelFile does, but through StreamExporter instead of
// excelize's in-memory SetCellValue API. SetCellValue keeps the whole
// sheet materialized as Go objects while building it, which is fine at
// 1M rows but risks exhausting RAM at 10M rows on a memory-constrained
// machine; ExportStream only holds a small in-flight window of rows at
// a time.
func createTestExcelFileStreaming(b *testing.B, numRows, numCols int) []byte {
	buf := new(bytes.Buffer)

	exp, err := NewStreamExporter(buf,
		WithExportWorkers(runtime.NumCPU()),
		WithExportHeadings(generateHeadings(numCols)),
	)
	if err != nil {
		b.Fatalf("NewStreamExporter: %v", err)
	}

	rows := make(chan []any, 5000)
	go func() {
		defer close(rows)
		for r := range numRows {
			rows <- generateRow(r, numCols)
		}
	}()

	if err := exp.ExportStream(context.Background(), rows); err != nil {
		b.Fatalf("build fixture: %v", err)
	}

	return buf.Bytes()
}

// createTestCSVFileStreaming builds a CSV benchmark fixture through
// StreamExporter, mirroring createTestExcelFileStreaming, so CSV and
// XLSX benchmarks exercise comparable data at the same row/column
// counts.
func createTestCSVFileStreaming(b *testing.B, numRows, numCols int) []byte {
	buf := new(bytes.Buffer)

	exp, err := NewStreamExporter(buf,
		WithExportFormat(config.FileFormatCSV),
		WithExportWorkers(runtime.NumCPU()),
		WithExportHeadings(generateHeadings(numCols)),
	)
	if err != nil {
		b.Fatalf("NewStreamExporter: %v", err)
	}

	rows := make(chan []any, 5000)
	go func() {
		defer close(rows)
		for r := range numRows {
			rows <- generateRow(r, numCols)
		}
	}()

	if err := exp.ExportStream(context.Background(), rows); err != nil {
		b.Fatalf("build fixture: %v", err)
	}

	return buf.Bytes()
}

// columnName converts a 0-based column index to its Excel column name
// (A, B, C, ..., Z, AA, AB, ...).
func columnName(col int) string {
	// Simple implementation for columns A-Z (enough for benchmarks)
	if col < 26 {
		return string(rune('A' + col))
	}

	// For columns beyond Z, use AA, AB, etc.
	return fmt.Sprintf("A%s", string(rune('A'+(col-26))))
}
