package excel

import (
	"time"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// Option is a function that modifies a Config
// Used for the functional options pattern to provide flexible configuration.
type Option func(*config.Config)

// WithSheetName sets the sheet name for export/import operations
// The sheet name must be valid according to Excel naming rules.
func WithSheetName(name string) Option {
	return func(c *config.Config) {
		c.SheetName = name
	}
}

// WithHeadings enables or disables header row handling
// When true, the first row is treated as column names.
func WithHeadings(enabled bool) Option {
	return func(c *config.Config) {
		c.WithHeadings = enabled
	}
}

// WithRows restricts processing to a 1-based, inclusive row range:
// rows numbered [first, last] are processed, where row 1 is the first
// data row (the heading row, if any, is never counted, regardless of
// WithHeadings). If last is 0, processes through the end of the sheet.
// Values of first below 1 are ignored, leaving the existing lower
// bound (default: no restriction) in place.
//
// WithRows replaces the removed WithRowRange, which was 0-indexed
// with an exclusive upper bound -- a different, incompatible
// convention from stream.WithImportRows' 1-based, inclusive one. The
// two were never unified into a shared convention because doing so
// would have silently changed which rows existing callers of one
// package or the other selected, with no compiler error to catch it.
func WithRows(first, last int) Option {
	return func(c *config.Config) {
		if first >= 1 {
			c.StartRow = first - 1
		}
		if last >= 0 {
			c.EndRow = last
		}
	}
}

// WithColumns restricts processing to a 1-based, inclusive column
// range: columns numbered [first, last] are processed, where column 1
// is the first column. If last is 0, processes through the end of the
// row. Values of first below 1 are ignored, leaving the existing
// lower bound (default: no restriction) in place.
//
// WithColumns replaces the removed WithColumnRange for the same
// reason WithRows replaces WithRowRange: it was 0-indexed with an
// exclusive upper bound, an incompatible convention worth a rename
// rather than a silent semantic change.
func WithColumns(first, last int) Option {
	return func(c *config.Config) {
		if first >= 1 {
			c.StartCol = first - 1
		}
		if last >= 0 {
			c.EndCol = last
		}
	}
}

// WithSkipEmptyRows enables or disables skipping of empty rows
// When true, completely empty rows are skipped during processing.
func WithSkipEmptyRows(skip bool) Option {
	return func(c *config.Config) {
		c.SkipEmptyRows = skip
	}
}

// WithTrimSpaces enables or disables trimming of cell values
// When true, leading/trailing whitespace is removed from values.
func WithTrimSpaces(trim bool) Option {
	return func(c *config.Config) {
		c.TrimSpaces = trim
	}
}

// WithValidation enables or disables min:/max:/pattern: tag enforcement
// during struct-mapped import. Required-column enforcement is
// unconditional and unaffected by this setting. See RowMapper.EnableValidation.
func WithValidation(enabled bool) Option {
	return func(c *config.Config) {
		c.ValidateData = enabled
	}
}

// WithStopOnError sets whether to stop on the first error
// If false, errors are collected and processing continues.
func WithStopOnError(stop bool) Option {
	return func(c *config.Config) {
		c.StopOnError = stop
	}
}

// WithTimeout sets the maximum duration for the operation
// If timeout is 0, no timeout is enforced.
func WithTimeout(timeout time.Duration) Option {
	return func(c *config.Config) {
		c.Timeout = timeout
	}
}

// WithPreallocation enables or disables row preallocation
// When enabled, memory is preallocated based on expected rows.
func WithPreallocation(enabled bool) Option {
	return func(c *config.Config) {
		c.PreallocateRows = enabled
	}
}

// WithLogger sets the logger for debug/info messages
// If nil, no logging is performed.
func WithLogger(logger Logger) Option {
	return func(c *config.Config) {
		c.Logger = logger
	}
}

// WithMetrics sets the metrics collector for monitoring
// If nil, no metrics are collected.
func WithMetrics(metrics Metric) Option {
	return func(c *config.Config) {
		c.Metrics = metrics
	}
}

// WithProgress sets the progress callback function
// Called periodically with progress updates (processed rows, total rows).
func WithProgress(callback func(processed int64, total int64)) Option {
	return func(c *config.Config) {
		c.ProgressCallback = callback
	}
}

// WithErrorHandler sets the error callback function
// Called when an error occurs during processing
// Return true to continue, false to stop.
func WithErrorHandler(callback func(row int, err error) bool) Option {
	return func(c *config.Config) {
		c.ErrorCallback = callback
	}
}

// WithConverters registers per-column Converters for struct-mapped
// import, keyed by column name. A column with an entry here has its
// cell value passed to that Converter instead of ConvertToType, for
// target types ConvertToType does not natively support. Columns with
// no entry are unaffected.
func WithConverters(converters map[string]Converter) Option {
	return func(c *config.Config) {
		c.Converters = converters
	}
}

// WithRowMap sets a function that transforms each data row (not the
// header row) before BasicExporter.Export writes it, letting a caller
// reshape or derive values without pre-processing the whole dataset up
// front. Its signature is distinct from stream.WithRowTransform, which
// predates this option and applies to the streaming export path with
// its own contract; the two are named differently, not unified, so
// that a caller migrating between excel/exporter and stream is not
// misled into believing they share behavior.
func WithRowMap(fn func(row []any) ([]any, error)) Option {
	return func(c *config.Config) {
		c.RowMap = fn
	}
}

// WithCustomStartCell shifts XLSX export to start writing at cell
// (an Excel reference such as "B3") instead of A1. Row and column
// numbers used for progress and error reporting still count from 1
// relative to the data, unaffected by the shift. Validated against
// excelize.CellNameToCoordinates by Config.Validate; a malformed
// reference surfaces there, not here. Has no effect on CSV export.
func WithCustomStartCell(cell string) Option {
	return func(c *config.Config) {
		c.StartCell = cell
	}
}

// ShouldAutoSize enables or disables measured column widths for XLSX
// export: when enabled, each column is widened to fit the longest cell
// written to it. This measures character length, not Excel's own
// font-aware autofit; a column with an explicit width from
// SetColumnWidth is unaffected. Has no effect on CSV export.
func ShouldAutoSize(enabled bool) Option {
	return func(c *config.Config) {
		c.AutoSizeColumns = enabled
	}
}

// WithProperties sets a subset of the exported XLSX workbook's OOXML
// core document properties (Title, Subject, Creator, Keywords,
// Description, Category, Version), applied via excelize's SetDocProps
// when the workbook is created, for both Export and ExportToSheets.
// Has no effect on CSV export, which has no document properties.
func WithProperties(props DocProperties) Option { //nolint:gocritic // props is passed once per Option chain, not per row; a pointer would only add an indirection for a value this short-lived, and a value keeps the excel.DocProperties{...} literal callers pass ergonomic
	return func(c *config.Config) {
		c.DocProperties = props
	}
}

// WithConfig replaces the entire configuration
// Useful for using a custom configuration object.
func WithConfig(cfg *config.Config) Option {
	return func(c *config.Config) {
		if cfg != nil {
			*c = *cfg
		}
	}
}

// WithFormat sets the document format explicitly. ExportToFile and
// ImportFromFile derive the format from the filename extension instead
// and ignore this option; it only takes effect for ExportToWriter,
// ExportToBuffer, and ImportFromReader, which have no filename to
// inspect.
func WithFormat(format FileFormat) Option {
	return func(c *config.Config) {
		c.Format = format
	}
}

// WithCSVDelimiter sets the field separator used for CSV export and
// import. Has no effect on XLSX operations. Default: ','.
func WithCSVDelimiter(delimiter rune) Option {
	return func(c *config.Config) {
		c.CSV.Delimiter = delimiter
	}
}

// WithCSVBOM enables or disables the UTF-8 byte order mark on exported
// CSV data. Excel misreads non-ASCII characters in a BOM-less UTF-8 CSV
// file, so this defaults to true. Has no effect on XLSX operations or on
// CSV import, which detects the BOM automatically if present.
func WithCSVBOM(enabled bool) Option {
	return func(c *config.Config) {
		c.CSV.UseBOM = enabled
	}
}

// WithCSVLineEnding sets the record terminator used for CSV export.
// Default: "\r\n". Has no effect on XLSX operations or on CSV import,
// which accepts either line ending regardless of this setting.
func WithCSVLineEnding(lineEnding string) Option {
	return func(c *config.Config) {
		c.CSV.LineEnding = lineEnding
	}
}

// WithCSVTimeLayout sets the time.Time.Format layout used to render
// time.Time values in CSV export. Default: "2006-01-02 15:04:05". Has no
// effect on XLSX operations.
func WithCSVTimeLayout(layout string) Option {
	return func(c *config.Config) {
		c.CSV.TimeLayout = layout
	}
}

// WithStrictFormat controls whether exporting to CSV with an XLSX-only
// feature configured (styles, column widths, a custom start cell,
// auto-sized columns, document properties, or more than one sheet
// passed to ExportToSheets) fails with an error wrapping
// ErrStrictFormatUnsupported, instead of dropping that feature and
// logging a one-time warning via Logger. Has no effect on XLSX
// operations, since none of those features are dropped there.
// Default: false (degrade and warn).
func WithStrictFormat(strict bool) Option {
	return func(c *config.Config) {
		c.StrictFormat = strict
	}
}
