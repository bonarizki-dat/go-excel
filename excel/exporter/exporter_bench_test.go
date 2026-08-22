package exporter

import (
	// bytes is used to create readers from buffers for realistic usage.
	"bytes"
	// context provides cancellation support similar to production paths.
	"context"
	// testing provides Go benchmark harness.
	"testing"

	// excel provides functional options used by exporter.
	"github.com/bonarizki-dat/go-excel/excel"
)

// genRows generates a synthetic dataset with a header to simulate real-world tables.
func genRows(n int) [][]any {
	// Preallocate slice to avoid reallocations during append
	rows := make([][]any, 0, n+1)
	// Header row improves parity with WithHeadings pathway
	header := []any{"c1", "c2", "c3"}
	// Append header first
	rows = append(rows, header)
	// Populate n data rows with deterministic integer content
	for i := range n {
		rows = append(rows, []any{i, i * 2, i * 3})
	}
	// Return generated rows for benchmarks
	return rows
}

// BenchmarkExporter1K measures small dataset performance and allocations.
func BenchmarkExporter1K(b *testing.B) {
	// Prepare dataset of 1,000 rows
	data := genRows(1000)
	// Use background context to mimic typical usage
	ctx := context.Background()
	// Report allocations to track GC pressure
	b.ReportAllocs()
	// Run benchmark iterations
	for i := 0; i < b.N; i++ {
		// Create exporter with headings enabled to exercise header path
		exp := NewExporter(data, excel.WithSheetName("Bench"), excel.WithHeadings(true))
		// Export to buffer to avoid disk I/O while exercising writer pipeline
		_, err := exp.ExportToBuffer(ctx)
		// Fail fast on errors for reliable benchmark results
		if err != nil {
			b.Fatal(err)
		}
		// Close resources to avoid leaks across iterations
		_ = exp.Close()
	}
}

// BenchmarkExporterCSV1K measures CSV export performance and
// allocations at the same dataset size as BenchmarkExporter1K, for a
// direct XLSX-vs-CSV comparison.
func BenchmarkExporterCSV1K(b *testing.B) {
	data := genRows(1000)
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		exp := NewExporter(data, excel.WithHeadings(true), excel.WithFormat(excel.FileFormatCSV))
		_, err := exp.ExportToBuffer(ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = exp.Close()
	}
}

// BenchmarkExporterCSV10K is the CSV counterpart of BenchmarkExporter10K.
func BenchmarkExporterCSV10K(b *testing.B) {
	data := genRows(10000)
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		exp := NewExporter(data, excel.WithHeadings(true), excel.WithFormat(excel.FileFormatCSV))
		_, err := exp.ExportToBuffer(ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = exp.Close()
	}
}

// BenchmarkExporter10K measures small-to-medium dataset performance.
func BenchmarkExporter10K(b *testing.B) {
	// Prepare dataset of 10,000 rows for granular performance tracking
	data := genRows(10000)
	// Use background context to mimic typical usage
	ctx := context.Background()
	// Report allocations to track GC pressure
	b.ReportAllocs()
	// Run benchmark iterations
	for i := 0; i < b.N; i++ {
		// Create exporter with headings enabled to exercise header path
		exp := NewExporter(data, excel.WithSheetName("Bench"), excel.WithHeadings(true))
		// Export to buffer to avoid disk I/O while exercising writer pipeline
		_, err := exp.ExportToBuffer(ctx)
		// Fail fast on errors for reliable benchmark results
		if err != nil {
			b.Fatal(err)
		}
		// Close resources to avoid leaks across iterations
		_ = exp.Close()
	}
}

// BenchmarkExporter100K measures medium dataset performance.
func BenchmarkExporter100K(b *testing.B) {
	// Prepare dataset of 100,000 rows
	data := genRows(100000)
	// Use background context to mimic typical usage
	ctx := context.Background()
	// Report allocations to track GC pressure
	b.ReportAllocs()
	// Run benchmark iterations
	for i := 0; i < b.N; i++ {
		// Create exporter with headings enabled to exercise header path
		exp := NewExporter(data, excel.WithSheetName("Bench"), excel.WithHeadings(true))
		// Export to buffer to avoid disk I/O while exercising writer pipeline
		_, err := exp.ExportToBuffer(ctx)
		// Fail fast on errors for reliable benchmark results
		if err != nil {
			b.Fatal(err)
		}
		// Close resources to avoid leaks across iterations
		_ = exp.Close()
	}
}

// genStyledExporter builds an exporter with header, data, and column
// styles configured, exercising the style-resolution path added for
// style ID caching (see resolveHeaderStyleID, resolveDataStyleID,
// resolveColumnStyleID in exporter.go).
func genStyledExporter(data [][]any) *BasicExporter {
	exp := NewExporter(data, excel.WithSheetName("Bench"), excel.WithHeadings(true))
	exp.SetHeaderStyle(excel.NewStyle().Bold(true).BackgroundColor(excel.ColorBlue))
	exp.SetDataStyle(excel.NewStyle().BorderAll(excel.BorderThin))
	exp.SetColumnStyle("B", excel.NewStyle().FontColor(excel.ColorGreen))
	return exp
}

// BenchmarkExporterStyled1K measures export performance with header, data,
// and column styles configured. Before style ID caching (Phase 4 of the
// remediation plan), every row called file.NewStyle() at least once and
// every "B" column cell called it again, registering thousands of
// redundant style records for a single style definition; this benchmark
// exists to catch that regression.
func BenchmarkExporterStyled1K(b *testing.B) {
	data := genRows(1000)
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		exp := genStyledExporter(data)
		_, err := exp.ExportToBuffer(ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = exp.Close()
	}
}

// BenchmarkExporterStyled10K is BenchmarkExporterStyled1K at 10x scale.
func BenchmarkExporterStyled10K(b *testing.B) {
	data := genRows(10000)
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		exp := genStyledExporter(data)
		_, err := exp.ExportToBuffer(ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = exp.Close()
	}
}

// BenchmarkExporter1M measures large dataset performance; skipped in short mode.
func BenchmarkExporter1M(b *testing.B) {
	// Skip in short mode to keep CI fast
	if testing.Short() {
		b.Skip("skip large benchmark in short mode")
	}
	// Prepare dataset of 1,000,000 rows
	data := genRows(1000000)
	// Use background context to mimic typical usage
	ctx := context.Background()
	// Report allocations to track GC pressure
	b.ReportAllocs()
	// Run benchmark iterations
	for i := 0; i < b.N; i++ {
		// Create exporter with headings enabled to exercise header path
		exp := NewExporter(data, excel.WithSheetName("Bench"), excel.WithHeadings(true))
		// Export to buffer to avoid disk I/O while exercising writer pipeline
		buf, err := exp.ExportToBuffer(ctx)
		// Fail fast on errors for reliable benchmark results
		if err != nil {
			b.Fatal(err)
		}
		// Construct reader from buffer to simulate downstream consumption
		_ = bytes.NewReader(buf)
		// Close resources to avoid leaks across iterations
		_ = exp.Close()
	}
}
