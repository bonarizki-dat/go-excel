package stream

import (
	"bytes"
	"context"
	"fmt"
	"testing"
)

// TestExportStream_RowOrderAtScale exports 10,000 rows across a range of
// worker counts and verifies the output file has every row, in input
// order, regardless of how workers interleave completion.
//
// Before the ordered pipeline rewrite, exporting 2,000 rows with the
// default worker count produced 7 to 204 rows of corrupted, out-of-order
// output across repeated runs, because workers wrote directly to a
// shared, non-thread-safe StreamWriter and the Excel row position was
// derived from scheduling order rather than input order.
func TestExportStream_RowOrderAtScale(t *testing.T) {
	const rowCount = 10000

	for _, workers := range []int{1, 4, 64} {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			var buf bytes.Buffer
			exporter, err := NewStreamExporter(&buf, WithExportWorkers(workers))
			if err != nil {
				t.Fatalf("NewStreamExporter: %v", err)
			}

			rows := make(chan []any, 10)
			go func() {
				defer close(rows)
				for i := 1; i <= rowCount; i++ {
					rows <- []any{fmt.Sprintf("row-%d", i)}
				}
			}()

			if err := exporter.ExportStream(context.Background(), rows); err != nil {
				t.Fatalf("ExportStream failed: %v", err)
			}

			got := readBackXLSX(t, &buf, "Sheet1")
			if len(got) != rowCount {
				t.Fatalf("rows written: %d, expected %d", len(got), rowCount)
			}

			for i, row := range got {
				want := fmt.Sprintf("row-%d", i+1)
				if len(row) == 0 || row[0] != want {
					t.Fatalf("row %d: want %q, got %v", i+1, want, row)
				}
			}
		})
	}
}
