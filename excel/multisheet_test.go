package excel

import (
	"testing"
)

// TestValidateSheetName tests sheet name validation according to Excel rules.
//
// Tests all validation rules:
// - Empty name
// - Length > 31 characters
// - Invalid characters: \ / ? * [ ]
// - Leading/trailing apostrophe.
func TestValidateSheetName(t *testing.T) {
	// Table-driven tests for all validation scenarios
	tests := []struct {
		name    string // Test case name
		input   string // Sheet name to validate
		wantErr bool   // Whether error is expected
	}{
		// Valid names
		{
			name:    "valid simple name",
			input:   "Sheet1",
			wantErr: false,
		},
		{
			name:    "valid name with spaces",
			input:   "My Sheet",
			wantErr: false,
		},
		{
			name:    "valid name with numbers",
			input:   "Sheet123",
			wantErr: false,
		},
		{
			name:    "valid name with underscore",
			input:   "My_Sheet",
			wantErr: false,
		},
		{
			name:    "valid name with hyphen",
			input:   "My-Sheet",
			wantErr: false,
		},
		{
			name:    "valid name at max length (31 chars)",
			input:   "1234567890123456789012345678901",
			wantErr: false,
		},

		// Invalid names - empty
		{
			name:    "empty name",
			input:   "",
			wantErr: true,
		},

		// Invalid names - too long
		{
			name:    "name too long (32 chars)",
			input:   "12345678901234567890123456789012",
			wantErr: true,
		},
		{
			name:    "name too long (50 chars)",
			input:   "12345678901234567890123456789012345678901234567890",
			wantErr: true,
		},

		// Invalid names - invalid characters
		{
			name:    "name with backslash",
			input:   "My\\Sheet",
			wantErr: true,
		},
		{
			name:    "name with forward slash",
			input:   "My/Sheet",
			wantErr: true,
		},
		{
			name:    "name with question mark",
			input:   "My?Sheet",
			wantErr: true,
		},
		{
			name:    "name with asterisk",
			input:   "My*Sheet",
			wantErr: true,
		},
		{
			name:    "name with opening bracket",
			input:   "My[Sheet",
			wantErr: true,
		},
		{
			name:    "name with closing bracket",
			input:   "My]Sheet",
			wantErr: true,
		},

		// Invalid names - apostrophe
		{
			name:    "name starting with apostrophe",
			input:   "'MySheet",
			wantErr: true,
		},
		{
			name:    "name ending with apostrophe",
			input:   "MySheet'",
			wantErr: true,
		},
		{
			name:    "name with apostrophe in middle (valid)",
			input:   "My'Sheet",
			wantErr: false,
		},
	}

	// Run all test cases
	for _, tt := range tests {
		// Run each test case as subtest
		t.Run(tt.name, func(t *testing.T) {
			// Call validation function
			err := validateSheetName(tt.input)

			// Check if error matches expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSheetName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestSheet_Struct tests Sheet struct initialization.
//
// Verifies that Sheet struct can be created with all fields.
func TestSheet_Struct(t *testing.T) {
	// Create sample data
	data := [][]any{
		{"ID", "Name", "Email"},
		{1, "John", "john@example.com"},
		{2, "Jane", "jane@example.com"},
	}

	// Create sheet with all fields
	sheet := &Sheet{
		Name:  "Users",
		Data:  data,
		Index: 0,
	}

	// Verify fields are set correctly
	if sheet.Name != "Users" {
		t.Errorf("Sheet.Name = %q, want %q", sheet.Name, "Users")
	}

	if len(sheet.Data) != 3 {
		t.Errorf("len(Sheet.Data) = %d, want %d", len(sheet.Data), 3)
	}

	if sheet.Index != 0 {
		t.Errorf("Sheet.Index = %d, want %d", sheet.Index, 0)
	}
}

// TestDefaultMultiSheetExportOptions tests default options creation.
//
// Verifies that default options have expected values.
func TestDefaultMultiSheetExportOptions(t *testing.T) {
	// Get default options
	opts := DefaultMultiSheetExportOptions()

	// Verify default values
	if opts.CreateDefaultSheet != false {
		t.Errorf("CreateDefaultSheet = %v, want %v", opts.CreateDefaultSheet, false)
	}

	if opts.DefaultSheetName != "Sheet1" {
		t.Errorf("DefaultSheetName = %q, want %q", opts.DefaultSheetName, "Sheet1")
	}

	if opts.SkipEmptySheets != false {
		t.Errorf("SkipEmptySheets = %v, want %v", opts.SkipEmptySheets, false)
	}

	if opts.ContinueOnError != false {
		t.Errorf("ContinueOnError = %v, want %v", opts.ContinueOnError, false)
	}

	if opts.ProgressCallback != nil {
		t.Error("ProgressCallback should be nil")
	}
}
