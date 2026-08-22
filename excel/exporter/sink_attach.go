package exporter

import (
	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// attachXLSXSink builds an xlsxSink that writes into file, a workbook
// the caller already owns, instead of creating one of its own.
// ExportToSheets uses this to keep BasicExporter.file as the single
// source of truth for the workbook across every sheet, so the sink's
// style-ID caches stay valid from one sheet to the next instead of
// being rebuilt per sheet. headerStyle, dataStyle, columnStyles, and
// columnWidths are referenced, not copied, matching newXLSXSink.
// startCell is an Excel cell reference such as "B3", or "" for the A1
// default; it must already be valid (Config.Validate checks this
// before a sink is built), since a malformed reference here would
// otherwise surface as a confusing excelize error deep inside WriteRow.
// Call setSheet before the first WriteRow to select a target sheet.
// autoSize mirrors Config.AutoSizeColumns.
func attachXLSXSink(file *excelize.File, headerStyle, dataStyle *excel.Style, columnStyles map[string]*excel.Style, columnWidths map[string]float64, startCell string, autoSize bool) (*xlsxSink, error) {
	colOffset, rowOffset := 0, 0
	if startCell != "" {
		col, row, err := excelize.CellNameToCoordinates(startCell)
		if err != nil {
			return nil, excel.NewConfigError("StartCell", startCell, "invalid cell reference: "+err.Error())
		}
		colOffset, rowOffset = col-1, row-1
	}

	return &xlsxSink{
		file:              file,
		headerStyle:       headerStyle,
		dataStyle:         dataStyle,
		columnStyles:      columnStyles,
		columnWidths:      columnWidths,
		colOffset:         colOffset,
		rowOffset:         rowOffset,
		autoSize:          autoSize,
		measuredColWidths: make(map[string]int),
		headerStyleID:     styleIDUnresolved,
		dataStyleID:       styleIDUnresolved,
		columnStyleIDs:    make(map[string]int),
	}, nil
}

// setSheet retargets the sink to write into sheetName within the same
// workbook and applies columnWidths to it. It does not create sheetName
// or change the workbook's active sheet; ExportToSheets already
// handles both before calling setSheet. Auto-size measurements taken
// against the sink's previous sheet (if any) are flushed first, so they
// land on that sheet instead of being attributed to sheetName.
func (s *xlsxSink) setSheet(sheetName string) error {
	if err := s.flushAutoSizeWidths(); err != nil {
		return err
	}
	s.sheetName = sheetName
	for col, width := range s.columnWidths {
		if err := s.file.SetColWidth(sheetName, col, col, width); err != nil {
			return excel.NewExportError("SetColWidth", sheetName, 0, 0, err)
		}
	}
	return nil
}
