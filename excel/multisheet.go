// This file implements multi-sheet support for exporting and importing
// multiple sheets in a single Excel file. See doc.go for the package
// godoc.

package excel

import (
	"context"
	"fmt"
	"slices"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// Sheet represents a single worksheet in an Excel file.
//
// Each sheet can have its own data, configuration, and styling.
// Sheets can be used with both BasicExporter and StreamExporter.
//
// Example:
//
//	sheet := excel.Sheet{
//	    Name: "Users",
//	    Data: userData,
//	    Config: userConfig,
//	}
type Sheet struct {
	// Name is the worksheet name. Excel limits it to 31 characters and
	// disallows the characters \ / ? * [ ]; see validateSheetName.
	Name string

	// Data holds the rows to export, one []any per row. For large
	// datasets, set DataChannel instead to stream rows without holding
	// them all in memory at once.
	Data [][]any

	// DataChannel is an optional streaming alternative to Data, for
	// large datasets that should not be loaded into memory all at
	// once. If both are set, DataChannel takes precedence.
	DataChannel <-chan []any

	// Config holds sheet-specific configuration, overriding the parent
	// exporter or importer's config for this sheet only. If nil, the
	// parent config applies unchanged.
	Config *config.Config

	// Index is the sheet's 0-based position; sheets with a lower Index
	// appear first in the Excel file. Set automatically when the sheet
	// is added to a SheetCollection.
	Index int
}

// MultiSheetExportOptions holds options for multi-sheet export.
//
// Allows customization of multi-sheet export behavior.
// Can be passed to ExportToSheets method.
type MultiSheetExportOptions struct {
	// CreateDefaultSheet, if true, creates a single default sheet when
	// no sheets are provided instead of returning an error. Defaults
	// to false.
	CreateDefaultSheet bool

	// DefaultSheetName names the sheet created when CreateDefaultSheet
	// is true. Defaults to "Sheet1"; unused otherwise.
	DefaultSheetName string

	// SkipEmptySheets, if true, omits sheets with no data from the
	// output instead of creating them empty. Defaults to false.
	SkipEmptySheets bool

	// ContinueOnError, if true, keeps exporting remaining sheets after
	// one sheet fails instead of stopping immediately. Defaults to
	// false.
	ContinueOnError bool

	// ProgressCallback, if non-nil, is invoked with a sheet's index
	// and name after that sheet has been exported, for progress
	// tracking.
	ProgressCallback func(ctx context.Context, index int, name string) error
}

// DefaultMultiSheetExportOptions returns the default
// MultiSheetExportOptions: no default sheet creation, no empty-sheet
// skipping, and stop-on-first-error behavior.
func DefaultMultiSheetExportOptions() MultiSheetExportOptions {
	return MultiSheetExportOptions{
		CreateDefaultSheet: false,
		DefaultSheetName:   "Sheet1",
		SkipEmptySheets:    false,
		ContinueOnError:    false,
		ProgressCallback:   nil,
	}
}

// validateSheetName returns an error wrapping ErrInvalidSheetName if
// name is empty, exceeds Excel's 31-character sheet name limit,
// contains any of the characters \ / ? * [ ], or starts or ends with
// an apostrophe (which Excel reserves for quoting sheet names in
// formulas). It returns nil if name is valid.
func validateSheetName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: sheet name cannot be empty", ErrInvalidSheetName)
	}

	if len(name) > 31 {
		return fmt.Errorf("%w: sheet name too long (max 31 characters): %d", ErrInvalidSheetName, len(name))
	}

	invalidChars := []rune{'\\', '/', '?', '*', '[', ']'}
	for _, char := range name {
		if slices.Contains(invalidChars, char) {
			return fmt.Errorf("%w: sheet name contains invalid character: %c", ErrInvalidSheetName, char)
		}
	}

	// Excel uses apostrophes for quoting sheet names in formulas, so a
	// leading/trailing one would make the name ambiguous to parse back.
	if name[0] == '\'' || name[len(name)-1] == '\'' {
		return fmt.Errorf("%w: sheet name cannot start or end with apostrophe", ErrInvalidSheetName)
	}

	return nil
}
