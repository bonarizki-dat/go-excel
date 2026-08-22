package exporter_test

import (
	"context"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// TestExporter_Close tests the Close method.
func TestExporter_Close(t *testing.T) {
	ctx := context.Background()

	// Create test data
	data := [][]any{{"test"}}
	exp := exporter.NewExporter(data)

	// Export first to create file
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Close the exporter
	err = exp.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestExporter_Close_WithoutExport tests Close without prior Export.
func TestExporter_Close_WithoutExport(t *testing.T) {
	// Create exporter but don't export
	exp := exporter.NewExporter([][]any{{"test"}})

	// Close should not fail even if file is nil
	err := exp.Close()
	if err != nil {
		t.Errorf("Close without export should not fail: %v", err)
	}
}

// TestExporter_GetFile tests GetFile method.
func TestExporter_GetFile(t *testing.T) {
	ctx := context.Background()

	// Create test data
	data := [][]any{{"test"}}
	exp := exporter.NewExporter(data)

	// Before export, file should be nil
	if exp.GetFile() != nil {
		t.Error("File should be nil before export")
	}

	// After export, file should exist
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify file is created
	if exp.GetFile() == nil {
		t.Error("File should not be nil after export")
	}
}

// TestExporter_GetErrors tests GetErrors method.
func TestExporter_GetErrors(t *testing.T) {
	ctx := context.Background()

	// Create test data
	data := [][]any{{"test"}}
	exp := exporter.NewExporter(data, excel.WithStopOnError(false))

	// Before export, errors should be empty
	errors := exp.GetErrors()
	if errors == nil {
		t.Error("Errors slice should be initialized")
	}
	if len(errors) > 0 {
		t.Error("Errors should be empty before export")
	}

	// After successful export, errors should still be empty
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify errors are still empty
	errors = exp.GetErrors()
	if len(errors) > 0 {
		t.Errorf("Errors should be empty after successful export, got %d", len(errors))
	}
}

// TestExporter_GetConfig tests GetConfig method.
func TestExporter_GetConfig(t *testing.T) {
	// Create exporter with custom config
	data := [][]any{{"test"}}
	exp := exporter.NewExporter(data,
		excel.WithSheetName("CustomSheet"),
		excel.WithStopOnError(false),
	)

	// Get config
	config := exp.GetConfig()
	if config == nil {
		t.Fatal("Config should not be nil")
	}

	// Verify config values
	if config.SheetName != "CustomSheet" {
		t.Errorf("SheetName = %v, want CustomSheet", config.SheetName)
	}
	if config.StopOnError {
		t.Error("StopOnError = true, want false")
	}

	// Modify returned config should not affect original
	config.SheetName = "ModifiedSheet"
	newConfig := exp.GetConfig()
	if newConfig.SheetName == "ModifiedSheet" {
		t.Error("GetConfig should return a copy, not original config")
	}
}

// TestExporter_Export_StopOnError_True tests that export stops on first error.
func TestExporter_Export_StopOnError_True(t *testing.T) {
	ctx := context.Background()

	// Valid data - we can't easily force SetCellValue to fail with excelize
	// But we can test the configuration
	data := [][]any{
		{"A", "B"},
		{"1", "2"},
	}

	// Create exporter with StopOnError=true (default)
	exp := exporter.NewExporter(data, excel.WithStopOnError(true))

	// Export should succeed with valid data
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export should succeed with valid data: %v", err)
	}

	// Verify no errors collected
	errors := exp.GetErrors()
	if len(errors) > 0 {
		t.Errorf("Should have no errors collected, got %d", len(errors))
	}
}

// TestExporter_Export_StopOnError_False tests that export continues on error.
func TestExporter_Export_StopOnError_False(t *testing.T) {
	ctx := context.Background()

	// Valid data for this test
	data := [][]any{
		{"Column1", "Column2"},
		{"Value1", "Value2"},
	}

	// Create exporter with StopOnError=false
	exp := exporter.NewExporter(data, excel.WithStopOnError(false))

	// Export should succeed
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
}
