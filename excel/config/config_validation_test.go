package config

import (
	// testing provides assertions for configuration validation rules.
	"testing"
)

// TestDefaultConfigValidate ensures default configuration passes validation.
func TestDefaultConfigValidate(t *testing.T) {
	// Acquire default configuration
	cfg := DefaultConfig()
	// Validate should return nil for sane defaults
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
}

// TestValidateSheetNameEmpty ensures empty sheet name is rejected.
func TestValidateSheetNameEmpty(t *testing.T) {
	// Acquire default configuration
	cfg := DefaultConfig()
	// Force empty sheet name to trigger validation error
	cfg.SheetName = ""
	// Expect an error for empty sheet name
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for empty SheetName")
	}
}

// TestValidateRanges ensures row/column bounds are checked correctly.
func TestValidateRanges(t *testing.T) {
	// StartRow cannot be negative
	cfg := DefaultConfig()
	cfg.StartRow = -1
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for StartRow")
	}
	// EndRow cannot be less than StartRow when specified
	cfg = DefaultConfig()
	cfg.StartRow = 10
	cfg.EndRow = 5
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for EndRow < StartRow")
	}
	// StartCol cannot be negative
	cfg = DefaultConfig()
	cfg.StartCol = -1
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for StartCol")
	}
	// EndCol cannot be less than StartCol when specified
	cfg = DefaultConfig()
	cfg.StartCol = 5
	cfg.EndCol = 3
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error for EndCol < StartCol")
	}
}

// TestValidateStartCell verifies StartCell is validated as an Excel
// cell reference: a malformed value is rejected, a well-formed one and
// the unset default both pass.
func TestValidateStartCell(t *testing.T) {
	cfg := DefaultConfig()
	cfg.StartCell = "not-a-cell"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for malformed StartCell")
	}

	cfg = DefaultConfig()
	cfg.StartCell = "B3"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error for valid StartCell: %v", err)
	}

	cfg = DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error for unset StartCell (default): %v", err)
	}
}
