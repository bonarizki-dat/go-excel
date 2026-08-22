package excel

import (
	"reflect"
	"testing"
)

func TestSheetCollection_Get(t *testing.T) {
	// Create collection with sheets
	sc := NewSheetCollection()
	sheet1 := &Sheet{Name: "Sheet1"}
	sheet2 := &Sheet{Name: "Sheet2"}
	sc.Add(sheet1)
	sc.Add(sheet2)

	tests := []struct {
		name       string // Test case name
		lookupName string // Name to look up
		want       *Sheet // Expected sheet
	}{
		{
			name:       "existing sheet 1",
			lookupName: "Sheet1",
			want:       sheet1,
		},
		{
			name:       "existing sheet 2",
			lookupName: "Sheet2",
			want:       sheet2,
		},
		{
			name:       "non-existing sheet",
			lookupName: "Sheet3",
			want:       nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get sheet by name
			got := sc.Get(tt.lookupName)

			// Compare pointers
			if got != tt.want {
				t.Errorf("Get(%q) = %v, want %v", tt.lookupName, got, tt.want)
			}
		})
	}
}

// TestSheetCollection_GetByIndex tests retrieving sheets by index.
func TestSheetCollection_GetByIndex(t *testing.T) {
	// Create collection with sheets
	sc := NewSheetCollection()
	sheet1 := &Sheet{Name: "Sheet1"}
	sheet2 := &Sheet{Name: "Sheet2"}
	sc.Add(sheet1)
	sc.Add(sheet2)

	tests := []struct {
		name  string // Test case name
		index int    // Index to look up
		want  *Sheet // Expected sheet
	}{
		{
			name:  "index 0",
			index: 0,
			want:  sheet1,
		},
		{
			name:  "index 1",
			index: 1,
			want:  sheet2,
		},
		{
			name:  "negative index",
			index: -1,
			want:  nil,
		},
		{
			name:  "index out of bounds",
			index: 2,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get sheet by index
			got := sc.GetByIndex(tt.index)

			// Compare pointers
			if got != tt.want {
				t.Errorf("GetByIndex(%d) = %v, want %v", tt.index, got, tt.want)
			}
		})
	}
}

// TestSheetCollection_Exists tests checking sheet existence.
func TestSheetCollection_Exists(t *testing.T) {
	// Create collection with sheets
	sc := NewSheetCollection()
	sc.Add(&Sheet{Name: "Sheet1"})
	sc.Add(&Sheet{Name: "Sheet2"})

	tests := []struct {
		name      string // Test case name
		sheetName string // Name to check
		want      bool   // Expected result
	}{
		{
			name:      "existing sheet",
			sheetName: "Sheet1",
			want:      true,
		},
		{
			name:      "non-existing sheet",
			sheetName: "Sheet3",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check existence
			got := sc.Exists(tt.sheetName)

			if got != tt.want {
				t.Errorf("Exists(%q) = %v, want %v", tt.sheetName, got, tt.want)
			}
		})
	}
}

// TestSheetCollection_All tests getting all sheets.
func TestSheetCollection_All(t *testing.T) {
	// Create collection with sheets
	sc := NewSheetCollection()
	sheet1 := &Sheet{Name: "Sheet1"}
	sheet2 := &Sheet{Name: "Sheet2"}
	sc.Add(sheet1)
	sc.Add(sheet2)

	// Get all sheets
	all := sc.All()

	// Verify count
	if len(all) != 2 {
		t.Errorf("len(All()) = %d, want 2", len(all))
	}

	// Verify order
	if all[0] != sheet1 {
		t.Errorf("All()[0] = %v, want %v", all[0], sheet1)
	}
	if all[1] != sheet2 {
		t.Errorf("All()[1] = %v, want %v", all[1], sheet2)
	}
}

// TestSheetCollection_Names tests getting sheet names.
func TestSheetCollection_Names(t *testing.T) {
	// Create collection with sheets
	sc := NewSheetCollection()
	sc.Add(&Sheet{Name: "Users"})
	sc.Add(&Sheet{Name: "Orders"})
	sc.Add(&Sheet{Name: "Products"})

	// Get names
	names := sc.Names()

	// Expected names in order
	want := []string{"Users", "Orders", "Products"}

	// Compare slices
	if !reflect.DeepEqual(names, want) {
		t.Errorf("Names() = %v, want %v", names, want)
	}
}

// TestSheetCollection_Clear tests clearing collection.
func TestSheetCollection_Clear(t *testing.T) {
	// Create collection with sheets
	sc := NewSheetCollection()
	sc.Add(&Sheet{Name: "Sheet1"})
	sc.Add(&Sheet{Name: "Sheet2"})

	// Verify not empty
	if sc.Count() != 2 {
		t.Fatalf("Count() before Clear = %d, want 2", sc.Count())
	}

	// Clear collection
	sc.Clear()

	// Verify empty
	if sc.Count() != 0 {
		t.Errorf("Count() after Clear = %d, want 0", sc.Count())
	}

	// Verify can add new sheets after clear
	if err := sc.Add(&Sheet{Name: "NewSheet"}); err != nil {
		t.Errorf("Add() after Clear error = %v", err)
	}

	if sc.Count() != 1 {
		t.Errorf("Count() after adding to cleared collection = %d, want 1", sc.Count())
	}
}

// contains checks if string contains substring.
//
