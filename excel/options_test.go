package excel

import (
	// reflect supplies the destination type argument in Converter test values.
	"reflect"
	// strings implements the sample Converter used in TestWithConverters.
	"strings"
	// testing provides assertions for functional options behavior.
	"testing"

	// config holds the Config type modified by options.
	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestOptionsModifyConfig verifies functional options correctly mutate Config fields.
func TestOptionsModifyConfig(t *testing.T) {
	// Start from sensible defaults
	cfg := config.DefaultConfig()
	// Apply sheet name to verify naming option wiring
	WithSheetName("Users")(cfg)
	// Disable headings to exercise alternative path
	WithHeadings(false)(cfg)
	// Limit row span to validate range assignment (1-based inclusive
	// first=3, last=10 maps to 0-indexed StartRow=2, EndRow=10)
	WithRows(3, 10)(cfg)
	// Limit column span to validate range assignment (1-based inclusive
	// first=2, last=5 maps to 0-indexed StartCol=1, EndCol=5)
	WithColumns(2, 5)(cfg)
	// Disable skipping empty rows to validate flag wiring
	WithSkipEmptyRows(false)(cfg)
	// Disable trimming to validate flag wiring
	WithTrimSpaces(false)(cfg)
	// Turn on validation to exercise validation flag
	WithValidation(true)(cfg)
	// Allow continuing on errors to validate error handling flag
	WithStopOnError(false)(cfg)
	// Disable preallocation to validate performance flag
	WithPreallocation(false)(cfg)
	// Validate sheet name applied
	if cfg.SheetName != "Users" {
		t.Fatalf("SheetName not applied")
	}
	// Validate data handling fields applied
	if cfg.WithHeadings || cfg.StartRow != 2 || cfg.EndRow != 10 || cfg.StartCol != 1 || cfg.EndCol != 5 {
		t.Fatalf("data handling fields not applied")
	}
	// Validate boolean flags applied
	if cfg.SkipEmptyRows || cfg.TrimSpaces {
		t.Fatalf("flags not applied")
	}
	// Validate validation and error handling flags applied
	if !cfg.ValidateData || cfg.StopOnError {
		t.Fatalf("validation/error flags not applied")
	}
	// Validate performance fields applied
	if cfg.PreallocateRows {
		t.Fatalf("memory fields not applied")
	}
}

// TestWithConverters verifies the option stores the map on Config
// unchanged, so RowMapper.SetConverters receives exactly what the
// caller registered.
func TestWithConverters(t *testing.T) {
	cfg := config.DefaultConfig()
	converters := map[string]Converter{
		"tags": func(raw string, _ reflect.Type) (any, error) {
			return strings.Split(raw, ";"), nil
		},
	}

	WithConverters(converters)(cfg)

	if len(cfg.Converters) != 1 {
		t.Fatalf("len(cfg.Converters) = %d, want 1", len(cfg.Converters))
	}
	if _, ok := cfg.Converters["tags"]; !ok {
		t.Error("cfg.Converters missing \"tags\" entry")
	}
}

// TestWithRowMap verifies the option stores the function on
// Config.RowMap unchanged.
func TestWithRowMap(t *testing.T) {
	cfg := config.DefaultConfig()
	called := false
	fn := func(row []any) ([]any, error) {
		called = true
		return row, nil
	}

	WithRowMap(fn)(cfg)

	if cfg.RowMap == nil {
		t.Fatal("cfg.RowMap = nil, want the function passed to WithRowMap")
	}
	if _, err := cfg.RowMap([]any{"x"}); err != nil {
		t.Fatalf("cfg.RowMap(...) error = %v, want nil", err)
	}
	if !called {
		t.Error("cfg.RowMap did not invoke the underlying function")
	}
}

// TestWithCustomStartCell verifies the option stores the cell reference
// on Config.StartCell unchanged.
func TestWithCustomStartCell(t *testing.T) {
	cfg := config.DefaultConfig()
	WithCustomStartCell("C4")(cfg)
	if cfg.StartCell != "C4" {
		t.Errorf("cfg.StartCell = %q, want %q", cfg.StartCell, "C4")
	}
}

// TestShouldAutoSize verifies the option stores the flag on
// Config.AutoSizeColumns unchanged.
func TestShouldAutoSize(t *testing.T) {
	cfg := config.DefaultConfig()
	ShouldAutoSize(true)(cfg)
	if !cfg.AutoSizeColumns {
		t.Error("cfg.AutoSizeColumns = false, want true")
	}
}

// TestWithProperties verifies the option stores the DocProperties
// value on Config.DocProperties unchanged.
func TestWithProperties(t *testing.T) {
	cfg := config.DefaultConfig()
	props := DocProperties{Title: "Report", Creator: "Test"}
	WithProperties(props)(cfg)
	if cfg.DocProperties != props {
		t.Errorf("cfg.DocProperties = %+v, want %+v", cfg.DocProperties, props)
	}
}
