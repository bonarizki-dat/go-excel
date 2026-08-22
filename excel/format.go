package excel

import (
	"path/filepath"
	"strings"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// FileFormat identifies the on-disk document format. See
// config.FileFormat for the full documentation; this is an alias so
// callers can write excel.FileFormatCSV instead of reaching into the
// config subpackage.
type FileFormat = config.FileFormat

// FileFormatXLSX and FileFormatCSV are the supported document formats.
// FileFormatXLSX also covers the other OOXML variants excelize accepts
// (.xlsm, .xltx, .xltm, .xlam).
const (
	FileFormatXLSX = config.FileFormatXLSX
	FileFormatCSV  = config.FileFormatCSV
)

// CSVOptions is an alias for config.CSVOptions; see that type for field
// documentation.
type CSVOptions = config.CSVOptions

// DefaultCSVOptions returns the CSV encoding/decoding defaults: comma
// delimiter, UTF-8 BOM enabled, CRLF line endings, and a
// "2006-01-02 15:04:05" time layout.
func DefaultCSVOptions() CSVOptions {
	return config.DefaultCSVOptions()
}

// DocProperties is an alias for config.DocProperties; see that type
// for field documentation.
type DocProperties = config.DocProperties

// csvExtensions lists file extensions that resolve to FileFormatCSV.
// Everything else, including unknown extensions, resolves to
// FileFormatXLSX so that ExportToFile keeps working the way it always
// has for OOXML variants excelize accepts.
var csvExtensions = map[string]bool{
	".csv": true,
}

// FormatFromExtension resolves the FileFormat implied by a filename's
// extension. The comparison is case-insensitive. Filenames with no
// recognized CSV extension resolve to FileFormatXLSX, which matches the
// library's historical behavior of delegating everything to
// excelize.SaveAs / excelize.OpenFile.
func FormatFromExtension(name string) FileFormat {
	ext := strings.ToLower(filepath.Ext(name))
	if csvExtensions[ext] {
		return FileFormatCSV
	}
	return FileFormatXLSX
}
