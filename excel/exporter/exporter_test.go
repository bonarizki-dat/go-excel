package exporter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// TestNewExporter tests the NewExporter constructor.
func TestNewExporter(t *testing.T) {
	// Test with nil data
	exp := exporter.NewExporter(nil)
	if exp == nil {
		t.Fatal("NewExporter should not return nil")
	}
	if exp.GetConfig() == nil {
		t.Error("Exporter config should not be nil")
	}
	if exp.GetErrors() == nil {
		t.Error("Exporter errors slice should be initialized")
	}

	// Test with valid data and options
	data := [][]any{
		{"Name", "Age"},
		{"John", 30},
	}
	exp = exporter.NewExporter(data,
		excel.WithSheetName("TestSheet"),
		excel.WithStopOnError(false),
	)
	if exp.GetConfig().SheetName != "TestSheet" {
		t.Errorf("SheetName = %v, want TestSheet", exp.GetConfig().SheetName)
	}
	if exp.GetConfig().StopOnError {
		t.Error("StopOnError = true, want false")
	}
}

// TestExporter_Export_EmptyData tests export with nil/empty data.
func TestExporter_Export_EmptyData(t *testing.T) {
	ctx := context.Background()

	// Test with nil data
	exp := exporter.NewExporter(nil)
	err := exp.Export(ctx)
	if !errors.Is(err, excel.ErrEmptyData) {
		t.Errorf("Export with nil data should return excel.ErrEmptyData, got %v", err)
	}
}

// TestExporter_Export_InvalidConfig tests export with invalid config.
func TestExporter_Export_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	// Create exporter with invalid config
	data := [][]any{{"test"}}
	exp := exporter.NewExporter(data, excel.WithSheetName("")) // Invalid empty sheet name

	err := exp.Export(ctx)
	if err == nil {
		t.Error("Export with invalid config should return error")
	}
}

// TestExporter_Export_SimpleData tests basic export functionality.
func TestExporter_Export_SimpleData(t *testing.T) {
	ctx := context.Background()

	// Prepare simple test data
	data := [][]any{
		{"Name", "Age", "City"},
		{"John", 30, "New York"},
		{"Jane", 25, "Los Angeles"},
	}

	// Create exporter
	exp := exporter.NewExporter(data)

	// Perform export
	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify file was created
	if exp.GetFile() == nil {
		t.Error("Excelize file should be created")
	}

	// Verify sheet exists
	sheets := exp.GetFile().GetSheetList()
	if len(sheets) == 0 {
		t.Fatal("No sheets found in exported file")
	}

	// Verify data was written (check first cell)
	cell, err := exp.GetFile().GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("Failed to get cell value: %v", err)
	}
	if cell != "Name" {
		t.Errorf("Cell A1 = %v, want Name", cell)
	}
}

// TestExporter_Export_WithHeadings tests export with headings enabled.
func TestExporter_Export_WithHeadings(t *testing.T) {
	ctx := context.Background()

	// Data where first row is headers
	data := [][]any{
		{"ID", "Name", "Email"},
		{1, "Alice", "alice@example.com"},
		{2, "Bob", "bob@example.com"},
	}

	// Create exporter with headings enabled
	exp := exporter.NewExporter(data, excel.WithHeadings(true))

	err := exp.Export(ctx)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify headers are in row 1
	header, err := exp.GetFile().GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("Failed to get header: %v", err)
	}
	if header != "ID" {
		t.Errorf("Header A1 = %v, want ID", header)
	}

	// Verify data starts from row 2
	dataCell, err := exp.GetFile().GetCellValue("Sheet1", "A2")
	if err != nil {
		t.Fatalf("Failed to get data cell: %v", err)
	}
	if dataCell != "1" {
		t.Errorf("Data A2 = %v, want 1", dataCell)
	}
}

// TestExporter_Export_ContextCancellation tests context cancellation.
func TestExporter_Export_ContextCancellation(t *testing.T) {
	// Create context that is already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Create large dataset to ensure context is checked
	data := make([][]any, 1000)
	for i := range data {
		data[i] = []any{i, "test", "data"}
	}

	exp := exporter.NewExporter(data)

	// Export should fail due to canceled context
	err := exp.Export(ctx)
	if err == nil {
		t.Error("Export should fail with canceled context")
	}
	// Error might be wrapped
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Error should be or wrap context.Canceled, got %v", err)
	}
}

// TestExporter_Export_WithTimeout tests export with timeout.
func TestExporter_Export_WithTimeout(t *testing.T) {
	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait a bit to ensure timeout
	time.Sleep(10 * time.Millisecond)

	data := [][]any{{"test"}}
	exp := exporter.NewExporter(data)

	err := exp.Export(ctx)
	// Should fail due to timeout
	if err == nil {
		t.Error("Export should fail with timeout")
	}
}
