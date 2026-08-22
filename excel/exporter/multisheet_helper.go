package exporter

import (
	"context"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// exportRowsToSheet writes rows through sink, applying cfg's heading,
// error-handling, and progress-callback behavior. It is the multi-sheet
// counterpart to exportRows, which writes through e.sink for the
// exporter's single active sheet instead of taking a RowSink argument;
// this path never dispatches BeforeRowExport/AfterRowExport the way
// exportRows does, matching ExportToSheets' documented contract.
func (e *BasicExporter) exportRowsToSheet(ctx context.Context, sink excel.RowSink, rows [][]any, cfg *config.Config) error {
	totalRows := int64(len(rows))
	currentRow := 0

	if cfg.WithHeadings && len(rows) > 0 {
		if err := sink.WriteRow(ctx, 1, rows[0], true); err != nil {
			return err
		}
		currentRow = 1
		rows = rows[1:]
	}

	for i, row := range rows {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rowNum := currentRow + i + 1

		if err := sink.WriteRow(ctx, rowNum, row, false); err != nil {
			if cfg.StopOnError {
				return err
			}
			e.errors = append(e.errors, err)

			if cfg.ErrorCallback != nil && !cfg.ErrorCallback(rowNum-1, err) {
				return err
			}
		}

		if cfg.ProgressCallback != nil && (i+1)%100 == 0 {
			cfg.ProgressCallback(int64(i+1), totalRows)
		}
	}

	if cfg.ProgressCallback != nil {
		cfg.ProgressCallback(totalRows, totalRows)
	}

	return nil
}
