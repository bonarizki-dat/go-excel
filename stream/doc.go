/*
Package stream provides row-at-a-time export and import for Excel files,
built on top of excelize's StreamWriter and streaming row reader.

Streaming reduces peak memory versus the excel package for large exports
by writing rows as they are produced instead of building the whole sheet
in memory first. It is not constant memory: internal buffers (the reorder
buffer, worker queues) scale with worker count, and StreamImporter loads
the workbook into memory via excelize.OpenReader when it opens the file.
Measure with the benchmarks in this package rather than assuming a fixed
memory ceiling.

# Basic Usage

Export rows from a channel:

	f, _ := os.Create("output.xlsx")
	defer f.Close()

	exp, err := stream.NewStreamExporter(f, stream.WithExportWorkers(4))
	if err != nil {
		log.Fatal(err)
	}

	rows := make(chan []any, 100)
	go func() {
		defer close(rows)
		for i := 0; i < 100000; i++ {
			rows <- []any{i, fmt.Sprintf("row-%d", i)}
		}
	}()

	if err := exp.ExportStream(context.Background(), rows); err != nil {
		log.Fatal(err)
	}

A StreamExporter is single-use: ExportStream writes and closes the
underlying file when it returns. Create a new StreamExporter for each
export.

Import rows one at a time:

	f, _ := os.Open("large.xlsx")
	defer f.Close()

	imp, err := stream.NewStreamImporter(f, stream.WithImportWorkers(4))
	if err != nil {
		log.Fatal(err)
	}
	defer imp.Close()

	err = imp.ImportStream(context.Background(), func(ctx context.Context, rowNum int, row []any) ([]any, error) {
		log.Printf("row %d: %v", rowNum, row)
		return row, nil
	})

# Row Transformation

ExportStream accepts an optional RowTransformer, run across a worker pool
(sized by WithExportWorkers) before rows are written in their original order:

	transform := func(ctx context.Context, rowNum uint64, row []any) ([]any, error) {
		return append(row, rowNum), nil
	}
	exp, err := stream.NewStreamExporter(f, stream.WithRowTransform(transform), stream.WithExportWorkers(8))

WithExportWorkers has no effect without WithRowTransform: rows still write in
order through a single writer goroutine either way.

# Multi-Sheet Support

StreamExporter.ExportToSheets writes multiple excel.Sheet values to one
workbook. Unlike ExportStream, each sheet is written synchronously by the
calling goroutine (no worker pool, no RowTransformer):

	sheets := []excel.Sheet{
		{Name: "Users", Data: userRows},
		{Name: "Orders", DataChannel: orderChan},
	}
	err := exp.ExportToSheets(context.Background(), sheets, excel.DefaultMultiSheetExportOptions())

StreamImporter.ImportFromSheet and ImportAllSheets read a named sheet or
every sheet from a file path, invoking a row handler synchronously per
row; they do not use the worker pool that ImportStream does.

# Concurrency

A single StreamExporter or StreamImporter is not safe for concurrent
reuse: each drives one export or import operation and mutates internal
state (the underlying excelize file, worker pool) as it runs. Create a
new instance per operation.

Within one ExportStream call, rows written to the underlying
excelize.StreamWriter go through exactly one writer goroutine, since
excelize.StreamWriter is not safe for concurrent use; any RowTransformer
runs on worker goroutines but never touches the writer directly. A
RowTransformer must not retain the row slice it is given after it
returns.

GetRowCount and GetStats are safe to call from another goroutine while an
export or import is in progress; they read atomic counters.

# Lower-Level Building Blocks

PoolManager and WorkerPool are the primitives StreamExporter and
StreamImporter are built from: PoolManager pools row slices, cell
buffers, and string builders via sync.Pool, and WorkerPool runs a
ProcessFunc across a fixed number of goroutines. Both moved to
internal/concurrency: they are implementation details of how rows are
processed concurrently, not something the happy-path API here ever
requires a caller to name directly, so they are not part of this
module's public API. WorkerStats is the one exception, kept as an
exported type alias in this package because GetStats returns it.

A third primitive, a multi-stage pipeline with named, independently
concurrent stages, lives at internal/pipeline. Neither StreamExporter
nor StreamImporter uses it, and it is not part of this module's public
API either.

# Limitations

  - CSV is supported via WithExportFormat/WithImportFormat, but only for
    a single sheet: ExportToSheets writes just the first sheet for CSV,
    and ImportFromSheet/ImportAllSheets reject CSV files outright.
  - No cell styling, number formatting, or column widths in the
    streaming write path; these are silently dropped, or logged once via
    WithExportLogger/WithImportLogger when the format cannot honor them.
  - No formula evaluation during streaming.
  - StreamImporter loads the full workbook when it opens the reader
    (excelize.OpenReader); only row iteration afterward is incremental.

# See Also

  - The excel package, for non-streaming export/import of files small
    enough to fit comfortably in memory.
  - Benchmarks in *_bench_test.go and *_bench_large_test.go for measured
    performance and allocations on this machine; re-run them rather than
    trusting historical numbers in docs.
*/
package stream
