package stream

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/xuri/excelize/v2"
)

// streamFromChannel writes each row received on dataChan to the
// current sheet via se.streamWriter, until the channel closes or ctx
// is canceled.
func (se *StreamExporter) streamFromChannel(ctx context.Context, dataChan <-chan []any) error {
	rowNum := 1

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case row, ok := <-dataChan:
			if !ok {
				return nil
			}

			cells := make([]any, len(row))
			for i, val := range row {
				cells[i] = excelize.Cell{Value: val}
			}

			cellAddr, err := excelize.CoordinatesToCellName(1, rowNum)
			if err != nil {
				return fmt.Errorf("failed to get cell address for row %d: %w", rowNum, err)
			}

			if err := se.streamWriter.SetRow(cellAddr, cells); err != nil {
				return fmt.Errorf("failed to write row %d: %w", rowNum, err)
			}

			rowNum++
			count := atomic.AddUint64(&se.rowCounter, 1)

			// Uses the value AddUint64 just returned rather than
			// re-reading se.rowCounter, which could have already been
			// changed by another writer by the time it is read back.
			if se.progressCallback != nil && count%se.progressInterval == 0 {
				se.progressCallback(count, fmt.Sprintf("Exported %d rows to sheet %s", count, se.sheetName))
			}
		}
	}
}

// streamFromStaticData writes each row of data to the current sheet
// via se.streamWriter, stopping early if ctx is canceled.
func (se *StreamExporter) streamFromStaticData(ctx context.Context, data [][]any) error {
	for rowNum, row := range data {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		cells := make([]any, len(row))
		for i, val := range row {
			cells[i] = excelize.Cell{Value: val}
		}

		cellAddr, err := excelize.CoordinatesToCellName(1, rowNum+1)
		if err != nil {
			return fmt.Errorf("failed to get cell address for row %d: %w", rowNum+1, err)
		}

		if err := se.streamWriter.SetRow(cellAddr, cells); err != nil {
			return fmt.Errorf("failed to write row %d: %w", rowNum+1, err)
		}

		count := atomic.AddUint64(&se.rowCounter, 1)

		// Uses the value AddUint64 just returned rather than
		// re-reading se.rowCounter, which could have already been
		// changed by another writer by the time it is read back.
		if se.progressCallback != nil && count%se.progressInterval == 0 {
			se.progressCallback(count, fmt.Sprintf("Exported %d rows to sheet %s", count, se.sheetName))
		}
	}

	return nil
}
