# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

**On dates:** released versions are dated from their git tag. Entries inside
pre-release sections carry the dates of a development log that predates this
repository's initial commit (`2f4bf6d`, 2026-01-20, a squash of earlier work),
so those dates do not line up with `git log`. They are kept as written rather
than back-dated to the squash.

Work that is planned but not started is not listed here; see
[docs/FEATURE_CHECKLIST.md](docs/FEATURE_CHECKLIST.md) Part 2.

## [0.3.0] - 2026-08-23

This release closes out the harden pass tracked in
[docs/planning](docs/planning): documentation claims corrected to match
measured behavior, two fail-silent contracts (invalid sheet name,
CSV export of an XLSX-only feature) turned into explicit errors, the
concurrency and event-dispatch primitives narrowed out of the public
API, and a first real-world "dirty file" corpus added under
`testdata/corpus/` with golden and typed-error tests
(`corpus_test.go`) — see "Added" below.

**API freeze**: with this release tagged, no further renames or
removals are planned before `1.0`; only bugfixes. `1.0` itself waits
on a 2-4 week soak period (or an equivalent public beta) pinning this
version in real usage and feeding any failing real-world files back
into `testdata/corpus/`, not on coverage or time alone.

### Breaking Changes

- **`excel.TagOptions.Index` changed from `int` to `[]int`.** It now
  holds a field's full path from the top-level struct (more than one
  element only for a field reached through an embedded struct), suitable
  for `reflect.Value.FieldByIndexErr` rather than `reflect.Value.Field`.
  Code that reads `TagOptions.Index` directly needs to use the first
  element (`Index[0]`) for a top-level field, or the full slice to reach
  a field through an embedded struct the way `excel.RowMapper.MapRow`
  now does.
- **`stream.NewStreamExporter` now returns `(*StreamExporter, error)`**
  instead of `*StreamExporter`. Previously, an invalid sheet name passed
  via `WithExportSheetName` was swallowed and the exporter silently fell
  back to `"Sheet1"`; the constructor now returns the error from
  excelize's `NewSheet` instead. Every call site needs an `err` check;
  see [docs/API_REFERENCE.md](docs/API_REFERENCE.md#newstreamexporter)
  for the updated signature.
- **`stream.WorkerPool`, `stream.PoolManager`, `stream.Task`,
  `stream.Result`, `stream.ProcessFunc`, and `stream.WorkerError`
  removed from the public API**, moved to `internal/concurrency`
  (`internal/concurrency.WorkerPool`, etc.). Nothing in the documented
  happy path (`NewStreamExporter`/`NewStreamImporter` plus the `With*`
  options) ever required naming these types directly; they were the
  concurrency primitives `StreamExporter`/`StreamImporter` are built
  from, not part of the contract callers depend on.
  `stream.WorkerStats` is unaffected — it stays exported (now a type
  alias for `internal/concurrency.WorkerStats`), since
  `StreamExporter.GetStats`/`StreamImporter.GetStats` return it. Code
  that called `stream.NewWorkerPool`/`stream.NewPoolManager` directly,
  or type-asserted against `*stream.WorkerError`, has no supported
  replacement in this module — that usage was always exercising an
  implementation detail, not the streaming contract.
- **`excel.EventDispatcher`/`excel.NewEventDispatcher` removed from the
  public API**, moved to `internal/events`
  (`internal/events.EventDispatcher`). `BasicExporter.On` and
  `BasicImporter.On` are unaffected — they already held the dispatcher
  in a private field and are the only supported way to register hooks.
  `excel.EventType`, `excel.Event`, and `excel.Hook` are unaffected and
  stay exported, since `Hook`'s signature is what `On` exposes.

### Fixed

- **`BasicExporter.ExportToSheets` ignored styles and column widths on
  XLSX.** It wrote cell values directly instead of going through the
  `RowSink` that `Export` uses, so `SetHeaderStyle`, `SetDataStyle`,
  `SetColumnStyle`, and `SetColumnWidth` had no effect on any sheet.
  `ExportToSheets` now attaches an `xlsxSink` to the workbook and reuses
  it across every sheet, so styles and column widths apply the same way
  `Export` applies them.
- **`ExportToSheets` leaked the previous call's workbook.** Calling it
  more than once on the same exporter overwrote `e.file`/`e.sink`
  without closing what they previously pointed at. It now closes the
  prior workbook first, the same leak prevention `Export` already
  applied to its own sink.
- **`excel.GetFieldTags`'s cache key collided for anonymous struct
  types.** The cache keyed on `t.PkgPath() + "." + t.Name()`, both of
  which are empty for an anonymous struct type, so every anonymous type
  shared the same cache entry and could return another type's field
  tags. The key is now `t.String()`.
- **`BasicImporter.GetHeadings()` and `StreamImporter.GetHeadings()`
  returned the internal headings slice directly**, so a caller mutating
  the returned slice corrupted the importer's own state — unlike
  `GetRows()`, which already returned a copy. Both now return a copy.

### Added

- `BasicExporter.SetNumberFormat(col string, fmtID int)` — merges a
  number format into `col`'s existing column style, or creates one via
  `excel.NewStyle().Format(fmtID)` if none exists yet.
- `excel.Style.FillPattern(n int)` — sets the fill pattern index passed
  through to `excelize.Style.Fill.Pattern` (previously only solid fill
  was reachable through the builder).
- **Embedded struct flattening for struct-mapped import.**
  `GetFieldTags` now recurses into anonymous (embedded) struct and
  pointer-to-struct fields, and `RowMapper.MapRow` reaches them via
  `reflect.Value.FieldByIndexErr`, which reports a nil embedded pointer
  as an error instead of panicking. Mirrors the flattening
  `reflectStructToRow` already did for export.
- `excel.WithStrictFormat(bool)` and `stream.WithExportStrictFormat(bool)`
  — when `true`, exporting to CSV with an XLSX-only feature configured
  (styles, column widths, a custom start cell, auto-sized columns,
  document properties, or more than one sheet passed to
  `ExportToSheets`) fails with an error wrapping the new
  `excel.ErrStrictFormatUnsupported` sentinel, instead of dropping the
  feature and logging a one-time warning. Default stays `false`
  (degrade and warn) for both.
- `excel.WithConverters(map[string]Converter)` and
  `type Converter func(raw string, dest reflect.Type) (any, error)` —
  registers a per-column override for struct-mapped import's
  string-to-Go-type conversion, consulted by `RowMapper.mapField` before
  it falls back to `ConvertToType`.
- `excel.WithRowMap(func(row []any) ([]any, error))` — transforms each
  data row (not the header row) before `Export` writes it; an error from
  the function aborts the export.
- `excel.WithCustomStartCell(cell string)` — shifts XLSX export's
  top-left cell to `cell` (e.g. `"B3"`) instead of `A1`; every
  subsequent row and column shifts by the same offset, while row/column
  numbers passed to `ProgressCallback`/`ErrorCallback` stay relative to
  the data. Ignored for CSV export, and rejected by `Config.Validate` if
  not a valid cell reference.
- `excel.ShouldAutoSize(bool)` — widens each XLSX column to its widest
  cell's character length, applied once every row for a sheet has been
  written; an explicit `SetColumnWidth` wins over the measured width.
  This is a character-count measurement, not Excel's own font-aware
  autofit. Ignored for CSV export.
- `excel.WithProperties(excel.DocProperties)` — sets a subset of the
  exported workbook's OOXML core document properties (`Title`,
  `Subject`, `Creator`, `Keywords`, `Description`, `Category`,
  `Version`) via excelize's `SetDocProps`. Ignored for CSV export.
- `Config.Clone()` now copies `Converters`, `RowMap`, `StartCell`,
  `AutoSizeColumns`, and `DocProperties`; `config_clone_test.go` gained
  a reflection-based test that fails if `Config` gains a field not yet
  wired into `Clone()`.
- **A real-world "dirty file" corpus** (`corpus_test.go`,
  `testdata/corpus/`): committed `.csv` fixtures covering a leading
  UTF-8 BOM, a `;` delimiter, a quoted embedded newline, and ragged
  rows with an empty heading cell; `.xlsx` fixtures built in-test via
  excelize covering merged cells (which shorten `GetRows()`' row
  length instead of padding it), a bare date serial vs. the same
  serial with a date format vs. a numeric-looking value stored as
  text, and a workbook with a non-ASCII sheet name plus a trailing
  empty sheet. Every case asserts either an exact golden result or a
  specific sentinel/typed error via `errors.Is` (`excel.ErrEmptyData`,
  `excel.ErrSheetNotFound`, `excelize.ErrMaxRows`) — never a bare
  `err != nil` and never a panic. Every prior test in this module built
  a clean `[][]any` fixture by hand; none of these quirks had a
  regression test until now.

### Notes

- **`internal/pipeline` tracking**: still has no production call path
  (see `internal/pipeline/doc.go`). Revisit 30 days after the `v0.3.0`
  tag is created; delete the package if it is still unused by anything
  outside its own tests. Recorded here rather than as a GitHub issue
  because `gh` was unavailable in the environment that did this pass.
- **Public git history was rewritten for this release.** `main` on
  `bonarizki-dat/go-excel` was replaced with a single orphan commit
  containing this tree (no parent), pushed with `--force`; the `v0.1.0`
  and `v0.2.0` tags were deleted from both the local clone and the
  remote. This was a deliberate one-time reset at the end of the harden
  pass, not a routine rebase.
  - Any existing clone or fork's history diverges permanently from
    `main` as of this rewrite; the old commits (`5799408` and earlier)
    no longer exist on the remote.
  - **If `v0.1.0`/`v0.2.0` were ever fetched through the public Go
    module proxy (`proxy.golang.org`), those versions remain
    retrievable there forever** — the module proxy caches tagged
    versions independently of the origin git history, and this rewrite
    cannot and does not remove them from it. `v0.3.0` is the first
    tag cut from the new history and the only one this project
    supports going forward.
  - Any GitHub Issues or Pull Requests that reference a pre-rewrite
    commit SHA are now orphaned (the SHA they link to no longer
    resolves on this repository). The repository name and module import
    path (`github.com/bonarizki-dat/go-excel`) are unchanged.

## [0.2.0] - 2026-08-22

### Breaking Changes & Migration Guide

This release contains several breaking changes, still permitted at this
pre-1.0 (`v0.x`) stage. Each is a compile-time break (a renamed or removed
identifier) rather than a silent behavior change: see
[docs/guides/CODING_STANDARDS.md](docs/guides/CODING_STANDARDS.md) on
preferring *fail loudly* over *fail silently* for breaking changes. If you
only use the documented "happy path" — `NewExporter`/`NewImporter`,
`WithHeadings`, `ExportToFile`/`ImportFromFile`, and
`stream.NewStreamExporter`/`NewStreamImporter` with `WithExport*`/`WithImport*`
options — fixing these should be a matter of running `go build` and
following the compiler's errors one rename at a time.

1. **Row/column range options renamed, and their semantics changed** from
   0-indexed/exclusive to 1-based/inclusive, matching how spreadsheet users
   already think about row numbers:
   - `excel.WithRowRange(startRow, endRow int)` → `excel.WithRows(first, last int)`
   - `excel.WithColumnRange(startCol, endCol int)` → `excel.WithColumns(first, last int)`
   - `stream.WithRowRange(start, end int)` → `stream.WithImportRows(start, end int)`
     (already 1-based; renamed only for the `WithImport*` convention below)

   Migrate `excel.WithRowRange(0, 500000)` to `excel.WithRows(1, 500000)`:
   shift both bounds up by one, and note `last` is now inclusive rather than
   exclusive.

2. **`stream` export options renamed to a consistent `WithExport*` prefix.**
   The previous names mixed `WithWorkers`/`WithRowTransform` with
   `WithStream*`, and some collided in shape (though not in package, since
   Go scopes them separately) with `excel.With*` import options of the same
   name:
   - `WithWorkers` → `WithExportWorkers`
   - `WithStreamHeadings` → `WithExportHeadings`
   - `WithStreamSheetName` → `WithExportSheetName`
   - `WithStreamProgress` → `WithExportProgress`
   - `WithStreamFormat` → `WithExportFormat`
   - `WithStreamCSVOptions` → `WithExportCSVOptions`
   - `WithStreamLogger` → `WithExportLogger`

3. **`stream.WithSkipEmptyRows()` (no-arg, always-on) → `stream.WithImportSkipEmptyRows(skip bool)`**,
   now symmetric with `excel.WithSkipEmptyRows(bool)`. Migrate
   `WithSkipEmptyRows()` to `WithImportSkipEmptyRows(true)`.

4. **`excel.NewValidationError` gained a 5th parameter.**
   `NewValidationError(row int, field string, value any, message string) error`
   is now
   `NewValidationError(row int, field string, value any, message string, cause error) error`.
   Pass `nil` for `cause` if there is no underlying error to wrap (most
   existing call sites already had one available and were not otherwise
   propagating it).

5. **10 sentinel errors removed** from `excel` (`errors_sentinel.go`)
   because each had no production call site, or was too generic to be
   meaningful (nearly any I/O failure could be called a "write failed"):
   `ErrInvalidConfig`, `ErrRowNotFound`, `ErrColumnNotFound`,
   `ErrInvalidRange`, `ErrPermissionDenied`, `ErrWriteFailed`,
   `ErrReadFailed`, `ErrContextCanceled`, `ErrTimeout`,
   `ErrValidationFailed`. Replacements:
   - Config-shaped failures: `errors.As(err, &excel.ConfigError{})` instead
     of `errors.Is(err, excel.ErrInvalidConfig)`.
   - Validation-shaped failures: `errors.As(err, &excel.ValidationError{})`
     instead of `errors.Is(err, excel.ErrValidationFailed)`.
   - Context cancellation/deadlines: `errors.Is(err, context.Canceled)` /
     `errors.Is(err, context.DeadlineExceeded)` instead of the removed
     `ErrContextCanceled`/`ErrTimeout` — this is also more correct, since
     the standard library's own sentinels are what `ctx.Err()` actually
     returns.
   - The remaining 7 (`ErrInvalidFormat`, `ErrEmptyData`,
     `ErrInvalidSheetName`, `ErrSheetNotFound`, `ErrFileNotFound`,
     `ErrInvalidFile`, `ErrTypeMismatch`) are unchanged.

6. **Dead API removed**: `stream.Task.Priority` (a field nothing ever
   read) and the `stream.PoolStats` type (its fields were never populated,
   since `sync.Pool` exposes no hit-count API to back them — see also the
   "Removed" section below for the 24 unused exported interfaces removed
   from `excel/interfaces.go` in this same release).

7. **`stream.Pipeline`, `stream.Stage`, and `stream.StageFunc` moved out
   of the public API** to `internal/pipeline` (`pipeline.Pipeline`,
   `pipeline.Stage`, `pipeline.StageFunc`, and `stream.NewPipeline` →
   `pipeline.New`). Neither `StreamExporter` nor `StreamImporter` has
   ever used `Pipeline` — they are built on `WorkerPool` instead — so it
   had no production call path, and its concurrency, shutdown, and
   backpressure handling were validated only by its own tests. There is
   no migration path for external callers using `stream.Pipeline`
   directly; copy `internal/pipeline`'s source into your own module if
   you depend on it. `PoolManager` and `WorkerPool` are unaffected and
   remain public.

See "Added", "Changed", "Removed", and "Fixed" below for the complete set
of changes in this release, including the non-breaking ones.

### Added

- **CSV format support** (2026-08-21):
  - `.csv` is now a first-class export/import format alongside `.xlsx`,
    added via a `RowSink`/`RowSource`-style abstraction shared by
    `excel/exporter`, `excel/importer`, and `stream` so the existing
    event, progress, error-handling, struct-mapping, and worker-pool
    machinery did not need to be duplicated.
  - `excel/config/format.go`: `FileFormat` type (`FileFormatXLSX`,
    `FileFormatCSV`) and `CSVOptions` (delimiter, UTF-8 BOM, line ending,
    time layout), aliased in package `excel` via `excel/format.go`
    together with `FormatFromExtension`.
  - New options: `WithFormat`, `WithCSVDelimiter`, `WithCSVBOM`,
    `WithCSVLineEnding`, `WithCSVTimeLayout` (`excel/options.go`), and
    their streaming equivalents `WithStreamFormat`,
    `WithStreamCSVOptions`, `WithStreamLogger`, `WithImportFormat`,
    `WithImportCSVOptions`, `WithImportLogger` (`stream/exporter.go`,
    `stream/importer_options.go`).
  - `excel/csvio`: a small shared package (`writer.go`, `reader.go`,
    `value.go`) wrapping `encoding/csv` for both the batch and streaming
    packages, handling BOM, custom delimiters/line endings, ragged
    records, and value formatting (floats never render in scientific
    notation).
  - `ExportToFile`/`ImportFromFile` infer the format from the filename
    extension; `ExportToWriter`, `ExportToBuffer`, `ImportFromReader`,
    and the streaming constructors have no filename to infer from and
    use `WithFormat`/`WithStreamFormat`/`WithImportFormat` instead
    (default: XLSX, unchanged from before).
  - **Degradation policy**: CSV has no concept of styles, column widths,
    merged cells, or multiple sheets. These are dropped rather than
    erroring, with exactly one `Logger.Warn` call per export/import
    regardless of row or sheet count. `ExportToSheets` (both
    `excel/exporter` and `stream`) writes only the first sheet to CSV;
    `ImportAllSheets`, `GetSheetNames`, `GetSheetCount`, and
    `SheetExists` report a single logical sheet for `.csv` inputs
    instead of failing with an excelize format error.
  - This reverses an earlier decision: `WithCSVDelimiter` and
    `WithEncoding` were previously removed from `excel/config` as unused
    dead code, and `.csv` was explicitly rejected with
    `ErrWorkbookFileFormat`. `WithCSVDelimiter` returns as a real
    functional option; `WithEncoding` was not reintroduced under that
    name, superseded by the `WithFormat`/CSV option set above.
  - Verified no measurable regression to the existing XLSX path: the
    `RowSink` extraction (moving `excelize`-specific styling/sheet setup
    out of `excel/exporter/exporter.go` into `excel/exporter/sink.go`)
    was benchmarked before CSV code was added, with `allocs/op` and
    `B/op` within measurement noise (<1%).
- **Re-verified all `stream` benchmarks and filled in missing allocation
  figures** (2026-08-22): re-ran `BenchmarkStreamExporter`/`BenchmarkStreamImporter`
  at 1K/10K/100K (previously time-only in `README.md`, now include
  `B/op`/`allocs/op`) plus `1M`, `MaxSheet`, `CSV1M`, and
  `WorkerScaling` on the same Apple M1 (8-core, 8GB RAM) machine class
  as the original measurements; updated `README.md`'s Performance
  Benchmarks table with the fresh numbers. The `excel/exporter` and
  `excel/importer` (`BasicExporter`/`BasicImporter`) benchmarks at
  1K/10K/100K were also re-run as a regression check; their 1M variants
  were intentionally skipped on this run, per `docs/PERFORMANCE_GUIDE.md`'s
  own note that those paths risk OOM on a memory-constrained machine.
- **Re-verified large-dataset benchmarks** (2026-08-21):
  - Re-ran `BenchmarkStreamExporter1M` / `BenchmarkStreamImporter1M` against
    the fixed ordered-pipeline export and restored real measured numbers to
    `README.md` (Export: 7.4s / 1.5GB alloc; Import: 30.8s / 10.5GB alloc).
  - Established that a literal "10M rows" single-sheet benchmark is not
    just slow but impossible: `.xlsx` enforces a hard cap of
    `excelize.TotalRows` = 1,048,576 rows per sheet, and `excelize` returns
    `ErrMaxRows` past that. Several historical documents referenced "10M
    rows" as an achievable single-sheet target; they were annotated with
    this limit at the time and have since been deleted (see Removed).
  - Added `BenchmarkStreamExporterMaxSheet` / `BenchmarkStreamImporterMaxSheet`
    (`stream/exporter_bench_large_test.go`, `stream/importer_bench_large_test.go`)
    covering the largest row count one sheet can hold
    (`excelize.TotalRows - 1`, after reserving row 1 for headings), and
    added `createTestExcelFileStreaming` (builds the benchmark fixture via
    `StreamExporter` instead of excelize's in-memory `SetCellValue` API) so
    the fixture itself doesn't risk exhausting RAM at that row count.
- Tests for streaming multi-sheet import, including a state-regression
  check that `ImportStream` still works after `ImportFromSheet`.
- `.github/workflows/ci.yml` running `go build`, `go vet`,
  `go test -race`, and `golangci-lint run` on Go 1.26.
- **CI hardening**: `govulncheck` as its own required job, the
  `golangci-lint-action` pinned to `v8.0.0`/linter `v2.13.1`, a build-test
  matrix across `ubuntu-latest`/`macos-latest`/`windows-latest` on Go
  `1.26`, and a coverage-gate job that fails the build below 80%
  total statement coverage.
- `golangci-lint` installed locally and wired into `make lint`/`make
  lint-fix`/`make check`, with `godot` (comments end in a period) and
  `revive` (including exported-identifier doc comments) enabled as
  permanent CI gates so a new narration or undocumented-export comment
  regression fails CI instead of surfacing only in a future audit.
- `SECURITY.md`, `.github/dependabot.yml` (weekly Go module + GitHub
  Actions updates), and a `make vulncheck` target wrapping `govulncheck`.
- Repository hygiene files: `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`
  (Contributor Covenant 2.1), `.github/ISSUE_TEMPLATE/` (bug report,
  feature request, config), `.github/PULL_REQUEST_TEMPLATE.md`, and
  `.editorconfig`.
- `Pipeline.DroppedErrors()`, returning the count of stage errors
  discarded because `errorChan` was full at send time, so callers can
  detect silent error loss instead of only inferring it from a suspiciously
  low error count.
- `WorkerError.Unwrap()`, so `errors.Is`/`errors.As` can see through a
  `WorkerError` to the `error` value in `Cause` (when `Cause` holds one,
  rather than an arbitrary `panic` value); `WorkerError.Error()` now
  includes `Cause` in its message instead of omitting it.
- 8 compiled, `go test`-verified `Example_` functions demonstrating the
  library end to end — `Example_basicExport`, `Example_csvExport`,
  `Example_styling`, `Example_multiSheet` (`excel/exporter`),
  `Example_basicImport`, `Example_structMapping` (`excel/importer`),
  `Example_streamExport`, `Example_streamImport` (`stream`) — so
  documentation drift in these code paths becomes a failing test instead
  of a future audit finding.
- `examples/stream_export/` and `examples/stream_import/`: the first
  runnable examples covering `stream.StreamExporter`/`StreamImporter`,
  demonstrating a concurrent `RowTransformer`, progress reporting,
  per-row error handling, context cancellation, and reliably deferring
  `Close()` (via a `run() error` pattern, so `log.Fatal` never skips a
  pending `defer`).
- `excel/csvio` test coverage raised from 60% to 100%: `NewReader`,
  `Read`, and `ReadAll` (previously 0%, entirely untested) now have
  dedicated tests for BOM stripping, custom delimiters, and ragged
  records; `FormatValue`'s missing `int8`/`int16`/`int32`/`uint8`/`uint16`/
  `uint32`/`uint64` cases were added; `NewWriter`'s BOM-write-error path
  is now covered via a `Write`-always-fails test double.
- Godoc comments for previously undocumented exported identifiers: every
  field of `Style`/`Font`/`Fill`/`Border`/`Alignment` (`excel/style.go`),
  the 13 `EventType` constants, the number-format constants
  (`excel/formatter.go`), and the fields of `Task`/`Result`/
  `WorkerStats`/`WorkerError` (`stream/worker_types.go`).

### Changed

- **BREAKING**: `excel.Logger` and `excel.Metric` are now type aliases for
  `config.Logger` and `config.Metric` instead of separate, structurally
  identical interface definitions. The two packages previously declared the
  same four/four methods under different names, so `WithLogger`/`WithMetrics`
  accepted values that satisfied either definition without any compile-time
  signal that they were meant to be the same type. This follows the existing
  `FileFormat`/`CSVOptions` alias pattern in `excel/format.go`. Not source-breaking
  for callers who only implement the methods; breaking for any explicit type
  assertion or embedding against the old `excel.Logger`/`excel.Metric`
  interface identity.
- `WithValidation` and `Config.ValidateData` godoc no longer refer to a
  `Validator` type, which never existed in this module. Both now point to
  `RowMapper.EnableValidation` and the `min:`/`max:`/`pattern:` struct tags
  that `ValidateData` actually gates.
- `StreamImporter.ImportFromSheet` / `ImportAllSheets` / `GetSheetNames` now
  apply the same row filters as `ImportStream` (`WithImportHeadings`,
  `WithImportRows`, `WithImportSkipEmptyRows`) and increment `GetRowCount` /
  progress callbacks. The heading row is skipped when `WithImportHeadings`
  is set but is **not** recorded into `GetHeadings()`, which still describes
  only the source the importer was constructed with.
- CSV paths (by filename extension or `WithImportFormat(FileFormatCSV)`)
  are rejected by those sheet-oriented methods with an explicit "CSV files
  have no sheets" error instead of failing inside `excelize.OpenFile`.
- Consolidated the parallel `excel_test/` black-box test tree into the
  package directories it was testing (`excel/`, `excel/exporter`,
  `excel/importer`, `excel/config`), using Go's `package foo_test`
  convention instead of a separate top-level directory. This removes
  the drift risk where the same test (e.g.
  `TestExporter_PrepareData_UnsupportedType`) existed twice and only one
  copy got fixed when the underlying behavior changed.
- Rewrote `stream/doc.go`, `docs/PERFORMANCE_GUIDE.md`, and
  `docs/guides/CODING_STANDARDS.md` to remove aspirational APIs and claims
  (e.g. "constant memory O(1)", a `WorkerPool.Errors()` method that does
  not exist, `perf`/`concerns` packages that were never built) and describe
  the actual public API instead.
- Removed the file-level 300-line limit from the coding standards, the
  README, and the Makefile's `check-lines` target, since it was unenforced
  at the time (6 files already exceeded it). Files were later brought back
  under 300 lines, but as a convention rather than a gate.
- Corrected godoc that contradicted measured behavior:
  - `StreamImporter`, `excel.StreamWriter`, and `excel.StreamReader` no
    longer claim constant/O(1) memory. `StreamImporter` now states that
    `excelize.OpenReader` loads the workbook and that `BasicImporter` is
    the lighter choice below ~500K rows.
  - `stream.StreamExporter.ExportToSheets` no longer documents a `writer`
    parameter that is not in its signature, and now states that
    `WithRowTransform` and the worker pool do not apply to that path.
  - `exporter.BasicExporter.ExportToSheets` now documents the known
    limitation that XLSX multi-sheet export ignores header, data, and
    column styles plus column widths, because it bypasses the `RowSink`.
- Reorganized this changelog: the two competing `[Unreleased]` sections
  were merged into one, `[0.1.0]` is dated from its git tag (2026-01-20)
  instead of the incorrect 2025-01-27, empty `N/A` subsections and the
  speculative "Version History" roadmap were dropped, and the comparison
  links at the bottom now point at tags that exist.
- `docs/API_REFERENCE.md` version/date header now reflects the last audit
  rather than a date that predated the CSV surface it documents.
- `docs/guides/CODING_STANDARDS.md` §1 now requires comments to use
  professional, general language: describe the code's contract in
  declarative third person, without dates, tool names, or a promise to
  fix something later, and without naming a type that does not exist in
  the codebase.
- **BREAKING**: `excelize` dependency bumped from `v2.10.0` to `v2.11.0`
  to pick up an upstream CVE fix; `go.mod` pins the module's minimum Go
  version at `1.26`, reflected in the CI matrix and
  `docs/guides/CODING_STANDARDS.md`.
- **BREAKING**: the option renames and removals in "Breaking Changes &
  Migration Guide" above (`WithRowRange`/`WithColumnRange` →
  `WithRows`/`WithColumns`; the `stream` `WithExport*`/`WithImport*`
  option renaming; `NewValidationError`'s new `cause` parameter; the
  sentinel-error and dead-API removals).
- A full sweep of the codebase (`stream/` then `excel/`) to remove
  roughly 310-350 narration and aspirational comments that violated this
  project's own coding standards (comments restating the next line,
  "for now"/"TODO"/"in the future" hedges, first-person "we" phrasing),
  starting with the four heaviest offenders (`stream/pool.go`,
  `stream/pipeline.go`, `excel/options.go`,
  `excel/multisheet_collection.go`) as their own commits, verified at
  each step with `git diff --stat` (comment-only) plus `go build`/`go vet`.
- Godoc comments unified on the concise prose style already used by
  about 70% of the codebase, replacing the remaining ~30% that used a
  Javadoc-style `Parameters:`/`Returns:` block, across both `excel/*` and
  `stream/*`.
- `Makefile`: `make ci` now runs `install check vulncheck` (matching the
  jobs in `.github/workflows/ci.yml`) instead of also running the full
  benchmark suite, which CI itself never ran; `make clean` no longer
  calls `go clean -modcache` (which deletes the module cache shared by
  every Go project on the machine, not just this one), only `-cache
  -testcache`.
- Numerous documentation drift fixes verified line-by-line against the
  current code: event names (`BeforeRow`/`AfterRow` → the real
  `BeforeRowExport`/`AfterRowExport`/etc.) and `DefaultConfig`'s package
  in `docs/API_REFERENCE.md`; `Operation` → `Op` in the `ExportError`/
  `ImportError`/`StreamError` docs; `RowProcessor`'s `rowNum` documented
  as the sequential task index it actually is, not an absolute sheet row
  number, in both `docs/API_REFERENCE.md` and
  `stream/importer_options.go`; the stale "XLSX only, no CSV support"
  claim in `stream/doc.go`; the duplicate package comments in
  `excel/multisheet.go` and `excel/multisheet_collection.go` (now
  `excel/doc.go` alone is canonical); and about 15 remaining
  `interface{}` → `any` spots across `docs/API_REFERENCE.md`,
  `docs/STYLING_GUIDE.md`, and `docs/planning/FEATURE_CHECKLIST.md`.

### Removed

- **BREAKING**: 24 exported interfaces from `excel/interfaces.go` that had
  no implementation, no call site, and no example anywhere in the module:
  `Exporter`, `Importer`, `Writer`, `Reader`, `StreamWriter`, `StreamReader`,
  `Importable`, `Validatable`, `Transformable`, `Configurable`,
  `ProgressReporter`, `ErrorHandler`, `Closable`, `ContextAware`,
  `FileWriter`, `FileReader`, `Flushable`, `Resettable`, `Poolable`,
  `Identifiable`, `Nameable`, `Versioned`, `Describable`, `Metadata`. Each
  duplicated a capability the library already exposes through a concrete
  mechanism: error handling through `WithErrorHandler`/`StopOnError`, row
  transformation through `WithRowTransform`, progress through
  `WithProgress`/`WithStreamProgress`, configuration through the functional
  options, pooling through the unexported `stream/pool.go`. `excel/interfaces.go`
  now declares only the five types with a real implementation: `RowSink`,
  `FileSaver`, `Exportable`, and the `Logger`/`Metric` aliases below. If a
  concrete second implementation of `Exporter`/`Importer` appears, the
  interface should be reintroduced against that implementation rather than
  restored from this entry.
- `excel.Converter`, an exported single-method interface that nothing in
  the module implemented, consumed, or documented. Struct-mapping type
  conversion goes through `excel.ConvertToType`, which is unaffected.
  Removing it also frees the name for the custom-converter design in
  [REMAINING_WORK.md](docs/planning/REMAINING_WORK.md), which needs
  `Converter` to be a func type rather than this interface.
- Documentation that no longer described the tree: all of
  `docs/performance/` (`BENCHMARK_RESULTS.md`,
  `PERFORMANCE_VERIFICATION.md`, `profiling_analysis.md`,
  `phase2_import_optimization_results.md`,
  `streaming_findings_and_roadmap.md`, `benchmark_comparison_pooling.md`)
  and `docs/planning/NEXT_STEPS.md`. Each carried pre-race-fix benchmark
  numbers, roadmaps toward files that were never built
  (`perf/pool_manager.go`, `perf/simd_amd64.go`, `stream/stream_exporter.go`),
  or targets since proven unreachable through excelize (`<500 MB per 10M
  rows`, "true O(1) streaming"); two also linked to raw log files
  (`docs/benchmark_full_*_pooling.txt`) that are not in the repository. The
  conclusions worth keeping live in `docs/PERFORMANCE_GUIDE.md` and
  FEATURE_CHECKLIST Part 3; the measured numbers live in `README.md`.
- `.clauderules` and `.claudeignore`. Both pointed at a `docs/AI_RULES.md`
  and `docs/ai-checklist.md` that do not exist, and `.clauderules` mandated
  three rules this project has since retired or disproved: a hard 300-line
  file limit as a generation-time gate, a comment on every significant
  line, and `<500 MB` memory for 10M rows.
- Makefile `check-comments` target, which scored files by
  comment-to-line ratio — the opposite of the current standard, where
  comments explain intent rather than narrate lines — and the `docs-check`
  target, which only printed "manual verification recommended".
- **BREAKING**: 10 sentinel errors with no production call site, or too
  generic to be meaningful (see "Breaking Changes & Migration Guide"
  above): `ErrInvalidConfig`, `ErrRowNotFound`, `ErrColumnNotFound`,
  `ErrInvalidRange`, `ErrPermissionDenied`, `ErrWriteFailed`,
  `ErrReadFailed`, `ErrContextCanceled`, `ErrTimeout`,
  `ErrValidationFailed`.
- **BREAKING**: `excel.WithRowRange`, `excel.WithColumnRange`,
  `stream.WithWorkers`, `stream.WithStreamHeadings`,
  `stream.WithStreamSheetName`, `stream.WithStreamProgress`,
  `stream.WithStreamFormat`, `stream.WithStreamCSVOptions`,
  `stream.WithStreamLogger`, and the no-arg `stream.WithSkipEmptyRows()`
  — all renamed or replaced; see "Breaking Changes & Migration Guide".
- **BREAKING**: `stream.Task.Priority` field (never read anywhere) and
  the `stream.PoolStats` type (its fields were never populated, since
  `sync.Pool` exposes no hit-count API to back them).
- Makefile `test-integration` target, which ran `go test -tags=integration`
  against a codebase with zero `//go:build integration` files — identical
  to plain `make test` but implying a separate integration suite existed.
- Aspirational/residue comments that described planned-but-never-built
  behavior or used first-person "we'll" phrasing (`stream/pool_stats.go`,
  `stream/worker_types.go`, `excel/tags.go`'s `omitempty` "future use"
  note, and an aspirational time-comparison comment in
  `excel/importer/struct_test.go`).

### Fixed

- **Stream export data loss and race condition** (2026-08-21):
  - `StreamExporter.ExportStream` had two compounding defects: the Excel row
    position was derived from worker scheduling order rather than input
    order, and multiple workers wrote to the same `excelize.StreamWriter`
    concurrently, which is not safe for concurrent use.
  - Reproduced before the fix: exporting 2,000 rows with the default
    worker count produced 7-204 rows of corrupted, out-of-order output
    across repeated runs; `go test -race` confirmed the data race.
  - Rewrote the export path as an ordered pipeline: workers only transform
    rows, and a single writer goroutine reassembles results by sequence
    number before writing, bounded by a semaphore sized to worker count so
    memory stays proportional to concurrency rather than dataset size.
  - All prior stream export tests asserted against `GetRowCount()`, the
    same counter the bug corrupted, so the suite stayed green throughout.
    Tests now read back the produced file with `excelize.OpenReader` and
    verify row count and per-row order.
  - The 1M-row benchmark figures previously in `README.md` were measured
    against the corrupted implementation and were removed rather than
    restated unverified, then re-measured (see Added).
- `StreamImporter.ImportFromSheet` / `ImportAllSheets` no longer assign the
  locally opened workbook onto `si.file` (and no longer overwrite
  `si.sheetName`). The previous assignment, combined with `defer file.Close()`,
  left `si.file` pointing at a closed handle — so a later `Close()`
  double-closed it — and leaked the workbook `NewStreamImporter` had opened.
- Stale test assertions that no longer matched behavior:
  `TestExporter_PrepareData_UnsupportedType` asserted an error for a plain
  struct, but `prepareData` had fallen back to reflection-based conversion
  (success) for that case since before the change; renamed to
  `TestExporter_PrepareData_BareStruct` and updated in both the internal
  and external test suites.
- Replaced `err != context.Canceled` / `err != context.DeadlineExceeded`
  comparisons with `errors.Is` in `excel/importer` and `stream` tests, and
  switched `excel/events_integration_test.go` to `t.TempDir()` instead of
  writing `test_events.xlsx` into the working directory.
- **`StreamImporter.ImportStream` goroutine leak**: the reader goroutine
  could block forever sending to `rowChan` after `ImportStream` returned
  early (e.g. on a processor error or context cancellation), since
  nothing was left to drain the channel. The reader now also selects on
  a cancellation signal so it exits instead of leaking.
- **Busy-wait polling removed** from `Stats()`-adjacent code paths: a
  `for { ... time.Sleep(...) }` polling loop waiting for worker
  completion was replaced with a channel close/signal, removing both the
  wasted CPU and the polling-interval latency.
- **`StreamExporter.Close()` made idempotent** via `sync.Once`, so
  calling it after `ExportStream` already closed the file on an error
  path — or calling it more than once directly — no longer risks a
  double-close.
- File handles are now closed on every constructor error path that opens
  one, instead of only on the success path; `BasicExporter.ExportToSheets`
  no longer leaks a `PoolManager` on an early return.
- **Data race** on `rowCounter` in `stream/multisheet_helper.go`: a plain
  `int` field was read from a goroutine other than the one incrementing
  it. Confirmed by `go test -race` before the fix and clean after;
  reading now goes through the same atomic access the rest of the
  package already used for shared counters.
- **`Pipeline` goroutine leak on `Stop`**: a stage worker could still be
  blocked sending to a full `errorChan` after `Stop` returned, keeping
  its goroutine alive indefinitely. Added a bounded `stopGracePeriod`
  during shutdown so a blocked send is abandoned (incrementing the new
  `DroppedErrors()` counter above) instead of leaking the goroutine
  forever.
- **`WorkerPool` shutdown**: workers now drain the remainder of
  `taskChan` before exiting instead of leaving submitted-but-unprocessed
  tasks stranded on `Stop`, and a `task.Data` type assertion that could
  panic on an unexpected type now returns a `WorkerError` instead.
- **`WorkerPool.Submit`**: errors from a full task queue are now
  returned to the caller instead of being silently dropped, and pending
  results are reported as errors instead of vanishing when a pool is
  stopped with work still in flight.

## [0.1.0] - 2026-01-20

First tagged release. Pre-release scope: XLSX export/import (basic and
streaming), multi-sheet, struct mapping, events, styling.

### Added

#### Core

- **BasicExporter** - export to files, `io.Writer`, or a byte buffer.
- **BasicImporter** - import from files or `io.Reader`.
- **StreamExporter** / **StreamImporter** - row-at-a-time export and
  import for large files, so caller-side memory does not scale with row
  count. This is not constant-memory (see Known Limitations).
- **Multi-Sheet Support** - `ExportToSheets()`, `ImportFromSheet()`,
  `ImportAllSheets()`, and the `GetSheetNames()` / `GetSheetCount()` /
  `SheetExists()` utilities.
- **Struct Mapping** - `ImportToStructs[T]()` with automatic type
  conversion (string → int, float, time, bool) and tag-based validation
  (`excel:"column,required,min:0,max:100"`).
- **Event System** - hooks for `BeforeExport`/`AfterExport`,
  `BeforeImport`/`AfterImport`, and the per-row events, with multiple
  hooks per event type and error propagation that cancels the operation.
- **Styling & Formatting** - fonts (bold, italic, size, family, color),
  background colors, borders, column widths, number formats, and
  header/data/per-column styles.

#### Performance

- Memory pooling via `sync.Pool` (`PoolManager`).
- Worker pool sized from `runtime.NumCPU()`.
- Composable `Pipeline` stages with backpressure.
- Progress callbacks and `context.Context` cancellation/timeout on every
  long-running operation.

#### Configuration

- Functional options pattern with config validation.
- Framework-agnostic boundaries (`io.Writer` / `io.Reader` only).

### Performance at release

Export 1M in 9.9s against a <30s target; import 1M in 16.4s against a
<45s target; smaller sizes proportionally further ahead.

**These figures are superseded.** The 1M numbers were measured before the
stream-export race fix in [Unreleased], so they describe a build that
produced corrupted output. See
[README.md](README.md#-performance-benchmarks) for re-measured figures.

### Test coverage at release

Per-package, counting only each package's own tests: `excel` 77.85%,
`stream` 85.3%, `excel/exporter` 85.0%, `excel/importer` 80.8%,
`excel/config` 91.4%. Per-package numbers understate shared code; the
figure worth tracking is the whole-module `-coverpkg` number, which was
only measured later (see
[FEATURE_CHECKLIST.md](docs/planning/FEATURE_CHECKLIST.md) Part 1).

### Known limitations

1. **Memory is bounded by excelize, not by this library.** Profiling put
   excelize internals at ~98% of allocations and this library's own code at
   ~2%.
2. **Basic is lighter than Stream below ~500K rows.** `GetRows()`
   pre-allocates more efficiently than excelize's row iterator, so
   `BasicImporter` uses less memory than `StreamImporter` at those sizes.
3. **True O(1) memory is not reachable** through excelize's current API.
   Streaming reduces peak memory versus collecting every row first; it does
   not make memory constant.

### Recommendations

- **< 500K rows**: `BasicImporter` / `BasicExporter`.
- **> 1M rows**: `StreamImporter` / `StreamExporter`, which is what keeps
  large inputs from failing outright.
- **Memory sizing**: see [PERFORMANCE_GUIDE.md](docs/PERFORMANCE_GUIDE.md).

### Documentation

- [API_REFERENCE.md](docs/API_REFERENCE.md),
  [PERFORMANCE_GUIDE.md](docs/PERFORMANCE_GUIDE.md),
  [STYLING_GUIDE.md](docs/STYLING_GUIDE.md), and runnable
  [examples/](examples/).
- A separate usage guide was written and later folded into the root README
  and `PERFORMANCE_GUIDE.md`.

### Development log (pre-release)

Condensed. The granular per-file line counts, per-phase coverage deltas,
and compliance claims that used to sit here were removed: several were
inaccurate when written, and the corrections are recorded once, in
[FEATURE_CHECKLIST.md](docs/planning/FEATURE_CHECKLIST.md) Part 3, rather
than in two places that could drift apart.

- **2026-01-26** — header style applied to row 1 even when
  `WithHeadings(false)`.
- **2026-01-22** — merge-cell support via `GetFile()` access to the
  underlying `*excelize.File`; web-export (`net/http`) examples.
- **2025-11-20** — multi-sheet core types: `Sheet`, `SheetCollection`,
  Excel name-rule validation (31 chars, invalid characters), duplicate
  detection, sheet ordering, `MultiSheetExportOptions`.
- **2025-11-20** — fixed a panic in the pool stress test:
  `largeBufferPool.New` returned `[]byte` where `GetLargeBuffer()`
  asserted `*[]byte`. Returning `&buf` also satisfies staticcheck SA6002
  by avoiding a boxing allocation.
- **2025-11-18** — test suites for `PoolManager`, `WorkerPool`, importer
  getters, and pool concurrency.
- **2025-11-17** — streaming architecture: `StreamExporter`,
  `StreamImporter`, `PoolManager`, `WorkerPool`, `Pipeline` and stages,
  with backpressure, context cancellation, progress callbacks, and
  graceful shutdown. New package; no change to `excel`.
- **2025-11-17** — benchmark suite at 1K/10K/100K/1M for export and
  import, all reporting allocations.
- **2025-11-14** — **breaking**: split `excel` into `excel/config`,
  `excel/exporter`, and `excel/importer` subpackages, which callers must
  now import directly. `Logger` and `Metric` moved to `excel/config` to
  break an import cycle. Done deliberately while the project had no
  external users.
- **2025-11-13** — initial `excel` package: error types, config and
  validation, functional options, exporter and importer cores.
- Project scaffolding: `go.mod` on excelize v2.10.0, Makefile,
  golangci-lint config, LICENSE, coding standards, feature checklist.

## [0.0.0] - 2025-11-13

Development-log marker for the initial repository setup and architecture
design. No `v0.0.0` tag exists in git.

[Unreleased]: https://github.com/bonarizki-dat/go-excel/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/bonarizki-dat/go-excel/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/bonarizki-dat/go-excel/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/bonarizki-dat/go-excel/releases/tag/v0.1.0
