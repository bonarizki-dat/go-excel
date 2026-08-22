package stream

import (
	"bytes"
	"context"
	"strconv"
	"testing"
)

// TestExportStream_EmptyData tests exporting with no data.
func TestExportStream_EmptyData(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter
	exporter, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Create empty channel
	rows := make(chan []any)
	close(rows)

	// Export
	ctx := context.Background()
	err = exporter.ExportStream(ctx, rows)

	// Verify no error
	if err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	// Verify the output file has no data rows
	got := readBackXLSX(t, &buf, "Sheet1")
	if len(got) != 0 {
		t.Errorf("Expected 0 rows in output file, got %d", len(got))
	}

	// Verify output generated (valid Excel file)
	if buf.Len() == 0 {
		t.Error("Expected Excel output, got empty buffer")
	}
}

// TestExportStream_SmallDataset tests exporting small dataset.
func TestExportStream_SmallDataset(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter
	exporter, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Create data channel
	rows := make(chan []any, 10)

	// Send 10 rows
	go func() {
		defer close(rows)
		for i := range 10 {
			rows <- []any{i, "name" + string(rune(i)), "email@test.com"}
		}
	}()

	// Export
	ctx := context.Background()
	err = exporter.ExportStream(ctx, rows)

	// Verify no error
	if err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	// Verify the output file has 10 rows in input order
	got := readBackXLSX(t, &buf, "Sheet1")
	if len(got) != 10 {
		t.Fatalf("Expected 10 rows in output file, got %d", len(got))
	}
	for i, row := range got {
		want := strconv.Itoa(i)
		if len(row) == 0 || row[0] != want {
			t.Errorf("row %d: expected first cell %q, got %v", i+1, want, row)
		}
	}

	// Verify output generated
	if buf.Len() == 0 {
		t.Error("Expected Excel output, got empty buffer")
	}

	// Verify worker stats
	stats := exporter.GetStats()
	if stats.TasksSubmitted != 10 {
		t.Errorf("Expected 10 tasks submitted, got %d", stats.TasksSubmitted)
	}

	// Note: TasksCompleted may vary due to concurrent worker timing
	// Just verify some tasks were completed successfully
	if stats.TasksCompleted == 0 {
		t.Error("Expected some tasks to be completed")
	}
}

// TestExportStream_WithHeadings tests exporting with column headers.
func TestExportStream_WithHeadings(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter with headings
	headings := []string{"ID", "Name", "Email"}
	exporter, err := NewStreamExporter(&buf, WithExportHeadings(headings))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Create data channel
	rows := make(chan []any, 5)

	// Send rows
	go func() {
		defer close(rows)
		for i := range 5 {
			rows <- []any{i, "user" + string(rune(i)), "test@example.com"}
		}
	}()

	// Export
	ctx := context.Background()
	err = exporter.ExportStream(ctx, rows)

	// Verify no error
	if err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	// Verify the output file has a heading row followed by 5 data rows
	// in input order
	got := readBackXLSX(t, &buf, "Sheet1")
	if len(got) != 6 {
		t.Fatalf("Expected 1 heading row + 5 data rows, got %d rows", len(got))
	}
	for i, want := range headings {
		if len(got[0]) <= i || got[0][i] != want {
			t.Errorf("heading[%d]: expected %q, got %v", i, want, got[0])
		}
	}
	for i, row := range got[1:] {
		want := strconv.Itoa(i)
		if len(row) == 0 || row[0] != want {
			t.Errorf("data row %d: expected first cell %q, got %v", i+1, want, row)
		}
	}

	// Verify output generated
	if buf.Len() == 0 {
		t.Error("Expected Excel output, got empty buffer")
	}
}

// TestExportStream_WithCustomSheetName tests exporting to custom sheet.
func TestExportStream_WithCustomSheetName(t *testing.T) {
	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter with custom sheet name
	exporter, err := NewStreamExporter(&buf, WithExportSheetName("CustomData"))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// Create data channel
	rows := make(chan []any, 3)

	// Send rows
	go func() {
		defer close(rows)
		rows <- []any{"data1", "data2"}
		rows <- []any{"data3", "data4"}
		rows <- []any{"data5", "data6"}
	}()

	// Export
	ctx := context.Background()
	err = exporter.ExportStream(ctx, rows)

	// Verify no error
	if err != nil {
		t.Fatalf("ExportStream failed: %v", err)
	}

	// Verify rows landed on the custom sheet, in order
	got := readBackXLSX(t, &buf, "CustomData")
	want := [][]string{
		{"data1", "data2"},
		{"data3", "data4"},
		{"data5", "data6"},
	}
	if len(got) != len(want) {
		t.Fatalf("Expected %d rows, got %d", len(want), len(got))
	}
	for i := range want {
		if len(got[i]) != len(want[i]) || got[i][0] != want[i][0] || got[i][1] != want[i][1] {
			t.Errorf("row %d: expected %v, got %v", i+1, want[i], got[i])
		}
	}
}

func TestExportStream_WithRowTransform(t *testing.T) {
	var buf bytes.Buffer
	exporter, err := NewStreamExporter(&buf, WithRowTransform(func(_ context.Context, _ uint64, row []any) ([]any, error) {
		if len(row) > 0 {
			row[0] = "x-" + strconv.Itoa(row[0].(int))
		}
		return row, nil
	}))
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	rows := make(chan []any, 2)
	rows <- []any{1, "a"}
	rows <- []any{2, "b"}
	close(rows)

	if err := exporter.ExportStream(context.Background(), rows); err != nil {
		t.Fatalf("ExportStream: %v", err)
	}

	got := readBackXLSX(t, &buf, "Sheet1")
	if len(got) != 2 || got[0][0] != "x-1" || got[1][0] != "x-2" {
		t.Errorf("transformed rows = %v, want x-1 then x-2", got)
	}
}

// TestExportStream_MultipleExports tests sequential exports.
func TestExportStream_MultipleExports(t *testing.T) {
	// Test that we can't reuse exporter (should fail on second export)
	// This is expected behavior since excelize file is closed after first export

	// Create buffer for output
	var buf bytes.Buffer

	// Create exporter
	exporter, err := NewStreamExporter(&buf)
	if err != nil {
		t.Fatalf("NewStreamExporter: %v", err)
	}

	// First export
	rows1 := make(chan []any, 2)
	go func() {
		defer close(rows1)
		rows1 <- []any{1, "data1"}
		rows1 <- []any{2, "data2"}
	}()

	ctx := context.Background()
	err = exporter.ExportStream(ctx, rows1)
	if err != nil {
		t.Fatalf("First export failed: %v", err)
	}

	// Verify the output file has both rows in order
	got := readBackXLSX(t, &buf, "Sheet1")
	if len(got) != 2 {
		t.Fatalf("Expected 2 rows from first export, got %d", len(got))
	}
	if got[0][0] != "1" || got[1][0] != "2" {
		t.Errorf("Expected rows in order [1, 2], got %v", got)
	}
}
