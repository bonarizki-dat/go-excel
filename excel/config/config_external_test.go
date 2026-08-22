package config_test

import (
	"testing"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestDefaultConfig verifies that config.DefaultConfig returns correct default values
// All defaults should match the documented values in config.go.
func TestDefaultConfig(t *testing.T) {
	// Get default configuration
	cfg := config.DefaultConfig()

	// Verify sheet configuration defaults
	if cfg.SheetName != "Sheet1" {
		t.Errorf("SheetName = %v, want %v", cfg.SheetName, "Sheet1")
	}

	// Verify data handling defaults
	if !cfg.WithHeadings {
		t.Error("WithHeadings = false, want true")
	}
	if cfg.StartRow != 0 {
		t.Errorf("StartRow = %v, want %v", cfg.StartRow, 0)
	}
	if cfg.EndRow != 0 {
		t.Errorf("EndRow = %v, want %v", cfg.EndRow, 0)
	}
	if cfg.StartCol != 0 {
		t.Errorf("StartCol = %v, want %v", cfg.StartCol, 0)
	}
	if cfg.EndCol != 0 {
		t.Errorf("EndCol = %v, want %v", cfg.EndCol, 0)
	}
	if !cfg.SkipEmptyRows {
		t.Error("SkipEmptyRows = false, want true")
	}
	if !cfg.TrimSpaces {
		t.Error("TrimSpaces = false, want true")
	}

	// Verify validation and error handling defaults
	if cfg.ValidateData {
		t.Error("ValidateData = true, want false")
	}
	if !cfg.StopOnError {
		t.Error("StopOnError = false, want true")
	}
	if cfg.Timeout != 0 {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 0)
	}

	// Verify memory management defaults
	if !cfg.PreallocateRows {
		t.Error("PreallocateRows = false, want true")
	}

	// Verify observability defaults (all should be nil)
	if cfg.Logger != nil {
		t.Error("Logger should be nil by default")
	}
	if cfg.Metrics != nil {
		t.Error("Metrics should be nil by default")
	}
	if cfg.ProgressCallback != nil {
		t.Error("ProgressCallback should be nil by default")
	}
	if cfg.ErrorCallback != nil {
		t.Error("ErrorCallback should be nil by default")
	}
}

// TestConfigValidate_Valid tests validation with valid configurations
// All valid configs should pass validation without errors.
func TestConfigValidate_Valid(t *testing.T) {
	// Test default config is valid
	cfg := config.DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("config.DefaultConfig validation failed: %v", err)
	}

	// Test custom valid config
	customCfg := &config.Config{
		SheetName: "CustomSheet",
		StartRow:  0,
		EndRow:    1000,
		StartCol:  0,
		EndCol:    10,
	}
	if err := customCfg.Validate(); err != nil {
		t.Errorf("Custom config validation failed: %v", err)
	}
}
