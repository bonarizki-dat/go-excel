package stream

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"github.com/bonarizki-dat/go-excel/excel"
)

// TestExportToSheets_ChannelAndStaticData verifies that ExportToSheets
// writes each sheet's rows to the output file in input order, whether
// the sheet is fed from a channel or from a static slice. Unlike
// ExportStream, this path writes directly from a single goroutine with
// no worker pool, but it had no read-back coverage at all before this
// test: prior assertions elsewhere in this package relied on
// GetRowCount(), which cannot detect a corrupt output file.
func TestExportToSheets_ChannelAndStaticData(t *testing.T) {
	var buf bytes.Buffer
	exporter, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	usersChan := make(chan []any, 10)
	go func() {
		defer close(usersChan)
		for i := range 5 {
			usersChan <- []any{i, "user"}
		}
	}()

	sheets := []excel.Sheet{
		{Name: "Users", DataChannel: usersChan},
		{Name: "Orders", Data: [][]any{
			{"order-1", 100},
			{"order-2", 200},
			{"order-3", 300},
		}},
	}

	err = exporter.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions())
	if err != nil {
		t.Fatalf("ExportToSheets failed: %v", err)
	}

	users := readBackXLSX(t, &buf, "Users")
	if len(users) != 5 {
		t.Fatalf("Users: expected 5 rows, got %d", len(users))
	}
	for i, row := range users {
		want := []string{strconv.Itoa(i), "user"}
		if row[0] != want[0] || row[1] != want[1] {
			t.Errorf("Users row %d: expected %v, got %v", i+1, want, row)
		}
	}

	orders := readBackXLSX(t, &buf, "Orders")
	want := [][]string{
		{"order-1", "100"},
		{"order-2", "200"},
		{"order-3", "300"},
	}
	if len(orders) != len(want) {
		t.Fatalf("Orders: expected %d rows, got %d", len(want), len(orders))
	}
	for i := range want {
		if orders[i][0] != want[i][0] || orders[i][1] != want[i][1] {
			t.Errorf("Orders row %d: expected %v, got %v", i+1, want[i], orders[i])
		}
	}
}
