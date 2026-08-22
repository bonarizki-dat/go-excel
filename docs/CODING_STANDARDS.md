# Coding Standards - Go Excel Library

> Go idioms and conventions for this library

---

## CORE CONVENTIONS

### 1. Comments explain intent, not narration

- Comments are in **English** only
- Explain **WHY** - intent, invariants, trade-offs, and constraints that
  the code itself cannot convey - not a restatement of the next line
- Document concurrency contracts explicitly wherever they exist: which
  goroutine owns a piece of mutable state, and the lifetime of any slice
  or buffer handed to a caller-supplied callback
- Do not leave narration comments like `// Increment counter` above
  `counter++`, historical/conversational residue (`// No longer used, kept
  for compatibility`), or aspirational notes standing in for real work
  (`// should add error handling in future`) - either implement it or
  open an issue
- Use professional, general language: declarative, third-person, present
  tense, describing the code's contract rather than the author or the
  process that produced it. No slang, jokes, emojis, exclamation marks, or
  hedging (`should probably work`). Do not reference dates, sprints, PR
  numbers, tool names, or people - a comment must stay accurate regardless
  of who reads it or when
- When documenting a limitation, state the condition, not a promise
  (`Styles are not applied on this path`, not `Styles broken, will fix
  next week`)
- Use the exact identifier names from the public API (`RowSink`,
  `BasicExporter`, header row, sheet) instead of ad hoc synonyms, and never
  reference a type that does not exist in the codebase

```go
// Good: explains a non-obvious constraint
// pending can never hold more than maxInFlight entries because a row
// occupies a slot from submission until the writer processes it.
pending := make(map[uint64][]any, maxInFlight)

// Bad: narrates what the next line already says
// Increment row counter
atomic.AddUint64(&rowCounter, 1)

// Good: professional, general, states the contract
// Finalize completes the document and writes it to w. Called exactly
// once, after every row has been written via WriteRow.

// Bad: conversational register, time-bound, narrates the process
// Finalize - we call this last. Was buggy last week, fixed now.
```

### 2. Every exported identifier has a godoc comment

Begin with the identifier name, state what it does, then document
parameters, return values, and error conditions where they are not
obvious from the signature.

### 3. Every behavior change ships with a test

A bug fix or behavior change should include a test that would have failed
before the fix. Performance-sensitive changes should include a benchmark
comparing before and after.

### 4. `go test -race ./...` must pass

This library processes rows across worker pools and goroutines; a green
test suite without `-race` does not prove the code is safe.

---

## PROJECT STRUCTURE

```
go-lib-matwebsite/
├── excel/               # Shared types: config, options, events, styling, struct-tag mapping
│   ├── exporter/        # BasicExporter
│   └── importer/        # BasicImporter
├── stream/              # StreamExporter/StreamImporter for row-at-a-time processing
├── examples/            # Runnable usage examples
├── docs/                # Documentation
├── go.mod
├── go.sum
├── README.md
├── CHANGELOG.md
├── LICENSE
├── Makefile             # Build automation
└── .golangci.yml        # Linter configuration
```

---

## NAMING CONVENTIONS

### Package Names
- **Use lowercase**, single word, no underscores
- **Keep short** and descriptive
- **Avoid generic names** like `util`, `common`, `helper`

```go
✅ GOOD
package excel
package exporter
package importer

❌ BAD
package excel_lib
package ExcelLib
package utils
```

### File Names
- **Use snake_case** for file names
- **Group related functionality** in the same file
- **Test files** end with `_test.go`

```go
✅ GOOD
exporter.go
importer_reader.go
exporter_test.go

❌ BAD
Exporter.go
importerReader.go
exporterTest.go
```

### Types & Structs
- **Use PascalCase** for exported types
- **Use camelCase** for unexported types
- **Be descriptive** but concise

```go
✅ GOOD
type BasicExporter struct { }
type StyleBuilder struct { }
type exportOptions struct { }  // unexported

❌ BAD
type exporter struct { }  // should be exported
type SB struct { }        // too short
type ExporterStruct struct { }  // redundant "Struct"
```

### Functions & Methods
- **PascalCase** for exported
- **camelCase** for unexported
- **Use verbs** for actions

```go
✅ GOOD
func NewExporter(data any, opts ...Option) *BasicExporter
func (e *BasicExporter) Export(ctx context.Context) error
func (e *BasicExporter) validateHeaders() error  // unexported

❌ BAD
func new_exporter() *BasicExporter
func (e *BasicExporter) export() error  // should be exported
func (e *BasicExporter) Exporting() error  // use Export, not Exporting
```

### Interfaces
- **Single method**: suffix with `-er` (Reader, Writer, Exporter)
- **Multiple methods**: descriptive names
- **Keep small and focused**

```go
✅ GOOD
type Exporter interface {
    Export() error
}

type HeadingsProvider interface {
    Headings() []string
}

type StyleApplier interface {
    ApplyStyle(*Style) error
}

❌ BAD
type ExporterInterface interface { }  // redundant "Interface"
type IExporter interface { }          // don't use "I" prefix
```

### Constants & Variables
- **Use PascalCase** for exported
- **Use camelCase** for unexported
- **Group related constants** using `const` blocks

```go
✅ GOOD
const (
    DefaultChunkSize = 1000
    MaxSheetNameLength = 31
)

var (
    ErrInvalidFormat = errors.New("invalid format")
    ErrEmptyData = errors.New("empty data")
)

❌ BAD
const DEFAULT_CHUNK_SIZE = 1000  // don't use snake_case
const defaultChunkSize = 1000    // should be exported
```

---

## ERROR HANDLING

### Custom Error Types
Create specific error types for better error handling.

```go
// errors.go
package excel

import "fmt"

// ExportError represents an error during export
type ExportError struct {
    Op    string // Operation (e.g., "Export", "WriteCell")
    Sheet string // Sheet name
    Row   int    // Row number
    Col   int    // Column number
    Err   error  // Underlying error
}

func (e *ExportError) Error() string {
    if e.Sheet != "" && e.Row > 0 {
        return fmt.Sprintf("%s [%s:%d:%d]: %v", e.Op, e.Sheet, e.Row, e.Col, e.Err)
    }
    return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *ExportError) Unwrap() error {
    return e.Err
}

// Sentinel errors
var (
    ErrInvalidFormat    = errors.New("invalid file format")
    ErrEmptyData        = errors.New("empty data")
    ErrInvalidSheetName = errors.New("invalid sheet name")
    ErrRowNotFound      = errors.New("row not found")
)
```

### Wrapping Errors
Always provide context when wrapping errors.

```go
func (e *Exporter) writeCell(row, col int, value any) error {
    if err := e.validate(value); err != nil {
        return &ExportError{
            Op:  "WriteCell",
            Row: row,
            Col: col,
            Err: err,
        }
    }
    return nil
}

func (e *Exporter) Export() error {
    if err := e.writeHeaders(); err != nil {
        return fmt.Errorf("export headers: %w", err)
    }
    return nil
}
```

### Error Checking
Check for specific errors using `errors.Is` and `errors.As`.

```go
// User code
if err := exporter.Export(); err != nil {
    if errors.Is(err, excel.ErrEmptyData) {
        log.Println("No data to export")
        return nil
    }

    var exportErr *excel.ExportError
    if errors.As(err, &exportErr) {
        log.Printf("Export failed at row %d: %v", exportErr.Row, exportErr.Err)
    }

    return err
}
```

---

## OPTIONS PATTERN (Functional Options)

Configuration uses a single `Option func(*config.Config)` type shared by
exporters and importers, defined in `excel/options.go`:

```go
// Option is a function that modifies a Config.
type Option func(*config.Config)

// WithSheetName sets the sheet name for export/import operations.
func WithSheetName(name string) Option {
    return func(c *config.Config) {
        c.SheetName = name
    }
}

// WithRows restricts processing to a 1-based, inclusive row range:
// rows numbered [first, last] are processed. If last is 0, processes
// through the end of the sheet.
func WithRows(first, last int) Option {
    return func(c *config.Config) {
        if first >= 1 {
            c.StartRow = first - 1
        }
        if last >= 0 {
            c.EndRow = last
        }
    }
}
```

Usage:

```go
exp := exporter.NewExporter(data,
    excel.WithSheetName("Users"),
    excel.WithHeadings(true),
    excel.WithRows(1, 1000),
)
```

Add new options by adding a field to `config.Config`, a `With*`
constructor in `excel/options.go`, and validation in
`excel/config/validation.go` if the value has constraints.

---

## CONTEXT SUPPORT

All long-running operations should accept `context.Context`.

```go
func (e *Exporter) Export(ctx context.Context) error {
    return e.export(ctx)
}

func (e *Exporter) export(ctx context.Context) error {
    rows, err := e.fetchData(ctx)
    if err != nil {
        return err
    }

    for i, row := range rows {
        // Check context cancellation
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        if err := e.writeRow(i, row); err != nil {
            return err
        }
    }

    return nil
}

// Usage with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := exporter.Export(ctx); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Export timeout")
    }
    return err
}
```

---

## TESTING STANDARDS

### Test File Organization
- Test files: `*_test.go`
- Same package for white-box testing: `package excel`
- Separate package for black-box testing: `package excel_test`

### Test Function Naming
```go
// Format: Test<Function>_<Scenario>_<ExpectedBehavior>
func TestExporter_Export_EmptyData_ReturnsError(t *testing.T) { }
func TestExporter_Export_ValidData_Success(t *testing.T) { }
```

### Table-Driven Tests
Prefer table-driven tests for multiple scenarios.

```go
func TestExporter_Export_TableDriven(t *testing.T) {
    tests := []struct {
        name    string
        data    any
        wantErr bool
    }{
        {name: "rows", data: [][]any{{"a", 1}, {"b", 2}}, wantErr: false},
        {name: "nil data", data: nil, wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            exp := NewExporter(tt.data)
            defer exp.Close()

            err := exp.Export(context.Background())
            if (err != nil) != tt.wantErr {
                t.Errorf("Export() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Helpers
Use `t.TempDir()` instead of `os.CreateTemp`/`os.MkdirTemp` for file-based
tests. It is automatically cleaned up, even if the test fails.

```go
func writeTempXLSX(t *testing.T, name string, content []byte) string {
    t.Helper()

    path := filepath.Join(t.TempDir(), name)
    if err := os.WriteFile(path, content, 0o600); err != nil {
        t.Fatal(err)
    }
    return path
}
```

### Benchmark Tests
```go
func BenchmarkExporter_ExportLarge(b *testing.B) {
    data := genRows(100000)
    ctx := context.Background()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        exp := exporter.NewExporter(data)
        if err := exp.Export(ctx); err != nil {
            b.Fatal(err)
        }
        _ = exp.Close()
    }
}
```

### Example Tests
Compiled `Example_` functions turn documentation drift into a build
failure instead of a later audit finding, so every runnable example
under `examples/` should have one alongside it (see
`examples/basic_export/example_test.go` and its siblings). Name them
`Example_<packageExample>` (e.g. `Example_basicExport`,
`Example_streamImport`) rather than the type-scoped
`ExampleType_Method` form, since these examples demonstrate a whole
workflow rather than one method in isolation.

```go
// Examples appear in godoc.
func Example_basicExport() {
    data := [][]any{
        {"ID", "Name"},
        {1, "John"},
        {2, "Jane"},
    }

    exp := exporter.NewExporter(data, excel.WithHeadings(true))
    defer exp.Close()

    if err := exp.ExportToFile(context.Background(), "users.xlsx"); err != nil {
        log.Fatal(err)
    }
    defer os.Remove("users.xlsx")

    // Output:
}
```

---

## DOCUMENTATION STANDARDS

### Package Documentation
Package-level docs live in `excel/doc.go` (and `stream/doc.go` for the
streaming package). Keep these in sync with the actual public API — see
`excel/doc.go` for the canonical example covering `NewExporter`,
`NewImporter`, `Exportable`, and context/options usage.

### Godoc Comments
All exported types, functions, and constants must have godoc comments that
describe real behavior, not aspirational behavior. Example from the
codebase:

```go
// WithRows restricts processing to a 1-based, inclusive row range:
// rows numbered [first, last] are processed, where row 1 is the first
// data row. If last is 0, processes through the end of the sheet.
func WithRows(first, last int) Option {
    return func(c *config.Config) {
        if first >= 1 {
            c.StartRow = first - 1
        }
        if last >= 0 {
            c.EndRow = last
        }
    }
}
```

When behavior is subtle (exclusive vs. inclusive bounds, thread-safety,
resource ownership), state it explicitly in the comment rather than leaving
it to be discovered by reading the implementation.

---

## CONCURRENCY PATTERNS

The patterns below are illustrative idioms for concurrent code in this
codebase, not existing public API. Check `excel/exporter` and
`excel/importer` before assuming a method like `ExportAsync` exists.

### Async Export with Channels
```go
func (e *Exporter) ExportAsync(ctx context.Context) <-chan error {
    errChan := make(chan error, 1)

    go func() {
        defer close(errChan)
        errChan <- e.Export(ctx)
    }()

    return errChan
}

// Usage
errChan := exporter.ExportAsync(ctx)

select {
case err := <-errChan:
    if err != nil {
        log.Println("Export failed:", err)
    }
case <-time.After(30 * time.Second):
    log.Println("Export timeout")
}
```

### Worker Pool for Batch Processing
```go
func (i *Importer) importWithWorkers(ctx context.Context, batchSize int) error {
    numWorkers := runtime.NumCPU()
    batches := make(chan []Row, numWorkers)
    errs := make(chan error, numWorkers)

    var wg sync.WaitGroup

    // Start workers
    for j := 0; j < numWorkers; j++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for batch := range batches {
                if err := i.processBatch(ctx, batch); err != nil {
                    errs <- err
                    return
                }
            }
        }()
    }

    // Feed batches
    go func() {
        defer close(batches)
        // ... batch generation logic
    }()

    // Wait and collect errors
    go func() {
        wg.Wait()
        close(errs)
    }()

    for err := range errs {
        if err != nil {
            return err
        }
    }

    return nil
}
```

### sync.Pool for Object Reuse
```go
var bufferPool = sync.Pool{
    New: func() any {
        return new(bytes.Buffer)
    },
}

func (e *Exporter) writeRow(row []any) error {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    // Use buffer...
}
```

---

## CODE ORGANIZATION PRINCIPLES

### 1. Separation of Concerns
Each package has a single responsibility.

### 2. Interface Segregation
Prefer many small interfaces over few large ones.

### 3. Dependency Inversion
Depend on abstractions (interfaces), not concrete types.

### 4. Composition Over Inheritance
Use struct embedding and interfaces.

```go
// Use embedding for composition
type StyledExporter struct {
    *Exporter
    style *Style
}

func (s *StyledExporter) Export(ctx context.Context) error {
    // Apply styles
    // Call embedded Exporter
    return s.Exporter.Export(ctx)
}
```

### 5. KISS (Keep It Simple, Stupid)
Simple code is better than clever code.

### 6. DRY (Don't Repeat Yourself)
But don't over-abstract prematurely.

---

## PERFORMANCE CONSIDERATIONS

### 1. Avoid Allocations in Hot Paths
```go
// BAD - allocates every call
func (e *Exporter) writeRow(row []any) error {
    buf := make([]byte, 1024)  // allocates
    // ...
}

// GOOD - reuse buffer
var bufferPool = sync.Pool{
    New: func() any {
        return make([]byte, 1024)
    },
}
```

### 2. Use Buffered Channels Appropriately
```go
// Size based on expected throughput
results := make(chan Result, 100)
```

### 3. Profile Before Optimizing
```go
// Use pprof
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

### 4. Benchmark Critical Paths
```go
func BenchmarkCriticalFunction(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        criticalFunction()
    }
}
```

---

## VERSION MANAGEMENT

### Semantic Versioning
- `v0.x.x` - Initial development, breaking changes allowed
- `v1.0.0` - First stable release
- `v1.x.x` - Minor versions (new features, backward compatible)
- `v1.x.y` - Patch versions (bug fixes)
- `v2.0.0` - Major version (breaking changes)

### Go Modules
```go
// go.mod
module github.com/bonarizki-dat/go-excel

go 1.26

require github.com/xuri/excelize/v2 v2.11.0
```

### Support Policy
- Support **Go 1.26 and later**, matching `go.mod` and CI's version
  matrix. `github.com/xuri/excelize/v2` requires `go >= 1.25.0`, so
  1.25 is a technical floor for the dependency, but this module's own
  `go.mod` pins 1.26 as the minimum it supports and tests against.
- Clearly document breaking changes in CHANGELOG
- Provide migration guide for major versions

---

## CODE QUALITY TOOLS

### Linting
See `.golangci.yml` at the repo root for the full, current configuration
(golangci-lint v2 schema). Run it with:

```bash
golangci-lint run
```

### Pre-commit Hooks
```makefile
# Makefile
.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: bench
bench:
	go test -bench=. -benchmem ./...

.PHONY: coverage
coverage:
	go tool cover -html=coverage.out
```

---

## CODE REVIEW CHECKLIST

Before submitting code:
- [ ] **All tests pass** (`go test -race ./...`)
- [ ] **New features have tests**, including error paths
- [ ] **Godoc comments** for all exported items
- [ ] **Comments use professional, general language** - no conversational
      register, dates, or references to types that don't exist
- [ ] **No linter warnings** (`golangci-lint run` passes)
- [ ] **Benchmarks added or updated** for performance-sensitive changes,
      with `b.ReportAllocs()`
- [ ] **Context cancellation** respected in any new long-running operation
- [ ] **Examples added or updated** for new public API
- [ ] **CHANGELOG.md updated** under `[Unreleased]`
- [ ] **Breaking changes documented** in the CHANGELOG and, if relevant,
      README
- [ ] **Error messages clear and actionable**, wrapped with `%w` and context
- [ ] **No claims of streaming, thread-safety, or performance** that aren't
      backed by the actual implementation and a benchmark/test

---

## PERFORMANCE BENCHMARKING TEMPLATE

Follow the pattern used in `excel/exporter/exporter_bench_test.go` and
`excel/importer/importer_bench_test.go`: a small `genRows`/`genFile` helper,
`b.ReportAllocs()`, and explicit `Close()` per iteration.

```go
package exporter

import (
    "context"
    "testing"

    "github.com/bonarizki-dat/go-excel/excel"
)

func genRows(n int) [][]any {
    rows := make([][]any, 0, n+1)
    rows = append(rows, []any{"c1", "c2", "c3"})
    for i := range n {
        rows = append(rows, []any{i, i * 2, i * 3})
    }
    return rows
}

func BenchmarkExporter1K(b *testing.B) {
    data := genRows(1000)
    ctx := context.Background()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        exp := NewExporter(data, excel.WithSheetName("Bench"), excel.WithHeadings(true))
        if _, err := exp.ExportToBuffer(ctx); err != nil {
            b.Fatal(err)
        }
        _ = exp.Close()
    }
}
```

Add sibling benchmarks (e.g. `BenchmarkExporter100K`) for larger row counts
instead of separate "small/medium/large" template functions, and compare
`allocs/op` and `B/op` before and after a change, not just wall time.

### Running Benchmarks
```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkExporter1K -benchmem ./excel/exporter/...

# Run benchmarks with CPU profile
go test -bench=. -benchmem -cpuprofile=cpu.prof

# Run benchmarks with memory profile
go test -bench=. -benchmem -memprofile=mem.prof

# Analyze CPU profile
go tool pprof cpu.prof

# Analyze memory profile
go tool pprof mem.prof

# Compare benchmarks
go test -bench=. -benchmem > old.txt
# Make changes...
go test -bench=. -benchmem > new.txt
benchstat old.txt new.txt
```

