package stream

import (
	"context"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// RowProcessor transforms a single imported row, returning the row to
// use (which may be row itself, unmodified) or an error that stops the
// import. rowNum is a 1-based sequential counter over the rows handed
// to processors, not the row's absolute position in the sheet: rows
// skipped by WithImportRows or WithImportSkipEmptyRows, or the heading
// row when WithImportHeadings is set, are never counted.
type RowProcessor func(ctx context.Context, rowNum int, row []any) ([]any, error)

// StreamImportOption configures StreamImporter behavior.
type StreamImportOption func(*StreamImporter) //nolint:revive // stutters intentionally: mirrors StreamExportOption naming

// WithImportWorkers sets the number of concurrent workers.
func WithImportWorkers(count int) StreamImportOption {
	return func(si *StreamImporter) {
		if count > 0 {
			si.numWorkers = count
		}
	}
}

// WithImportHeadings enables header row parsing.
func WithImportHeadings() StreamImportOption {
	return func(si *StreamImporter) {
		si.withHeadings = true
	}
}

// WithImportSheetName sets the source sheet name.
func WithImportSheetName(name string) StreamImportOption {
	return func(si *StreamImporter) {
		if name != "" {
			si.sheetName = name
		}
	}
}

// WithImportSkipEmptyRows enables or disables skipping of completely
// empty rows during import. Takes a bool, symmetric with excel
// package's WithSkipEmptyRows(bool), rather than the previous no-arg
// WithSkipEmptyRows() that could only ever enable the behavior.
func WithImportSkipEmptyRows(skip bool) StreamImportOption {
	return func(si *StreamImporter) {
		si.skipEmptyRows = skip
	}
}

// WithImportRows restricts import to rows from start through end,
// both 1-based; end of 0 means read through the last row.
func WithImportRows(start, end int) StreamImportOption {
	return func(si *StreamImporter) {
		if start > 0 {
			si.startRow = start
		}
		if end >= 0 {
			si.endRow = end
		}
	}
}

// WithImportProgress enables progress reporting with callback.
func WithImportProgress(callback ProgressCallback, interval uint64) StreamImportOption {
	return func(si *StreamImporter) {
		si.progressCallback = callback
		if interval > 0 {
			si.progressInterval = interval
		}
	}
}

// WithImportFormat selects the on-disk document format read from the
// importer's reader. Default is FileFormatXLSX.
func WithImportFormat(format config.FileFormat) StreamImportOption {
	return func(si *StreamImporter) {
		si.format = format
	}
}

// WithImportCSVOptions overrides the CSV decoding settings used when
// format is FileFormatCSV. Has no effect for FileFormatXLSX.
func WithImportCSVOptions(opts config.CSVOptions) StreamImportOption {
	return func(si *StreamImporter) {
		si.csvOptions = opts
	}
}

// WithImportLogger sets the logger used to report XLSX-only features,
// such as multiple sheets, that a CSV input cannot provide. Without a
// logger, degradation happens silently.
func WithImportLogger(logger excel.Logger) StreamImportOption {
	return func(si *StreamImporter) {
		si.logger = logger
	}
}
