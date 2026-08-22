package stream

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bonarizki-dat/go-excel/excel/config"
)

// TestImportStream_ProcessorErrorDoesNotLeakReaderGoroutine is a
// regression test for a goroutine leak: when a processor returns an
// error, ImportStream used to return immediately without canceling
// the context passed to the reader goroutine (importer_read.go). With
// enough buffered rows, the reader would then block forever on
// `rowChan <- rowData` because nothing was left to drain rowChan,
// leaking the goroutine (and the row buffer, and everything it
// closed over) for the lifetime of the process.
//
// CSV format is used so a large row count can be generated quickly
// without going through excelize's XLSX writer.
func TestImportStream_ProcessorErrorDoesNotLeakReaderGoroutine(t *testing.T) {
	const numRows = 200_000

	var sb strings.Builder
	for i := 0; i < numRows; i++ {
		fmt.Fprintf(&sb, "%d,row-%d\n", i, i)
	}

	importer, err := NewStreamImporter(strings.NewReader(sb.String()),
		WithImportFormat(config.FileFormatCSV),
		WithImportWorkers(2),
	)
	if err != nil {
		t.Fatalf("NewStreamImporter failed: %v", err)
	}
	defer importer.Close()

	before := goroutineCountAfterSettling()

	wantErr := errors.New("processor error")
	processor := func(_ context.Context, _ int, _ []any) ([]any, error) {
		// Fail immediately so most of the 200k rows are still
		// unread when ImportStream returns, maximizing the chance
		// the reader goroutine is still blocked sending if the
		// leak were present.
		return nil, wantErr
	}

	if err := importer.ImportStream(context.Background(), processor); err == nil {
		t.Fatal("expected ImportStream to return an error, got nil")
	}

	after := goroutineCountAfterSettling()

	if after > before {
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		t.Fatalf("goroutine count grew from %d to %d after ImportStream returned; "+
			"reader goroutine likely leaked:\n%s", before, after, buf[:n])
	}
}

// goroutineCountAfterSettling returns runtime.NumGoroutine() after
// giving recently-finished goroutines a chance to be scheduled off,
// so the count reflects steady state rather than a transient peak.
func goroutineCountAfterSettling() int {
	var n int
	for i := 0; i < 5; i++ {
		runtime.Gosched()
		time.Sleep(10 * time.Millisecond)
		n = runtime.NumGoroutine()
	}
	return n
}
