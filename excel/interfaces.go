package excel

import (
	"context"
	"io"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// RowSink receives prepared rows and produces a document in a specific
// on-disk format (XLSX, CSV, ...). It is the seam BasicExporter writes
// through instead of talking to a concrete format library directly, so
// adding a format means adding an implementation of this interface, not
// changing BasicExporter itself.
//
// The caller owns row ordering: WriteRow is expected to be called once
// per row, in ascending rowNum order, with the header row (if any) first.
// A RowSink implementation does not need to buffer or reorder rows
// itself.
type RowSink interface {
	// WriteRow writes one row. rowNum is 1-indexed to match Excel's
	// convention, even for formats like CSV where the number is only
	// used for error messages. isHeader is true for the header row so
	// implementations that support per-role styling (headers vs. data)
	// can apply it; implementations without styling ignore it.
	WriteRow(ctx context.Context, rowNum int, row []any, isHeader bool) error

	// Finalize completes the document and writes it to w. Called exactly
	// once, after every row has been written via WriteRow. Behavior of
	// calling WriteRow after Finalize is undefined.
	Finalize(ctx context.Context, w io.Writer) error

	// Close releases any resources the sink holds (open files, buffers).
	// Safe to call after Finalize, and safe to call without a prior
	// Finalize if the export was abandoned partway through.
	Close() error
}

// FileSaver is an optional capability of a RowSink for formats with a
// native "save to path" operation. Implementing it lets ExportToFile
// write the document straight to disk instead of buffering the whole
// thing in memory and copying it into an os.File. RowSink
// implementations without a native file writer (there are none among the
// formats this library supports today, but the seam exists for future
// ones) simply do not implement FileSaver, and ExportToFile falls back
// to Finalize.
type FileSaver interface {
	// SaveAs writes the document to the given path, creating parent
	// directories is the caller's responsibility, not SaveAs's.
	SaveAs(name string) error
}

// Exportable defines the base interface for exportable data
// Any type implementing this can be exported to Excel.
type Exportable interface {
	// Data returns the data to be exported
	// Can return various types: [][]interface{}, []struct{}, etc.
	Data() any
}

// Logger is an alias for config.Logger. The config package owns the
// definition because Config stores a Logger value; this alias lets
// callers write excel.Logger without importing the config package
// directly, matching the FileFormat and CSVOptions aliases in format.go.
type Logger = config.Logger

// Metric is an alias for config.Metric. The config package owns the
// definition for the same reason as Logger.
type Metric = config.Metric

// Converter is an alias for config.Converter. The config package owns
// the definition because Config.Converters stores values of this type.
type Converter = config.Converter
