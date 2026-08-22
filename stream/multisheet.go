package stream

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/xuri/excelize/v2"
)

// ExportToSheets writes multiple sheets to the io.Writer this exporter was
// constructed with, giving each sheet its own excelize StreamWriter so rows
// can be consumed from per-sheet channels instead of collected up front.
//
// This is a separate write path from ExportStream, which has two
// consequences worth knowing before choosing it: the row transformer set by
// WithRowTransform does not run, and neither does the worker pool, so
// WithExportWorkers has no effect here. Rows are written in the order they arrive
// on each sheet's channel.
//
// For FileFormatCSV the call degrades to writing only the first sheet, with
// one warning through the configured logger, because CSV has no concept of
// multiple sheets.
//
// Example:
//
//	userChan := make(chan []interface{}, 100)
//	orderChan := make(chan []interface{}, 100)
//
//	sheets := []excel.Sheet{
//	    {Name: "Users", DataChannel: userChan},
//	    {Name: "Orders", DataChannel: orderChan},
//	}
//
//	exporter, err := stream.NewStreamExporter(writer)
//	if err != nil {
//		log.Fatal(err)
//	}
//	err = exporter.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())
func (se *StreamExporter) ExportToSheets(ctx context.Context, sheets []excel.Sheet, options excel.MultiSheetExportOptions) error {
	sheets, err := resolveStreamSheets(sheets, options)
	if err != nil {
		return err
	}

	if err := validateSheets(sheets); err != nil {
		return fmt.Errorf("sheet validation failed: %w", err)
	}

	// CSV has no concept of multiple sheets, so it degrades to writing
	// only the first sheet instead of failing outright.
	if se.format == config.FileFormatCSV {
		return se.exportToSheetsCSV(ctx, sheets, options)
	}

	if err := se.initWorkbook(sheets[0].Name); err != nil {
		return err
	}

	var exportErrors []error
	if err := se.exportSheetsLoop(ctx, sheets, options, &exportErrors); err != nil {
		return err
	}

	if err := se.file.Write(se.writer); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if options.ContinueOnError && len(exportErrors) > 0 {
		return fmt.Errorf("export completed with %d errors: first error: %w",
			len(exportErrors), exportErrors[0])
	}

	return nil
}

// resolveStreamSheets returns sheets unchanged unless it is empty, in
// which case it either synthesizes a single default sheet (when
// options.CreateDefaultSheet is set) or reports an error.
func resolveStreamSheets(sheets []excel.Sheet, options excel.MultiSheetExportOptions) ([]excel.Sheet, error) {
	if len(sheets) > 0 {
		return sheets, nil
	}
	if !options.CreateDefaultSheet {
		return nil, fmt.Errorf("no sheets provided for export")
	}
	return []excel.Sheet{
		{
			Name:        options.DefaultSheetName,
			DataChannel: nil,
		},
	}, nil
}

// initWorkbook replaces se.file with a fresh excelize workbook and
// renames its default "Sheet1" to firstSheetName. NewStreamExporter
// already created se.file, so the old one is closed first to avoid
// leaking its resources.
func (se *StreamExporter) initWorkbook(firstSheetName string) error {
	if se.file != nil {
		_ = se.file.Close()
	}
	se.file = excelize.NewFile()
	if se.file == nil {
		return fmt.Errorf("failed to create excelize file")
	}

	defaultSheet := se.file.GetSheetName(0)
	if defaultSheet != "" && defaultSheet != firstSheetName {
		if err := se.file.SetSheetName(defaultSheet, firstSheetName); err != nil {
			return fmt.Errorf("failed to rename default sheet: %w", err)
		}
	}
	return nil
}

// exportSheetsLoop exports each sheet in turn, honoring
// options.SkipEmptySheets, options.ContinueOnError (appending to
// exportErrors instead of stopping), and options.ProgressCallback.
func (se *StreamExporter) exportSheetsLoop(ctx context.Context, sheets []excel.Sheet, options excel.MultiSheetExportOptions, exportErrors *[]error) error {
	for i, sheet := range sheets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if options.SkipEmptySheets && sheet.DataChannel == nil && len(sheet.Data) == 0 {
			continue
		}

		if err := se.exportSheetStream(ctx, &sheet, i); err != nil {
			if options.ContinueOnError {
				*exportErrors = append(*exportErrors, fmt.Errorf("sheet '%s': %w", sheet.Name, err))
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

	return nil
}

// exportToSheetsCSV degrades ExportToSheets for FileFormatCSV: CSV has
// no concept of multiple sheets, so only the first sheet is streamed,
// through the same ExportStream path a single-sheet CSV export uses.
// With WithExportStrictFormat(true), a second sheet fails the call
// with an error wrapping excel.ErrStrictFormatUnsupported instead.
// Otherwise, a warning is emitted once per call, not once per dropped
// sheet, and only when a sheet was actually dropped.
func (se *StreamExporter) exportToSheetsCSV(ctx context.Context, sheets []excel.Sheet, options excel.MultiSheetExportOptions) error {
	if len(sheets) > 1 {
		if se.strictFormat {
			return fmt.Errorf("%w: CSV export supports only one sheet, got %d", excel.ErrStrictFormatUnsupported, len(sheets))
		}
		if se.logger != nil {
			se.logger.Warn("CSV export supports only one sheet; writing the first sheet and dropping the rest",
				"written", sheets[0].Name, "dropped", len(sheets)-1)
		}
	}

	first := sheets[0]

	rows := first.DataChannel
	if rows == nil {
		ch := make(chan []any, len(first.Data))
		for _, row := range first.Data {
			ch <- row
		}
		close(ch)
		rows = ch
	}

	if err := se.ExportStream(ctx, rows); err != nil {
		return fmt.Errorf("failed to export sheet '%s': %w", first.Name, err)
	}

	if options.ProgressCallback != nil {
		if err := options.ProgressCallback(ctx, 0, first.Name); err != nil {
			return fmt.Errorf("progress callback failed for sheet '%s': %w", first.Name, err)
		}
	}

	return nil
}

// exportSheetStream streams sheet's rows into the workbook at position
// index, creating a new worksheet for it unless index is 0 (the first
// sheet reuses the workbook's already-renamed default sheet).
func (se *StreamExporter) exportSheetStream(ctx context.Context, sheet *excel.Sheet, index int) error {
	// The first sheet reuses the workbook's default sheet (already
	// renamed by the caller); later sheets need to be created.
	if index > 0 {
		if _, err := se.file.NewSheet(sheet.Name); err != nil {
			return fmt.Errorf("failed to create sheet '%s': %w", sheet.Name, err)
		}
	}

	streamWriter, err := se.file.NewStreamWriter(sheet.Name)
	if err != nil {
		return fmt.Errorf("failed to create stream writer for sheet '%s': %w", sheet.Name, err)
	}

	se.streamWriter = streamWriter
	se.sheetName = sheet.Name
	atomic.StoreUint64(&se.rowCounter, 0)

	if sheet.DataChannel != nil {
		if err := se.streamFromChannel(ctx, sheet.DataChannel); err != nil {
			// Best-effort flush of whatever was buffered; the original
			// error is what matters here, so a flush failure is ignored.
			_ = streamWriter.Flush()
			return fmt.Errorf("failed to stream from channel: %w", err)
		}
	} else if sheet.Data != nil {
		if err := se.streamFromStaticData(ctx, sheet.Data); err != nil {
			_ = streamWriter.Flush()
			return fmt.Errorf("failed to stream from static data: %w", err)
		}
	}

	if err := streamWriter.Flush(); err != nil {
		return fmt.Errorf("failed to flush stream writer: %w", err)
	}

	return nil
}

// validateSheets checks sheet names for Excel-rule validity and
// uniqueness by feeding them through excel.SheetCollection, which
// performs both checks as a side effect of Add.
func validateSheets(sheets []excel.Sheet) error {
	collection := excel.NewSheetCollection()
	for i := range sheets {
		if err := collection.Add(&sheets[i]); err != nil {
			return err
		}
	}
	return nil
}
