// Package importer implements the non-streaming import pipeline: reading
// rows from an xlsx or CSV source, filtering/trimming them, and mapping
// them onto structs or plain string slices.
package importer

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

// BasicImporter implements the core import functionality for
// non-streaming use cases.
type BasicImporter struct {
	config *config.Config
	file   *excelize.File

	// rows holds the imported rows; each row is a slice of interface{}
	// for flexibility.
	rows [][]any

	// headings stores column headings if WithHeadings is true.
	headings []string

	// currentRow tracks the current row being read.
	currentRow int

	// totalRows tracks the total number of rows read.
	totalRows int64

	// errors collects errors during processing if StopOnError is false.
	errors []error

	// shouldCloseFile is true if the file was opened by this importer
	// (and must be closed on Close()), false if it was provided
	// externally via NewImporterWithFile.
	shouldCloseFile bool

	// format records which format the current or most recent import
	// used. CSV imports never set file, so GetSheetNames, GetSheetCount,
	// and SheetExists consult this to report a single logical sheet
	// instead of the "no file opened" error a nil file would otherwise
	// imply. Default: FileFormatXLSX.
	format config.FileFormat

	dispatcher *events.EventDispatcher
}

// NewImporter creates a new basic importer with the given options. The
// file parameter can be nil if you plan to use ImportFromFile or
// ImportFromReader.
func NewImporter(opts ...Option) *BasicImporter {
	cfg := config.DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &BasicImporter{
		config:          cfg,
		rows:            make([][]any, 0),
		headings:        make([]string, 0),
		errors:          make([]error, 0),
		shouldCloseFile: false,
		dispatcher:      events.NewEventDispatcher(),
	}
}

// NewImporterWithFile creates a new importer with an existing excelize
// file instance instead of one opened by ImportFromFile.
func NewImporterWithFile(file *excelize.File, opts ...Option) *BasicImporter {
	importer := NewImporter(opts...)
	importer.file = file
	importer.shouldCloseFile = false
	return importer
}

// On registers a hook for a specific event type.
func (i *BasicImporter) On(eventType excel.EventType, hook excel.Hook) {
	i.dispatcher.On(eventType, hook)
}

// Import performs the XLSX import operation with the given context,
// reading rows via the excelize file previously set by ImportFromFile,
// ImportFromReader, or NewImporterWithFile. Formats without an excelize
// file (CSV) do not go through this method; see importCSVFromReader in
// source_csv.go for that path. Both converge on importStringRows once
// rows are plain strings.
// Returns error if import fails.
func (i *BasicImporter) Import(ctx context.Context) error {
	ctx, cancel, err := i.prepareImport(ctx)
	defer cancel()
	if err != nil {
		return err
	}

	if i.file == nil {
		return excel.ErrInvalidFile
	}

	if i.config.Logger != nil {
		i.config.Logger.Info("Starting import",
			"sheet", i.config.SheetName)
	}

	// excelize.GetRows loads the entire sheet into memory; see the
	// stream package for a row-at-a-time alternative on large files.
	sheetRows, err := i.file.GetRows(i.config.SheetName)
	if err != nil {
		return excel.NewImportError("GetRows", i.config.SheetName, 0, 0, err)
	}

	return i.importStringRows(ctx, sheetRows)
}

// GetRows returns a copy of all imported rows.
// Each row is a slice of interface{} for flexibility. The returned outer
// slice is independent of the importer's internal state: appending to it
// or reassigning its elements does not affect a later call to GetRows or
// a subsequent Import on the same importer. Individual row slices are
// not imported into by this package after Import returns, so they are
// safe to read but should still be treated as owned by the caller only
// after this copy, not mutated in place.
func (i *BasicImporter) GetRows() [][]any {
	rows := make([][]any, len(i.rows))
	copy(rows, i.rows)
	return rows
}

// GetHeadings returns a copy of the column headings if WithHeadings was
// enabled, or an empty slice if no headings were extracted. The
// returned slice is independent of the importer's internal state, the
// same guarantee GetRows makes: mutating it does not affect a later
// call to GetHeadings or a subsequent Import on the same importer.
func (i *BasicImporter) GetHeadings() []string {
	headings := make([]string, len(i.headings))
	copy(headings, i.headings)
	return headings
}

// GetErrors returns all errors collected during import. Only populated
// if StopOnError is false.
func (i *BasicImporter) GetErrors() []error {
	return i.errors
}

// GetConfig returns a copy of the importer configuration, safe to
// modify without affecting the importer.
func (i *BasicImporter) GetConfig() *config.Config {
	return i.config.Clone()
}

// Close releases the importer's resources. Should be called when done
// with the importer.
func (i *BasicImporter) Close() error {
	if i.file != nil && i.shouldCloseFile {
		if err := i.file.Close(); err != nil {
			return fmt.Errorf("failed to close file: %w", err)
		}
	}

	return nil
}
