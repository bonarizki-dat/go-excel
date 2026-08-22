// This file implements SheetCollection for managing multiple sheets.
// See doc.go for the package godoc.

package excel

import (
	"fmt"
)

// SheetCollection manages multiple sheets for export/import operations.
//
// Provides validation, ordering, and access to sheets by name or index.
// Ensures sheet names are unique and valid according to Excel rules.
type SheetCollection struct {
	// sheets stores all sheets in order
	sheets []*Sheet

	// nameIndex maps sheet names to sheet pointers for fast lookup and
	// uniqueness checks.
	nameIndex map[string]*Sheet
}

// NewSheetCollection creates a new empty sheet collection.
//
// Returns a collection ready to accept sheets via Add method.
func NewSheetCollection() *SheetCollection {
	return &SheetCollection{
		// Capacity for the common case of 3-5 sheets.
		sheets:    make([]*Sheet, 0, 5),
		nameIndex: make(map[string]*Sheet),
	}
}

// Add appends sheet to the collection, setting its Index field based on
// insertion order. It returns an error, without modifying the
// collection, if sheet.Name is empty, exceeds 31 characters, contains
// characters Excel disallows in sheet names, or duplicates a name
// already present in the collection.
func (sc *SheetCollection) Add(sheet *Sheet) error {
	if err := validateSheetName(sheet.Name); err != nil {
		return fmt.Errorf("invalid sheet name: %w", err)
	}

	if _, exists := sc.nameIndex[sheet.Name]; exists {
		return fmt.Errorf("%w: sheet name '%s' already exists", ErrInvalidSheetName, sheet.Name)
	}

	sheet.Index = len(sc.sheets)
	sc.sheets = append(sc.sheets, sheet)
	sc.nameIndex[sheet.Name] = sheet

	return nil
}

// Get returns the sheet named name, or nil if no sheet with that name
// exists.
func (sc *SheetCollection) Get(name string) *Sheet {
	return sc.nameIndex[name]
}

// GetByIndex returns the sheet at the given 0-based index, or nil if
// index is out of bounds.
func (sc *SheetCollection) GetByIndex(index int) *Sheet {
	if index < 0 || index >= len(sc.sheets) {
		return nil
	}
	return sc.sheets[index]
}

// Exists reports whether a sheet named name exists in the collection.
func (sc *SheetCollection) Exists(name string) bool {
	_, exists := sc.nameIndex[name]
	return exists
}

// Count returns the number of sheets in the collection.
func (sc *SheetCollection) Count() int {
	return len(sc.sheets)
}

// All returns a copy of the collection's sheets, ordered by their Index
// field. Callers may freely modify the returned slice without affecting
// the collection.
func (sc *SheetCollection) All() []*Sheet {
	result := make([]*Sheet, len(sc.sheets))
	copy(result, sc.sheets)
	return result
}

// Names returns the collection's sheet names in the order the sheets
// will appear in the exported Excel file.
func (sc *SheetCollection) Names() []string {
	names := make([]string, len(sc.sheets))
	for i, sheet := range sc.sheets {
		names[i] = sheet.Name
	}
	return names
}

// Clear resets the collection to empty, discarding all sheets, so it
// can be reused as if newly created by NewSheetCollection.
func (sc *SheetCollection) Clear() {
	sc.sheets = sc.sheets[:0]
	// Reallocate rather than clearing in place so the old entries'
	// memory can be released instead of merely unreferenced.
	sc.nameIndex = make(map[string]*Sheet)
}
