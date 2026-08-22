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

// BenchmarkStreamImporter1M benchmarks streaming import of 1,000,000 rows.
//
// This is the primary performance target: < 45 seconds.
// Memory usage must stay below 500MB.
func BenchmarkStreamImporter1M(b *testing.B) {
	// Skip this benchmark in short mode (too slow)
	if testing.Short() {
		b.Skip("skipping 1M row benchmark in short mode")
	}

	// Number of rows and columns for this benchmark
	const (
		numRows = 1000000
		numCols = 10
	)

	// Create test Excel file once (this takes time)
	b.Log("Creating test file with 1M rows...")
	excelData := createTestExcelFile(b, numRows, numCols)
	b.Logf("Test file size: %d MB", len(excelData)/(1024*1024))

	// Reset timer to exclude setup
	b.ResetTimer()

	// Report memory allocations
	b.ReportAllocs()

	// Run benchmark (likely only once due to time)
	for i := 0; i < b.N; i++ {
		// Stop timer during setup
		b.StopTimer()

		// Create reader from test data
		reader := bytes.NewReader(excelData)

		// Create streaming importer with max workers
		importer, err := NewStreamImporter(reader,
			WithImportWorkers(runtime.NumCPU()),
			WithImportHeadings(),
			WithImportSkipEmptyRows(true),
		)
		if err != nil {
			b.Fatalf("create importer: %v", err)
		}

		// Define row processor
		processor := func(_ context.Context, _ int, row []any) ([]any, error) {
			return row, nil
		}

		// Resume timer for actual import
		b.StartTimer()

		// Execute import
		ctx := context.Background()
		if err := importer.ImportStream(ctx, processor); err != nil {
			b.Fatalf("import failed: %v", err)
		}

		// Stop timer before cleanup
		b.StopTimer()

		// Close importer
		importer.Close()

		// Log result
		b.Logf("Imported %d rows", importer.GetRowCount())
	}
}

// BenchmarkStreamImporterCSV1M benchmarks streaming import of 1,000,000
// CSV rows, for comparison against BenchmarkStreamImporter1M's XLSX
// numbers. Unlike XLSX, CSV has no per-sheet row limit, so this is the
// direct 1M-row counterpart rather than needing a "MaxSheet" variant.
func BenchmarkStreamImporterCSV1M(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping 1M row benchmark in short mode")
	}

	const (
		numRows = 1000000
		numCols = 10
	)

	b.Log("Creating CSV test file with 1M rows...")
	csvData := createTestCSVFileStreaming(b, numRows, numCols)
	b.Logf("Test file size: %d MB", len(csvData)/(1024*1024))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		reader := bytes.NewReader(csvData)

		importer, err := NewStreamImporter(reader,
			WithImportFormat(config.FileFormatCSV),
			WithImportWorkers(runtime.NumCPU()),
			WithImportHeadings(),
			WithImportSkipEmptyRows(true),
		)
		if err != nil {
			b.Fatalf("create importer: %v", err)
		}

		processor := func(_ context.Context, _ int, row []any) ([]any, error) {
			return row, nil
		}

		b.StartTimer()

		ctx := context.Background()
		if err := importer.ImportStream(ctx, processor); err != nil {
			b.Fatalf("import failed: %v", err)
		}

		b.StopTimer()

		importer.Close()

		b.Logf("Imported %d rows", importer.GetRowCount())
	}
}

// BenchmarkStreamImporter10M is intentionally not implemented as a
// literal 10M-row single-sheet benchmark: XLSX enforces a hard limit of
// excelize.TotalRows (1,048,576) rows per sheet, so that dataset cannot
// exist as one sheet. See BenchmarkStreamImporterMaxSheet for the largest
// single-sheet row count that fits, or split the dataset across sheets /
// files for genuinely larger imports.

// BenchmarkStreamImporterMaxSheet benchmarks streaming import at the
// largest row count a single .xlsx sheet can hold.
//
// The fixture is built via createTestExcelFileStreaming (StreamExporter),
// not the SetCellValue-based createTestExcelFile: materializing ~1M rows
// through excelize's in-memory API before writing risks exhausting RAM on
// a memory-constrained machine, whereas ExportStream only holds a bounded
// in-flight window of rows.
//
// StreamImporter itself still loads the whole workbook via
// excelize.OpenReader when it opens the file (see stream/doc.go), so this
// benchmark's memory usage reflects that read-side cost, not just the
// per-row iteration.
func BenchmarkStreamImporterMaxSheet(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping max-sheet row benchmark in short mode")
	}

	const (
		numRows = excelize.TotalRows - 1 // row 1 is reserved for headings
		numCols = 10
	)

	b.Log("Creating test file with max sheet rows...")
	excelData := createTestExcelFileStreaming(b, numRows, numCols)
	b.Logf("Test file size: %d MB", len(excelData)/(1024*1024))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		reader := bytes.NewReader(excelData)

		importer, err := NewStreamImporter(reader,
			WithImportWorkers(runtime.NumCPU()),
			WithImportHeadings(),
			WithImportSkipEmptyRows(true),
		)
		if err != nil {
			b.Fatalf("create importer: %v", err)
		}

		processor := func(_ context.Context, _ int, row []any) ([]any, error) {
			return row, nil
		}

		b.StartTimer()

		ctx := context.Background()
		if err := importer.ImportStream(ctx, processor); err != nil {
			b.Fatalf("import failed: %v", err)
		}

		b.StopTimer()

		importer.Close()

		b.Logf("Imported %d rows", importer.GetRowCount())
	}
}

// BenchmarkStreamImporterWorkerScaling tests worker pool scaling.
//
// Measures performance with different worker counts to find optimal configuration.
func BenchmarkStreamImporterWorkerScaling(b *testing.B) {
	// Test different worker counts
	workerCounts := []int{1, 2, 4, 8, 16}

	// Create test data once
	const (
		numRows = 10000
		numCols = 10
	)
	excelData := createTestExcelFile(b, numRows, numCols)

	for _, workers := range workerCounts {
		// Create sub-benchmark for each worker count
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			// Reset timer
			b.ResetTimer()

			// Report allocations
			b.ReportAllocs()

			// Run benchmark
			for i := 0; i < b.N; i++ {
				// Stop timer during setup
				b.StopTimer()

				// Create reader
				reader := bytes.NewReader(excelData)

				// Create importer with specific worker count
				importer, err := NewStreamImporter(reader,
					WithImportWorkers(workers),
					WithImportHeadings(),
					WithImportSkipEmptyRows(true),
				)
				if err != nil {
					b.Fatalf("create importer: %v", err)
				}

				// Define processor
				processor := func(_ context.Context, _ int, row []any) ([]any, error) {
					return row, nil
				}

				// Start timer
				b.StartTimer()

				// Execute import
				ctx := context.Background()
				if err := importer.ImportStream(ctx, processor); err != nil {
					b.Fatalf("import failed: %v", err)
				}

				// Stop timer before cleanup
				b.StopTimer()

				// Close importer
				importer.Close()
			}
		})
	}
}
