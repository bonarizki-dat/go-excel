package exporter

import (
	"bytes"
	"context"
	"io"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/config"
	"github.com/bonarizki-dat/go-excel/excel/csvio"
)

// csvSink is the excel.RowSink implementation for config.FileFormatCSV.
// CSV has no native concept of styling, merged cells, column widths, or
// multiple sheets, so it only ever writes the values passed to WriteRow;
// warning about dropped features is BasicExporter's responsibility, not
// this sink's.
//
// Rows are buffered in memory and only handed to the caller's io.Writer
// in Finalize, matching how xlsxSink (and excelize generally) behaves:
// BasicExporter is a batch exporter, not a streaming one. The stream
// package has its own CSV sink that writes incrementally.
type csvSink struct {
	buf    bytes.Buffer
	writer *csvio.Writer
	layout string
}

// newCSVSink creates a csvSink using opts for delimiter, BOM, line
// ending, and time formatting.
func newCSVSink(opts config.CSVOptions) (*csvSink, error) {
	s := &csvSink{layout: opts.TimeLayout}

	w, err := csvio.NewWriter(&s.buf, opts)
	if err != nil {
		return nil, excel.NewExportError("NewWriter", "", 0, 0, err)
	}
	s.writer = w

	return s, nil
}

// WriteRow formats each value with csvio.FormatValue and writes the
// resulting record. rowNum and isHeader are accepted to satisfy
// excel.RowSink; CSV has no per-role styling, so both are only used for
// error context.
func (s *csvSink) WriteRow(ctx context.Context, rowNum int, row []any, _ bool) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fields := make([]string, len(row))
	for i, v := range row {
		fields[i] = csvio.FormatValue(v, s.layout)
	}

	if err := s.writer.WriteRecord(fields); err != nil {
		return excel.NewExportError("WriteRecord", "", rowNum, 0, err)
	}
	return nil
}

// Finalize flushes the buffered CSV data to w.
func (s *csvSink) Finalize(_ context.Context, w io.Writer) error {
	if err := s.writer.Flush(); err != nil {
		return excel.NewExportError("Flush", "", 0, 0, err)
	}
	if _, err := w.Write(s.buf.Bytes()); err != nil {
		return excel.NewExportError("Write", "", 0, 0, err)
	}
	return nil
}

// Close is a no-op: csvSink holds only an in-memory buffer, nothing that
// needs releasing.
func (s *csvSink) Close() error {
	return nil
}
