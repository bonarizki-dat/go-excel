# Feature Checklist - Go Excel Library

> Feature parity tracker, originally derived from the
> [Maatwebsite/Laravel-Excel](https://github.com/SpartnerNL/Laravel-Excel)
> feature set.
>
> **This document has three parts, and only the first one is current:**
>
> | Part | Contents | Trust it? |
> |------|----------|-----------|
> | [Part 1 — Current Status](#part-1--current-status-verified-2026-08-22) | Verified against the code | Yes, as of the stated date |
> | [Part 2 — Not Implemented](#part-2--not-implemented) | Designs for work never started | Yes as intent, not as status |
> | [Part 3 — Historical Log](#part-3--historical-log-2025-11) | The original 2025-11 plan and progress claims | No — snapshot only |
>
> Earlier revisions of this file mixed all three together, so ✅ marks
> drifted away from reality in both directions (quality gates claimed as
> met but violated; documentation claimed as pending but finished). Part 3
> records those corrections instead of hiding them.

---

# Part 1 — Current Status (verified 2026-08-22)

## Verification method

Everything in Part 1 was produced by running these against the working
tree, not copied from an earlier revision:

```bash
go build ./...                 # BUILD OK
go vet ./...                   # clean, no findings
go test ./... -count=1         # all packages ok
go test ./excel/... ./stream -coverpkg=./excel/...,./stream \
  -coverprofile=all.out        # whole-module coverage
go tool cover -func=all.out
find . -name '*.go' | xargs wc -l | sort -rn
```

Toolchain: go1.26.0 darwin/arm64.

## Build, tests, benchmarks

- [x] `go build ./...` succeeds (library + all 9 `examples/` binaries)
- [x] `go vet ./...` clean
- [x] All tests passing: **830 tests/subtests, 0 failures**
- [x] `go test -race ./excel/... ./stream` clean
- [x] Benchmarks defined: **26 benchmark funcs** across 6 files
  - `excel/exporter/exporter_bench_test.go` (8), `excel/importer/importer_bench_test.go` (4)
  - `stream/exporter_bench_test.go` (3), `stream/exporter_bench_large_test.go` (4)
  - `stream/importer_bench_test.go` (3), `stream/importer_bench_large_test.go` (4)
- [x] Released tags `v0.1.0` and `v0.2.0` exist in git

## Test coverage

**Whole-module, all tests contributing (`-coverpkg`): 97.0%** — the number
to track against the >80% goal.

Per-package figures below count only each package's *own* tests, so they
understate shared code (e.g. `excel/csvio` is largely exercised through
`excel/exporter`, `excel/importer`, and `stream`):

| Package | Own-test coverage |
|---------|------------------|
| `excel` | 99.3% |
| `excel/config` | 100.0% |
| `excel/csvio` | 100.0% |
| `excel/exporter` | 94.3% |
| `excel/importer` | 98.8% |
| `stream` | 96.8% |

- [x] Whole-module coverage measured and reproducible
- [x] Whole-module coverage >80% (97.0%)

### Exported API with zero test coverage

None. No function in the module, exported or unexported, has 0% coverage.

### Weakly covered paths (below 70%, still tested)

- [x] `convertUsingReflection` / `reflectStructToRow` — covered (slice,
  array, pointer, embedded, scalar, non-struct element)
- [x] `csvSink.Close` — covered
- [x] `stream/multisheet.go` `StreamExporter.ExportToSheets` — 94.1%
- [x] `excel/exporter/multisheet_helper.go` `exportRowsToSheet` — 100.0%
- [x] `excel/exporter/multisheet.go` `exportToSheetsCSV` — 100.0%
- [ ] `excel/exporter/sink.go` `xlsxSink.SaveAs` — 66.7%
- [ ] `excel/exporter/sink.go` `xlsxSink.Close` — 66.7%
- [ ] `stream/exporter_write.go` `writeToOutput` — 66.7%

## 300-line file limit

The file-level 300-line limit was removed as a coding-standards gate (see
[CHANGELOG.md](../CHANGELOG.md), "Removed the file-level 300-line
limit"); it is now a convention that guides when to split a file for
readability, not something CI enforces. Several files currently exceed
it, mostly table-driven test files:

- `excel/mapper_test.go` (531 lines), `stream/multisheet_error_test.go`
  (452), `excel/exporter/multisheet_error_test.go` (446),
  `excel/exporter/sink.go` (383), `excel/exporter/multisheet.go` (301)

- [ ] Every `.go` file within 300 lines — **not a goal.** Tracked here only
  so the largest files are visible, not as work to do.

## Implemented features

### Formats

- [x] `.xlsx` and the other OOXML variants excelize accepts (`.xlsm`, `.xltx`, `.xltm`, `.xlam`)
- [x] `.csv` as a first-class format for export, import, and streaming
  - [x] `excel/csvio` package: `Reader`, `Writer`, `FormatValue`
  - [x] Options: `WithFormat`, `WithCSVDelimiter`, `WithCSVBOM`, `WithCSVLineEnding`, `WithCSVTimeLayout`
  - [x] Stream options: `WithExportFormat`, `WithExportCSVOptions`, `WithImportFormat`, `WithImportCSVOptions`
  - [x] Format inference from filename via `FormatFromExtension`
  - [x] Sink/source abstraction: `excel.RowSink`, `xlsxSink`, `csvSink`, `streamRowSource`
  - [x] Documented degradation: CSV + `ExportToSheets` writes the first sheet only and warns once
- [ ] `.xls` — **rejected by design.** No Go library both reads and writes it under a compatible license.

### Core export / import

- [x] `BasicExporter`: `Export`, `ExportToFile`, `ExportToWriter`, `ExportToBuffer`
- [x] `BasicImporter`: `Import`, `ImportFromFile`, `ImportFromReader`, `GetRows`, `GetHeadings`
- [x] Framework-agnostic `io.Writer` / `io.Reader` boundaries
- [x] `context.Context` on every operation, with cancellation honored mid-run
- [x] Progress and error callbacks (`WithProgress`, `WithErrorHandler`)
- [x] Functional options (`excel/options.go`, 25 options)
- [x] Typed errors: `ExportError`, `ImportError`, `ValidationError`, `ConfigError`, `StreamError`
- [x] `GetFile()` escape hatch to raw `*excelize.File` (enables merge cells — see `examples/merge_cells`)

### Multi-sheet

- [x] `excel.Sheet`, `excel.SheetCollection`, name validation, duplicate detection
- [x] `MultiSheetExportOptions`: `CreateDefaultSheet`, `SkipEmptySheets`, `ContinueOnError`, `ProgressCallback`
- [x] `BasicExporter.ExportToSheets`
- [x] `BasicImporter.ImportFromSheet`, `ImportAllSheets`, `GetSheetNames`, `GetSheetCount`, `SheetExists`
- [x] Per-sheet config override, `DataChannel` per sheet
- [x] `StreamExporter.ExportToSheets` — implemented, 94.1% covered
- [x] `StreamImporter.ImportFromSheet`, `ImportAllSheets`, `GetSheetNames` — implemented and tested; honors importer row filters; does not mutate `si.file` / `si.sheetName`; rejects CSV

### Struct mapping

- [x] Tag parser (`excel/tags.go`): `excel:"name,required,min:N,max:N,pattern:re,omitempty"`
- [x] Type conversion (`excel/converter.go`): int/int32/int64, uint/uint64, float32/64, bool, `time.Time`
- [x] Row mapper (`excel/mapper.go`) with header-to-field resolution
- [x] Generic `importer.ImportToStructs[T any](ctx, filename, opts...) ([]T, error)`
- [x] `required` enforced unconditionally
- [x] `min` / `max` / `pattern` enforced (gated by `ValidateData`; regex cached in `sync.Map`)
- [x] Custom converter interface — `type Converter func(raw string, dest reflect.Type) (any, error)`, registered per-column via `WithConverters`, consulted by `RowMapper.mapField` before falling back to `ConvertToType`
- [x] Nested / embedded struct support — `GetFieldTags` flattens anonymous (embedded) struct and pointer-to-struct fields recursively; `MapRow` reaches them via `reflect.Value.FieldByIndexErr`

### Events

- [x] Event types + `EventType.String()` (`excel/events.go`)
- [x] `EventDispatcher` (`internal/events/dispatcher.go` as of `v0.3.0`; not part of the public API — see `On()` below), multiple hooks per event, in-order execution
- [x] `Hook func(ctx context.Context, event *Event) error`, error cancels the operation
- [x] `On()` wired into both `BasicExporter` and `BasicImporter`

### Styling

- [x] `Style` builder: `Bold`, `Italic`, `FontSize`, `FontColor`, `FontFamily`, `BackgroundColor`, `BorderAll`, `BorderColor`, `AlignHorizontal`, `AlignVertical`, `WrapText`, `Format`
- [x] `SetHeaderStyle`, `SetDataStyle`, `SetColumnStyle`, `SetColumnWidth`
- [x] Header style applied to row 1 even when `WithHeadings(false)`
- [x] Number-format **constants** in `excel/formatter.go` (`FormatCurrency`, `FormatPercentage`, `FormatDate`, …)
- [x] `SetNumberFormat(col, format)` convenience method on `BasicExporter`, merging into an existing `columnStyles[col]` or creating one via `excel.NewStyle().Format(fmtID)`
- [x] Pattern fill — `Style.FillPattern(n int)` sets the fill pattern index passed through to `excelize.Style.Fill.Pattern`

### Streaming

- [x] `StreamExporter` / `StreamImporter` process rows one at a time
  (not materializing every row up front); peak memory is not bounded —
  it still scales with dataset size, dominated by excelize internally
  (see `docs/PERFORMANCE_GUIDE.md`)
- [x] `PoolManager` (`sync.Pool` for rows, cell buffers, builders, buffers) — moved to `internal/concurrency` as of `v0.3.0`, not part of the public API
- [x] `WorkerPool` sized from `runtime.NumCPU()` — moved to `internal/concurrency` as of `v0.3.0`, not part of the public API; `stream.WorkerStats`/`GetStats()` are the one exception, kept exported
- [x] `Pipeline` + stages with backpressure (`internal/pipeline`; not part of the public API, no production call path — see `internal/pipeline/doc.go`)
- [x] Package docs in `stream/doc.go`

## Correct API examples

Replacing the examples in earlier revisions that referenced methods which
never existed.

```go
// Multi-sheet export
sheets := []excel.Sheet{
    {Name: "Users", Data: userData, Config: userConfig},
    {Name: "Orders", Data: orderData},
}
exp := exporter.NewExporter(nil)
if err := exp.ExportToSheets(ctx, sheets, excel.DefaultMultiSheetExportOptions()); err != nil {
    return err
}
if err := exp.GetFile().SaveAs("report.xlsx"); err != nil {
    return err
}

// Import one sheet, then all sheets
imp := importer.NewImporter()
users, err := imp.ImportFromSheet(ctx, "report.xlsx", "Users")
all, err := imp.ImportAllSheets(ctx, "report.xlsx")

// Struct mapping; min/max/pattern require WithValidation(true)
type User struct {
    ID        int       `excel:"id,required"`
    Name      string    `excel:"name,required"`
    Email     string    `excel:"email,pattern:^[a-z0-9@._-]+$"`
    Age       int       `excel:"age,min:18,max:100"`
    Active    bool      `excel:"active"`
    CreatedAt time.Time `excel:"created_at"`
}
records, err := importer.ImportToStructs[User](ctx, "users.xlsx", excel.WithValidation(true))

// Styling, including number formats via column styles
exp.SetHeaderStyle(excel.NewStyle().Bold(true).BackgroundColor("#4472C4").FontColor("#FFFFFF"))
exp.SetDataStyle(excel.NewStyle().FontSize(10).BorderAll(excel.BorderThin))
exp.SetColumnStyle("C", excel.NewStyle().Format(excel.FormatCurrency))
exp.SetColumnWidth("A", 15.0)

// CSV export
csvExp := exporter.NewExporter(data,
    excel.WithFormat(excel.FileFormatCSV),
    excel.WithCSVDelimiter(';'),
    excel.WithCSVBOM(true),
)
err = csvExp.ExportToFile(ctx, "report.csv")
```

## Laravel-Excel parity — export concerns

Data sources:

- [x] **FromArray** — `[][]any`
- [x] **FromCollection** — slices, including `[]struct` via reflection
- [ ] **FromGenerator**
- [ ] **FromIterator**
- [ ] **FromQuery**
- [ ] **FromView** (HTML template)

Formatting and transformation:

- [x] **WithHeadings** — `WithHeadings()` + `Config.WithHeadings`
- [x] **WithCustomCsvSettings** — `WithCSVDelimiter`, `WithCSVBOM`, `WithCSVLineEnding`, `WithCSVTimeLayout` (no `WithEncoding`; output is UTF-8 with optional BOM)
- [x] **WithMapping** — `excel.WithRowMap(func(row []any) ([]any, error))`, called in `exportRows` on every data row before it is written; `stream.WithRowTransform` covers the streaming case
- [x] **WithColumnFormatting** — `BasicExporter.SetNumberFormat(col, fmtID)`, on top of `SetColumnStyle` + `Style.Format`
- [ ] **WithCustomValueBinder**

Styling and layout:

- [x] **WithStyles** — header, data, and per-column styles
- [x] **WithColumnWidths** — `SetColumnWidth`
- [x] **ShouldAutoSize** — `excel.ShouldAutoSize(true)` widens each XLSX column to its widest cell's character length once every row for a sheet has been written; an explicit `SetColumnWidth` wins over the measurement
- [x] **WithCustomStartCell** — `excel.WithCustomStartCell("B3")` shifts XLSX export's top-left cell; default remains A1

Advanced:

- [x] **WithMultipleSheets** — `ExportToSheets` on both basic and stream exporters
- [x] **WithEvents** — full hook system, plus progress/error callbacks
- [ ] **HasReferencesToOtherSheets**
- [ ] **WithCharts**
- [ ] **WithDrawings**
- [x] **WithProperties** — `excel.WithProperties(excel.DocProperties{...})` sets Title/Subject/Creator/Keywords/Description/Category/Version via excelize's `SetDocProps`
- [ ] **WithPreCalculateFormulas**
- [ ] **WithCustomQuerySize**
- [ ] **WithStrictNullComparison**
- [ ] ~~**WithCustomChunkSize**~~ — `WithChunkSize` was deleted as dead code; no chunk-size option exists
- [ ] **Exportable** trait / **RegistersEventListeners** trait

## Laravel-Excel parity — import concerns

Destinations:

- [x] **ToCollection** — `GetRows()`
- [x] **ToArray**
- [x] **ToModel** — `ImportToStructs[T]`
- [x] **OnEachRow** — row handlers and callbacks; `StreamImporter.ImportStream` with a `RowProcessor`

Structure and headers:

- [x] **WithHeadingRow** — `WithHeadings` + `GetHeadings()`
- [ ] **WithGroupedHeadingRow**
- [x] **WithMapping** — partial: `excel.WithConverters(map[string]Converter)` overrides per-column string-to-Go-type conversion during struct-mapped import; there is no whole-row remap for import the way `WithRowMap` provides for export

Filtering and limits:

- [x] **WithLimit** — `WithRows` (last row)
- [x] **WithStartRow** — `WithRows` (first row)
- [x] **WithColumnLimit** — `WithColumns`
- [x] **SkipsEmptyRows** — `WithSkipEmptyRows`

Validation and errors:

- [x] **WithValidation** — `WithValidation(true)` → `Config.ValidateData`, enforced by the struct mapper
- [x] **SkipsOnFailure** — `WithStopOnError(false)`
- [x] **SkipsOnError** — `WithStopOnError` + error callback
- [ ] **WithUpserts** / **WithUpsertColumns**

Batching:

- [ ] **WithBatchInserts** — **no such option.** `Config.BatchSize` was deleted as dead code.
- [ ] **WithChunkReading** — **no such option.** `Config.ChunkSize` was deleted as dead code. Row-at-a-time reading (not bounded-memory; see `docs/PERFORMANCE_GUIDE.md`) is served by `StreamImporter` instead.

## Performance

Measured figures live in [README.md](../README.md#-performance-benchmarks);
memory guidance and the Basic-vs-Stream decision tree live in
[PERFORMANCE_GUIDE.md](PERFORMANCE_GUIDE.md). Re-run `go test -bench`
before quoting either anywhere user-facing.

- [x] Export 1M rows < 30s
- [x] Import 1M rows < 45s
- [x] Row-at-a-time processing for both export and import, so caller
  code never holds every row at once — but peak memory during the
  operation is still dominated by excelize (98% per
  [PERFORMANCE_GUIDE.md](PERFORMANCE_GUIDE.md#understanding-memory-usage)),
  not bounded/constant. `StreamImporter` loads the whole workbook via
  `excelize.OpenReader` at construction time; see `stream/doc.go`.
- [x] Multi-core utilization via `internal/concurrency.WorkerPool`
- [x] Memory pooling via `internal/concurrency.PoolManager`
- [x] Benchmarks at 1K / 10K / 100K / 1M, all with `b.ReportAllocs()`
- [x] Memory and CPU profiling done; conclusions kept in
  [PERFORMANCE_GUIDE.md](PERFORMANCE_GUIDE.md#memory-profiling-summary)
  and in "Findings still worth keeping" below. The raw profiling write-ups
  were deleted once their roadmaps were finished or rejected.
- [ ] Zero-copy hot paths — bounded by excelize's API (`rows.Columns()` is ~41% of import allocations)
- [ ] Batch-size tuning via benchmarks

A single `.xlsx` sheet cannot hold more than `excelize.TotalRows` =
1,048,576 rows, so any "10M+ rows" goal from the original plan can only
mean a dataset spread across multiple sheets or files.

## Documentation and examples

- [x] [README.md](../README.md) — install, quick start, benchmarks, memory guidance
- [x] [CHANGELOG.md](../CHANGELOG.md)
- [x] [docs/API_REFERENCE.md](API_REFERENCE.md) — audited against real signatures
- [x] [docs/PERFORMANCE_GUIDE.md](PERFORMANCE_GUIDE.md)
- [x] [docs/CODING_STANDARDS.md](CODING_STANDARDS.md)
- [x] [docs/README.md](README.md) — documentation index
- [x] `examples/`: `basic_export`, `basic_import`, `csv_format`, `events`, `merge_cells`, `multisheet`, `struct_mapping`, `styling`, `web_export`
- [x] Production code commented in English
- [ ] `docs/USAGE_GUIDE.md` — **deleted.** Its content was folded into `README.md` and `PERFORMANCE_GUIDE.md`; earlier revisions of this file still linked to it.
- [ ] `docs/STYLING_GUIDE.md` — **deleted 2026-08-22.** Duplicated the Style
  section of `docs/API_REFERENCE.md` with the same code samples, and had
  already drifted (it still said pattern fills were unimplemented after
  `Style.FillPattern` shipped). Its unique content — color/alignment
  constants, merge cells, best practices, limitations — was folded into
  `API_REFERENCE.md`'s Style section instead of kept as a second copy.
- [ ] `docs/guides/CODING_STANDARDS.md` and `docs/planning/` — **flattened
  2026-08-22.** `guides/` and `planning/` each held one live file after the
  `STYLING_GUIDE.md`/`REMAINING_WORK.md` deletions noted here, so the
  subfolders were dropped: `CODING_STANDARDS.md` and this file now live
  directly under `docs/`.
- [ ] `docs/planning/REMAINING_WORK.md` — **deleted 2026-08-22.** It was the
  Phase 1/2 implementation plan; both phases finished the same day (see
  `CHANGELOG.md` `[0.3.0]`). Its only unfinished item, Phase 3
  (database/query design intent), was already duplicated in Part 2 above,
  so nothing was lost.

## Next work, in priority order

1. Raise coverage on `xlsxSink.SaveAs`, `xlsxSink.Close`, and `stream/exporter_write.go`'s `writeToOutput` (66.7% each)
2. Database integration and query/generator export (Part 2), if that surface is wanted

---

# Part 2 — Not Implemented

Never started. Kept as design intent, not as status. Both depend on
decisions about how much database surface this library should own.

## Database integration (P2)

Files this would add: `excel/database.go`, `excel/batch.go`,
`excel/database_test.go`, `examples/gorm_integration`, `examples/sqlx_integration`.

- [ ] `ToModel` interface (`TableName() string`, `Validate() error`)
- [ ] `ImportToDatabase(db *sql.DB, model ToModel, filename string, opts DBOptions) error`
- [ ] Batch insert with configurable batch size
- [ ] Upsert / `OnConflict` support
- [ ] Transaction handling with rollback on error
- [ ] GORM helper
- [ ] Progress callback during inserts

```go
err := excel.ImportToDatabase(db, &User{}, "users.xlsx", excel.DBOptions{
    BatchSize:  1000,
    OnConflict: "UPDATE",
})
```

## Advanced export sources (P2)

Files this would add: `excel/query.go`, `excel/generator.go`, `excel/query_test.go`.

- [ ] `ExportFromQuery(db *sql.DB, query string, args []any, filename string) error` with auto-chunking
- [ ] `ExportFromGenerator(gen Generator, filename string) error`
- [ ] Public `DataSource` interface (`Next() ([]any, bool)`, `Close() error`) — note the existing `streamRowSource` in `stream/` is unexported and format-internal, not this
- [ ] Streaming variants of both

---

# Part 3 — Historical Log (2025-11)

A snapshot of the original plan and its progress claims. **Line counts,
coverage percentages, file counts, and dates below were true only when
written.** Part 1 supersedes all of them.

## Phase timeline

| Phase | Scope | Claimed |
|-------|-------|---------|
| 1 | Core package: basic exporter/importer, config, errors, options, interfaces | Complete |
| 2 | Testing foundation | In progress, 65.4% excel / 85.3% stream |
| 3 | Benchmark suite at 1K/10K/100K/1M | Complete 2025-11-17 |
| 4 | Streaming architecture: exporter, importer, pool, worker, pipeline | Complete 2025-11-19 |
| 5 | Memory optimization and profiling | Complete 2025-11-19 |
| 6 | Documentation and release prep | In progress 2025-11-19 |
| 7 | Feature implementation before v0.1.0 (multi-sheet, struct mapping, events, styling, database, query export) | Multi-sheet complete 2025-11-20 |

The original schedule was 3-4 weeks from 2025-11-19 targeting mid-December
2025, organized as P0 multi-sheet and struct mapping (days 1-7), P1 events
and styling (days 8-12), P2 database and query export (days 13-17), then
testing, docs, and release (days 18-23). P0 and P1 shipped; P2 did not.

## Findings still worth keeping

- Memory profiling identified excelize's `rows.Columns()` as the import
  bottleneck (1.54 GB, ~41% of allocations). True O(1) import is not
  reachable through excelize's current API, so the streaming importer is
  already at the library's floor.
- `StreamImporter` uses **more** memory than `BasicImporter` below ~500K
  rows: 872 MB vs 551 MB at 100K, because `GetRows` pre-allocates more
  efficiently than the `Columns()` iterator. Use `BasicImporter` under
  500K rows, `StreamImporter` above ~1M.
- Revised memory targets: the original <500MB per 1M rows was unreachable;
  actual is roughly 2.5 GB export and 8.5 GB import, dominated by
  excelize's internal buffers.
- A pipeline deadlock and an `ImportStream` race (`errChan` vs `rowChan`
  ordering) were both found and fixed during Phase 2/4 testing.

## Corrections to earlier claims in this file

Recorded so the same drift is not reintroduced:

1. **"XLSX/XLS/CSV format support"** — `.xls` was never supported, and CSV
   was not supported at the time. CSV support landed later; `.xls` remains
   out of scope.
2. **"300-line limit enforced, 100% compliance"** — untrue when written:
   13 files exceeded it after Phase 4. Since fixed; every `.go` file is
   within 300 lines again, as recorded in Part 1.
3. **Multi-sheet streaming marked "✅ Complete, all tests passing, 100%"** —
   the code existed with `stream/multisheet_import.go` at zero coverage,
   while another section listed streaming as "Phase 4 (Future): PENDING".
   Superseded by commit `042dbf9`, which fixed the importer-state bug there
   and added the missing tests.
4. **`WithBatchInserts` and `WithChunkReading` marked done** — the backing
   `Config.BatchSize` and `Config.ChunkSize` fields were deleted as dead
   code, as was `WithChunkSize`.
5. **Styling example used `exporter.SetNumberFormat(...)`** — that method
   never existed. Number formats go through `SetColumnStyle` +
   `Style.Format`.
6. **`docs/USAGE_GUIDE.md` marked created** — later deleted and merged into
   `README.md` and `PERFORMANCE_GUIDE.md`; the links were left dangling.
7. **Phase 6 documentation and release items left unchecked** — README,
   API reference, performance guide, changelog, and the `v0.1.0` tag were
   all finished; the checklist simply was not updated.
8. **`min`/`max`/`pattern` tags marked "parsed but not enforced"** — they
   are enforced now in `excel/mapper.go`.
9. **CSV format support has no entry of its own** — it was a substantial
   feature (new `excel/csvio` package, sink/source abstraction, 9 new
   options) that only ever appeared as footnotes on unrelated lines.
10. **Per-package coverage was reported as if it were overall coverage** —
    the meaningful figure is the `-coverpkg` whole-module number.
11. **`docs/performance/` and `docs/planning/NEXT_STEPS.md` were kept long
    after they stopped describing the tree** — six profiling/roadmap files
    plus a planning snapshot, all carrying pre-race-fix numbers, roadmaps
    to files that were never built (`perf/pool_manager.go`,
    `perf/simd_amd64.go`), and targets since proven unreachable
    (`<500 MB per 10M rows`, "true O(1) streaming"). Deleted; the findings
    worth keeping are in the section above and in `PERFORMANCE_GUIDE.md`.
    The same applies to `.clauderules` / `.claudeignore`, which pointed at
    a `docs/AI_RULES.md` that never existed and enforced the retired
    300-line and comment-every-line rules.
12. **Phase 1's "interfaces" scope (line 383 above) was never audited
    against actual usage** — `excel/interfaces.go` grew to 29 exported
    interfaces, 24 of which had no implementation or call site anywhere in
    the module. Reduced to the 5 with a real implementation: `RowSink`,
    `FileSaver`, `Exportable`, and `Logger`/`Metric` (now aliases of
    `config.Logger`/`config.Metric` instead of duplicate definitions). See
    `CHANGELOG.md` `[0.3.0]`.

## Status legend

- `[ ]` Not started · `[/]` In progress · `[x]` Complete
