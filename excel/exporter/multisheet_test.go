package exporter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/xuri/excelize/v2"
)

// TestExportToSheets_Basic tests basic multi-sheet export functionality.
func TestExportToSheets_Basic(t *testing.T) {
	// Create test data for multiple sheets
	usersData := [][]any{
		{"ID", "Name", "Email"},
		{1, "John Doe", "john@example.com"},
		{2, "Jane Smith", "jane@example.com"},
	}

	ordersData := [][]any{
		{"Order ID", "User ID", "Amount"},
		{101, 1, 250.50},
		{102, 2, 175.25},
		{103, 1, 99.99},
	}

	// Create sheets
	sheets := []excel.Sheet{
		{
			Name: "Users",
			Data: usersData,
		},
		{
			Name: "Orders",
			Data: ordersData,
		},
	}

	// Create exporter
	exporter := NewExporter(nil)

	// Create temp file
	tempFile := filepath.Join(os.TempDir(), "test_multisheet.xlsx")
	defer os.Remove(tempFile)

	// Export to sheets
	ctx := context.Background()
	err := exporter.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())
	if err != nil {
		t.Fatalf("ExportToSheets failed: %v", err)
	}

	// Save file using excelize's SaveAs method directly
	err = exporter.GetFile().SaveAs(tempFile)
	if err != nil {
		t.Fatalf("SaveAs failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Fatal("Output file was not created")
	}

	// Verify file contents
	f, err := excelize.OpenFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to open output file: %v", err)
	}
	defer f.Close()

	// Check sheet names
	sheetList := f.GetSheetList()
	if len(sheetList) != 2 {
		t.Fatalf("Expected 2 sheets, got %d", len(sheetList))
	}

	if sheetList[0] != "Users" {
		t.Errorf("Expected first sheet to be 'Users', got '%s'", sheetList[0])
	}

	if sheetList[1] != "Orders" {
		t.Errorf("Expected second sheet to be 'Orders', got '%s'", sheetList[1])
	}

	// Verify Users sheet data
	usersRows, err := f.GetRows("Users")
	if err != nil {
		t.Fatalf("Failed to read Users sheet: %v", err)
	}

	if len(usersRows) != 3 {
		t.Errorf("Expected 3 rows in Users sheet, got %d", len(usersRows))
	}

	// Verify Orders sheet data
	ordersRows, err := f.GetRows("Orders")
	if err != nil {
		t.Fatalf("Failed to read Orders sheet: %v", err)
	}

	if len(ordersRows) != 4 {
		t.Errorf("Expected 4 rows in Orders sheet, got %d", len(ordersRows))
	}
}

// TestExportToSheets_DataChannel_ContextCancelled verifies that
// canceling the context while a sheet's DataChannel is still being read
// surfaces as an error from ExportToSheets, rather than silently
// exporting whatever partial data had been read so far. Before this fix,
// readFromChannel discarded ctx.Err() and returned only the rows read up
// to that point, with no way for the caller to detect the truncation.
func TestExportToSheets_DataChannel_ContextCancelled(t *testing.T) {
	rows := make(chan []any)
	ctx, cancel := context.WithCancel(context.Background())

	// Send one row, then cancel before the channel is closed, simulating
	// a producer that never finishes.
	go func() {
		rows <- []any{"only-row"}
		cancel()
	}()

	sheets := []excel.Sheet{
		{Name: "Streamed", DataChannel: rows},
	}

	exp := NewExporter(nil)
	err := exp.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())
	if err == nil {
		t.Fatal("expected ExportToSheets to fail when the context is canceled mid-stream")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected error to wrap context.Canceled, got: %v", err)
	}
}
