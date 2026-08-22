package stream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
)

// readRows reads rows from si.source and sends them to rowChan,
// running in its own goroutine to decouple reading from processing.
// si.source was already created in NewStreamImporter and, if
// withHeadings is set, already had its first row consumed by
// readHeadings, so this only ever sees data rows. It returns an error
// if a row fails to decode or ctx is canceled.
func (si *StreamImporter) readRows(ctx context.Context, rowChan chan<- []any) error {
	startRow := si.startRow

	// rowNum counts the heading row as row 1, matching the row numbers
	// callers see for XLSX. The heading row itself was already consumed
	// by readHeadings and will not appear again here, so rowNum starts
	// pre-advanced past it instead of being recounted.
	rowNum := 0
	if si.withHeadings {
		rowNum = 1
	}

	for {
		row, err := si.source.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("read row %d: %w", rowNum+1, err)
		}

		rowNum++

		if rowNum < startRow {
			continue
		}

		if si.endRow > 0 && rowNum > si.endRow {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if si.skipEmptyRows && si.isRowEmpty(row) {
			continue
		}

		if err := si.sendRow(ctx, rowChan, row); err != nil {
			return err
		}
	}

	return nil
}

// sendRow copies row into a []any buffer (reused from si.poolManager
// when possible) and sends it on rowChan, then reports progress.
//
// rowData is not returned to the pool once it reaches the processor,
// since the processor may retain it; see the taskProcessor comment in
// importer_import.go.
func (si *StreamImporter) sendRow(ctx context.Context, rowChan chan<- []any, row []string) error {
	rowData := si.poolManager.GetRow()

	// Buffer too small for this row (rare with proper pool
	// configuration): return it to the pool before replacing it,
	// otherwise it is silently dropped instead of recycled.
	if cap(rowData) < len(row) {
		si.poolManager.PutRow(rowData)
		rowData = make([]any, 0, len(row))
	}

	rowData = rowData[:len(row)]
	for i, cell := range row {
		rowData[i] = cell
	}

	select {
	case rowChan <- rowData:
	case <-ctx.Done():
		return ctx.Err()
	}

	count := atomic.AddUint64(&si.rowCounter, 1)
	if si.progressCallback != nil && count%si.progressInterval == 0 {
		si.progressCallback(count, fmt.Sprintf("Imported %d rows", count))
	}

	return nil
}

// readHeadings reads the column headers from the first row of
// si.source into si.headings, or returns an error if the read fails.
func (si *StreamImporter) readHeadings() error {
	row, err := si.source.next()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("read headings: %w", err)
	}
	si.headings = row
	return nil
}

// isRowEmpty reports whether every cell in row is an empty string.
func (si *StreamImporter) isRowEmpty(row []string) bool {
	for _, cell := range row {
		if cell != "" {
			return false
		}
	}
	return true
}
