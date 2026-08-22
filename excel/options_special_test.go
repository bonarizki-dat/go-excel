package excel_test

import (
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestWithProgress verifies progress callback is set correctly.
func TestWithProgress(t *testing.T) {
	// Track if callback was set
	callbackSet := false
	// Create progress callback
	progressCallback := func(_, _ int64) {
		callbackSet = true
	}

	// Apply option
	cfg := config.DefaultConfig()
	excel.WithProgress(progressCallback)(cfg)

	// Verify callback is set
	if cfg.ProgressCallback == nil {
		t.Fatal("ProgressCallback should not be nil")
	}

	// Test callback works
	cfg.ProgressCallback(10, 100)
	if !callbackSet {
		t.Error("Progress callback was not called")
	}
}

// TestWithErrorHandler verifies error callback is set correctly.
func TestWithErrorHandler(t *testing.T) {
	// Track if callback was set
	callbackCalled := false
	// Create error callback
	errorCallback := func(_ int, _ error) bool {
		callbackCalled = true
		return true // Continue on error
	}

	// Apply option
	cfg := config.DefaultConfig()
	excel.WithErrorHandler(errorCallback)(cfg)

	// Verify callback is set
	if cfg.ErrorCallback == nil {
		t.Fatal("ErrorCallback should not be nil")
	}

	// Test callback works
	result := cfg.ErrorCallback(1, nil)
	if !callbackCalled {
		t.Error("Error callback was not called")
	}
	if !result {
		t.Error("Error callback should return true")
	}
}

// TestWithConfig verifies entire config replacement works.
func TestWithConfig(t *testing.T) {
	// Create custom config
	customConfig := &config.Config{
		SheetName:   "CustomSheet",
		StartRow:    5,
		StopOnError: false,
	}

	// Apply WithConfig option
	cfg := config.DefaultConfig()
	excel.WithConfig(customConfig)(cfg)

	// Verify all fields were copied
	if cfg.SheetName != "CustomSheet" {
		t.Errorf("SheetName = %v, want CustomSheet", cfg.SheetName)
	}
	if cfg.StartRow != 5 {
		t.Errorf("StartRow = %v, want 5", cfg.StartRow)
	}
	if cfg.StopOnError {
		t.Error("StopOnError = true, want false")
	}
}

// TestWithConfig_Nil verifies nil config is handled gracefully.
func TestWithConfig_Nil(t *testing.T) {
	// Apply excel.WithConfig with nil
	cfg := config.DefaultConfig()
	originalSheetName := cfg.SheetName
	excel.WithConfig(nil)(cfg)

	// Verify config unchanged
	if cfg.SheetName != originalSheetName {
		t.Error("Config should not change when excel.WithConfig(nil) is applied")
	}
}

// TestMultipleOptions verifies multiple options can be combined.
func TestMultipleOptions(t *testing.T) {
	// Start with default config
	cfg := config.DefaultConfig()

	// Apply multiple options
	options := []excel.Option{
		excel.WithSheetName("MultiTest"),
		excel.WithRows(3, 10),
		excel.WithStopOnError(false),
		excel.WithValidation(true),
	}

	// Apply all options
	for _, opt := range options {
		opt(cfg)
	}

	// Verify all options were applied
	if cfg.SheetName != "MultiTest" {
		t.Errorf("SheetName = %v, want MultiTest", cfg.SheetName)
	}
	if cfg.StartRow != 2 || cfg.EndRow != 10 {
		t.Errorf("row range = [%v,%v), want [2,10)", cfg.StartRow, cfg.EndRow)
	}
	if cfg.StopOnError {
		t.Error("StopOnError should be false")
	}
	if !cfg.ValidateData {
		t.Error("ValidateData should be true")
	}
}

// TestOptionsOverride verifies later options override earlier ones.
func TestOptionsOverride(t *testing.T) {
	// Start with default config
	cfg := config.DefaultConfig()

	// Apply conflicting options
	excel.WithSheetName("First")(cfg)
	excel.WithSheetName("Second")(cfg)
	excel.WithSheetName("Third")(cfg)

	// Last one should win
	if cfg.SheetName != "Third" {
		t.Errorf("SheetName = %v, want Third (last option)", cfg.SheetName)
	}
}
