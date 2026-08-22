package config_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestConfigClone verifies that Clone creates an independent copy
// Modifications to the clone should not affect the original.
func TestConfigClone(t *testing.T) {
	// Create original config with specific values
	original := config.DefaultConfig()
	original.SheetName = "Original"
	original.StartRow = 5
	original.WithHeadings = false
	original.Timeout = 30 * time.Second

	// Clone the configuration
	clone := original.Clone()

	// Verify clone has same values as original
	if clone.SheetName != original.SheetName {
		t.Errorf("clone.SheetName = %v, want %v", clone.SheetName, original.SheetName)
	}
	if clone.StartRow != original.StartRow {
		t.Errorf("clone.StartRow = %v, want %v", clone.StartRow, original.StartRow)
	}
	if clone.WithHeadings != original.WithHeadings {
		t.Errorf("clone.WithHeadings = %v, want %v", clone.WithHeadings, original.WithHeadings)
	}
	if clone.Timeout != original.Timeout {
		t.Errorf("clone.Timeout = %v, want %v", clone.Timeout, original.Timeout)
	}

	// Modify clone values
	clone.SheetName = "Modified"
	clone.StartRow = 10
	clone.WithHeadings = true
	clone.Timeout = 60 * time.Second

	// Verify original is unchanged (deep copy)
	if original.SheetName != "Original" {
		t.Errorf("original.SheetName changed to %v, should remain 'Original'", original.SheetName)
	}
	if original.StartRow != 5 {
		t.Errorf("original.StartRow changed to %v, should remain 5", original.StartRow)
	}
	if original.WithHeadings != false {
		t.Error("original.WithHeadings changed, should remain false")
	}
	if original.Timeout != 30*time.Second {
		t.Errorf("original.Timeout changed to %v, should remain 30s", original.Timeout)
	}
}

// TestConfigClone_AllFields verifies all fields are properly cloned
// This ensures Clone() is updated when new fields are added.
func TestConfigClone_AllFields(t *testing.T) {
	// Create config with all fields set to non-default values
	original := &config.Config{
		SheetName:        "TestSheet",
		WithHeadings:     false,
		StartRow:         10,
		EndRow:           100,
		StartCol:         2,
		EndCol:           20,
		SkipEmptyRows:    false,
		TrimSpaces:       false,
		ValidateData:     true,
		StopOnError:      false,
		Timeout:          60 * time.Second,
		PreallocateRows:  false,
		Logger:           nil,
		Metrics:          nil,
		ProgressCallback: nil,
		ErrorCallback:    nil,
	}

	// Clone the configuration
	clone := original.Clone()

	// Verify all string fields
	if clone.SheetName != original.SheetName {
		t.Errorf("SheetName not cloned correctly")
	}

	// Verify all int fields
	if clone.StartRow != original.StartRow {
		t.Errorf("StartRow not cloned correctly")
	}
	if clone.EndRow != original.EndRow {
		t.Errorf("EndRow not cloned correctly")
	}
	if clone.StartCol != original.StartCol {
		t.Errorf("StartCol not cloned correctly")
	}
	if clone.EndCol != original.EndCol {
		t.Errorf("EndCol not cloned correctly")
	}

	// Verify all bool fields
	if clone.WithHeadings != original.WithHeadings {
		t.Errorf("excel.WithHeadings not cloned correctly")
	}
	if clone.SkipEmptyRows != original.SkipEmptyRows {
		t.Errorf("SkipEmptyRows not cloned correctly")
	}
	if clone.TrimSpaces != original.TrimSpaces {
		t.Errorf("TrimSpaces not cloned correctly")
	}
	if clone.ValidateData != original.ValidateData {
		t.Errorf("ValidateData not cloned correctly")
	}
	if clone.StopOnError != original.StopOnError {
		t.Errorf("StopOnError not cloned correctly")
	}
	if clone.PreallocateRows != original.PreallocateRows {
		t.Errorf("PreallocateRows not cloned correctly")
	}

	// Verify time.Duration fields
	if clone.Timeout != original.Timeout {
		t.Errorf("Timeout not cloned correctly")
	}
}

// TestConfigClone_ConvertersAndRowMap verifies Clone carries over the
// Converters map and RowMap function, added after
// TestConfigClone_AllFields was originally written, so a future field
// left out of Clone is caught here rather than silently dropped.
func TestConfigClone_ConvertersAndRowMap(t *testing.T) {
	original := config.DefaultConfig()
	original.Converters = map[string]config.Converter{
		"col": func(raw string, _ reflect.Type) (any, error) { return raw, nil },
	}
	original.RowMap = func(row []any) ([]any, error) { return row, nil }
	original.StartCell = "B3"
	original.AutoSizeColumns = true
	original.DocProperties = config.DocProperties{Title: "Report"}

	clone := original.Clone()

	if len(clone.Converters) != 1 {
		t.Errorf("clone.Converters = %v, want the single entry from original", clone.Converters)
	}
	if clone.RowMap == nil {
		t.Error("clone.RowMap = nil, want the function from original")
	}
	if clone.StartCell != "B3" {
		t.Errorf("clone.StartCell = %q, want %q", clone.StartCell, "B3")
	}
	if !clone.AutoSizeColumns {
		t.Error("clone.AutoSizeColumns = false, want true")
	}
	if clone.DocProperties.Title != "Report" {
		t.Errorf("clone.DocProperties.Title = %q, want %q", clone.DocProperties.Title, "Report")
	}
}

// knownConfigFields lists every field Config currently has. Adding a
// field to Config without adding its name here (and updating Clone())
// makes TestConfigClone_NoFieldForgotten fail with the field's name,
// catching the omission this whole test file exists to prevent instead
// of relying on someone remembering to hand-write a new assertion.
var knownConfigFields = map[string]bool{
	"SheetName": true, "WithHeadings": true, "StartRow": true, "EndRow": true,
	"StartCol": true, "EndCol": true, "SkipEmptyRows": true, "TrimSpaces": true,
	"ValidateData": true, "StopOnError": true, "Timeout": true, "PreallocateRows": true,
	"Logger": true, "Metrics": true, "ProgressCallback": true, "ErrorCallback": true,
	"Format": true, "CSV": true, "Converters": true, "RowMap": true,
	"StartCell": true, "AutoSizeColumns": true, "DocProperties": true,
	"StrictFormat": true,
}

// TestConfigClone_NoFieldForgotten cross-checks Config's actual field
// list against knownConfigFields, so a field added to Config but never
// wired into Clone() (and this list) surfaces immediately by name
// instead of silently losing its value on every Clone() call.
func TestConfigClone_NoFieldForgotten(t *testing.T) {
	typ := reflect.TypeOf(config.Config{})

	seen := make(map[string]bool, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		seen[name] = true
		if !knownConfigFields[name] {
			t.Errorf("Config gained field %q; add it to Clone(), to knownConfigFields, and ideally to TestConfigClone_ConvertersAndRowMap", name)
		}
	}
	for name := range knownConfigFields {
		if !seen[name] {
			t.Errorf("knownConfigFields lists %q, which no longer exists on Config; remove the stale entry", name)
		}
	}
}
