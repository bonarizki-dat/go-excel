package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/xuri/excelize/v2"
)

// iterateSheet applies the importer's row filters (heading skip, row
// range, and skip-empty-rows) to src and invokes rowHandler for each
// surviving row, mirroring the semantics readRows applies in
// importer_read.go for ImportStream. Row numbering is 1-based and
// counts the heading row, matching the numbering readRows uses and
// users see in Excel.
//
// src is owned by the caller: iterateSheet neither closes it nor stores
// it on si, so importing a specific sheet or file cannot disturb the
// source and file si.source/si.file point to for ImportStream. The
// heading row, when withHeadings is set, is consumed and discarded
// rather than recorded into si.headings, since si.headings describes
// the source si was constructed with, not whichever sheet is being
// iterated here.
func (si *StreamImporter) iterateSheet(ctx context.Context, src streamRowSource, rowHandler func([]any) error) error {
	rowNum := 0

	for {
		row, err := src.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("read row %d: %w", rowNum+1, err)
		}
		rowNum++

		skip, stop := si.rangeFilterSheetRow(rowNum)
		if stop {
			return nil
		}
		if skip {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if si.skipEmptyRows && si.isRowEmpty(row) {
			continue
		}

		if err := si.handleSheetRow(row, rowHandler); err != nil {
			return err
		}
	}
}

// rangeFilterSheetRow reports whether rowNum should be skipped (the
// heading row, or before startRow) or whether iteration should stop
// entirely (past endRow), mirroring the row-range semantics
// readRows applies for ImportStream.
func (si *StreamImporter) rangeFilterSheetRow(rowNum int) (skip, stop bool) {
	if si.withHeadings && rowNum == 1 {
		return true, false
	}
	if rowNum < si.startRow {
		return true, false
	}
	if si.endRow > 0 && rowNum > si.endRow {
		return false, true
	}
	return false, false
}

// handleSheetRow converts row into a fresh []any (never pool-backed,
// since rowHandler may retain it after returning) and invokes
// rowHandler, then reports progress.
func (si *StreamImporter) handleSheetRow(row []string, rowHandler func([]any) error) error {
	rowData := make([]any, len(row))
	for i, cell := range row {
		rowData[i] = cell
	}

	if err := rowHandler(rowData); err != nil {
		return fmt.Errorf("row handler error: %w", err)
	}

	count := atomic.AddUint64(&si.rowCounter, 1)
	if si.progressCallback != nil && si.progressInterval > 0 && count%si.progressInterval == 0 {
		si.progressCallback(count, fmt.Sprintf("Imported %d rows", count))
	}

	return nil
}

// ImportFromSheet streams rows from a single named sheet in filename,
// applying the same row filters (WithImportHeadings, WithImportRows,
// WithImportSkipEmptyRows) and progress reporting as ImportStream.
//
// filename is opened independently of the reader passed to
// NewStreamImporter, so this can be called on an importer built from a
// different source, or called repeatedly for several sheets or files,
// without disturbing si's own file or sheet. The heading row, when
// WithImportHeadings is set, is skipped but not exposed through
// GetHeadings, which continues to reflect only si's original source.
//
// Returns an error if filename or sheetName is empty, rowHandler is
// nil, filename resolves to CSV (which has no sheets), the file cannot
// be opened, the sheet does not exist, a row fails to decode,
// rowHandler returns an error, or ctx is canceled.
func (si *StreamImporter) ImportFromSheet(ctx context.Context, filename, sheetName string, rowHandler func([]any) error) (err error) {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if sheetName == "" {
		return fmt.Errorf("%w: sheet name cannot be empty", excel.ErrInvalidSheetName)
	}
	if rowHandler == nil {
		return fmt.Errorf("row handler cannot be nil")
	}
	if err := rejectCSVSheets(filename, si.format); err != nil {
		return err
	}

	file, err := excelize.OpenFile(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close file: %w", cerr)
		}
	}()

	if sheetIndex, sErr := file.GetSheetIndex(sheetName); sErr != nil || sheetIndex < 0 {
		return fmt.Errorf("%w: '%s'", excel.ErrSheetNotFound, sheetName)
	}

	src, err := newXLSXStreamSource(file, sheetName)
	if err != nil {
		return fmt.Errorf("failed to get rows iterator: %w", err)
	}
	defer func() {
		if cerr := src.close(); cerr != nil && err == nil {
			err = fmt.Errorf("close rows: %w", cerr)
		}
	}()

	return si.iterateSheet(ctx, src, rowHandler)
}

// ImportAllSheets streams rows from every sheet in filename, in
// workbook order, calling sheetHandler (if non-nil) before each sheet's
// rows and rowHandler for each row surviving the same filters
// ImportFromSheet applies. It stops at the first sheetHandler,
// row-decode, or rowHandler error rather than attempting the remaining
// sheets, so sheets after a failure are left unprocessed.
//
// sheetHandler, if non-nil, is invoked with each sheet name before that
// sheet's rows are streamed to rowHandler.
//
// Returns an error if filename is empty, rowHandler is nil, filename
// resolves to CSV, the file cannot be opened, it has no sheets, a
// handler returns an error, or ctx is canceled.
func (si *StreamImporter) ImportAllSheets(ctx context.Context, filename string, sheetHandler func(string) error, rowHandler func([]any) error) (err error) {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if rowHandler == nil {
		return fmt.Errorf("row handler cannot be nil")
	}
	if err := rejectCSVSheets(filename, si.format); err != nil {
		return err
	}

	file, err := excelize.OpenFile(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close file: %w", cerr)
		}
	}()

	sheetList := file.GetSheetList()
	if len(sheetList) == 0 {
		return fmt.Errorf("no sheets found in file")
	}

	for _, sheetName := range sheetList {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := si.importOneSheet(ctx, file, sheetName, sheetHandler, rowHandler); err != nil {
			return err
		}
	}

	return nil
}

// importOneSheet runs sheetHandler (if non-nil) and then streams
// sheetName's rows through rowHandler via iterateSheet, for a single
// sheet within ImportAllSheets's loop.
func (si *StreamImporter) importOneSheet(ctx context.Context, file *excelize.File, sheetName string, sheetHandler func(string) error, rowHandler func([]any) error) error {
	if sheetHandler != nil {
		if hErr := sheetHandler(sheetName); hErr != nil {
			return fmt.Errorf("sheet handler error for '%s': %w", sheetName, hErr)
		}
	}

	src, srcErr := newXLSXStreamSource(file, sheetName)
	if srcErr != nil {
		return fmt.Errorf("failed to get rows iterator for sheet '%s': %w", sheetName, srcErr)
	}

	iterErr := si.iterateSheet(ctx, src, rowHandler)
	closeErr := src.close()
	if iterErr != nil {
		return fmt.Errorf("sheet '%s': %w", sheetName, iterErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close rows for sheet '%s': %w", sheetName, closeErr)
	}
	return nil
}

// GetSheetNames returns the sheet names in filename, in workbook order.
// It does not touch si's own file or sheet.
//
// Returns an error if filename is empty, resolves to CSV, or cannot be
// opened.
func (si *StreamImporter) GetSheetNames(filename string) (names []string, err error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}
	if rejectErr := rejectCSVSheets(filename, si.format); rejectErr != nil {
		return nil, rejectErr
	}

	file, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close file: %w", cerr)
		}
	}()

	return file.GetSheetList(), nil
}

// rejectCSVSheets reports an error when filename or the importer format
// identifies CSV, which has no sheet concept and would otherwise fail
// inside excelize.OpenFile with a less useful message.
func rejectCSVSheets(filename string, format config.FileFormat) error {
	if format == config.FileFormatCSV || excel.FormatFromExtension(filename) == config.FileFormatCSV {
		return fmt.Errorf("%w: CSV files have no sheets; use ImportStream instead", excel.ErrInvalidFormat)
	}
	return nil
}
