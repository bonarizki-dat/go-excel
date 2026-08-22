package importer

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// ImportFromFile imports data from a file with the given filename.
// This is a convenience method that opens the file and calls Import().
// The format is derived from filename's extension: ".csv" reads CSV,
// anything else (including no extension) reads XLSX via excelize.
// Returns error if file cannot be opened or import fails.
func (i *BasicImporter) ImportFromFile(ctx context.Context, filename string) error {
	if filename == "" {
		return excel.NewImportError("ImportFromFile", i.config.SheetName, 0, 0,
			fmt.Errorf("filename cannot be empty"))
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if i.config.Logger != nil {
		i.config.Logger.Info("Starting import from file",
			"filename", filename,
			"sheet", i.config.SheetName)
	}

	var startTime time.Time
	if i.config.Metrics != nil {
		startTime = time.Now()
	}

	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return excel.NewImportError("Stat", i.config.SheetName, 0, 0,
				fmt.Errorf("%w: %s", excel.ErrFileNotFound, filename))
		}
		return excel.NewImportError("Stat", i.config.SheetName, 0, 0, err)
	}

	// Open and import according to the format the extension implies.
	// Each helper mirrors the historical error shape: the "open the
	// source" step (OpenFile/os.Open) returns its *excel.ImportError
	// as-is, while a failure from the import pipeline beyond that point
	// is wrapped with "import failed: %w" - callers that type-assert on
	// *excel.ImportError directly, instead of using errors.As, depend on
	// that distinction.
	var err error
	if excel.FormatFromExtension(filename) == excel.FileFormatCSV {
		err = i.importCSVFile(ctx, filename)
	} else {
		err = i.importXLSXFile(ctx, filename)
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
		i.config.Logger.Info("Import from file completed successfully",
			"filename", filename,
			"rows", i.currentRow)
	}

	return nil
}

// importXLSXFile opens filename with excelize and imports it via
// Import(). OpenFile failures are returned as-is, an *excel.ImportError;
// a failure from Import() itself is wrapped with "import failed: %w".
func (i *BasicImporter) importXLSXFile(ctx context.Context, filename string) error {
	file, err := excelize.OpenFile(filename)
	if err != nil {
		if i.config.Logger != nil {
			i.config.Logger.Error("Failed to open file", "error", err)
		}
		return excel.NewImportError("OpenFile", i.config.SheetName, 0, 0, err)
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

// importCSVFile opens filename and imports it as CSV via
// importCSVFromReader. The os.Open failure is returned as-is; a failure
// from importCSVFromReader itself is wrapped with "import failed: %w",
// mirroring importXLSXFile's OpenFile/Import() split.
func (i *BasicImporter) importCSVFile(ctx context.Context, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return excel.NewImportError("Open", i.config.SheetName, 0, 0, err)
	}
	defer func() {
		_ = f.Close()
	}()

	if err := i.importCSVFromReader(ctx, f); err != nil {
		if i.config.Logger != nil {
			i.config.Logger.Error("Import failed", "error", err)
		}
		return fmt.Errorf("import failed: %w", err)
	}
	return nil
}
