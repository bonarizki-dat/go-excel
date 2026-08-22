package stream

import (
	"fmt"
	"io"
	"runtime"
	"sync/atomic"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/internal/concurrency"
	"github.com/xuri/excelize/v2"
)

// StreamImporter reads rows one at a time instead of materializing every
// row before the caller sees any of them, which is what BasicImporter's
// GetRows does.
//
// This is not constant-memory import. excelize.OpenReader loads the
// workbook when the importer is constructed, and rows.Columns() allocates
// per row inside excelize; measurements put those two at roughly 98% of
// total allocations. Below ~500K rows BasicImporter is actually the
// lighter choice, because GetRows pre-allocates more efficiently than the
// iterator. Reach for StreamImporter above ~1M rows, where holding every
// row at once is the thing that breaks. See docs/PERFORMANCE_GUIDE.md.
//
// Features:
//   - Row-at-a-time processing, so caller-side memory does not scale with row count
//   - Parallel row processing with worker pool
//   - Memory pooling to minimize allocations
//   - Context cancellation support
//   - Progress callbacks
//   - Row filtering and transformation
//
// Usage:
//
//	importer := stream.NewStreamImporter(reader, stream.WithImportWorkers(8))
//	err := importer.ImportStream(ctx, rowProcessor)
type StreamImporter struct { //nolint:revive // stutters intentionally: distinguishes it from StreamImportOption and friends in the same package
	// reader is the input source for Excel data
	// Can be file, network stream, or any io.Reader
	reader io.Reader

	// file is the excelize file instance for Excel operations
	// Created once and reused for all rows
	file *excelize.File

	// sheetName is the source sheet for import
	// Default: first sheet in workbook
	sheetName string

	// poolManager manages memory pools for efficient allocation
	poolManager *concurrency.PoolManager

	// workerPool processes rows in parallel
	workerPool *concurrency.WorkerPool

	// numWorkers specifies concurrent worker count
	// Default: runtime.NumCPU()
	numWorkers int

	// withHeadings indicates if first row contains headers
	withHeadings bool

	// headings contains column headers if withHeadings is true
	headings []string

	// skipEmptyRows indicates whether to skip empty rows
	skipEmptyRows bool

	// startRow specifies which row to start reading from
	// Default: 1 (or 2 if withHeadings is true)
	startRow int

	// endRow specifies which row to stop reading at
	// 0 means read until end of file
	endRow int

	// rowCounter tracks number of rows processed
	// Used for progress reporting and debugging
	rowCounter uint64

	// progressCallback is called periodically to report progress
	// nil if progress reporting is disabled
	progressCallback ProgressCallback

	// progressInterval specifies how often to report progress
	// Default: every 100 rows
	progressInterval uint64

	// format selects the on-disk document format read from reader.
	// Default: FileFormatXLSX.
	format config.FileFormat

	// csvOptions controls CSV decoding when format is FileFormatCSV.
	// Ignored for FileFormatXLSX.
	csvOptions config.CSVOptions

	// logger receives a warning when an XLSX-only feature (multiple
	// sheets) cannot be honored because format does not support it.
	// Nil means no warning is emitted.
	logger excel.Logger

	// source yields decoded rows regardless of format. Created once in
	// NewStreamImporter and shared by readHeadings and readRows.
	source streamRowSource
}

// RowProcessor, StreamImportOption, and all With* option functions are defined in importer_options.go

// NewStreamImporter creates a StreamImporter that reads Excel data from
// reader and hands rows to the caller one at a time instead of
// materializing all of them up front, the way BasicImporter.GetRows
// does. This is not constant-memory: excelize.OpenReader loads the
// whole workbook when this constructor runs, so peak memory still
// scales with file size. See the type doc above and
// docs/PERFORMANCE_GUIDE.md for when this is actually the lighter
// choice over BasicImporter. opts customizes its behavior; see the
// With* functions in importer_options.go.
func NewStreamImporter(reader io.Reader, opts ...StreamImportOption) (*StreamImporter, error) {
	si := &StreamImporter{
		reader:           reader,
		poolManager:      concurrency.NewPoolManager(),
		numWorkers:       runtime.NumCPU(),
		startRow:         1,
		endRow:           0, // 0 = read to end
		progressInterval: 100,
		csvOptions:       config.DefaultCSVOptions(),
	}

	for _, opt := range opts {
		opt(si)
	}

	if si.format == config.FileFormatCSV {
		si.source = newCSVStreamSource(reader, si.csvOptions)
	} else if err := si.initXLSXSource(reader); err != nil {
		return nil, err
	}

	if si.withHeadings {
		if err := si.readHeadings(); err != nil {
			_ = si.Close()
			return nil, fmt.Errorf("read headings: %w", err)
		}
	}

	return si, nil
}

// initXLSXSource opens reader as an xlsx workbook, resolves si.sheetName
// to the first sheet when unset, and builds si.source from it. On any
// error, the opened *excelize.File (if any) is closed before returning.
func (si *StreamImporter) initXLSXSource(reader io.Reader) error {
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return fmt.Errorf("open excel file: %w", err)
	}
	si.file = file

	if si.sheetName == "" {
		sheets := file.GetSheetList()
		if len(sheets) == 0 {
			_ = file.Close()
			return fmt.Errorf("no sheets found in workbook")
		}
		si.sheetName = sheets[0]
	}

	source, err := newXLSXStreamSource(file, si.sheetName)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("get rows iterator: %w", err)
	}
	si.source = source
	return nil
}

// readRows, readHeadings, and isRowEmpty are defined in importer_read.go.

// GetHeadings returns a copy of the column headers read from the
// source's first row, or an empty slice if WithImportHeadings was not
// used. The returned slice is independent of the importer's internal
// state: mutating it does not affect a later call to GetHeadings.
func (si *StreamImporter) GetHeadings() []string {
	headings := make([]string, len(si.headings))
	copy(headings, si.headings)
	return headings
}

// GetRowCount returns the number of rows processed so far. It is safe
// to call concurrently with a running ImportStream.
func (si *StreamImporter) GetRowCount() uint64 {
	return atomic.LoadUint64(&si.rowCounter)
}

// GetStats returns a snapshot of the underlying worker pool's
// statistics, for monitoring import progress and performance. It
// returns the zero WorkerStats if ImportStream has not yet started.
func (si *StreamImporter) GetStats() WorkerStats {
	if si.workerPool != nil {
		return si.workerPool.Stats()
	}
	return WorkerStats{}
}

// Close releases the underlying file and row source. Callers should
// call it once import is complete.
func (si *StreamImporter) Close() error {
	if si.source != nil {
		if err := si.source.close(); err != nil {
			return err
		}
	}
	if si.file != nil {
		return si.file.Close()
	}
	return nil
}
