package config

import (
	"strings"

	"github.com/xuri/excelize/v2"
)

// Validate checks if the configuration is valid, filling in CSV defaults
// for any zero-value fields along the way.
// Returns an error if any configuration value is invalid.
func (c *Config) Validate() error {
	if c.SheetName == "" {
		return NewConfigError("SheetName", c.SheetName, "sheet name cannot be empty")
	}
	if c.StartRow < 0 {
		return NewConfigError("StartRow", c.StartRow, "start row cannot be negative")
	}
	if c.EndRow > 0 && c.EndRow < c.StartRow {
		return NewConfigError("EndRow", c.EndRow, "end row must be >= start row")
	}
	if c.StartCol < 0 {
		return NewConfigError("StartCol", c.StartCol, "start column cannot be negative")
	}
	if c.EndCol > 0 && c.EndCol < c.StartCol {
		return NewConfigError("EndCol", c.EndCol, "end column must be >= start column")
	}
	if c.StartCell != "" {
		if _, _, err := excelize.CellNameToCoordinates(c.StartCell); err != nil {
			return NewConfigError("StartCell", c.StartCell, "invalid cell reference: "+err.Error())
		}
	}

	return c.applyCSVDefaults()
}

// applyCSVDefaults fills zero-value CSV fields with DefaultCSVOptions and
// rejects delimiters that would corrupt the output: a newline or carriage
// return would be indistinguishable from a record separator, and a
// double quote would collide with encoding/csv's own quoting character.
func (c *Config) applyCSVDefaults() error {
	defaults := DefaultCSVOptions()

	if c.CSV.Delimiter == 0 {
		c.CSV.Delimiter = defaults.Delimiter
	}
	if c.CSV.LineEnding == "" {
		c.CSV.LineEnding = defaults.LineEnding
	}
	if c.CSV.TimeLayout == "" {
		c.CSV.TimeLayout = defaults.TimeLayout
	}

	switch c.CSV.Delimiter {
	case '\n', '\r', '"':
		return NewConfigError("CSV.Delimiter", c.CSV.Delimiter,
			"delimiter cannot be a newline, carriage return, or double quote")
	}
	if strings.ContainsAny(c.CSV.LineEnding, "\"") {
		return NewConfigError("CSV.LineEnding", c.CSV.LineEnding,
			"line ending cannot contain a double quote")
	}

	return nil
}

// Clone creates a deep copy of the configuration
// Useful for creating modified copies without affecting the original.
func (c *Config) Clone() *Config {
	clone := &Config{
		SheetName:        c.SheetName,
		WithHeadings:     c.WithHeadings,
		StartRow:         c.StartRow,
		EndRow:           c.EndRow,
		StartCol:         c.StartCol,
		EndCol:           c.EndCol,
		SkipEmptyRows:    c.SkipEmptyRows,
		TrimSpaces:       c.TrimSpaces,
		ValidateData:     c.ValidateData,
		StopOnError:      c.StopOnError,
		Timeout:          c.Timeout,
		PreallocateRows:  c.PreallocateRows,
		Logger:           c.Logger,
		Metrics:          c.Metrics,
		ProgressCallback: c.ProgressCallback,
		ErrorCallback:    c.ErrorCallback,
		Format:           c.Format,
		CSV:              c.CSV,
		Converters:       c.Converters,
		RowMap:           c.RowMap,
		StartCell:        c.StartCell,
		AutoSizeColumns:  c.AutoSizeColumns,
		DocProperties:    c.DocProperties,
		StrictFormat:     c.StrictFormat,
	}

	return clone
}
