package exporter

import (
	"context"
	"fmt"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/xuri/excelize/v2"
)

// ExportToSheets writes sheets to a single Excel file, each with its
// own data and an optional Config that overrides the exporter's own
// config for that sheet only. For XLSX, header/data/column styles and
// column widths set via SetHeaderStyle/SetDataStyle/SetColumnStyle/
// SetColumnWidth apply to every sheet, the same as Export. Unlike
// Export, this path does not dispatch BeforeRowExport/AfterRowExport
// (or BeforeExport/AfterExport, since it is not Export) for any sheet;
// use MultiSheetExportOptions.ProgressCallback or a sheet's own
// Config.ProgressCallback to observe progress instead. The CSV path
// routes back through Export, which does dispatch those events for the
// single sheet it writes.
//
// Example:
//
//	sheets := []excel.Sheet{
//	    {Name: "Users", Data: userData, Config: userConfig},
//	    {Name: "Orders", Data: orderData, Config: orderConfig},
//	}
//	err := exporter.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())
func (e *BasicExporter) ExportToSheets(ctx context.Context, sheets []excel.Sheet, options excel.MultiSheetExportOptions) error {
	sheets, err := resolveSheets(sheets, options)
	if err != nil {
		return err
	}

	if err := e.validateSheets(sheets); err != nil {
		return fmt.Errorf("sheet validation failed: %w", err)
	}

	// CSV has no concept of multiple sheets, so it degrades to writing
	// only the first sheet instead of failing outright.
	if e.config.Format == config.FileFormatCSV {
		return e.exportToSheetsCSV(ctx, sheets, options)
	}

	if err := e.initWorkbook(sheets[0].Name); err != nil {
		return err
	}

	if err := e.exportSheetsLoop(ctx, sheets, options); err != nil {
		return err
	}

	// exportSheet's setSheet call only flushes auto-size measurements
	// for the sheet being left behind, so the last sheet written still
	// needs an explicit flush here.
	if xs, ok := e.sink.(*xlsxSink); ok {
		if err := xs.flushAutoSizeWidths(); err != nil {
			return err
		}
	}
	return nil
}

// resolveSheets returns sheets unchanged unless it is empty, in which
// case it either synthesizes a single default sheet (when
// options.CreateDefaultSheet is set) or reports an error.
func resolveSheets(sheets []excel.Sheet, options excel.MultiSheetExportOptions) ([]excel.Sheet, error) {
	if len(sheets) > 0 {
		return sheets, nil
	}
	if !options.CreateDefaultSheet {
		return nil, fmt.Errorf("no sheets provided for export")
	}
	return []excel.Sheet{
		{
			Name:   options.DefaultSheetName,
			Data:   [][]any{},
			Config: nil,
		},
	}, nil
}

// initWorkbook creates e.file, renames excelize's default "Sheet1" to
// firstSheetName instead of deleting it and creating a new one, and
// builds the xlsxSink that exportSheet retargets to each sheet in turn
// via setSheet. It closes and discards whatever the previous
// Export()-family call on this exporter left behind first, the same
// leak prevention Export() applies to its own sink, since ExportToSheets
// assigns e.file/e.sink directly rather than going through Export.
func (e *BasicExporter) initWorkbook(firstSheetName string) error {
	if err := e.closeCurrent(); err != nil {
		return fmt.Errorf("close previous export file: %w", err)
	}
	e.sink = nil
	e.file = nil

	e.file = excelize.NewFile()
	if e.file == nil {
		return fmt.Errorf("failed to create excelize file")
	}
	if err := applyDocProperties(e.file, e.config.DocProperties); err != nil {
		return excel.NewExportError("SetDocProps", firstSheetName, 0, 0, err)
	}

	defaultSheet := e.file.GetSheetName(0)
	if defaultSheet != "" && defaultSheet != firstSheetName {
		if err := e.file.SetSheetName(defaultSheet, firstSheetName); err != nil {
			return excel.NewExportError("SetSheetName", firstSheetName, 0, 0, err)
		}
	}

	sink, err := attachXLSXSink(e.file, e.headerStyle, e.dataStyle, e.columnStyles, e.columnWidths, e.config.StartCell, e.config.AutoSizeColumns)
	if err != nil {
		return err
	}
	e.sink = sink
	return nil
}

// exportSheetsLoop exports each sheet in turn, honoring
// options.SkipEmptySheets, options.ContinueOnError, and
// options.ProgressCallback.
func (e *BasicExporter) exportSheetsLoop(ctx context.Context, sheets []excel.Sheet, options excel.MultiSheetExportOptions) error {
	var exportErrors []error

	for i, sheet := range sheets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if options.SkipEmptySheets && len(sheet.Data) == 0 {
			continue
		}

		if err := e.exportSheet(ctx, &sheet, i); err != nil {
			if options.ContinueOnError {
				exportErrors = append(exportErrors, fmt.Errorf("sheet '%s': %w", sheet.Name, err))
				continue
			}
			return fmt.Errorf("failed to export sheet '%s': %w", sheet.Name, err)
		}

		if options.ProgressCallback != nil {
			if err := options.ProgressCallback(ctx, i, sheet.Name); err != nil {
				return fmt.Errorf("progress callback failed for sheet '%s': %w", sheet.Name, err)
			}
		}
	}

	if options.ContinueOnError && len(exportErrors) > 0 {
		return fmt.Errorf("export completed with %d errors: first error: %w",
			len(exportErrors), exportErrors[0])
	}

	return nil
}

// exportToSheetsCSV degrades ExportToSheets for FileFormatCSV: CSV has
// no concept of multiple sheets, so only the first sheet is written,
// matching Maatwebsite and PhpSpreadsheet's behavior for the same
// situation. With WithStrictFormat(true), a second sheet fails the
// call with an error wrapping excel.ErrStrictFormatUnsupported instead.
// Otherwise, a warning is emitted once per call, not once per dropped
// sheet, and only when a sheet was actually dropped.
func (e *BasicExporter) exportToSheetsCSV(ctx context.Context, sheets []excel.Sheet, options excel.MultiSheetExportOptions) error {
	if len(sheets) > 1 {
		if e.config.StrictFormat {
			return fmt.Errorf("%w: CSV export supports only one sheet, got %d", excel.ErrStrictFormatUnsupported, len(sheets))
		}
		if e.config.Logger != nil {
			e.config.Logger.Warn("CSV export supports only one sheet; writing the first sheet and dropping the rest",
				"written", sheets[0].Name, "dropped", len(sheets)-1)
		}
	}

	first := sheets[0]

	var rows [][]any
	if first.DataChannel != nil {
		var err error
		rows, err = e.readFromChannel(ctx, first.DataChannel)
		if err != nil {
			return fmt.Errorf("read sheet '%s' data channel: %w", first.Name, err)
		}
	} else {
		rows = first.Data
	}

	// Export() reads e.config directly, so a per-sheet config override
	// is applied by temporarily swapping it, the same trick exportSheet
	// uses for the XLSX multi-sheet path.
	originalConfig := e.config
	if first.Config != nil {
		if err := first.Config.Validate(); err != nil {
			return fmt.Errorf("sheet config validation failed: %w", err)
		}
		e.config = first.Config
	}
	defer func() { e.config = originalConfig }()

	e.data = rows
	if err := e.Export(ctx); err != nil {
		return fmt.Errorf("failed to export sheet '%s': %w", first.Name, err)
	}

	if options.ProgressCallback != nil {
		if err := options.ProgressCallback(ctx, 0, first.Name); err != nil {
			return fmt.Errorf("progress callback failed for sheet '%s': %w", first.Name, err)
		}
	}

	return nil
}

// exportSheet writes sheet's rows into the workbook at position index,
// creating a new worksheet for it unless index is 0 (the first sheet
// reuses the workbook's already-renamed default sheet), then retargets
// e.sink to that sheet via setSheet before writing through it.
func (e *BasicExporter) exportSheet(ctx context.Context, sheet *excel.Sheet, index int) error {
	// The first sheet reuses the workbook's default sheet (already
	// renamed by ExportToSheets); later sheets need to be created.
	if index > 0 {
		sheetIndex, err := e.file.NewSheet(sheet.Name)
		if err != nil {
			return excel.NewExportError("NewSheet", sheet.Name, 0, 0, err)
		}
		e.file.SetActiveSheet(sheetIndex)
	}

	// initWorkbook always builds e.sink as an xlsxSink for the non-CSV
	// path exportSheet is only reachable from, so this assertion cannot
	// fail in practice; it guards against a future change to initWorkbook
	// silently breaking that invariant instead of panicking here.
	sink, ok := e.sink.(*xlsxSink)
	if !ok {
		return fmt.Errorf("XLSX multi-sheet export requires an xlsxSink, got %T", e.sink)
	}
	if err := sink.setSheet(sheet.Name); err != nil {
		return err
	}

	sheetConfig := e.config
	if sheet.Config != nil {
		sheetConfig = sheet.Config
		if err := sheetConfig.Validate(); err != nil {
			return fmt.Errorf("sheet config validation failed: %w", err)
		}
	}

	var rows [][]any
	if sheet.DataChannel != nil {
		var err error
		rows, err = e.readFromChannel(ctx, sheet.DataChannel)
		if err != nil {
			return fmt.Errorf("read sheet '%s' data channel: %w", sheet.Name, err)
		}
	} else {
		rows = sheet.Data
	}

	if err := e.exportRowsToSheet(ctx, sink, rows, sheetConfig); err != nil {
		return fmt.Errorf("failed to export rows: %w", err)
	}

	return nil
}

// readFromChannel reads and returns all rows sent on dataChan until it
// closes, letting a sheet's data stream in without being loaded into
// memory ahead of time by the caller. It returns ctx.Err() if the
// context is canceled before the channel closes, so a caller cannot
// mistake a truncated read for a complete one; the error is always nil
// once the channel closes normally.
func (e *BasicExporter) readFromChannel(ctx context.Context, dataChan <-chan []any) ([][]any, error) {
	var rows [][]any

	for {
		select {
		case <-ctx.Done():
			// Context canceled before the channel closed: rows read so
			// far are incomplete, so the caller must treat this as a
			// failure rather than silently exporting a partial sheet.
			return rows, ctx.Err()
		case row, ok := <-dataChan:
			if !ok {
				return rows, nil
			}
			rows = append(rows, row)
		}
	}
}

// validateSheets returns an error if any sheet in sheets has an
// invalid or duplicate name, per the rules enforced by
// excel.SheetCollection.Add.
func (e *BasicExporter) validateSheets(sheets []excel.Sheet) error {
	collection := excel.NewSheetCollection()
	for i := range sheets {
		if err := collection.Add(&sheets[i]); err != nil {
			return err
		}
	}
	return nil
}
