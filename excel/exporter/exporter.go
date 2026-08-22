// Package exporter implements the non-streaming export pipeline: converting
// input data into rows, applying styling and headers, and writing the
// result to a file, io.Writer, or in-memory buffer.
package exporter

import (
	"context"
	"fmt"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/internal/events"
	"github.com/xuri/excelize/v2"
)

// Option is imported from parent excel package for constructor.
type Option = excel.Option

// BasicExporter implements the core export functionality for
// non-streaming use cases.
type BasicExporter struct {
	config *config.Config
	data   any

	// file mirrors the excelize file when the current or most recent
	// export used FileFormatXLSX; nil otherwise (e.g. after a CSV
	// export). Kept for GetFile() and for ExportToSheets, which manages
	// its own excelize file directly rather than going through sink.
	file *excelize.File

	// sink is the format-specific destination Export() writes rows to.
	// Built fresh by newSink() at the start of every Export() call, so
	// it always reflects the exporter's current config.Format.
	sink excel.RowSink

	// currentRow tracks the current row being written.
	currentRow int

	// totalRows tracks the total number of rows (if known).
	totalRows int64

	// errors collects errors during processing if StopOnError is false.
	errors []error

	dispatcher *events.EventDispatcher

	// Styles. Only xlsxSink honors these; formats without styling
	// support (CSV) ignore them entirely rather than erroring.
	headerStyle  *excel.Style
	dataStyle    *excel.Style
	columnStyles map[string]*excel.Style
	columnWidths map[string]float64
}

// NewExporter creates a new basic exporter with the given data and
// options. data can be [][]interface{} for raw data, []struct{} for
// structured data, or any type implementing the Exportable interface.
func NewExporter(data any, opts ...Option) *BasicExporter {
	cfg := config.DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &BasicExporter{
		config:       cfg,
		data:         data,
		errors:       make([]error, 0),
		dispatcher:   events.NewEventDispatcher(),
		columnStyles: make(map[string]*excel.Style),
		columnWidths: make(map[string]float64),
	}
}

// On registers a hook for a specific event type.
func (e *BasicExporter) On(eventType excel.EventType, hook excel.Hook) {
	e.dispatcher.On(eventType, hook)
}

// SetHeaderStyle sets the style for the header row. Only takes effect
// for FileFormatXLSX exports; CSV has no styling concept.
func (e *BasicExporter) SetHeaderStyle(style *excel.Style) {
	e.headerStyle = style
}

// SetDataStyle sets the default style for data rows. Only takes effect
// for FileFormatXLSX exports; CSV has no styling concept.
func (e *BasicExporter) SetDataStyle(style *excel.Style) {
	e.dataStyle = style
}

// SetColumnStyle sets the style for a specific column (e.g., "A", "B").
// Only takes effect for FileFormatXLSX exports; CSV has no styling
// concept.
func (e *BasicExporter) SetColumnStyle(col string, style *excel.Style) {
	e.columnStyles[col] = style
}

// SetNumberFormat sets the number format for a specific column (e.g.,
// "A", "B") by merging fmtID into that column's existing style from
// SetColumnStyle, or creating a new column style if none is set yet.
// Only takes effect for FileFormatXLSX exports; CSV has no styling
// concept. See the Format* constants in formatter.go for common
// built-in format IDs.
func (e *BasicExporter) SetNumberFormat(col string, fmtID int) {
	style, ok := e.columnStyles[col]
	if !ok {
		style = excel.NewStyle()
	}
	style.Format(fmtID)
	e.columnStyles[col] = style
}

// SetColumnWidth sets the width for a specific column.
func (e *BasicExporter) SetColumnWidth(col string, width float64) {
	e.columnWidths[col] = width
}

// Export performs the export operation with the given context. This is
// the main entry point for export operations.
func (e *BasicExporter) Export(ctx context.Context) error {
	// Bound the whole operation if a timeout was configured via WithTimeout.
	if e.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.config.Timeout)
		defer cancel()
	}

	if err := e.dispatcher.Dispatch(ctx, &excel.Event{Type: excel.BeforeExport, Data: e.data}); err != nil {
		return err
	}

	if err := e.config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// Check if data is empty. Beyond a nil interface, a non-nil slice
	// with zero elements (e.g. [][]any{} or []any{}) is just as empty
	// from the caller's point of view and should be rejected the same
	// way, not silently produce a file with no rows.
	if isEmptyExportData(e.data) {
		return excel.ErrEmptyData
	}

	// Close any sink left over from a previous Export()-family call on
	// this exporter before replacing it, otherwise its resources leak:
	// GetFile() and Close() only ever reach the current sink/file, so
	// nothing else can reach the old one once we overwrite the fields.
	if err := e.closeCurrent(); err != nil {
		return fmt.Errorf("close previous export file: %w", err)
	}
	e.sink = nil
	e.file = nil

	// Build the sink for the configured format. A new sink means any
	// style IDs an xlsxSink cached belonged to the discarded file and
	// would be meaningless (or point at an unrelated style) in the new
	// one; xlsxSink starts that cache empty on every construction.
	sink, err := e.newSink()
	if err != nil {
		return err
	}
	e.sink = sink
	if xs, ok := sink.(*xlsxSink); ok {
		e.file = xs.file
	} else if err := e.warnCSVDegradation(); err != nil {
		return err
	}

	rows, err := e.prepareData(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare data: %w", err)
	}

	if err := e.exportRows(ctx, rows); err != nil {
		return fmt.Errorf("failed to export rows: %w", err)
	}

	// Auto-size widths are only known once every row has been written,
	// so they are computed here rather than incrementally; Finalize/
	// SaveAs also flush, making this a no-op if that already happened
	// (flushAutoSizeWidths empties measuredColWidths once applied).
	if xs, ok := e.sink.(*xlsxSink); ok {
		if err := xs.flushAutoSizeWidths(); err != nil {
			return err
		}
	}

	if !e.config.StopOnError && len(e.errors) > 0 {
		return fmt.Errorf("export completed with %d errors: first error: %w",
			len(e.errors), e.errors[0])
	}

	// Trigger AfterExport event. Data is e.file for FileFormatXLSX
	// exports, matching pre-CSV-support behavior exactly; it is nil for
	// formats such as CSV that have no excelize file at all.
	return e.dispatcher.Dispatch(ctx, &excel.Event{Type: excel.AfterExport, Data: e.file})
}

// Close releases the exporter's resources. Should be called when done
// with the exporter.
func (e *BasicExporter) Close() error {
	if err := e.closeCurrent(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}
	return nil
}

// GetFile returns the underlying excelize.File instance for the current
// or most recent FileFormatXLSX export. Returns nil if no export has run
// yet, or if the current/most recent export used a different format
// (e.g. CSV), since those formats have no excelize file at all.
func (e *BasicExporter) GetFile() *excelize.File {
	return e.file
}

// GetSink returns the excel.RowSink built by the current or most recent
// Export()-family call, or nil if no export has run yet. This is the
// non-XLSX counterpart to GetFile: ExportToSheets has no io.Writer
// parameter of its own, so retrieving CSV (or any future non-XLSX
// format) output after it requires calling GetSink().Finalize(ctx, w)
// directly, the same way XLSX output is retrieved via
// GetFile().SaveAs(...).
func (e *BasicExporter) GetSink() excel.RowSink {
	return e.sink
}

// GetErrors returns all errors collected during export. Only populated
// if StopOnError is false.
func (e *BasicExporter) GetErrors() []error {
	return e.errors
}

// GetConfig returns a copy of the exporter configuration, safe to
// modify without affecting the exporter.
func (e *BasicExporter) GetConfig() *config.Config {
	return e.config.Clone()
}
