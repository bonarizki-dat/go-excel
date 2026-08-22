// Command stream_export demonstrates StreamExporter: writing a large
// number of rows to an xlsx file through a channel, transforming rows
// concurrently across a worker pool, reporting progress, and reliably
// releasing resources with Close() even when the export fails partway
// through.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bonarizki-dat/go-excel/stream"
)

const rowCount = 50_000

func main() {
	// run holds the resource-owning code so its defers (closing the
	// file and the exporter) always execute before main exits,
	// including on the error paths below - calling log.Fatal directly
	// inside a function with pending defers would skip them.
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	f, err := os.Create("stream_export_output.xlsx")
	if err != nil {
		return err
	}
	// Close the destination file only after the exporter (and its
	// buffered excelize writer) has finished writing to it.
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("close output file: %v", err)
		}
	}()

	exp, err := stream.NewStreamExporter(f,
		stream.WithExportHeadings([]string{"ID", "Name", "Score"}),
		stream.WithExportWorkers(4),
		stream.WithRowTransform(scoreBand),
		stream.WithExportProgress(reportProgress, 10_000),
	)
	if err != nil {
		return fmt.Errorf("create stream exporter: %w", err)
	}
	// Close() releases the underlying excelize file's resources. It is
	// idempotent and safe to defer immediately: if ExportStream below
	// fails, the exporter has already closed the file on that error
	// path, and this call becomes a no-op instead of a double-close.
	defer func() {
		if err := exp.Close(); err != nil {
			log.Printf("close exporter: %v", err)
		}
	}()

	// A real caller would derive ctx from an incoming request or from
	// signal.NotifyContext; a deadline is used here only so the example
	// terminates on its own if something unexpected stalls it.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows := make(chan []any, 256)
	go generateRows(rows)

	if err := exp.ExportStream(ctx, rows); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return fmt.Errorf("export canceled: %w", err)
		}
		return fmt.Errorf("export failed: %w", err)
	}

	fmt.Printf("exported %d rows to stream_export_output.xlsx\n", exp.GetRowCount())
	return nil
}

// generateRows sends rowCount synthetic rows to rows and closes it when
// done, the same shape a caller would use to stream rows from a
// database cursor or paginated API instead of loading them all upfront.
func generateRows(rows chan<- []any) {
	defer close(rows)
	for i := 1; i <= rowCount; i++ {
		rows <- []any{i, fmt.Sprintf("user-%d", i), i % 101}
	}
}

// scoreBand is a RowTransformer: it runs concurrently across the
// exporter's worker pool, so it must not mutate row (the caller may
// reuse it) and must be safe for concurrent use.
func scoreBand(_ context.Context, _ uint64, row []any) ([]any, error) {
	score, _ := row[2].(int)
	band := "low"
	switch {
	case score >= 80:
		band = "high"
	case score >= 40:
		band = "medium"
	}
	return []any{row[0], row[1], band}, nil
}

func reportProgress(current uint64, message string) {
	fmt.Printf("progress: %d rows (%s)\n", current, message)
}
