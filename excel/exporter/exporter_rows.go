package exporter

import (
	"context"
	"fmt"

	"github.com/bonarizki-dat/go-excel/excel"
)

// Row-write loop for BasicExporter: events, progress, and StopOnError
// handling against the current sink.

// exportRows writes the prepared data rows to the Excel sheet.
func (e *BasicExporter) exportRows(ctx context.Context, rows [][]any) error {
	e.totalRows = int64(len(rows))

	if e.config.WithHeadings && len(rows) > 0 {
		if err := e.writeHeaderRow(ctx, rows[0]); err != nil {
			return err
		}
		rows = rows[1:]
	}

	for i, row := range rows {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rowNum := e.currentRow + i + 1

		if e.config.RowMap != nil {
			mapped, err := e.config.RowMap(row)
			if err != nil {
				return fmt.Errorf("row %d: RowMap: %w", rowNum, err)
			}
			row = mapped
		}

		if err := e.writeDataRow(ctx, rowNum, row); err != nil {
			return err
		}

		if e.config.ProgressCallback != nil && (i+1)%100 == 0 {
			e.config.ProgressCallback(int64(i+1), e.totalRows)
		}
	}

	if e.config.ProgressCallback != nil {
		e.config.ProgressCallback(e.totalRows, e.totalRows)
	}

	e.currentRow += len(rows)

	return nil
}

// writeHeaderRow dispatches BeforeRowExport/AfterRowExport around writing
// headerRow as Excel row 1, and advances e.currentRow past it.
func (e *BasicExporter) writeHeaderRow(ctx context.Context, headerRow []any) error {
	if err := e.dispatcher.Dispatch(ctx, &excel.Event{Type: excel.BeforeRowExport, Data: headerRow}); err != nil {
		return err
	}

	if err := e.sink.WriteRow(ctx, 1, headerRow, true); err != nil {
		return err
	}

	if err := e.dispatcher.Dispatch(ctx, &excel.Event{Type: excel.AfterRowExport, Data: headerRow}); err != nil {
		return err
	}

	e.currentRow = 1
	return nil
}

// writeDataRow dispatches BeforeRowExport/AfterRowExport around writing a
// single non-header row at rowNum, applying StopOnError/ErrorCallback
// handling around a write failure the same way importRow does for
// imports.
func (e *BasicExporter) writeDataRow(ctx context.Context, rowNum int, row []any) error {
	if err := e.dispatcher.Dispatch(ctx, &excel.Event{Type: excel.BeforeRowExport, Data: row}); err != nil {
		return err
	}

	if err := e.sink.WriteRow(ctx, rowNum, row, false); err != nil {
		if e.config.StopOnError {
			return err
		}
		e.errors = append(e.errors, err)

		if e.config.ErrorCallback != nil && !e.config.ErrorCallback(rowNum-1, err) {
			return err
		}
	}

	return e.dispatcher.Dispatch(ctx, &excel.Event{Type: excel.AfterRowExport, Data: row})
}
