package stream

import (
	"context"
	"io"
	"runtime"
	"testing"
)

// BenchmarkStreamExporter1K benchmarks streaming export of 1,000 rows.
//
// This represents a small dataset that fits entirely in memory.
// Used as baseline for comparison with BasicExporter.
func BenchmarkStreamExporter1K(b *testing.B) {
	// Number of rows and columns for this benchmark
	const (
		numRows = 1000
		numCols = 10
	)

	// Reset timer to exclude setup
	b.ResetTimer()

	// Report memory allocations
	b.ReportAllocs()

	// Run benchmark N times
	for i := 0; i < b.N; i++ {
		// Stop timer during setup
		b.StopTimer()

		// Create row channel
		rows := make(chan []any, 100)

		// Create streaming exporter (write to discard for speed)
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

// BenchmarkStreamExporter10K benchmarks streaming export of 10,000 rows.
//
// This represents a medium dataset that still fits in memory but
// benefits from streaming to reduce peak memory usage.
func BenchmarkStreamExporter10K(b *testing.B) {
	// Number of rows and columns for this benchmark
	const (
		numRows = 10000
		numCols = 10
	)

	// Reset timer to exclude setup
	b.ResetTimer()

	// Report memory allocations
	b.ReportAllocs()

	// Run benchmark N times
	for i := 0; i < b.N; i++ {
		// Stop timer during setup
		b.StopTimer()

		// Create row channel with larger buffer for throughput
		rows := make(chan []any, 1000)

		// Create streaming exporter
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

// BenchmarkStreamExporter100K benchmarks streaming export of 100,000 rows.
//
// This represents a large dataset where streaming becomes essential.
// Memory usage should remain constant regardless of row count.
func BenchmarkStreamExporter100K(b *testing.B) {
	// Number of rows and columns for this benchmark
	const (
		numRows = 100000
		numCols = 10
	)

	// Reset timer to exclude setup
	b.ResetTimer()

	// Report memory allocations
	b.ReportAllocs()

	// Run benchmark (may only run once due to time)
	for i := 0; i < b.N; i++ {
		// Stop timer during setup
		b.StopTimer()

		// Create row channel with large buffer
		rows := make(chan []any, 2000)

		// Create streaming exporter
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
