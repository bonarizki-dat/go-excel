package stream

import (
	"context"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// Functional options and callback types for StreamExporter.

// RowTransformer transforms a row before it is written to the output
// file. Transformers may run concurrently across worker goroutines and
// must be safe for concurrent use; they must not retain a reference to
// row after returning, since row may be reused by the caller.
//
// rowNum is the 1-based position of the row within the exported dataset,
// counting only data rows (the heading row, if any, is not counted).
type RowTransformer func(ctx context.Context, rowNum uint64, row []any) ([]any, error)

// ProgressCallback is invoked periodically during export or import to
// report progress: current is the number of rows processed so far, and
// message is an optional human-readable status string.
type ProgressCallback func(current uint64, message string)

// StreamExportOption configures StreamExporter behavior.
type StreamExportOption func(*StreamExporter) //nolint:revive // stutters intentionally: mirrors StreamImportOption naming

// WithExportWorkers sets the number of concurrent workers used to run the row
// transformer set by WithRowTransform. Has no effect on write ordering,
// since a single writer goroutine always reassembles rows in order.
//
// Default is runtime.NumCPU().
func WithExportWorkers(count int) StreamExportOption {
	return func(se *StreamExporter) {
		if count > 0 {
			se.numWorkers = count
		}
	}
}

// WithRowTransform sets a function to transform each row concurrently
// before it is written. Use this for CPU-bound per-row work; without a
// transformer, increasing worker count has no effect since writing
// itself is inherently sequential.
func WithRowTransform(transform RowTransformer) StreamExportOption {
	return func(se *StreamExporter) {
		se.transform = transform
	}
}

// WithExportHeadings enables a header row with the specified column names.
func WithExportHeadings(headings []string) StreamExportOption {
	return func(se *StreamExporter) {
		se.withHeadings = true
		se.headings = headings
	}
}

// WithExportSheetName sets the target sheet name.
func WithExportSheetName(name string) StreamExportOption {
	return func(se *StreamExporter) {
		if name != "" {
			se.sheetName = name
		}
	}
}

// WithExportProgress enables progress reporting with a callback.
//
// The callback is invoked every interval rows by default.
func WithExportProgress(callback ProgressCallback, interval uint64) StreamExportOption {
	return func(se *StreamExporter) {
		se.progressCallback = callback
		if interval > 0 {
			se.progressInterval = interval
		}
	}
}

// WithExportFormat selects the on-disk document format written by
// ExportStream. Default is FileFormatXLSX.
func WithExportFormat(format config.FileFormat) StreamExportOption {
	return func(se *StreamExporter) {
		se.format = format
	}
}

// WithExportCSVOptions overrides the CSV encoding settings used when
// format is FileFormatCSV. Has no effect for FileFormatXLSX.
func WithExportCSVOptions(opts config.CSVOptions) StreamExportOption {
	return func(se *StreamExporter) {
		se.csvOptions = opts
	}
}

// WithExportLogger sets the logger used to report XLSX-only features
// (styles, column widths, merged cells, multiple sheets) that are
// dropped when format does not support them, such as FileFormatCSV.
// Without a logger, degradation happens silently. Ignored when
// WithExportStrictFormat(true) is also set.
func WithExportLogger(logger excel.Logger) StreamExportOption {
	return func(se *StreamExporter) {
		se.logger = logger
	}
}

// WithExportStrictFormat controls whether passing more than one sheet
// to ExportToSheets on a FileFormatCSV exporter fails with an error
// wrapping excel.ErrStrictFormatUnsupported, instead of writing only
// the first sheet and logging a one-time warning via WithExportLogger.
// Has no effect on ExportStream, or on FileFormatXLSX exports.
// Default: false (degrade and warn).
func WithExportStrictFormat(strict bool) StreamExportOption {
	return func(se *StreamExporter) {
		se.strictFormat = strict
	}
}
