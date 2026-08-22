package stream

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

// readBackXLSX opens buf as an XLSX file and returns the rows of sheet
// exactly as excelize reports them after a round trip through the file
// format.
//
// Stream export assertions must use this helper rather than
// exporter.GetRowCount(), which only reflects the writer's own
// bookkeeping. Before the ordered pipeline rewrite, GetRowCount agreed
// with itself while the actual output file was missing or misordered
// rows, which is why that bug shipped with a green test suite.
func readBackXLSX(t *testing.T, buf *bytes.Buffer, sheet string) [][]string {
	t.Helper()

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("readBackXLSX: open: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("readBackXLSX: close: %v", err)
		}
	}()

	rows, err := f.GetRows(sheet)
	if err != nil {
		t.Fatalf("readBackXLSX: get rows: %v", err)
	}
	return rows
}
