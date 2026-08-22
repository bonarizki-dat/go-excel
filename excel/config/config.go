// Package config defines the shared Config type, functional options, and
// validation logic used by the exporter, importer, and stream packages to
// configure sheet/row/column ranges, formatting, and I/O behavior.
package config

import (
	"reflect"
	"time"
)

// Logger interface for logging operations
// Implement this interface to provide custom logging.
type Logger interface {
	// Debug logs a debug message
	Debug(msg string, fields ...any)

	// Info logs an info message
	Info(msg string, fields ...any)

	// Warn logs a warning message
	Warn(msg string, fields ...any)

	// Error logs an error message
	Error(msg string, fields ...any)
}

// Converter converts a cell's raw string value to dest, the Go type of
// the struct field it will be assigned to during struct-mapped import.
// A Converter registered in Config.Converters for a column runs
// instead of the built-in string/int/float/bool/time.Time conversion,
// for column types that conversion does not natively handle.
type Converter func(raw string, dest reflect.Type) (any, error)

// Metric interface for metrics collection
// Implement this interface to collect performance metrics.
type Metric interface {
	// Inc increments a counter metric
	Inc(name string, value int64)

	// Gauge sets a gauge metric
	Gauge(name string, value float64)

	// Histogram records a histogram value
	Histogram(name string, value float64)

	// Timing records a timing value (duration in milliseconds)
	Timing(name string, duration int64)
}

// Config holds the configuration for export/import operations
// All fields have sensible defaults and can be customized via options.
type Config struct {
	// SheetName is the name of the sheet to export/import
	// Default: "Sheet1"
	SheetName string

	// WithHeadings indicates whether the first row contains headings
	// If true, the first row will be treated as column names
	// Default: true
	WithHeadings bool

	// StartRow is the absolute, 0-indexed row position to start reading
	// from (after the heading row, if any, has already been removed).
	// Default: 0
	StartRow int

	// EndRow is the absolute, 0-indexed, exclusive upper bound for rows
	// to read: rows in [StartRow, EndRow) are kept, independent of how
	// StartRow was set. If 0, reads through the end of the sheet.
	// Default: 0 (no upper bound)
	EndRow int

	// StartCol is the absolute, 0-indexed column position to start
	// reading from.
	// Default: 0
	StartCol int

	// EndCol is the absolute, 0-indexed, exclusive upper bound for
	// columns to read: columns in [StartCol, EndCol) are kept,
	// independent of how StartCol was set. If 0, reads through the end
	// of the row.
	// Default: 0 (no upper bound)
	EndCol int

	// SkipEmptyRows skips rows that are completely empty
	// Useful for handling sparse data
	// Default: true
	SkipEmptyRows bool

	// TrimSpaces trims leading/trailing whitespace from cell values
	// Useful for cleaning up data during import
	// Default: true
	TrimSpaces bool

	// ValidateData enables min:/max:/pattern: tag enforcement during
	// struct-mapped import (see RowMapper.EnableValidation). Required-column
	// enforcement is unconditional and unaffected by this setting.
	// Default: false
	ValidateData bool

	// StopOnError stops processing when an error occurs
	// If false, errors are collected and processing continues
	// Default: true
	StopOnError bool

	// Timeout is the maximum duration for the operation
	// If 0, no timeout is set
	// Default: 0 (no timeout)
	Timeout time.Duration

	// PreallocateRows preallocates memory for the expected number of rows
	// Improves performance by avoiding reallocations
	// Default: true
	PreallocateRows bool

	// Logger is the logger instance for debug/info messages
	// If nil, no logging is performed
	// Default: nil
	Logger Logger

	// Metrics is the metrics collector for performance monitoring
	// If nil, no metrics are collected
	// Default: nil
	Metrics Metric

	// ProgressCallback is called periodically with progress updates
	// Useful for showing progress bars in UI
	// Default: nil
	ProgressCallback func(processed int64, total int64)

	// ErrorCallback is called when an error occurs
	// Allows custom error handling logic
	// Return true to continue, false to stop
	// Default: nil
	ErrorCallback func(row int, err error) bool

	// Format selects the on-disk document format. ExportToFile and
	// ImportFromFile derive it from the filename extension instead, so
	// this field only matters for ExportToWriter, ExportToBuffer, and
	// ImportFromReader, which have no filename to inspect.
	// Default: FileFormatXLSX
	Format FileFormat

	// CSV controls encoding/decoding behavior when Format is
	// FileFormatCSV (or the extension resolves to CSV). Ignored for
	// FileFormatXLSX.
	// Default: DefaultCSVOptions()
	CSV CSVOptions

	// Converters maps a column name to a Converter that handles that
	// column's string-to-Go-type conversion during struct-mapped
	// import, in place of ConvertToType's built-in conversion. Columns
	// with no entry keep using ConvertToType.
	// Default: nil (ConvertToType handles every column)
	Converters map[string]Converter

	// RowMap, if set, transforms each data row (not the header row)
	// before it is written during export, receiving and returning a
	// row in the same []any shape BasicExporter.Export accepts. An
	// error aborts the export with that error.
	// Default: nil (rows are written unmodified)
	RowMap func(row []any) ([]any, error)

	// StartCell shifts where XLSX export writes the top-left cell of
	// the header (or first data row, if WithHeadings is false),
	// expressed as an Excel cell reference such as "B3". Every
	// subsequent row and column shifts by the same offset. Row and
	// column numbers passed to ProgressCallback and ErrorCallback are
	// unaffected: they still count from 1 relative to the data, not
	// the sheet. Ignored for CSV export, which has no concept of a
	// starting cell.
	// Default: "" (write starting at A1)
	StartCell string

	// AutoSizeColumns measures the character length of every cell
	// written to XLSX export and widens each column to fit its widest
	// cell, applied once every row for that sheet has been written.
	// This is a character-count measurement, not Excel's own font-aware
	// autofit (which excelize cannot reproduce reliably across fonts);
	// a column with an explicit width from SetColumnWidth keeps that
	// width instead of the measured one. Ignored for CSV export.
	// Default: false
	AutoSizeColumns bool

	// DocProperties sets a subset of the exported XLSX workbook's OOXML
	// core document properties. Ignored for CSV export.
	// Default: DocProperties{} (no properties set; excelize's own
	// defaults apply)
	DocProperties DocProperties

	// StrictFormat, when true, makes Export/ExportToSheets fail with an
	// error instead of silently degrading when an XLSX-only feature
	// (styles, column widths, start cell, auto-sized columns, document
	// properties, or a second sheet passed to ExportToSheets) is
	// configured for a CSV export. When false (the default), those
	// features are dropped and a one-time warning is logged via
	// Logger, if one is set.
	// Default: false (degrade and warn)
	StrictFormat bool
}

// DefaultConfig returns a Config with sensible default values
// These defaults are optimized for typical use cases.
func DefaultConfig() *Config {
	return &Config{
		// Sheet configuration
		SheetName: "Sheet1",

		// Data handling
		WithHeadings:  true,
		StartRow:      0,
		EndRow:        0,
		StartCol:      0,
		EndCol:        0,
		SkipEmptyRows: true,
		TrimSpaces:    true,

		// Validation and error handling
		ValidateData: false,
		StopOnError:  true,
		Timeout:      0,

		// Memory management
		PreallocateRows: true,

		// Observability (disabled by default)
		Logger:           nil,
		Metrics:          nil,
		ProgressCallback: nil,
		ErrorCallback:    nil,

		// Document format
		Format: FileFormatXLSX,
		CSV:    DefaultCSVOptions(),
	}
}
