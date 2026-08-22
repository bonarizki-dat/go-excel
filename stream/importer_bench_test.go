package stream

import (
	"bytes"
	"context"
	"runtime"
	"testing"
)

// BenchmarkStreamImporter1K benchmarks streaming import of 1,000 rows.
//
// This represents a small dataset that fits entirely in memory.
// Used as baseline for comparison with BasicImporter.
func BenchmarkStreamImporter1K(b *testing.B) {
	// Number of rows and columns for this benchmark
	const (
		numRows = 1000
		numCols = 10
	)

	// Create test Excel file once
	excelData := createTestExcelFile(b, numRows, numCols)

	// Reset timer to exclude setup
	b.ResetTimer()

	// Report memory allocations
	b.ReportAllocs()

	// Run benchmark N times
	for i := 0; i < b.N; i++ {
		// Stop timer during setup
		b.StopTimer()

		// Create reader from test data
		reader := bytes.NewReader(excelData)

		// Create streaming importer
		importer, err := NewStreamImporter(reader,
			WithImportWorkers(runtime.NumCPU()),
			WithImportHeadings(),
			WithImportSkipEmptyRows(true),
		)
		if err != nil {
			b.Fatalf("create importer: %v", err)
		}

		// Define row processor (minimal processing for benchmark)
		processor := func(_ context.Context, _ int, row []any) ([]any, error) {
			// Just return row unchanged (simulates read-only processing)
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
	}
}

// BenchmarkStreamImporter10K benchmarks streaming import of 10,000 rows.
//
// This represents a medium dataset that still fits in memory but
// benefits from streaming to reduce peak memory usage.
func BenchmarkStreamImporter10K(b *testing.B) {
	// Number of rows and columns for this benchmark
	const (
		numRows = 10000
		numCols = 10
	)

	// Create test Excel file once
	excelData := createTestExcelFile(b, numRows, numCols)

	// Reset timer to exclude setup
	b.ResetTimer()

	// Report memory allocations
	b.ReportAllocs()

	// Run benchmark N times
	for i := 0; i < b.N; i++ {
		// Stop timer during setup
		b.StopTimer()

		// Create reader from test data
		reader := bytes.NewReader(excelData)

		// Create streaming importer
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
	}
}

// BenchmarkStreamImporter100K benchmarks streaming import of 100,000 rows.
//
// This represents a large dataset where streaming becomes essential.
// Memory usage should remain constant regardless of row count.
func BenchmarkStreamImporter100K(b *testing.B) {
	// Number of rows and columns for this benchmark
	const (
		numRows = 100000
		numCols = 10
	)

	// Create test Excel file once
	excelData := createTestExcelFile(b, numRows, numCols)

	// Reset timer to exclude setup
	b.ResetTimer()

	// Report memory allocations
	b.ReportAllocs()

	// Run benchmark (may only run once due to time)
	for i := 0; i < b.N; i++ {
		// Stop timer during setup
		b.StopTimer()

		// Create reader from test data
		reader := bytes.NewReader(excelData)

		// Create streaming importer
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
	}
}
