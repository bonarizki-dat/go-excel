package stream

import (
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/excel/csvio"
	"github.com/bonarizki-dat/go-excel/internal/concurrency"
	"github.com/xuri/excelize/v2"
)

// StreamExporter exports rows to an Excel file without buffering the
// entire dataset in memory at once.
//
// Rows are written to the underlying excelize.StreamWriter through a
// single writer goroutine, which is the only goroutine permitted to touch
// the writer, since excelize.StreamWriter is not safe for concurrent use.
// An optional RowTransformer can run concurrently across a worker pool;
// the writer goroutine reassembles transformed rows into their original
// order before writing them, so output order always matches input order
// regardless of worker count.
//
// Usage:
//
//	exporter, err := stream.NewStreamExporter(writer, stream.WithExportWorkers(8))
//	if err != nil {
//		log.Fatal(err)
//	}
//	err = exporter.ExportStream(ctx, rowChannel)
type StreamExporter struct { //nolint:revive // stutters intentionally: distinguishes it from StreamExportOption and friends in the same package
	// writer is the output destination for Excel data.
	// Can be a file, network stream, or any io.Writer.
	writer io.Writer

	// file is the excelize file instance for Excel operations.
	// Created once and reused for all rows.
	file *excelize.File

	// sheetName is the target sheet for export.
	// Default: "Sheet1".
	sheetName string

	// workerPool runs the row transformer concurrently, if one is set.
	// It never touches streamWriter directly.
	workerPool *concurrency.WorkerPool

	// numWorkers specifies concurrent worker count.
	// Default: runtime.NumCPU().
	numWorkers int

	// transform optionally transforms each row before it is written.
	// Runs concurrently across workers; nil means rows pass through
	// unchanged.
	transform RowTransformer

	// withHeadings indicates if the first row should be a header row.
	withHeadings bool

	// headings contains column headers if withHeadings is true.
	headings []string

	// rowCounter tracks the number of rows written so far.
	// Only the writer goroutine mutates this during export; readers use
	// atomic loads since export runs concurrently with caller code.
	rowCounter uint64

	// progressCallback is called periodically to report progress.
	// nil if progress reporting is disabled. Invoked only from the
	// writer goroutine, so callers never observe concurrent calls.
	progressCallback ProgressCallback

	// progressInterval specifies how often to report progress.
	// Default: every 100 rows.
	progressInterval uint64

	// streamWriter manages streaming writes to excelize.
	// Owned exclusively by the writer goroutine started in ExportStream.
	streamWriter *excelize.StreamWriter

	// format selects the on-disk document format written by
	// ExportStream. Default: FileFormatXLSX.
	format config.FileFormat

	// csvOptions controls CSV encoding when format is FileFormatCSV.
	// Ignored for FileFormatXLSX.
	csvOptions config.CSVOptions

	// logger receives a warning when an XLSX-only feature (styles,
	// column widths, merged cells, multiple sheets) is dropped because
	// format does not support it. Nil means no warning is emitted.
	// Ignored when strictFormat is true.
	logger excel.Logger

	// strictFormat, when true, makes ExportToSheets fail with an error
	// wrapping excel.ErrStrictFormatUnsupported instead of degrading
	// to a single sheet when more than one sheet is passed for a CSV
	// export. Default: false (degrade and warn via logger).
	strictFormat bool

	// sink is the row destination used by ExportStream, resolved from
	// format when the export starts.
	sink streamRowSink

	// closeOnce guards Close so a double call, or a call after
	// ExportStream already closed the file on an error path, never
	// double-closes se.file.
	closeOnce sync.Once
}

// NewStreamExporter creates a StreamExporter that writes Excel data
// directly to writer without buffering all data in memory, suitable
// for exporting large datasets. opts customizes its behavior; see the
// With* functions in exporter_options.go.
//
// Returns an error if WithExportSheetName set a name excelize's
// NewSheet rejects (see excelize's sheet-naming rules), instead of
// silently falling back to the default sheet name.
func NewStreamExporter(writer io.Writer, opts ...StreamExportOption) (*StreamExporter, error) {
	se := &StreamExporter{
		writer:           writer,
		sheetName:        "Sheet1",
		numWorkers:       runtime.NumCPU(),
		progressInterval: 100,
		csvOptions:       config.DefaultCSVOptions(),
	}

	for _, opt := range opts {
		opt(se)
	}

	// Kept unconditional even for FileFormatCSV, which never touches
	// it, since ExportToSheets requires a fresh file on every call
	// regardless of ExportStream's format and existing callers rely on
	// this field being non-nil immediately after construction.
	se.file = excelize.NewFile()

	if se.sheetName != "Sheet1" {
		if _, err := se.file.NewSheet(se.sheetName); err != nil {
			_ = se.file.Close()
			return nil, fmt.Errorf("create sheet %q: %w", se.sheetName, err)
		}
	}

	return se, nil
}

// Close releases the underlying excelize file's resources (temp files
// and in-memory buffers). Idempotent: only the first call has any
// effect, so it is safe to call after ExportStream already closed the
// file on an error path, or to call more than once directly.
//
// Returns any error from closing the underlying file.
func (se *StreamExporter) Close() error {
	var err error
	se.closeOnce.Do(func() {
		if se.file != nil {
			err = se.file.Close()
		}
	})
	return err
}

// hasHeadingRow reports whether a heading row was actually written,
// which determines the row offset applied to data rows.
func (se *StreamExporter) hasHeadingRow() bool {
	return se.withHeadings && len(se.headings) > 0
}

// dataRowOffset returns the 1-based Excel row number for the data row
// at the given sequence position (1-based, counting only data rows).
func (se *StreamExporter) dataRowOffset(seq uint64) int {
	row := int(seq)
	if se.hasHeadingRow() {
		row++
	}
	return row
}

// newSink resolves the row destination for ExportStream from se.format.
// XLSX keeps writing through se.file's stream writer so multi-sheet and
// GetFile-style access are unaffected; CSV writes straight to se.writer
// since it has no document wrapper for writeToOutput to finalize.
func (se *StreamExporter) newSink() (streamRowSink, error) {
	if se.format == config.FileFormatCSV {
		cw, err := csvio.NewWriter(se.writer, se.csvOptions)
		if err != nil {
			return nil, fmt.Errorf("create csv writer: %w", err)
		}
		return newCSVStreamSink(cw, se.csvOptions.TimeLayout), nil
	}

	sw, err := se.file.NewStreamWriter(se.sheetName)
	if err != nil {
		return nil, fmt.Errorf("create stream writer: %w", err)
	}
	se.streamWriter = sw
	return newXLSXStreamSink(sw), nil
}
