package importer

import (
	"context"
	"fmt"
	"strings"

	"github.com/bonarizki-dat/go-excel/excel"
)

// Shared import tail: timeout/events, heading/range filtering, per-row events.

// prepareImport applies the configured timeout, dispatches BeforeImport,
// and validates configuration. Shared by the XLSX path (Import) and the
// CSV path (importCSVFromReader in source_csv.go) so both formats run
// the same lifecycle events and validation before diverging on how they
// obtain string rows. The returned cancel func is always safe to defer,
// even when timeout is 0 or an error is returned.
func (i *BasicImporter) prepareImport(ctx context.Context) (context.Context, context.CancelFunc, error) {
	cancel := func() {}
	if i.config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, i.config.Timeout)
	}

	if err := i.dispatcher.Dispatch(ctx, &excel.Event{Type: excel.BeforeImport, Data: i.file}); err != nil {
		return ctx, cancel, err
	}

	if err := i.config.Validate(); err != nil {
		return ctx, cancel, fmt.Errorf("config validation failed: %w", err)
	}

	return ctx, cancel, nil
}

// importStringRows runs the format-agnostic tail of an import: heading
// extraction, row-range filtering, preallocation, per-row processing via
// importRows, error aggregation, and the AfterImport event. XLSX (via
// Import) and CSV (via importCSVFromReader) both reduce to plain string
// rows before reaching this method, which is what lets the two formats
// share every step from here on, including struct mapping in
// ImportToStructs.
func (i *BasicImporter) importStringRows(ctx context.Context, sheetRows [][]string) error {
	if len(sheetRows) == 0 {
		return excel.ErrEmptyData
	}

	sheetRows = i.extractHeadings(sheetRows)
	sheetRows = i.applyRowRange(sheetRows)
	i.totalRows = int64(len(sheetRows))

	if i.config.PreallocateRows && len(sheetRows) > 0 {
		i.rows = make([][]any, 0, len(sheetRows))
	}

	if err := i.importRows(ctx, sheetRows); err != nil {
		return fmt.Errorf("failed to import rows: %w", err)
	}

	if !i.config.StopOnError && len(i.errors) > 0 {
		return fmt.Errorf("import completed with %d errors: first error: %w",
			len(i.errors), i.errors[0])
	}

	return i.dispatcher.Dispatch(ctx, &excel.Event{Type: excel.AfterImport, Data: i.rows})
}

// extractHeadings pulls the first row off sheetRows into i.headings when
// WithHeadings is configured, returning the remaining data rows. It is a
// no-op (returning sheetRows unchanged) when WithHeadings is false.
func (i *BasicImporter) extractHeadings(sheetRows [][]string) [][]string {
	if !i.config.WithHeadings || len(sheetRows) == 0 {
		return sheetRows
	}

	firstRow := sheetRows[0]
	i.headings = make([]string, len(firstRow))
	for j, cell := range firstRow {
		cellValue := cell
		if i.config.TrimSpaces {
			cellValue = strings.TrimSpace(cellValue)
		}
		i.headings[j] = cellValue
	}

	if i.config.Logger != nil {
		i.config.Logger.Debug("Extracted headings",
			"count", len(i.headings),
			"headings", i.headings)
	}

	return sheetRows[1:]
}

// applyRowRange slices sheetRows down to [StartRow, EndRow). Both bounds
// are absolute, 0-indexed positions into sheetRows (the heading row, if
// any, has already been removed by extractHeadings), and EndRow is
// exclusive. Computing both bounds against the original sheetRows length,
// rather than re-slicing between them, keeps EndRow absolute instead of
// accidentally relative to StartRow.
func (i *BasicImporter) applyRowRange(sheetRows [][]string) [][]string {
	rangeEnd := len(sheetRows)
	if i.config.EndRow > 0 && i.config.EndRow < rangeEnd {
		rangeEnd = i.config.EndRow
	}
	rangeStart := i.config.StartRow
	if rangeStart < 0 {
		rangeStart = 0
	}
	if rangeStart > rangeEnd {
		rangeStart = rangeEnd
	}
	return sheetRows[rangeStart:rangeEnd]
}

// importRows processes and imports all rows from the sheet. Extended in
// importer_reader.go with the processRow helper.
func (i *BasicImporter) importRows(ctx context.Context, sheetRows [][]string) error {
	for rowIdx, row := range sheetRows {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		skip, err := i.importRow(ctx, rowIdx, row)
		if err != nil {
			return err
		}
		if skip {
			continue
		}

		// Reporting on every row would dominate runtime for large
		// imports, so progress is only reported every 100 rows.
		if i.config.ProgressCallback != nil && (rowIdx+1)%100 == 0 {
			i.config.ProgressCallback(int64(rowIdx+1), i.totalRows)
		}
	}

	if i.config.ProgressCallback != nil {
		i.config.ProgressCallback(i.totalRows, i.totalRows)
	}

	if i.config.Logger != nil {
		i.config.Logger.Info("Import completed",
			"rows", i.currentRow,
			"errors", len(i.errors))
	}

	return nil
}

// importRow processes a single row: dispatching its BeforeRowImport/
// AfterRowImport events, converting it via processRow, and appending it
// to i.rows unless it is skipped as empty. skip is true when the row was
// handled (either skipped or recorded as a non-fatal error) and the
// caller should move on to the next row without further per-row work;
// err is non-nil only when the caller must abort the whole import.
func (i *BasicImporter) importRow(ctx context.Context, rowIdx int, row []string) (skip bool, err error) {
	if err := i.dispatcher.Dispatch(ctx, &excel.Event{Type: excel.BeforeRowImport, Data: row}); err != nil {
		return false, err
	}

	processedRow, err := i.processRow(ctx, row)
	if err != nil {
		if i.config.StopOnError {
			return false, err
		}

		i.errors = append(i.errors, err)

		if i.config.ErrorCallback != nil && !i.config.ErrorCallback(rowIdx, err) {
			return false, err
		}

		return true, nil
	}

	if i.config.SkipEmptyRows && i.isEmptyRow(processedRow) {
		return true, nil
	}

	i.rows = append(i.rows, processedRow)
	i.currentRow++

	if err := i.dispatcher.Dispatch(ctx, &excel.Event{Type: excel.AfterRowImport, Data: processedRow}); err != nil {
		return false, err
	}

	return false, nil
}
