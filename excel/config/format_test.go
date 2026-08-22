package config

import "testing"

func TestFileFormat_String(t *testing.T) {
	tests := []struct {
		name string
		f    FileFormat
		want string
	}{
		{name: "xlsx", f: FileFormatXLSX, want: "xlsx"},
		{name: "csv", f: FileFormatCSV, want: "csv"},
		{name: "unknown", f: FileFormat(99), want: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigError_Error(t *testing.T) {
	err := &ConfigError{Field: "SheetName", Value: "", Message: "sheet name cannot be empty"}

	got := err.Error()
	want := "config error: field 'SheetName' (value: ): sheet name cannot be empty"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestConfig_Validate_CSVLineEnding_RejectsDoubleQuote(t *testing.T) {
	c := DefaultConfig()
	c.CSV.LineEnding = "\"\n"

	if err := c.Validate(); err == nil {
		t.Error("Validate() = nil, want error for line ending containing a double quote")
	}
}

func TestConfig_Validate_CSVDelimiter_RejectsUnsafeValues(t *testing.T) {
	tests := []struct {
		name      string
		delimiter rune
		wantErr   bool
	}{
		{name: "comma", delimiter: ',', wantErr: false},
		{name: "semicolon", delimiter: ';', wantErr: false},
		{name: "tab", delimiter: '\t', wantErr: false},
		{name: "zero value falls back to default", delimiter: 0, wantErr: false},
		{name: "newline rejected", delimiter: '\n', wantErr: true},
		{name: "carriage return rejected", delimiter: '\r', wantErr: true},
		{name: "double quote rejected", delimiter: '"', wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DefaultConfig()
			c.CSV.Delimiter = tt.delimiter
			err := c.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_CSVZeroValues_FilledWithDefaults(t *testing.T) {
	c := DefaultConfig()
	c.CSV = CSVOptions{}

	if err := c.Validate(); err != nil {
		t.Fatalf("Validate() unexpected error: %v", err)
	}

	defaults := DefaultCSVOptions()
	if c.CSV.Delimiter != defaults.Delimiter {
		t.Errorf("Delimiter = %q, want %q", c.CSV.Delimiter, defaults.Delimiter)
	}
	if c.CSV.LineEnding != defaults.LineEnding {
		t.Errorf("LineEnding = %q, want %q", c.CSV.LineEnding, defaults.LineEnding)
	}
	if c.CSV.TimeLayout != defaults.TimeLayout {
		t.Errorf("TimeLayout = %q, want %q", c.CSV.TimeLayout, defaults.TimeLayout)
	}
}

func TestConfig_Clone_CopiesFormatAndCSVFields(t *testing.T) {
	c := DefaultConfig()
	c.Format = FileFormatCSV
	c.CSV.Delimiter = ';'

	clone := c.Clone()

	if clone.Format != FileFormatCSV {
		t.Errorf("Format = %v, want %v", clone.Format, FileFormatCSV)
	}
	if clone.CSV.Delimiter != ';' {
		t.Errorf("CSV.Delimiter = %q, want %q", clone.CSV.Delimiter, ';')
	}
}

func TestDefaultConfig_Format_DefaultsToXLSX(t *testing.T) {
	c := DefaultConfig()
	if c.Format != FileFormatXLSX {
		t.Errorf("Format = %v, want %v", c.Format, FileFormatXLSX)
	}
	if !c.CSV.UseBOM {
		t.Error("CSV.UseBOM = false, want true by default")
	}
}
