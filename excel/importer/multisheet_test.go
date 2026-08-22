package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// createMultiSheetTestFile creates a test Excel file with multiple sheets.
func createMultiSheetTestFile(t *testing.T) string {
	t.Helper()

	// Create excelize file
	f := excelize.NewFile()

	// Create Users sheet (rename default sheet)
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, "Users"); err != nil {
		t.Fatalf("Failed to rename sheet: %v", err)
	}

	// Add Users data
	usersData := [][]any{
		{"ID", "Name", "Email"},
		{1, "John Doe", "john@example.com"},
		{2, "Jane Smith", "jane@example.com"},
	}

	for rowIdx, row := range usersData {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			f.SetCellValue("Users", cell, value)
		}
	}

	// Create Orders sheet
	_, err := f.NewSheet("Orders")
	if err != nil {
		t.Fatalf("Failed to create Orders sheet: %v", err)
	}

	// Add Orders data
	ordersData := [][]any{
		{"Order ID", "User ID", "Amount"},
		{101, 1, 250.50},
		{102, 2, 175.25},
		{103, 1, 99.99},
	}

	for rowIdx, row := range ordersData {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			f.SetCellValue("Orders", cell, value)
		}
	}

	// Create Products sheet
	_, err = f.NewSheet("Products")
	if err != nil {
		t.Fatalf("Failed to create Products sheet: %v", err)
	}

	// Add Products data
	productsData := [][]any{
		{"Product ID", "Name", "Price"},
		{1001, "Laptop", 1200.00},
		{1002, "Mouse", 25.00},
	}

	for rowIdx, row := range productsData {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			f.SetCellValue("Products", cell, value)
		}
	}

	// Save to temp file
	tempFile := filepath.Join(os.TempDir(), "test_multisheet_import.xlsx")
	if err := f.SaveAs(tempFile); err != nil {
		t.Fatalf("Failed to save test file: %v", err)
	}

	return tempFile
}

// TestImportFromSheet_Basic tests importing from a specific sheet.
func TestImportFromSheet_Basic(t *testing.T) {
	// Create test file
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	// Create importer
	// Default config has WithHeadings=true, so header row is skipped
	importer := NewImporter()
	defer importer.Close()

	// Import from Users sheet
	ctx := context.Background()
	rows, err := importer.ImportFromSheet(ctx, tempFile, "Users")
	if err != nil {
		t.Fatalf("ImportFromSheet failed: %v", err)
	}

	// Verify data (should be 2 rows since header is skipped by default)
	if len(rows) != 2 {
		t.Errorf("Expected 2 rows (header skipped), got %d", len(rows))
	}

	// Verify first data row
	if len(rows) > 0 {
		// First row should be John Doe (header was skipped)
		if rows[0][1] != "John Doe" {
			t.Errorf("Expected 'John Doe', got %v", rows[0][1])
		}
	}

	// Verify second data row
	if len(rows) > 1 {
		if rows[1][1] != "Jane Smith" {
			t.Errorf("Expected 'Jane Smith', got %v", rows[1][1])
		}
	}
}

// TestImportFromSheet_DifferentSheets tests importing from different sheets.
func TestImportFromSheet_DifferentSheets(t *testing.T) {
	// Create test file
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	ctx := context.Background()

	// Test Users sheet
	importer1 := NewImporter()
	defer importer1.Close()

	usersRows, err := importer1.ImportFromSheet(ctx, tempFile, "Users")
	if err != nil {
		t.Fatalf("Failed to import Users: %v", err)
	}

	// Header is skipped, so 2 data rows
	if len(usersRows) != 2 {
		t.Errorf("Expected 2 rows in Users, got %d", len(usersRows))
	}

	// Test Orders sheet
	importer2 := NewImporter()
	defer importer2.Close()

	ordersRows, err := importer2.ImportFromSheet(ctx, tempFile, "Orders")
	if err != nil {
		t.Fatalf("Failed to import Orders: %v", err)
	}

	// Header is skipped, so 3 data rows
	if len(ordersRows) != 3 {
		t.Errorf("Expected 3 rows in Orders, got %d", len(ordersRows))
	}

	// Test Products sheet
	importer3 := NewImporter()
	defer importer3.Close()

	productsRows, err := importer3.ImportFromSheet(ctx, tempFile, "Products")
	if err != nil {
		t.Fatalf("Failed to import Products: %v", err)
	}

	// Header is skipped, so 2 data rows
	if len(productsRows) != 2 {
		t.Errorf("Expected 2 rows in Products, got %d", len(productsRows))
	}
}

// TestImportFromSheet_NonExistentSheet tests error handling for non-existent sheet.
func TestImportFromSheet_NonExistentSheet(t *testing.T) {
	// Create test file
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	// Create importer
	importer := NewImporter()
	defer importer.Close()

	// Try to import from non-existent sheet
	ctx := context.Background()
	_, err := importer.ImportFromSheet(ctx, tempFile, "NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent sheet, got nil")
	}
}

// TestImportAllSheets_Basic tests importing all sheets.
func TestImportAllSheets_Basic(t *testing.T) {
	// Create test file
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	// Create importer
	importer := NewImporter()
	defer importer.Close()

	// Import all sheets
	ctx := context.Background()
	sheetsData, err := importer.ImportAllSheets(ctx, tempFile)
	if err != nil {
		t.Fatalf("ImportAllSheets failed: %v", err)
	}

	// Verify we got 3 sheets
	if len(sheetsData) != 3 {
		t.Fatalf("Expected 3 sheets, got %d", len(sheetsData))
	}

	// Verify Users sheet (header skipped, so 2 rows)
	usersData, exists := sheetsData["Users"]
	if !exists {
		t.Error("Users sheet not found in result")
	} else if len(usersData) != 2 {
		t.Errorf("Expected 2 rows in Users, got %d", len(usersData))
	}

	// Verify Orders sheet (header skipped, so 3 rows)
	ordersData, exists := sheetsData["Orders"]
	if !exists {
		t.Error("Orders sheet not found in result")
	} else if len(ordersData) != 3 {
		t.Errorf("Expected 3 rows in Orders, got %d", len(ordersData))
	}

	// Verify Products sheet (header skipped, so 2 rows)
	productsData, exists := sheetsData["Products"]
	if !exists {
		t.Error("Products sheet not found in result")
	} else if len(productsData) != 2 {
		t.Errorf("Expected 2 rows in Products, got %d", len(productsData))
	}
}
