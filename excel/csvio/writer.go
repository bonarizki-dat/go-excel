package csvio

import (
	"encoding/csv"
	"io"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// bom is the three-byte UTF-8 byte order mark. Prepending it to a CSV
// file signals UTF-8 encoding explicitly, which stops Excel guessing a
// legacy codepage and mangling accented or otherwise non-ASCII
// characters.
var bom = []byte{0xEF, 0xBB, 0xBF}

// Writer encodes records as CSV using the delimiter, line ending, and
// BOM settings from a config.CSVOptions. It wraps encoding/csv.Writer,
// which buffers internally, so Flush must be called after the last
// WriteRecord.
type Writer struct {
	csv *csv.Writer
}

// NewWriter wraps w for CSV output. If opts.UseBOM is set, the BOM is
// written to w immediately, before any record, regardless of when the
// first WriteRecord call happens.
func NewWriter(w io.Writer, opts config.CSVOptions) (*Writer, error) {
	if opts.UseBOM {
		if _, err := w.Write(bom); err != nil {
			return nil, err
		}
	}

	cw := csv.NewWriter(w)
	if opts.Delimiter != 0 {
		cw.Comma = opts.Delimiter
	}
	cw.UseCRLF = opts.LineEnding == "\r\n"

	return &Writer{csv: cw}, nil
}

// WriteRecord writes one CSV record. Fields containing the delimiter, a
// quote, or a newline are quoted automatically by encoding/csv.
func (w *Writer) WriteRecord(fields []string) error {
	return w.csv.Write(fields)
}

// Flush flushes buffered data to the underlying writer and returns the
// first write error encountered, if any. Must be called after the last
// WriteRecord; encoding/csv.Writer does not write through immediately.
func (w *Writer) Flush() error {
	w.csv.Flush()
	return w.csv.Error()
}
