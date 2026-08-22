package stream

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestXLSXStreamSink_WriteRow_InvalidRowNum verifies writeRow surfaces
// excelize.CoordinatesToCellName's error for a non-positive rowNum
// instead of it going unnoticed.
func TestXLSXStreamSink_WriteRow_InvalidRowNum(t *testing.T) {
	f := excelize.NewFile()
	sw, err := f.NewStreamWriter("Sheet1")
	if err != nil {
		t.Fatalf("NewStreamWriter: %v", err)
	}
	sink := newXLSXStreamSink(sw)

	if err := sink.writeRow(0, []any{"a"}); err == nil {
		t.Error("writeRow(0, ...) = nil error, want error")
	}
}

// TestNewXLSXStreamSource_NonexistentSheet verifies newXLSXStreamSource
// surfaces excelize.File.Rows' error for a sheet that does not exist.
func TestNewXLSXStreamSource_NonexistentSheet(t *testing.T) {
	f := excelize.NewFile()

	if _, err := newXLSXStreamSource(f, "DoesNotExist"); err == nil {
		t.Error("newXLSXStreamSource for a nonexistent sheet = nil error, want error")
	}
}
