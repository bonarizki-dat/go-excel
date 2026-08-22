package importer

import (
	"context"
	"io"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/csvio"
)

// importCSVFromReader decodes r as CSV and runs the result through
// importStringRows, the same pipeline an XLSX import uses once rows are
// plain strings. This is the CSV counterpart to Import() as a whole
// (open/decode plus pipeline): it does not call Import() itself because
// Import() requires an excelize file, which a CSV source never has.
// Errors are returned as-is, unwrapped; callers wrap them with
// "import failed: %w" the same way they wrap a failing Import() call.
func (i *BasicImporter) importCSVFromReader(ctx context.Context, r io.Reader) error {
	ctx, cancel, err := i.prepareImport(ctx)
	defer cancel()
	if err != nil {
		return err
	}

	i.format = excel.FileFormatCSV

	if i.config.Logger != nil {
		i.config.Logger.Info("Starting CSV import", "sheet", i.config.SheetName)
	}

	rows, err := csvio.ReadAll(r, i.config.CSV)
	if err != nil {
		return excel.NewImportError("ReadAll", i.config.SheetName, 0, 0, err)
	}

	return i.importStringRows(ctx, rows)
}
