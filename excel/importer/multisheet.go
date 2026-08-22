package importer

import (
	"context"
	"fmt"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// ImportFromSheet imports data from the worksheet named sheetName in
// filename, instead of the importer's configured default sheet. It
// returns excel.ErrSheetNotFound if no sheet by that name exists.
//
// Example:
//
//	importer := importer.NewImporter()
//	rows, err := importer.ImportFromSheet(ctx, "report.xlsx", "Users")
func (i *BasicImporter) ImportFromSheet(ctx context.Context, filename, sheetName string) ([][]any, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	if sheetName == "" {
		return nil, fmt.Errorf("%w: sheet name cannot be empty", excel.ErrInvalidSheetName)
	}

	file, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, excel.NewImportError("OpenFile", sheetName, 0, 0, err)
	}

	i.file = file
	i.shouldCloseFile = true

	sheetIndex, err := file.GetSheetIndex(sheetName)
	if err != nil || sheetIndex < 0 {
		return nil, fmt.Errorf("%w: '%s'", excel.ErrSheetNotFound, sheetName)
	}

	// Temporarily override the configured sheet name so Import()
	// targets sheetName instead of i.config.SheetName.
	originalSheetName := i.config.SheetName
	i.config.SheetName = sheetName

	if err := i.Import(ctx); err != nil {
		i.config.SheetName = originalSheetName
		return nil, fmt.Errorf("failed to import from sheet '%s': %w", sheetName, err)
	}

	i.config.SheetName = originalSheetName

	// Return a copy of the imported rows; see GetRows for why.
	return i.GetRows(), nil
}

// ImportAllSheets imports every worksheet in filename, returning a map
// from sheet name to that sheet's data rows. For a CSV file, which has
// no concept of multiple sheets, it returns a single-entry map keyed by
// the importer's configured sheet name.
//
// Example:
//
//	importer := importer.NewImporter()
//	sheetsData, err := importer.ImportAllSheets(ctx, "report.xlsx")
//	usersData := sheetsData["Users"]
//	ordersData := sheetsData["Orders"]
func (i *BasicImporter) ImportAllSheets(ctx context.Context, filename string) (map[string][][]any, error) {
	if filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	// CSV has no concept of multiple sheets, so it degrades to a single
	// logical sheet keyed by config.SheetName instead of failing with an
	// excelize format error.
	if excel.FormatFromExtension(filename) == excel.FileFormatCSV {
		return i.importAllSheetsCSV(ctx, filename)
	}

	file, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, excel.NewImportError("OpenFile", "", 0, 0, err)
	}

	i.file = file
	i.shouldCloseFile = true

	sheetList := file.GetSheetList()
	if len(sheetList) == 0 {
		return nil, fmt.Errorf("no sheets found in file")
	}

	result := make(map[string][][]any, len(sheetList))
	originalSheetName := i.config.SheetName

	for _, sheetName := range sheetList {
		select {
		case <-ctx.Done():
			i.config.SheetName = originalSheetName
			return nil, ctx.Err()
		default:
		}

		i.config.SheetName = sheetName

		// Reset per-sheet state so a subsequent Import() call starts
		// clean instead of appending to the previous sheet's rows.
		i.rows = make([][]any, 0)
		i.headings = make([]string, 0)
		i.currentRow = 0
		i.totalRows = 0

		if err := i.Import(ctx); err != nil {
			i.config.SheetName = originalSheetName
			return nil, fmt.Errorf("failed to import sheet '%s': %w", sheetName, err)
		}

		// Copy out of i.rows since the next iteration's reset above
		// would otherwise invalidate a slice this map still holds.
		sheetData := make([][]any, len(i.rows))
		copy(sheetData, i.rows)
		result[sheetName] = sheetData
	}

	i.config.SheetName = originalSheetName

	return result, nil
}

// importAllSheetsCSV is the CSV counterpart to ImportAllSheets: a CSV
// file only ever has one logical sheet, so it imports filename normally
// and returns a single-entry map keyed by config.SheetName, warning once
// that no further sheets exist to import.
func (i *BasicImporter) importAllSheetsCSV(ctx context.Context, filename string) (map[string][][]any, error) {
	if i.config.Logger != nil {
		i.config.Logger.Warn("CSV has no concept of multiple sheets; importing as a single logical sheet",
			"sheet", i.config.SheetName)
	}

	if err := i.ImportFromFile(ctx, filename); err != nil {
		return nil, err
	}

	return map[string][][]any{i.config.SheetName: i.GetRows()}, nil
}

// GetSheetNames returns the names of all sheets in the currently
// opened file. A file must already be open via ImportFromFile or
// ImportFromSheet, or this returns an error.
//
// Example:
//
//	importer := importer.NewImporter()
//	_, _ = importer.ImportFromSheet(ctx, "report.xlsx", "Users")
//	sheetNames, err := importer.GetSheetNames()
func (i *BasicImporter) GetSheetNames() ([]string, error) {
	// CSV has exactly one logical sheet, named after config.SheetName,
	// and never sets i.file, so it is handled before the nil check below.
	if i.format == excel.FileFormatCSV {
		return []string{i.config.SheetName}, nil
	}

	if i.file == nil {
		return nil, fmt.Errorf("no file opened")
	}

	return i.file.GetSheetList(), nil
}

// GetSheetCount returns the number of sheets in the currently opened
// file, or an error if no file is opened.
func (i *BasicImporter) GetSheetCount() (int, error) {
	// CSV always has exactly one logical sheet.
	if i.format == excel.FileFormatCSV {
		return 1, nil
	}

	if i.file == nil {
		return 0, fmt.Errorf("no file opened")
	}

	return len(i.file.GetSheetList()), nil
}

// SheetExists reports whether a sheet named sheetName exists in the
// currently opened file, or returns an error if no file is opened.
func (i *BasicImporter) SheetExists(sheetName string) (bool, error) {
	// CSV's single logical sheet is named after config.SheetName; there
	// is no excelize file to consult GetSheetIndex against.
	if i.format == excel.FileFormatCSV {
		return sheetName == i.config.SheetName, nil
	}

	if i.file == nil {
		return false, fmt.Errorf("no file opened")
	}

	// GetSheetIndex only errors on an invalid sheetName (empty, too long,
	// disallowed characters); it returns (-1, nil) for a syntactically
	// valid name that simply is not present. Propagate that error instead
	// of reporting a plain "not found" false, since the two cases mean
	// different things to a caller.
	sheetIndex, err := i.file.GetSheetIndex(sheetName)
	if err != nil {
		return false, fmt.Errorf("%w: check sheet %q: %w", excel.ErrInvalidSheetName, sheetName, err)
	}

	return sheetIndex >= 0, nil
}
