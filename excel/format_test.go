package excel

import "testing"

func TestFormatFromExtension_VariousInputs_ResolvesFormat(t *testing.T) {
	tests := []struct {
		name string
		file string
		want FileFormat
	}{
		{name: "csv lowercase", file: "report.csv", want: FileFormatCSV},
		{name: "csv uppercase", file: "REPORT.CSV", want: FileFormatCSV},
		{name: "csv mixed case", file: "Report.Csv", want: FileFormatCSV},
		{name: "xlsx", file: "report.xlsx", want: FileFormatXLSX},
		{name: "xlsm", file: "report.xlsm", want: FileFormatXLSX},
		{name: "xltx", file: "report.xltx", want: FileFormatXLSX},
		{name: "no extension", file: "report", want: FileFormatXLSX},
		{name: "unknown extension", file: "report.txt", want: FileFormatXLSX},
		{name: "empty filename", file: "", want: FileFormatXLSX},
		{name: "path with csv extension", file: "/tmp/exports/report.csv", want: FileFormatCSV},
		{name: "dotfile with no extension", file: ".gitignore", want: FileFormatXLSX},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatFromExtension(tt.file); got != tt.want {
				t.Errorf("FormatFromExtension(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

func TestDefaultCSVOptions_MatchesConfigDefaults(t *testing.T) {
	got := DefaultCSVOptions()
	if got.Delimiter != ',' || !got.UseBOM || got.LineEnding != "\r\n" || got.TimeLayout != "2006-01-02 15:04:05" {
		t.Errorf("DefaultCSVOptions() = %+v, want comma/BOM/CRLF/datetime", got)
	}
}

func TestFileFormat_String_ReturnsReadableName(t *testing.T) {
	tests := []struct {
		format FileFormat
		want   string
	}{
		{FileFormatXLSX, "xlsx"},
		{FileFormatCSV, "csv"},
		{FileFormat(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.format.String(); got != tt.want {
				t.Errorf("FileFormat(%d).String() = %q, want %q", tt.format, got, tt.want)
			}
		})
	}
}
