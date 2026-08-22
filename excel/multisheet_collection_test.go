package excel

import (
	"testing"
)

// TestNewSheetCollection tests SheetCollection creation.
//
// Verifies that new collection is empty and ready to use.
func TestNewSheetCollection(t *testing.T) {
	// Create new collection
	sc := NewSheetCollection()

	// Verify collection is empty
	if sc.Count() != 0 {
		t.Errorf("Count() = %d, want 0", sc.Count())
	}

	// Verify internal structures are initialized
	if sc.sheets == nil {
		t.Error("sheets slice is nil")
	}

	if sc.nameIndex == nil {
		t.Error("nameIndex map is nil")
	}
}

// TestSheetCollection_Add tests adding sheets to collection.
//
// Tests successful addition and error cases.
func TestSheetCollection_Add(t *testing.T) {
	tests := []struct {
		name    string // Test case name
		sheet   *Sheet // Sheet to add
		wantErr bool   // Whether error is expected
		errMsg  string // Expected error message substring
	}{
		{
			name: "valid sheet",
			sheet: &Sheet{
				Name: "Sheet1",
				Data: [][]any{{"test"}},
			},
			wantErr: false,
		},
		{
			name: "valid sheet with long name",
			sheet: &Sheet{
				Name: "1234567890123456789012345678901", // 31 chars (max)
			},
			wantErr: false,
		},
		{
			name: "empty sheet name",
			sheet: &Sheet{
				Name: "",
			},
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name: "sheet name too long",
			sheet: &Sheet{
				Name: "12345678901234567890123456789012", // 32 chars
			},
			wantErr: true,
			errMsg:  "too long",
		},
		{
			name: "sheet name with invalid char",
			sheet: &Sheet{
				Name: "My/Sheet",
			},
			wantErr: true,
			errMsg:  "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create new collection for each test
			sc := NewSheetCollection()

			// Try to add sheet
			err := sc.Add(tt.sheet)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If error expected, verify error message
			if tt.wantErr && err != nil {
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Add() error = %q, want substring %q", err.Error(), tt.errMsg)
				}
			}

			// If no error, verify sheet was added
			if !tt.wantErr {
				if sc.Count() != 1 {
					t.Errorf("Count() = %d, want 1", sc.Count())
				}

				// Verify index was set
				if tt.sheet.Index != 0 {
					t.Errorf("Sheet.Index = %d, want 0", tt.sheet.Index)
				}
			}
		})
	}
}

// TestSheetCollection_Add_Duplicate tests duplicate sheet name handling.
//
// Verifies that duplicate names are rejected.
func TestSheetCollection_Add_Duplicate(t *testing.T) {
	// Create collection
	sc := NewSheetCollection()

	// Add first sheet
	sheet1 := &Sheet{Name: "Sheet1"}
	if err := sc.Add(sheet1); err != nil {
		t.Fatalf("Add() first sheet error = %v", err)
	}

	// Try to add duplicate
	sheet2 := &Sheet{Name: "Sheet1"}
	err := sc.Add(sheet2)

	// Should return error
	if err == nil {
		t.Error("Add() duplicate sheet should return error")
	}

	// Should contain "already exists"
	if !contains(err.Error(), "already exists") {
		t.Errorf("Add() error = %q, want substring 'already exists'", err.Error())
	}

	// Collection should still have only 1 sheet
	if sc.Count() != 1 {
		t.Errorf("Count() = %d, want 1", sc.Count())
	}
}

// TestSheetCollection_Get tests retrieving sheets by name.
// Helper function for error message validation.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

// findSubstring searches for substring in string.
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
