package excel

import (
	"fmt"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// ExportError represents an error that occurred during export operations
// Provides detailed context about where the error occurred.
type ExportError struct {
	// Op is the operation that failed (e.g., "Export", "WriteCell")
	Op string

	// Sheet is the name of the sheet where the error occurred
	Sheet string

	// Row is the row number where the error occurred (0-indexed)
	Row int

	// Col is the column number where the error occurred (0-indexed)
	Col int

	// Err is the underlying error that caused this error
	Err error
}

// Error implements the error interface
// Returns a formatted error message with context.
func (e *ExportError) Error() string {
	if e.Sheet != "" && e.Row > 0 && e.Col > 0 {
		return fmt.Sprintf("%s [sheet:%s row:%d col:%d]: %v", e.Op, e.Sheet, e.Row, e.Col, e.Err)
	}
	if e.Sheet != "" && e.Row > 0 {
		return fmt.Sprintf("%s [sheet:%s row:%d]: %v", e.Op, e.Sheet, e.Row, e.Err)
	}
	if e.Sheet != "" {
		return fmt.Sprintf("%s [sheet:%s]: %v", e.Op, e.Sheet, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
// Allows errors.Is and errors.As to work correctly.
func (e *ExportError) Unwrap() error {
	return e.Err
}

// ImportError represents an error that occurred during import operations
// Provides detailed context about where the error occurred.
type ImportError struct {
	// Op is the operation that failed (e.g., "Import", "ReadCell")
	Op string

	// Sheet is the name of the sheet where the error occurred
	Sheet string

	// Row is the row number where the error occurred (0-indexed)
	Row int

	// Col is the column number where the error occurred (0-indexed)
	Col int

	// Err is the underlying error that caused this error
	Err error
}

// Error implements the error interface
// Returns a formatted error message with context.
func (e *ImportError) Error() string {
	if e.Sheet != "" && e.Row > 0 && e.Col > 0 {
		return fmt.Sprintf("%s [sheet:%s row:%d col:%d]: %v", e.Op, e.Sheet, e.Row, e.Col, e.Err)
	}
	if e.Sheet != "" && e.Row > 0 {
		return fmt.Sprintf("%s [sheet:%s row:%d]: %v", e.Op, e.Sheet, e.Row, e.Err)
	}
	if e.Sheet != "" {
		return fmt.Sprintf("%s [sheet:%s]: %v", e.Op, e.Sheet, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
// Allows errors.Is and errors.As to work correctly.
func (e *ImportError) Unwrap() error {
	return e.Err
}

// ValidationError represents a validation error for a specific row
// Used when data validation fails during import/export.
type ValidationError struct {
	// Row is the row number where validation failed (0-indexed)
	Row int

	// Field is the field name that failed validation
	Field string

	// Value is the invalid value
	Value any

	// Message is the validation error message
	Message string

	// Err is the underlying error (if any)
	Err error
}

// Error implements the error interface
// Returns a formatted validation error message.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed at row %d, field '%s' (value: %v): %s",
			e.Row, e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("validation failed at row %d: %s", e.Row, e.Message)
}

// Unwrap returns the underlying error
// Allows errors.Is and errors.As to work correctly.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// ConfigError is an alias for config.ConfigError, kept here so callers
// that only import excel (not excel/config directly) can still use
// errors.As(err, &excel.ConfigError{}) regardless of which package's
// constructor produced the error: both excel.NewConfigError and
// config.NewConfigError return the same underlying type.
type ConfigError = config.ConfigError

// StreamError represents an error that occurred during streaming operations
// Used for errors in stream package.
type StreamError struct {
	// Op is the operation that failed (e.g., "WriteRow", "ReadRow")
	Op string

	// Row is the row number where the error occurred (0-indexed)
	Row int

	// Err is the underlying error
	Err error
}

// Error implements the error interface
// Returns a formatted streaming error message.
func (e *StreamError) Error() string {
	if e.Row > 0 {
		return fmt.Sprintf("stream %s failed at row %d: %v", e.Op, e.Row, e.Err)
	}
	return fmt.Sprintf("stream %s failed: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error
// Allows errors.Is and errors.As to work correctly.
func (e *StreamError) Unwrap() error {
	return e.Err
}

// NewExportError creates a new ExportError with the given details.
// Exported for use by subpackages.
func NewExportError(op, sheet string, row, col int, err error) error {
	return &ExportError{
		Op:    op,
		Sheet: sheet,
		Row:   row,
		Col:   col,
		Err:   err,
	}
}

// NewImportError creates a new ImportError with the given details.
// Exported for use by subpackages.
func NewImportError(op, sheet string, row, col int, err error) error {
	return &ImportError{
		Op:    op,
		Sheet: sheet,
		Row:   row,
		Col:   col,
		Err:   err,
	}
}

// NewValidationError creates a new ValidationError with the given
// details. cause is the underlying error that triggered the validation
// failure, if any (e.g. a type conversion error); pass nil when the
// message is self-contained and there is no wrapped cause. Exported for
// use by subpackages.
func NewValidationError(row int, field string, value any, message string, cause error) error {
	return &ValidationError{
		Row:     row,
		Field:   field,
		Value:   value,
		Message: message,
		Err:     cause,
	}
}

// NewConfigError creates a new ConfigError. Delegates to
// config.NewConfigError so both packages' constructors produce values
// of the exact same underlying type.
func NewConfigError(field string, value any, message string) error {
	return config.NewConfigError(field, value, message)
}

// NewStreamError creates a new StreamError with the given details.
// Exported for use by subpackages.
func NewStreamError(op string, row int, err error) error {
	return &StreamError{
		Op:  op,
		Row: row,
		Err: err,
	}
}
