package stream

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/xuri/excelize/v2"
)

// BenchmarkStreamExporter1M benchmarks streaming export of 1,000,000 rows.
//
// This is the primary performance target: < 30 seconds.
// Memory usage must stay below 500MB.
func BenchmarkStreamExporter1M(b *testing.B) {
	// Number of rows and columns for this benchmark
	const (
		numRows = 1000000
		numCols = 10
	)

	// Reset timer to exclude setup
	b.ResetTimer()

	// Report memory allocations
	b.ReportAllocs()

	// Run benchmark (likely only once due to time)
	for i := 0; i < b.N; i++ {
		// Stop timer during setup
		b.StopTimer()

		// Create row channel with large buffer for max throughput
		rows := make(chan []any, 5000)

		// Create streaming exporter with max workers
		exporter, err := NewStreamExporter(io.Discard,
			WithExportWorkers(runtime.NumCPU()),
			WithExportHeadings(generateHeadings(numCols)),
		)
		if err != nil {
			b.Fatalf("NewStreamExporter: %v", err)
		}

		// Start feeding rows in background
		go func() {
			defer close(rows)
			for r := range numRows {
				rows <- generateRow(r, numCols)
			}
		}()

		// Resume timer for actual export
		b.StartTimer()

		// Execute export
		ctx := context.Background()
		if err := exporter.ExportStream(ctx, rows); err != nil {
			b.Fatalf("export failed: %v", err)
		}
	}
}

// BenchmarkStreamExporterCSV1M benchmarks streaming export of 1,000,000
// rows to CSV, for comparison against BenchmarkStreamExporter1M's XLSX
// numbers. CSV has no per-sheet row limit, so unlike the XLSX
// benchmarks this scenario has no "MaxSheet" counterpart.
func BenchmarkStreamExporterCSV1M(b *testing.B) {
	const (
		numRows = 1000000
		numCols = 10
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		rows := make(chan []any, 5000)

		exporter, err := NewStreamExporter(io.Discard,
			WithExportFormat(config.FileFormatCSV),
			WithExportWorkers(runtime.NumCPU()),
			WithExportHeadings(generateHeadings(numCols)),
		)
		if err != nil {
			b.Fatalf("NewStreamExporter: %v", err)
		}

		go func() {
			defer close(rows)
			for r := range numRows {
				rows <- generateRow(r, numCols)
			}
		}()

		b.StartTimer()

		ctx := context.Background()
		if err := exporter.ExportStream(ctx, rows); err != nil {
			b.Fatalf("export failed: %v", err)
		}
	}
}

// BenchmarkStreamExporterMaxSheet benchmarks streaming export at the
// largest row count a single .xlsx sheet can hold.
//
// XLSX has a hard limit of excelize.TotalRows (1,048,576) rows per sheet,
// enforced by excelize itself (ErrMaxRows); a 10M-row single-sheet
// benchmark is not just slow, it is impossible. This benchmark uses the
// most data rows that fit once row 1 is reserved for headings, as a
// realistic ceiling for single-sheet export. For genuinely larger
// datasets, split across multiple sheets (see ExportToSheets) or files.
//
// Writes to io.Discard, so this measures CPU/allocation cost of the export
// path itself rather than disk I/O.
func BenchmarkStreamExporterMaxSheet(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping max-sheet row benchmark in short mode")
	}

	const (
		numRows = excelize.TotalRows - 1 // row 1 is reserved for headings
		numCols = 10
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		rows := make(chan []any, 5000)

		exporter, err := NewStreamExporter(io.Discard,
			WithExportWorkers(runtime.NumCPU()),
			WithExportHeadings(generateHeadings(numCols)),
		)
		if err != nil {
			b.Fatalf("NewStreamExporter: %v", err)
		}

		go func() {
			defer close(rows)
			for r := range numRows {
				rows <- generateRow(r, numCols)
			}
		}()

		b.StartTimer()

		ctx := context.Background()
		if err := exporter.ExportStream(ctx, rows); err != nil {
			b.Fatalf("export failed: %v", err)
		}
	}
}

// BenchmarkStreamExporterWorkerScaling tests worker pool scaling.
//
// Measures performance with different worker counts to find optimal configuration.
func BenchmarkStreamExporterWorkerScaling(b *testing.B) {
	// Test different worker counts
	workerCounts := []int{1, 2, 4, 8, 16}

	for _, workers := range workerCounts {
		// Create sub-benchmark for each worker count
		b.Run(fmt.Sprintf("Workers_%d", workers), func(b *testing.B) {
			// Number of rows for scaling test
			const (
				numRows = 10000
				numCols = 10
			)

			// Reset timer
			b.ResetTimer()

			// Report allocations
			b.ReportAllocs()

			// Run benchmark
			for i := 0; i < b.N; i++ {
				// Stop timer during setup
				b.StopTimer()

				// Create row channel
				rows := make(chan []any, 1000)

				// Create exporter with specific worker count
				exporter, err := NewStreamExporter(io.Discard,
					WithExportWorkers(workers),
					WithExportHeadings(generateHeadings(numCols)),
				)
				if err != nil {
					b.Fatalf("NewStreamExporter: %v", err)
				}

				// Feed rows
				go func() {
					defer close(rows)
					for r := range numRows {
						rows <- generateRow(r, numCols)
					}
				}()

				// Start timer
				b.StartTimer()

				// Execute export
				ctx := context.Background()
				if err := exporter.ExportStream(ctx, rows); err != nil {
					b.Fatalf("export failed: %v", err)
				}
			}
		})
	}
}
