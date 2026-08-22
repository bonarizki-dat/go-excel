package importer

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// ImportFromReader and the XLSX reader path.

// ImportFromReader imports data from an io.Reader.
// This method makes the library framework-agnostic:
// - Gin: importer.ImportFromReader(ctx, c.Request.Body)
// - Echo: importer.ImportFromReader(ctx, c.Request().Body)
// - Fiber: importer.ImportFromReader(ctx, c.Request().BodyStream())
// - stdlib: importer.ImportFromReader(ctx, r.Body)
// There is no filename here for the format to be inferred from, so it
// uses whatever excel.WithFormat set on the importer (default:
// FileFormatXLSX).
// Returns error if reading fails or import fails.
func (i *BasicImporter) ImportFromReader(ctx context.Context, r io.Reader) error {
	if r == nil {
		return excel.NewImportError("ImportFromReader", i.config.SheetName, 0, 0,
			fmt.Errorf("reader cannot be nil"))
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if i.config.Logger != nil {
		i.config.Logger.Info("Starting import from reader",
			"sheet", i.config.SheetName)
	}

	var startTime time.Time
	if i.config.Metrics != nil {
		startTime = time.Now()
	}

	// Same error-shape split as ImportFromFile: importXLSXFromReader
	// keeps OpenReader failures raw and wraps only the Import()
	// failure; the CSV branch wraps the whole importCSVFromReader
	// call, since a reader has no separate "open" step to keep raw.
	var err error
	if i.config.Format == excel.FileFormatCSV {
		if err = i.importCSVFromReader(ctx, r); err != nil {
			if i.config.Logger != nil {
				i.config.Logger.Error("Import failed", "error", err)
			}
			err = fmt.Errorf("import failed: %w", err)
		}
	} else {
		err = i.importXLSXFromReader(ctx, r)
	}
	if err != nil {
		return err
	}

	if i.config.Metrics != nil {
		duration := time.Since(startTime).Milliseconds()
		i.config.Metrics.Timing("import.duration", duration)
		i.config.Metrics.Inc("import.rows", int64(i.currentRow))
		i.config.Metrics.Inc("import.success", 1)
	}

	if i.config.Logger != nil {
		i.config.Logger.Info("Import from reader completed successfully",
			"rows", i.currentRow)
	}

	return nil
}

// importXLSXFromReader opens r with excelize.OpenReader, which reads
// the entire input into memory, and imports it via Import(). OpenReader
// failures are returned as-is, an *excel.ImportError; a failure from
// Import() itself is wrapped with "import failed: %w".
func (i *BasicImporter) importXLSXFromReader(ctx context.Context, r io.Reader) error {
	file, err := excelize.OpenReader(r)
	if err != nil {
		if i.config.Logger != nil {
			i.config.Logger.Error("Failed to open reader", "error", err)
		}
		return excel.NewImportError("OpenReader", i.config.SheetName, 0, 0, err)
	}

	i.file = file
	i.shouldCloseFile = true
	i.format = excel.FileFormatXLSX

	if err := i.Import(ctx); err != nil {
		if i.config.Logger != nil {
			i.config.Logger.Error("Import failed", "error", err)
		}
		return fmt.Errorf("import failed: %w", err)
	}
	return nil
}
