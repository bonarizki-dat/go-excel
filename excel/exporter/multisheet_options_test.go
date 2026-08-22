package exporter

import (
	"context"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestExportToSheets_EmptySheets tests handling of empty sheets.
func TestExportToSheets_EmptySheets(t *testing.T) {
	sheets := []excel.Sheet{}

	exporter := NewExporter(nil)
	ctx := context.Background()

	// Should fail with no sheets
	err := exporter.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())
	if err == nil {
		t.Error("Expected error for empty sheets, got nil")
	}
}

// TestExportToSheets_CreateDefaultSheet tests CreateDefaultSheet option.
func TestExportToSheets_CreateDefaultSheet(t *testing.T) {
	sheets := []excel.Sheet{}

	exporter := NewExporter(nil)
	ctx := context.Background()

	// Configure options to create default sheet
	options := excel.DefaultMultiSheetExportOptions()
	options.CreateDefaultSheet = true
	options.DefaultSheetName = "DefaultSheet"

	err := exporter.ExportToSheets(ctx, sheets, options)
	if err != nil {
		t.Fatalf("ExportToSheets with CreateDefaultSheet failed: %v", err)
	}

	// Verify default sheet was created
	sheetList := exporter.GetFile().GetSheetList()
	if len(sheetList) != 1 {
		t.Fatalf("Expected 1 sheet, got %d", len(sheetList))
	}

	if sheetList[0] != "DefaultSheet" {
		t.Errorf("Expected sheet name 'DefaultSheet', got '%s'", sheetList[0])
	}
}

// TestExportToSheets_SkipEmptySheets tests SkipEmptySheets option.
func TestExportToSheets_SkipEmptySheets(t *testing.T) {
	sheets := []excel.Sheet{
		{
			Name: "Sheet1",
			Data: [][]any{{"A", "B"}, {1, 2}},
		},
		{
			Name: "EmptySheet",
			Data: [][]any{},
		},
		{
			Name: "Sheet2",
			Data: [][]any{{"C", "D"}, {3, 4}},
		},
	}

	exporter := NewExporter(nil)
	ctx := context.Background()

	// Configure to skip empty sheets
	options := excel.DefaultMultiSheetExportOptions()
	options.SkipEmptySheets = true

	err := exporter.ExportToSheets(ctx, sheets, options)
	if err != nil {
		t.Fatalf("ExportToSheets failed: %v", err)
	}

	// Should only have 2 sheets (empty one skipped)
	sheetList := exporter.GetFile().GetSheetList()
	if len(sheetList) != 2 {
		t.Fatalf("Expected 2 sheets (empty skipped), got %d", len(sheetList))
	}
}

// TestExportToSheets_InvalidSheetName tests validation of sheet names.
func TestExportToSheets_InvalidSheetName(t *testing.T) {
	testCases := []struct {
		name      string
		sheetName string
		wantError bool
	}{
		{
			name:      "valid name",
			sheetName: "ValidSheet",
			wantError: false,
		},
		{
			name:      "empty name",
			sheetName: "",
			wantError: true,
		},
		{
			name:      "name too long",
			sheetName: "ThisIsAVeryLongSheetNameThatExceedsTheMaximumAllowedLength",
			wantError: true,
		},
		{
			name:      "name with invalid char /",
			sheetName: "Invalid/Sheet",
			wantError: true,
		},
		{
			name:      "name with invalid char \\",
			sheetName: "Invalid\\Sheet",
			wantError: true,
		},
		{
			name:      "name with invalid char ?",
			sheetName: "Invalid?Sheet",
			wantError: true,
		},
		{
			name:      "name with invalid char *",
			sheetName: "Invalid*Sheet",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sheets := []excel.Sheet{
				{
					Name: tc.sheetName,
					Data: [][]any{{"A"}, {1}},
				},
			}

			exporter := NewExporter(nil)
			ctx := context.Background()

			err := exporter.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())

			if tc.wantError && err == nil {
				t.Error("Expected error for invalid sheet name, got nil")
			}

			if !tc.wantError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestExportToSheets_DuplicateNames tests handling of duplicate sheet names.
func TestExportToSheets_DuplicateNames(t *testing.T) {
	sheets := []excel.Sheet{
		{
			Name: "Sheet1",
			Data: [][]any{{"A"}, {1}},
		},
		{
			Name: "Sheet1", // Duplicate name
			Data: [][]any{{"B"}, {2}},
		},
	}

	exporter := NewExporter(nil)
	ctx := context.Background()

	err := exporter.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())
	if err == nil {
		t.Error("Expected error for duplicate sheet names, got nil")
	}
}

// TestExportToSheets_WithConfig tests sheet-specific configuration.
func TestExportToSheets_WithConfig(t *testing.T) {
	// Create config with headings for first sheet
	configWithHeadings := config.DefaultConfig()
	configWithHeadings.WithHeadings = true

	// Create config without headings for second sheet
	configNoHeadings := config.DefaultConfig()
	configNoHeadings.WithHeadings = false

	sheets := []excel.Sheet{
		{
			Name:   "WithHeadings",
			Data:   [][]any{{"Header1", "Header2"}, {1, 2}, {3, 4}},
			Config: configWithHeadings,
		},
		{
			Name:   "NoHeadings",
			Data:   [][]any{{"Data1", "Data2"}, {5, 6}},
			Config: configNoHeadings,
		},
	}

	exporter := NewExporter(nil)
	ctx := context.Background()

	err := exporter.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions())
	if err != nil {
		t.Fatalf("ExportToSheets failed: %v", err)
	}

	// Verify both sheets were created
	sheetList := exporter.GetFile().GetSheetList()
	if len(sheetList) != 2 {
		t.Fatalf("Expected 2 sheets, got %d", len(sheetList))
	}
}
