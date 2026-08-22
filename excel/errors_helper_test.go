package excel_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestConfigError_Error tests excel.ConfigError error message formatting.
func TestConfigError_Error(t *testing.T) {
	// Create config error
	err := &excel.ConfigError{
		Field:   "ChunkSize",
		Value:   0,
		Message: "chunk size must be positive",
	}

	// Get error message
	msg := err.Error()

	// Verify expected strings are present
	expectedStrings := []string{"config error", "field 'ChunkSize'", "value: 0", "chunk size must be positive"}
	for _, expected := range expectedStrings {
		if !strings.Contains(msg, expected) {
			t.Errorf("error message %q should contain %q", msg, expected)
		}
	}
}

// TestStreamError_Error tests StreamError error message formatting.
func TestStreamError_Error(t *testing.T) {
	tests := []struct {
		name     string             // Test case name
		err      *excel.StreamError // Error to test
		contains []string           // Strings that should be in error message
	}{
		{
			name: "with row number",
			err: &excel.StreamError{
				Op:  "WriteRow",
				Row: 100,
				Err: errors.New("buffer full"),
			},
			contains: []string{"stream WriteRow failed", "row 100", "buffer full"},
		},
		{
			name: "without row number",
			err: &excel.StreamError{
				Op:  "CloseStream",
				Err: errors.New("already closed"),
			},
			contains: []string{"stream CloseStream failed", "already closed"},
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

// TestStreamError_Unwrap tests StreamError unwrapping.
func TestStreamError_Unwrap(t *testing.T) {
	baseErr := errors.New("stream base error")
	streamErr := &excel.StreamError{
		Op:  "WriteRow",
		Err: baseErr,
	}

	// Test Unwrap
	if !errors.Is(streamErr.Unwrap(), baseErr) {
		t.Error("Unwrap() should return the underlying error")
	}
}

func TestNewStreamError(t *testing.T) {
	base := errors.New("write failed")
	err := excel.NewStreamError("WriteRow", 7, base)

	var streamErr *excel.StreamError
	if !errors.As(err, &streamErr) {
		t.Fatalf("NewStreamError type = %T, want *StreamError", err)
	}
	if streamErr.Op != "WriteRow" || streamErr.Row != 7 {
		t.Errorf("NewStreamError fields = %+v", streamErr)
	}
	if !errors.Is(err, base) {
		t.Error("NewStreamError should wrap the cause")
	}
}

// TestErrorHelperFunctions tests all error constructor helper functions
// Note: Helper functions are not exported, so we create errors manually.
func TestErrorHelperFunctions(t *testing.T) {
	// Test ExportError creation
	exportErr := &excel.ExportError{
		Op:    "Export",
		Sheet: "Sheet1",
		Row:   10,
		Col:   5,
		Err:   excel.ErrEmptyData,
	}
	wantExport := fmt.Sprintf("Export [sheet:Sheet1 row:10 col:5]: %v", excel.ErrEmptyData)
	if got := exportErr.Error(); got != wantExport {
		t.Errorf("ExportError.Error() = %q, want %q", got, wantExport)
	}
	if !errors.Is(exportErr, excel.ErrEmptyData) {
		t.Error("ExportError should wrap ErrEmptyData")
	}

	// Test ImportError creation
	readFailed := errors.New("read failed")
	importErr := &excel.ImportError{
		Op:    "Import",
		Sheet: "Sheet2",
		Row:   20,
		Col:   3,
		Err:   readFailed,
	}
	wantImport := fmt.Sprintf("Import [sheet:Sheet2 row:20 col:3]: %v", readFailed)
	if got := importErr.Error(); got != wantImport {
		t.Errorf("ImportError.Error() = %q, want %q", got, wantImport)
	}
	if !errors.Is(importErr, readFailed) {
		t.Error("ImportError should wrap its underlying error")
	}

	// Test ValidationError creation
	valErr := &excel.ValidationError{
		Row:     5,
		Field:   "email",
		Value:   "invalid",
		Message: "invalid email format",
	}
	wantVal := "validation failed at row 5, field 'email' (value: invalid): invalid email format"
	if got := valErr.Error(); got != wantVal {
		t.Errorf("ValidationError.Error() = %q, want %q", got, wantVal)
	}

	// Test ConfigError creation
	cfgErr := &excel.ConfigError{
		Field:   "ChunkSize",
		Value:   0,
		Message: "must be positive",
	}
	wantCfg := "config error: field 'ChunkSize' (value: 0): must be positive"
	if got := cfgErr.Error(); got != wantCfg {
		t.Errorf("ConfigError.Error() = %q, want %q", got, wantCfg)
	}

	// Test StreamError creation
	streamErr := &excel.StreamError{
		Op:  "WriteRow",
		Row: 100,
		Err: errors.New("buffer full"),
	}
	wantStream := "stream WriteRow failed at row 100: buffer full"
	if got := streamErr.Error(); got != wantStream {
		t.Errorf("StreamError.Error() = %q, want %q", got, wantStream)
	}
}

// TestErrorsAs tests that errors.As works correctly with custom error types.
func TestErrorsAs(t *testing.T) {
	// Create an excel.ExportError
	err := &excel.ExportError{
		Op:    "Export",
		Sheet: "Sheet1",
		Row:   10,
		Col:   5,
		Err:   excel.ErrEmptyData,
	}

	// Test errors.As with excel.ExportError
	var exportErr *excel.ExportError
	if !errors.As(err, &exportErr) {
		t.Error("errors.As should find *excel.ExportError")
	}
	if exportErr.Op != "Export" {
		t.Errorf("excel.ExportError.Op = %v, want Export", exportErr.Op)
	}

	// Create an excel.ImportError
	importErrInstance := &excel.ImportError{
		Op:    "Import",
		Sheet: "Sheet2",
		Row:   20,
		Col:   3,
		Err:   errors.New("read failed"),
	}

	// Test errors.As with excel.ImportError
	var importErr *excel.ImportError
	if !errors.As(importErrInstance, &importErr) {
		t.Error("errors.As should find *excel.ImportError")
	}
	if importErr.Op != "Import" {
		t.Errorf("excel.ImportError.Op = %v, want Import", importErr.Op)
	}
}

// TestSentinelErrors tests that all sentinel errors are defined.
func TestSentinelErrors(t *testing.T) {
	// List of all sentinel errors that should exist
	sentinelErrors := []error{
		excel.ErrInvalidFormat,
		excel.ErrEmptyData,
		excel.ErrInvalidSheetName,
		excel.ErrSheetNotFound,
		excel.ErrFileNotFound,
		excel.ErrInvalidFile,
		excel.ErrTypeMismatch,
	}

	// Verify each sentinel error is not nil
	for _, err := range sentinelErrors {
		if err == nil {
			t.Error("Sentinel error should not be nil")
		}
		// Verify error message is not empty
		if err.Error() == "" {
			t.Error("Sentinel error message should not be empty")
		}
	}
}
