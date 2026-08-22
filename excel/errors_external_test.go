package excel_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestExportError_Error tests excel.ExportError error message formatting.
func TestExportError_Error(t *testing.T) {
	tests := []struct {
		name     string             // Test case name
		err      *excel.ExportError // Error to test
		contains []string           // Strings that should be in error message
	}{
		{
			name: "full context",
			err: &excel.ExportError{
				Op:    "WriteCell",
				Sheet: "Sheet1",
				Row:   10,
				Col:   5,
				Err:   excel.ErrInvalidFormat,
			},
			contains: []string{"WriteCell", "Sheet1", "row:10", "col:5", "invalid file format"},
		},
		{
			name: "sheet and row only",
			err: &excel.ExportError{
				Op:    "Export",
				Sheet: "Sheet1",
				Row:   10,
				Err:   excel.ErrEmptyData,
			},
			contains: []string{"Export", "Sheet1", "row:10", "empty data"},
		},
		{
			name: "sheet only",
			err: &excel.ExportError{
				Op:    "Export",
				Sheet: "Sheet1",
				Err:   excel.ErrInvalidSheetName,
			},
			contains: []string{"Export", "Sheet1", "invalid sheet name"},
		},
		{
			name: "operation and error only",
			err: &excel.ExportError{
				Op:  "Export",
				Err: excel.ErrEmptyData,
			},
			contains: []string{"Export", "empty data"},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get error message
			msg := tt.err.Error()
			// Verify all expected strings are present
			for _, expected := range tt.contains {
				if !strings.Contains(msg, expected) {
					t.Errorf("error message %q should contain %q", msg, expected)
				}
			}
		})
	}
}

// TestExportError_Unwrap tests excel.ExportError unwrapping.
func TestExportError_Unwrap(t *testing.T) {
	// Create base error
	baseErr := errors.New("base error")
	// Wrap in excel.ExportError
	exportErr := &excel.ExportError{
		Op:  "Export",
		Err: baseErr,
	}

	// Unwrap should return the base error
	if !errors.Is(exportErr.Unwrap(), baseErr) {
		t.Error("Unwrap() should return the underlying error")
	}

	// errors.Is should work
	if !errors.Is(exportErr, baseErr) {
		t.Error("errors.Is() should find wrapped error")
	}
}

// TestImportError_Error tests excel.ImportError error message formatting.
func TestImportError_Error(t *testing.T) {
	tests := []struct {
		name     string             // Test case name
		err      *excel.ImportError // Error to test
		contains []string           // Strings that should be in error message
	}{
		{
			name: "full context",
			err: &excel.ImportError{
				Op:    "ReadCell",
				Sheet: "Data",
				Row:   20,
				Col:   3,
				Err:   excel.ErrInvalidFile,
			},
			contains: []string{"ReadCell", "Data", "row:20", "col:3", "invalid file"},
		},
		{
			name: "sheet and row only",
			err: &excel.ImportError{
				Op:    "Import",
				Sheet: "Data",
				Row:   5,
				Err:   excel.ErrSheetNotFound,
			},
			contains: []string{"Import", "Data", "row:5", "sheet not found"},
		},
		{
			name: "sheet only",
			err: &excel.ImportError{
				Op:    "Import",
				Sheet: "Data",
				Err:   excel.ErrInvalidSheetName,
			},
			contains: []string{"Import", "Data", "invalid sheet name"},
		},
		{
			name: "operation and error only",
			err: &excel.ImportError{
				Op:  "Import",
				Err: excel.ErrEmptyData,
			},
			contains: []string{"Import", "empty data"},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(msg, expected) {
					t.Errorf("error message %q should contain %q", msg, expected)
				}
			}
		})
	}
}

// TestImportError_Unwrap tests excel.ImportError unwrapping.
func TestImportError_Unwrap(t *testing.T) {
	baseErr := errors.New("import base error")
	importErr := &excel.ImportError{
		Op:  "Import",
		Err: baseErr,
	}

	// Test Unwrap
	if !errors.Is(importErr.Unwrap(), baseErr) {
		t.Error("Unwrap() should return the underlying error")
	}

	// Test errors.Is
	if !errors.Is(importErr, baseErr) {
		t.Error("errors.Is() should find wrapped error")
	}
}

// TestValidationError_Error tests ValidationError error message formatting.
func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string                 // Test case name
		err      *excel.ValidationError // Error to test
		contains []string               // Strings that should be in error message
	}{
		{
			name: "with field and value",
			err: &excel.ValidationError{
				Row:     5,
				Field:   "age",
				Value:   -1,
				Message: "age must be positive",
			},
			contains: []string{"row 5", "field 'age'", "value: -1", "age must be positive"},
		},
		{
			name: "without field",
			err: &excel.ValidationError{
				Row:     10,
				Message: "invalid data format",
			},
			contains: []string{"row 10", "invalid data format"},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			for _, expected := range tt.contains {
				if !strings.Contains(msg, expected) {
					t.Errorf("error message %q should contain %q", msg, expected)
				}
			}
		})
	}
}

// TestValidationError_Unwrap tests ValidationError unwrapping.
func TestValidationError_Unwrap(t *testing.T) {
	baseErr := errors.New("validation base error")
	valErr := &excel.ValidationError{
		Row: 1,
		Err: baseErr,
	}

	// Test Unwrap
	if !errors.Is(valErr.Unwrap(), baseErr) {
		t.Error("Unwrap() should return the underlying error")
	}
}
