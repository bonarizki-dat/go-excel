package stream

import (
	"io"

	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/excel/csvio"
	"github.com/xuri/excelize/v2"
)

// streamRowSource yields decoded rows one at a time regardless of the
// on-disk format, so readHeadings and readRows in importer_read.go stay
// format-agnostic. A single instance is created once in NewStreamImporter
// and shared between readHeadings and readRows, since a CSV stream can
// only be read forward once; reusing the same instance for XLSX keeps
// both formats' row numbering identical instead of needing separate
// code paths.
type streamRowSource interface {
	// next returns the next row's cells, or io.EOF once the source is
	// exhausted.
	next() ([]string, error)

	// close releases resources held by the source.
	close() error
}

// xlsxStreamSource adapts excelize's row iterator to streamRowSource.
type xlsxStreamSource struct {
	rows *excelize.Rows
}

// newXLSXStreamSource opens a row iterator over sheetName in file.
func newXLSXStreamSource(file *excelize.File, sheetName string) (*xlsxStreamSource, error) {
	rows, err := file.Rows(sheetName)
	if err != nil {
		return nil, err
	}
	return &xlsxStreamSource{rows: rows}, nil
}

func (s *xlsxStreamSource) next() ([]string, error) {
	if !s.rows.Next() {
		if err := s.rows.Error(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}
	return s.rows.Columns()
}

func (s *xlsxStreamSource) close() error {
	return s.rows.Close()
}

// csvStreamSource adapts csvio.Reader to streamRowSource.
type csvStreamSource struct {
	r *csvio.Reader
}

// newCSVStreamSource wraps r for row-at-a-time CSV decoding using opts.
func newCSVStreamSource(r io.Reader, opts config.CSVOptions) *csvStreamSource {
	return &csvStreamSource{r: csvio.NewReader(r, opts)}
}

func (s *csvStreamSource) next() ([]string, error) {
	return s.r.Read()
}

// close is a no-op: csvStreamSource does not own the underlying reader.
func (s *csvStreamSource) close() error {
	return nil
}
