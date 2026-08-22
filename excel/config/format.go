package config

// FileFormat identifies the on-disk document format that an export writes
// or an import reads. It is distinct from the Excel number-format codes
// in excel/formatter.go (FormatDate, FormatCurrency, and so on), which
// describe how a cell's value is displayed, not what kind of file it
// lives in.
type FileFormat int

const (
	// FileFormatXLSX is the default format, handled by excelize. It also
	// covers the other OOXML variants excelize.SaveAs accepts (.xlsm,
	// .xltx, .xltm, .xlam).
	FileFormatXLSX FileFormat = iota

	// FileFormatCSV is a plain-text, single-sheet format with no styling,
	// merged cells, or formulas. Features that do not translate to CSV
	// are dropped and reported through Config.Logger.Warn instead of
	// failing the export or import.
	FileFormatCSV
)

// String returns a human-readable name for the format, used in log
// messages and error text.
func (f FileFormat) String() string {
	switch f {
	case FileFormatXLSX:
		return "xlsx"
	case FileFormatCSV:
		return "csv"
	default:
		return "unknown"
	}
}

// CSVOptions controls how CSV documents are encoded and decoded. It has
// no effect on FileFormatXLSX operations.
type CSVOptions struct {
	// Delimiter separates fields on a line. Default: ','.
	Delimiter rune

	// UseBOM prepends a UTF-8 byte order mark to exported CSV data so
	// that Excel detects the encoding correctly instead of misreading
	// accented and non-ASCII characters. Default: true.
	UseBOM bool

	// LineEnding terminates each record. Default: "\r\n", which matches
	// the CSV RFC and what Excel itself writes. Use "\n" for Unix-style
	// output.
	LineEnding string

	// TimeLayout formats time.Time values using time.Time.Format.
	// Default: "2006-01-02 15:04:05".
	TimeLayout string
}

// DefaultCSVOptions returns the CSVOptions used when a Config has not
// customized them: comma-delimited, BOM enabled, CRLF line endings, and
// a date-time layout that both Excel and common CSV consumers parse
// unambiguously.
func DefaultCSVOptions() CSVOptions {
	return CSVOptions{
		Delimiter:  ',',
		UseBOM:     true,
		LineEnding: "\r\n",
		TimeLayout: "2006-01-02 15:04:05",
	}
}

// DocProperties sets a subset of an XLSX workbook's OOXML core
// properties (the fields visible under a spreadsheet application's
// "Document Properties" or "Info" panel). It is a typed struct rather
// than a map[string]string so a misspelled property name is a compiler
// error instead of a silently ignored key; it does not expose the full
// OOXML core-properties surface (ContentStatus, Identifier, Revision,
// Language, and the Created/Modified timestamps are not included).
// Zero-value fields are left unset, leaving excelize's own defaults in
// place. Ignored for CSV export, which has no document properties.
type DocProperties struct {
	Title       string
	Subject     string
	Creator     string
	Keywords    string
	Description string
	Category    string
	Version     string
}
