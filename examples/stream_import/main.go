// Command stream_import demonstrates StreamImporter: reading rows from
// an xlsx file one at a time through a RowProcessor callback, handling
// per-row errors without aborting the whole import, and reliably
// releasing resources with Close().
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/bonarizki-dat/go-excel/stream"
)

func main() {
	// run holds the resource-owning code so its defers (closing the
	// file and the importer) always execute before main exits,
	// including on the error paths below - calling log.Fatal directly
	// inside a function with pending defers would skip them.
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	path := "stream_export_output.xlsx"
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("%s not found; run examples/stream_export first: %w", path, err)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("close input file: %v", err)
		}
	}()

	imp, err := stream.NewStreamImporter(f,
		stream.WithImportHeadings(),
		stream.WithImportWorkers(4),
	)
	if err != nil {
		return err
	}
	// Close() releases the importer's underlying reader and worker
	// pool. Deferring it immediately after a successful constructor
	// call, rather than only on the success path below, ensures
	// resources are released even if ImportStream returns an error or
	// the context is canceled partway through.
	defer func() {
		if err := imp.Close(); err != nil {
			log.Printf("close importer: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var highScores int
	processor := func(ctx context.Context, rowNum int, row []any) ([]any, error) {
		// A processor can itself stop the import early by returning
		// ctx.Err() once it observes cancellation, instead of doing
		// more unnecessary work first.
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if len(row) < 3 {
			return nil, fmt.Errorf("row %d: expected 3 columns, got %d", rowNum, len(row))
		}
		if row[2] == "high" {
			highScores++
		}
		return row, nil
	}

	if err := imp.ImportStream(ctx, processor); err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("import canceled: %w", err)
		}
		return fmt.Errorf("import failed: %w", err)
	}

	stats := imp.GetStats()
	fmt.Printf("processed %d rows (%d high scores), %d worker(s), %d failed\n",
		imp.GetRowCount(), highScores, stats.TotalWorkers, stats.TasksFailed)
	return nil
}
