package importer

import (
	"context"
	"os"
	"testing"
)

// TestGetSheetNames tests retrieving sheet names.
func TestGetSheetNames(t *testing.T) {
	// Create test file
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	// Create importer and open file
	importer := NewImporter()
	defer importer.Close()

	ctx := context.Background()
	_, err := importer.ImportFromSheet(ctx, tempFile, "Users")
	if err != nil {
		t.Fatalf("ImportFromSheet failed: %v", err)
	}

	// Get sheet names
	sheetNames, err := importer.GetSheetNames()
	if err != nil {
		t.Fatalf("GetSheetNames failed: %v", err)
	}

	// Verify we got 3 sheet names
	if len(sheetNames) != 3 {
		t.Fatalf("Expected 3 sheet names, got %d", len(sheetNames))
	}

	// Verify sheet names
	expectedNames := map[string]bool{
		"Users":    false,
		"Orders":   false,
		"Products": false,
	}

	for _, name := range sheetNames {
		if _, exists := expectedNames[name]; exists {
			expectedNames[name] = true
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("Expected sheet '%s' not found", name)
		}
	}
}

// TestGetSheetCount tests counting sheets.
func TestGetSheetCount(t *testing.T) {
	// Create test file
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	// Create importer and open file
	importer := NewImporter()
	defer importer.Close()

	ctx := context.Background()
	_, err := importer.ImportFromSheet(ctx, tempFile, "Users")
	if err != nil {
		t.Fatalf("ImportFromSheet failed: %v", err)
	}

	// Get sheet count
	count, err := importer.GetSheetCount()
	if err != nil {
		t.Fatalf("GetSheetCount failed: %v", err)
	}

	// Verify count
	if count != 3 {
		t.Errorf("Expected 3 sheets, got %d", count)
	}
}

// TestSheetExists tests checking if sheet exists.
func TestSheetExists(t *testing.T) {
	// Create test file
	tempFile := createMultiSheetTestFile(t)
	defer os.Remove(tempFile)

	// Create importer and open file
	importer := NewImporter()
	defer importer.Close()

	ctx := context.Background()
	_, err := importer.ImportFromSheet(ctx, tempFile, "Users")
	if err != nil {
		t.Fatalf("ImportFromSheet failed: %v", err)
	}

	// Test existing sheet
	exists, err := importer.SheetExists("Users")
	if err != nil {
		t.Fatalf("SheetExists failed: %v", err)
	}

	if !exists {
		t.Error("Expected Users sheet to exist")
	}

	// Test non-existing sheet
	exists, err = importer.SheetExists("NonExistent")
	if err != nil {
		t.Fatalf("SheetExists failed: %v", err)
	}

	if exists {
		t.Error("Expected NonExistent sheet to not exist")
	}

	// Test with a syntactically invalid sheet name (contains a disallowed
	// character): this must surface as an error, not be reported as a
	// plain "sheet not found" false, since the two failure modes mean
	// different things to a caller.
	_, err = importer.SheetExists("invalid[name]")
	if err == nil {
		t.Error("Expected error for invalid sheet name, got nil")
	}
}
