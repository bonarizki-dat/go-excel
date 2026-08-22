package importer

import (
	// bytes is used to provide an in-memory reader for excelize.OpenReader.
	"bytes"
	// context provides cancellation support similar to production paths.
	"context"
	// testing provides Go benchmark harness.
	"testing"

	// excelize is the underlying Excel engine used to generate synthetic files.
	"github.com/xuri/excelize/v2"
	// excel provides functional options used by importer.
	"github.com/bonarizki-dat/go-excel/excel"
)

// makeExcelBuffer builds an Excel file with a header and rows, returning bytes for ImportFromReader.
func makeExcelBuffer(rows int) []byte {
	// Create a new Excel file instance
	f := excelize.NewFile()
	// Ensure default sheet exists and rename to a known sheet name for tests
	sheet := f.GetSheetName(0)
	if sheet == "" {
		sheet = "Sheet1"
		_, _ = f.NewSheet(sheet)
	}
	// Use a deterministic sheet name to match importer configuration
	_ = f.SetSheetName(sheet, "Bench")
	// Write header cells for WithHeadings path
	_ = f.SetCellValue("Bench", "A1", "c1")
	_ = f.SetCellValue("Bench", "B1", "c2")
	_ = f.SetCellValue("Bench", "C1", "c3")
	// Populate data rows using coordinate conversion for correctness
	for i := range rows {
		// Excel rows are 1-indexed; start at 2 due to header at row 1
		r := i + 2
		// Compute A/B/C column cell addresses for row r
		c1, _ := excelize.CoordinatesToCellName(1, r)
		c2, _ := excelize.CoordinatesToCellName(2, r)
		c3, _ := excelize.CoordinatesToCellName(3, r)
		// Write deterministic integer data to cells
		_ = f.SetCellValue("Bench", c1, i)
		_ = f.SetCellValue("Bench", c2, i*2)
		_ = f.SetCellValue("Bench", c3, i*3)
	}
	// Serialize file to buffer for in-memory import
	buf, _ := f.WriteToBuffer()
	// Close to release resources
	_ = f.Close()
	// Return raw bytes for reader construction
	return buf.Bytes()
}

// BenchmarkImporter1K measures small dataset import performance and allocations.
func BenchmarkImporter1K(b *testing.B) {
	// Prepare Excel buffer with 1,000 data rows
	data := makeExcelBuffer(1000)
	// Use background context to mimic typical usage
	ctx := context.Background()
	// Report allocations to track GC pressure
	b.ReportAllocs()
	// Run benchmark iterations
	for i := 0; i < b.N; i++ {
		// Create importer configured to read headings from first row
		imp := NewImporter(excel.WithSheetName("Bench"), excel.WithHeadings(true))
		// Use reader over buffer to exercise OpenReader pathway
		r := bytes.NewReader(data)
		// Import rows from reader
		err := imp.ImportFromReader(ctx, r)
		// Fail fast on errors for reliable benchmark results
		if err != nil {
			b.Fatal(err)
		}
		// Close resources to avoid leaks across iterations
		_ = imp.Close()
	}
}

// BenchmarkImporter10K measures small-to-medium dataset import performance.
func BenchmarkImporter10K(b *testing.B) {
	// Prepare Excel buffer with 10,000 data rows for granular performance tracking
	data := makeExcelBuffer(10000)
	// Use background context to mimic typical usage
	ctx := context.Background()
	// Report allocations to track GC pressure
	b.ReportAllocs()
	// Run benchmark iterations
	for i := 0; i < b.N; i++ {
		// Create importer configured to read headings from first row
		imp := NewImporter(excel.WithSheetName("Bench"), excel.WithHeadings(true))
		// Use reader over buffer to exercise OpenReader pathway
		r := bytes.NewReader(data)
		// Import rows from reader
		err := imp.ImportFromReader(ctx, r)
		// Fail fast on errors for reliable benchmark results
		if err != nil {
			b.Fatal(err)
		}
		// Close resources to avoid leaks across iterations
		_ = imp.Close()
	}
}

// BenchmarkImporter100K measures medium dataset import performance.
func BenchmarkImporter100K(b *testing.B) {
	// Prepare Excel buffer with 100,000 data rows
	data := makeExcelBuffer(100000)
	// Use background context to mimic typical usage
	ctx := context.Background()
	// Report allocations to track GC pressure
	b.ReportAllocs()
	// Run benchmark iterations
	for i := 0; i < b.N; i++ {
		// Create importer configured to read headings from first row
		imp := NewImporter(excel.WithSheetName("Bench"), excel.WithHeadings(true))
		// Use reader over buffer to exercise OpenReader pathway
		r := bytes.NewReader(data)
		// Import rows from reader
		err := imp.ImportFromReader(ctx, r)
		// Fail fast on errors for reliable benchmark results
		if err != nil {
			b.Fatal(err)
		}
		// Close resources to avoid leaks across iterations
		_ = imp.Close()
	}
}

// BenchmarkImporter1M measures large dataset import performance; skipped in short mode.
func BenchmarkImporter1M(b *testing.B) {
	// Skip in short mode to keep CI fast
	if testing.Short() {
		b.Skip("skip large benchmark in short mode")
	}
	// Prepare Excel buffer with 1,000,000 data rows
	data := makeExcelBuffer(1000000)
	// Use background context to mimic typical usage
	ctx := context.Background()
	// Report allocations to track GC pressure
	b.ReportAllocs()
	// Run benchmark iterations
	for i := 0; i < b.N; i++ {
		// Create importer configured to read headings from first row
		imp := NewImporter(excel.WithSheetName("Bench"), excel.WithHeadings(true))
		// Use reader over buffer to exercise OpenReader pathway
		r := bytes.NewReader(data)
		// Import rows from reader
		err := imp.ImportFromReader(ctx, r)
		// Fail fast on errors for reliable benchmark results
		if err != nil {
			b.Fatal(err)
		}
		// Close resources to avoid leaks across iterations
		_ = imp.Close()
	}
}
