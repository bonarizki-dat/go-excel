package exporter

import (
	"context"
	"fmt"
	"io"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// styleIDUnresolved marks headerStyleID/dataStyleID as not yet resolved
// for the current sink. 0 is not used as the sentinel because it is the
// conventional "no style" value used throughout this file.
const styleIDUnresolved = -1

// autoSizeColumnPadding is added to a column's measured character
// length before it becomes a SetColWidth value, leaving room so a
// cell's text does not butt directly against the column's edge.
const autoSizeColumnPadding = 2.0

// xlsxSink is the excel.RowSink implementation backed by excelize. It
// owns worksheet setup (sheet naming, column widths) and style-ID
// caching, so it is the only place in this package that talks to
// excelize directly; BasicExporter itself is format-agnostic.
type xlsxSink struct {
	file      *excelize.File
	sheetName string

	// ownsFile is true when this sink created file itself (via
	// newXLSXSink) and is therefore responsible for closing it. A sink
	// built by attachXLSXSink writes into a workbook the caller already
	// owns (BasicExporter.file, for ExportToSheets), so Close leaves
	// that workbook open for the caller to manage.
	ownsFile bool

	headerStyle  *excel.Style
	dataStyle    *excel.Style
	columnStyles map[string]*excel.Style
	columnWidths map[string]float64

	// colOffset and rowOffset shift every cell WriteRow writes, so that
	// logical row 1, column 1 (as WriteRow's own arguments and
	// exportRows' progress/error reporting see them) lands at
	// Config.StartCell instead of always at A1. Both are 0 when
	// StartCell is unset, the A1 default.
	colOffset int
	rowOffset int

	// autoSize mirrors Config.AutoSizeColumns. When true, writeCellValues
	// records each cell's character length into measuredColWidths, and
	// flushAutoSizeWidths later turns that into SetColWidth calls for
	// s.sheetName specifically, so measurements from one sheet are never
	// applied to another when a single sink writes multiple sheets.
	autoSize          bool
	measuredColWidths map[string]int

	// Cached excelize style IDs, resolved lazily on first use and reused
	// for the rest of the sink's lifetime instead of calling
	// file.NewStyle() again for every row or cell. A style ID is scoped
	// to the *excelize.File it was registered against, not to a single
	// sheet, so these caches stay valid across setSheet calls when one
	// sink writes multiple sheets in the same workbook.
	headerStyleID  int
	dataStyleID    int
	columnStyleIDs map[string]int
}

// newXLSXSink creates a fresh excelize workbook, ensures sheetName
// exists and is active, and applies columnWidths. headerStyle,
// dataStyle, and columnStyles are referenced, not copied: the sink
// reflects whatever was set on the exporter via
// SetHeaderStyle/SetDataStyle/SetColumnStyle at the moment Export()
// built it. startCell is an Excel cell reference such as "B3", or ""
// for the A1 default; the caller (BasicExporter.newSink) is
// responsible for validating it beforehand via Config.Validate.
// autoSize mirrors Config.AutoSizeColumns. docProperties mirrors
// Config.DocProperties; a zero-value DocProperties (WithProperties
// never called) leaves excelize's own defaults untouched.
func newXLSXSink(sheetName string, headerStyle, dataStyle *excel.Style, columnStyles map[string]*excel.Style, columnWidths map[string]float64, startCell string, autoSize bool, docProperties excel.DocProperties) (*xlsxSink, error) { //nolint:gocritic // docProperties is copied once per Export() call, not per row; a pointer would only add an indirection for a value this short-lived
	file := excelize.NewFile()

	if err := applyDocProperties(file, docProperties); err != nil {
		return nil, excel.NewExportError("SetDocProps", sheetName, 0, 0, err)
	}

	defaultSheet := file.GetSheetName(0)
	if defaultSheet == "" {
		if _, err := file.NewSheet(sheetName); err != nil {
			return nil, excel.NewExportError("NewSheet", sheetName, 0, 0, err)
		}
	} else if err := file.SetSheetName(defaultSheet, sheetName); err != nil {
		return nil, excel.NewExportError("SetSheetName", sheetName, 0, 0, err)
	}
	file.SetActiveSheet(0)

	sink, err := attachXLSXSink(file, headerStyle, dataStyle, columnStyles, columnWidths, startCell, autoSize)
	if err != nil {
		return nil, err
	}
	sink.ownsFile = true
	if err := sink.setSheet(sheetName); err != nil {
		return nil, err
	}
	return sink, nil
}

// WriteRow writes a single row to the Excel sheet.
// Row number is 1-indexed (Excel convention).
func (s *xlsxSink) WriteRow(ctx context.Context, rowNum int, row []any, isHeader bool) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Priority: headerStyle (if isHeader OR if headerStyle is set and this is the first row).
	// Note: If WithHeadings(false) but headerStyle is set, apply it to first row.
	isHeaderRow := isHeader || (rowNum == 1 && s.headerStyle != nil)

	var rowStyleID int
	var err error
	if isHeaderRow {
		rowStyleID, err = s.resolveHeaderStyleID()
	} else {
		rowStyleID, err = s.resolveDataStyleID()
	}
	if err != nil {
		return excel.NewExportError("NewStyle", s.sheetName, rowNum, 0, err)
	}

	// physicalRow is where rowNum actually lands on the sheet once
	// Config.StartCell's row offset is applied; rowNum itself keeps
	// counting from 1 for error reporting below, unaffected by the
	// shift.
	physicalRow := rowNum + s.rowOffset

	if err := s.writeCellValues(rowNum, physicalRow, row); err != nil {
		return err
	}

	if len(row) == 0 {
		return nil
	}

	// Header rows never take column-specific overrides, and data rows
	// with no column styles configured use the same style for every
	// cell. Both cases can be styled with a single contiguous-range
	// SetCellStyle call instead of one call per cell.
	if isHeaderRow || len(s.columnStyles) == 0 {
		return s.applyUniformRowStyle(rowNum, physicalRow, len(row), rowStyleID)
	}

	// Column-specific styles are configured, so adjacent cells may need
	// different styles: fall back to resolving and applying style per cell.
	return s.applyPerCellStyles(rowNum, physicalRow, row, rowStyleID)
}

// writeCellValues writes every cell's value in row at physicalRow,
// before any styling is applied. rowNum (unshifted) is only used to
// annotate errors.
func (s *xlsxSink) writeCellValues(rowNum, physicalRow int, row []any) error {
	for colNum, value := range row {
		cell, err := excelize.CoordinatesToCellName(colNum+1+s.colOffset, physicalRow)
		if err != nil {
			return excel.NewExportError("CoordinatesToCellName", s.sheetName, rowNum, colNum+1, err)
		}
		if err := s.file.SetCellValue(s.sheetName, cell, value); err != nil {
			return excel.NewExportError("SetCellValue", s.sheetName, rowNum, colNum+1, err)
		}
		if s.autoSize {
			s.measureCellWidth(colNum, value)
		}
	}
	return nil
}

// measureCellWidth records value's rendered character length against
// the running maximum for its physical column, for flushAutoSizeWidths
// to turn into a SetColWidth call later. colNum is the row-relative,
// 0-based column index before s.colOffset is applied.
func (s *xlsxSink) measureCellWidth(colNum int, value any) {
	colName, err := excelize.ColumnNumberToName(colNum + 1 + s.colOffset)
	if err != nil {
		return
	}
	if length := len(fmt.Sprint(value)); length > s.measuredColWidths[colName] {
		s.measuredColWidths[colName] = length
	}
}

// flushAutoSizeWidths turns measuredColWidths accumulated so far into
// SetColWidth calls against whatever sheet s.sheetName currently names,
// skipping any column with an explicit SetColumnWidth entry (which
// always wins over a measured width). Called once every row for the
// current sheet has been written (from Export/ExportToSheets), before
// setSheet retargets the sink to a different sheet, and again from
// Finalize/SaveAs as a harmless no-op safety net, so each sheet's own
// measurements land on that sheet rather than leaking into the next
// one or being lost if a caller never calls Finalize/SaveAs at all.
// No-op unless auto-sizing is enabled and at least one cell has been
// measured since the last flush.
func (s *xlsxSink) flushAutoSizeWidths() error {
	if !s.autoSize || s.sheetName == "" || len(s.measuredColWidths) == 0 {
		return nil
	}

	for colName, maxLen := range s.measuredColWidths {
		if _, explicit := s.columnWidths[colName]; explicit {
			continue
		}
		width := float64(maxLen) + autoSizeColumnPadding
		if err := s.file.SetColWidth(s.sheetName, colName, colName, width); err != nil {
			return excel.NewExportError("SetColWidth", s.sheetName, 0, 0, err)
		}
	}

	s.measuredColWidths = make(map[string]int)
	return nil
}

// applyPerCellStyles resolves and applies a style to each cell in row at
// physicalRow individually, falling back to rowStyleID for columns with
// no column-specific style configured. Column-specific styles are keyed
// by the physical column letter, the same one SetColumnStyle's caller
// used, so they are unaffected by the column offset.
func (s *xlsxSink) applyPerCellStyles(rowNum, physicalRow int, row []any, rowStyleID int) error {
	for colNum := range row {
		colName, err := excelize.ColumnNumberToName(colNum + 1 + s.colOffset)
		if err != nil {
			return excel.NewExportError("ColumnNumberToName", s.sheetName, rowNum, colNum+1, err)
		}

		styleID := rowStyleID
		colStyleID, hasColStyle, err := s.resolveColumnStyleID(colName)
		if err != nil {
			return excel.NewExportError("NewStyle", s.sheetName, rowNum, colNum+1, err)
		}
		if hasColStyle {
			styleID = colStyleID
		}
		if styleID == 0 {
			continue
		}

		cell, err := excelize.CoordinatesToCellName(colNum+1+s.colOffset, physicalRow)
		if err != nil {
			return excel.NewExportError("CoordinatesToCellName", s.sheetName, rowNum, colNum+1, err)
		}
		if err := s.file.SetCellStyle(s.sheetName, cell, cell, styleID); err != nil {
			return excel.NewExportError("SetCellStyle", s.sheetName, rowNum, colNum+1, err)
		}
	}

	return nil
}

// applyUniformRowStyle applies styleID to every cell in row physicalRow
// using a single range-based SetCellStyle call, since excelize accepts a
// first-cell/last-cell range for uniform styling. No-op if styleID is 0
// (the "no style" sentinel used throughout this file). rowNum
// (unshifted) is only used to annotate errors.
func (s *xlsxSink) applyUniformRowStyle(rowNum, physicalRow, colCount, styleID int) error {
	if styleID == 0 {
		return nil
	}

	first, err := excelize.CoordinatesToCellName(1+s.colOffset, physicalRow)
	if err != nil {
		return excel.NewExportError("CoordinatesToCellName", s.sheetName, rowNum, 1, err)
	}
	last, err := excelize.CoordinatesToCellName(colCount+s.colOffset, physicalRow)
	if err != nil {
		return excel.NewExportError("CoordinatesToCellName", s.sheetName, rowNum, colCount, err)
	}
	if err := s.file.SetCellStyle(s.sheetName, first, last, styleID); err != nil {
		return excel.NewExportError("SetCellStyle", s.sheetName, rowNum, 0, err)
	}
	return nil
}

// resolveHeaderStyleID returns the cached style ID for headerStyle,
// registering it with the file on first use. Returns 0 (no style) if
// headerStyle is unset.
func (s *xlsxSink) resolveHeaderStyleID() (int, error) {
	if s.headerStyle == nil {
		return 0, nil
	}
	if s.headerStyleID == styleIDUnresolved {
		id, err := s.file.NewStyle(s.headerStyle.ToExcelizeStyle())
		if err != nil {
			return 0, err
		}
		s.headerStyleID = id
	}
	return s.headerStyleID, nil
}

// resolveDataStyleID returns the cached style ID for dataStyle,
// registering it with the file on first use. Returns 0 (no style) if
// dataStyle is unset.
func (s *xlsxSink) resolveDataStyleID() (int, error) {
	if s.dataStyle == nil {
		return 0, nil
	}
	if s.dataStyleID == styleIDUnresolved {
		id, err := s.file.NewStyle(s.dataStyle.ToExcelizeStyle())
		if err != nil {
			return 0, err
		}
		s.dataStyleID = id
	}
	return s.dataStyleID, nil
}

// resolveColumnStyleID returns the cached style ID for colName's
// column-specific style, registering it with the file on first use. The
// second return value is false if no style is configured for colName.
func (s *xlsxSink) resolveColumnStyleID(colName string) (int, bool, error) {
	style, ok := s.columnStyles[colName]
	if !ok {
		return 0, false, nil
	}
	if id, cached := s.columnStyleIDs[colName]; cached {
		return id, true, nil
	}
	id, err := s.file.NewStyle(style.ToExcelizeStyle())
	if err != nil {
		return 0, false, err
	}
	s.columnStyleIDs[colName] = id
	return id, true, nil
}

// Finalize writes the complete workbook to w.
func (s *xlsxSink) Finalize(_ context.Context, w io.Writer) error {
	if err := s.flushAutoSizeWidths(); err != nil {
		return err
	}
	if err := s.file.Write(w); err != nil {
		return excel.NewExportError("Write", s.sheetName, 0, 0, err)
	}
	return nil
}

// SaveAs writes the workbook directly to the given path, implementing
// excel.FileSaver so ExportToFile can skip buffering the whole document
// before copying it to disk.
func (s *xlsxSink) SaveAs(name string) error {
	if err := s.flushAutoSizeWidths(); err != nil {
		return err
	}
	return s.file.SaveAs(name)
}

// applyDocProperties sets file's OOXML core document properties from
// props, unless props is the zero value (WithProperties was never
// called for this export), in which case excelize's own defaults are
// left untouched.
func applyDocProperties(file *excelize.File, props excel.DocProperties) error { //nolint:gocritic // props is copied once per Export() call, not per row; a pointer would only add an indirection for a value this short-lived
	if props == (excel.DocProperties{}) {
		return nil
	}
	return file.SetDocProps(&excelize.DocProperties{
		Title:       props.Title,
		Subject:     props.Subject,
		Creator:     props.Creator,
		Keywords:    props.Keywords,
		Description: props.Description,
		Category:    props.Category,
		Version:     props.Version,
	})
}

// Close releases the underlying excelize file, unless this sink was
// built by attachXLSXSink onto a workbook the caller owns, in which
// case the caller closes it and this is a no-op.
func (s *xlsxSink) Close() error {
	if !s.ownsFile {
		return nil
	}
	return s.file.Close()
}
