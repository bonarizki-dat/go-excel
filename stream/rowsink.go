package stream

import (
	"github.com/bonarizki-dat/go-excel/excel/csvio"
	"github.com/xuri/excelize/v2"
)

// streamRowSink receives rows during ExportStream, independent of the
// on-disk format. It lets exporter_write.go drive XLSX and CSV output
// through the same code, instead of branching on format at every write.
//
// rowNum is 1-indexed and includes the heading row, matching how
// dataRowOffset numbers rows for XLSX; csvStreamSink ignores it since a
// CSV writer only ever appends.
type streamRowSink interface {
	writeRow(rowNum int, row []any) error
	flush() error
}

// xlsxStreamSink adapts excelize.StreamWriter to streamRowSink.
type xlsxStreamSink struct {
	sw *excelize.StreamWriter
}

func newXLSXStreamSink(sw *excelize.StreamWriter) *xlsxStreamSink {
	return &xlsxStreamSink{sw: sw}
}

func (s *xlsxStreamSink) writeRow(rowNum int, row []any) error {
	cellRef, err := excelize.CoordinatesToCellName(1, rowNum)
	if err != nil {
		return err
	}
	return s.sw.SetRow(cellRef, row)
}

func (s *xlsxStreamSink) flush() error {
	return s.sw.Flush()
}

// csvStreamSink adapts csvio.Writer to streamRowSink. Unlike the batch
// csvSink in excel/exporter, it writes each row straight to the
// destination writer as it arrives instead of buffering the whole
// document, since that buffering is exactly what streaming exists to
// avoid.
type csvStreamSink struct {
	w      *csvio.Writer
	layout string
}

func newCSVStreamSink(w *csvio.Writer, layout string) *csvStreamSink {
	return &csvStreamSink{w: w, layout: layout}
}

func (s *csvStreamSink) writeRow(_ int, row []any) error {
	fields := make([]string, len(row))
	for i, v := range row {
		fields[i] = csvio.FormatValue(v, s.layout)
	}
	return s.w.WriteRecord(fields)
}

func (s *csvStreamSink) flush() error {
	return s.w.Flush()
}
