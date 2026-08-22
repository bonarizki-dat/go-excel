package csvio

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"io"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// Reader decodes CSV records one at a time. It exists alongside ReadAll
// for streaming importers that cannot afford to load the whole input
// into memory the way ReadAll does.
type Reader struct {
	csv *csv.Reader
}

// NewReader wraps r for CSV decoding using opts.Delimiter. A leading
// UTF-8 BOM is stripped if present, and ragged records - rows with a
// different field count than the first - are tolerated; see ReadAll for
// why both matter for real-world CSV data.
func NewReader(r io.Reader, opts config.CSVOptions) *Reader {
	br := bufio.NewReader(r)

	if peeked, err := br.Peek(len(bom)); err == nil && bytes.Equal(peeked, bom) {
		_, _ = br.Discard(len(bom))
	}

	cr := csv.NewReader(br)
	if opts.Delimiter != 0 {
		cr.Comma = opts.Delimiter
	}
	cr.FieldsPerRecord = -1

	return &Reader{csv: cr}
}

// Read returns the next record, or io.EOF once the input is exhausted.
func (r *Reader) Read() ([]string, error) {
	return r.csv.Read()
}

// ReadAll decodes r as CSV into [][]string using opts.Delimiter.
//
// Ragged records - rows with a different field count than the first -
// are tolerated rather than rejected, since real-world CSV data
// (including files with no header row, or hand-edited exports) commonly
// has short trailing rows; encoding/csv's default behavior of erroring
// on those would make CSV import far stricter than XLSX import ever was.
//
// A leading UTF-8 BOM is stripped if present, regardless of
// opts.UseBOM: BOM detection on read does not need to match what the
// writer chose, since a byte-for-byte BOM either is or isn't there.
func ReadAll(r io.Reader, opts config.CSVOptions) ([][]string, error) {
	return NewReader(r, opts).csv.ReadAll()
}
