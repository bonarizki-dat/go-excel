# Go Excel Library (go-excel)

> Excel import/export library for Go, inspired by Laravel's Maatwebsite/Excel

[![Go Reference](https://pkg.go.dev/badge/github.com/bonarizki-dat/go-excel.svg)](https://pkg.go.dev/github.com/bonarizki-dat/go-excel)
[![Go Version](https://img.shields.io/badge/Go-1.26%2B-blue)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-v0.3.0-green.svg)](CHANGELOG.md)

---

## 🎯 Project Goals

Build a **fast** Excel library for Go with:
- ⚡ **Strong performance**: See [measured benchmarks](#-performance-benchmarks) below
- 🌊 **Streaming Support**: A `stream` package for handing rows to the caller one at a time instead of materializing them all up front, via a worker pool and `excelize.StreamWriter`/row iterator (see the [Memory Requirements](#-memory-requirements) caveat below — this is **not constant memory**; peak memory still scales with dataset size and is still dominated by excelize internally)
- 🔧 **Framework-agnostic**: Works with any `io.Writer`/`io.Reader`, so it plugs into Gin, Echo, Fiber, or the standard library equally
- 🔧 **Extensible**: Event hooks (`BeforeExport`/`AfterExport`/etc.), plus an `Exportable` interface (a single `Data() any` method) so your own type can hand rows to the exporter

## 📊 Performance Benchmarks

Measured with `go test -bench` on Apple M1 (8-core, 8GB RAM), streaming 10-column rows to/from an in-memory buffer. 1K-100K figures are averaged over the default benchmark duration; 1M and max-sheet figures are single-run (`-benchtime 1x`) measurements on the same memory-constrained development machine, so treat them as order-of-magnitude rather than precise:

| Dataset | Export Time | Export Allocs | Import Time | Import Allocs |
|---------|-------------|----------------|--------------|----------------|
| 1K rows | 9.3ms | 3.8 MB / 65.8K allocs | 17.9ms | 10.0 MB / 179.1K allocs |
| 10K rows | 89.2ms | 25.8 MB / 668.9K allocs | 193.2ms | 100.5 MB / 1.79M allocs |
| 100K rows | 935.5ms | 183.1 MB / 6.7M allocs | 1.95s | 1.0 GB / 17.9M allocs |
| 1M rows | 9.5s | 1.5 GB / 67.0M allocs | 24.3s | 10.1 GB / 179.0M allocs |
| 1,048,575 rows (max rows in one `.xlsx` sheet) | 9.8s | 1.6 GB / 70.3M allocs | 25.2s | 12.1 GB / 222.3M allocs |

Re-run these yourself with:

```bash
# 1K-100K rows (averaged)
go test -bench 'BenchmarkStream(Exporter|Importer)(1K|10K|100K)$' -benchmem -run '^$' ./stream/...

# 1M and max-sheet rows (single-run; slow, and needs a lot of RAM)
go test -bench 'BenchmarkStream(Exporter|Importer)(1M|MaxSheet)$' -benchmem -benchtime 1x -run '^$' ./stream/...
```

**Hard limit:** the `.xlsx` format caps every worksheet at `excelize.TotalRows` = 1,048,576 rows (2^20). A single sheet cannot hold 10M rows regardless of implementation — `excelize` returns `ErrMaxRows` if you try. For datasets larger than that, split across multiple sheets with `ExportToSheets` (see [examples/multisheet/](examples/multisheet/)) or across multiple files.

## 💾 Memory Requirements

| Dataset Size | BasicExporter | BasicImporter | StreamExporter | StreamImporter | Recommendation |
|--------------|---------------|---------------|----------------|----------------|----------------|
| < 100K rows | ~200 MB | ~550 MB | ~250 MB | ~870 MB | **Use Basic** |
| 100K-500K | ~1 GB | ~2.8 GB | ~1.2 GB | ~4.4 GB | **Use Basic** |
| 500K - 1,048,576 (max per sheet) | ~2 GB | ~5 GB | ~2.5 GB | ~8.5 GB | **Use Stream** |
| > 1,048,576 rows | Not possible in one sheet | Not possible in one sheet | Split across sheets/files | Split across sheets/files | **Use `ExportToSheets`** |

**Note**: these are older peak-memory estimates predating the [stream export race-condition fix](CHANGELOG.md), not re-measured in this release, and are not the same measurement as the allocation totals (`B/op`) in the [Performance Benchmarks](#-performance-benchmarks) table above (re-verified for v0.2.0), which report cumulative bytes allocated over the run rather than peak resident memory. Re-measuring actual peak RSS for the `BasicImporter`/`BasicExporter` rows above 500K would risk OOM on a memory-constrained machine (see `docs/PERFORMANCE_GUIDE.md`'s own "may OOM" notes for those paths), so this table is left as an unverified estimate rather than publishing a number from a run that could crash before completing. Memory usage is primarily limited by the underlying excelize library. See [docs/PERFORMANCE_GUIDE.md](docs/PERFORMANCE_GUIDE.md) for details.

## 🚫 When Not to Use This Library

This is a pre-1.0 (`v0.x`) library. Breaking changes are still possible between minor versions (see [CHANGELOG.md](CHANGELOG.md)), and the API surface is not yet frozen. Concretely, avoid it if:

- **You need proven behavior on unpredictable, human-produced files.** This library's test suite is built from clean, programmatically generated fixtures; it has not yet been soaked against a corpus of real-world spreadsheets with merged cells, inconsistent headers, mixed date/number formatting, or unusual sheet names in production.
- **You are expecting "streaming" to mean bounded/constant memory.** The `stream` package is row-at-a-time processing so caller code never holds every row in a `[][]any` at once, but peak process memory is still dominated by excelize internally (~98%, see [PERFORMANCE_GUIDE.md](docs/PERFORMANCE_GUIDE.md#understanding-memory-usage)) and still scales with dataset size. If your goal is O(1) memory regardless of file size, this library does not provide that for either XLSX export or import.
- **You need charts, drawings, formula evaluation, or `.xls` (the legacy binary format).** None of these are implemented; see [docs/FEATURE_CHECKLIST.md](docs/FEATURE_CHECKLIST.md) Part 2 for what is intentionally out of scope.
- **Your toolchain is below Go 1.26.** `go.mod` pins `go 1.26` as a hard minimum; this is a deliberate toolchain requirement, not a claim that the library needs 1.26 language features for performance.
- **You need a stable, frozen API today.** Functional options and even constructor signatures have been renamed across recent releases (see the Breaking Changes sections in [CHANGELOG.md](CHANGELOG.md)). Pin an exact version and read the changelog before upgrading.

## 🚀 Installation

```bash
go get github.com/bonarizki-dat/go-excel
```

## 🚀 Quick Start

### Basic Export

**Copy and run this example:**

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/bonarizki-dat/go-excel/excel"
    "github.com/bonarizki-dat/go-excel/excel/exporter"
)

func main() {
    ctx := context.Background()
    
    // Prepare data
    data := [][]interface{}{
        {"Name", "Age", "Email"},
        {"John Doe", 30, "john@example.com"},
        {"Jane Smith", 25, "jane@example.com"},
    }
    
    // Create exporter with headings
    exp := exporter.NewExporter(data, excel.WithHeadings(true))
    
    // Export to a file on local disk
    if err := exp.ExportToFile(ctx, "output.xlsx"); err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("✅ Excel file created: output.xlsx")
}
```

**Run it:**
```bash
go run main.go
```

**📥 Download complete example:** [examples/basic_export/main.go](examples/basic_export/main.go)

### Export for Web Applications (Download to Browser)

**For web applications, use `ExportToWriter` to stream the file directly to the client instead of writing it to disk:**

**Gin Framework Example** (requires `go get github.com/gin-gonic/gin`; this library itself has no framework dependency):

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/bonarizki-dat/go-excel/excel"
    "github.com/bonarizki-dat/go-excel/excel/exporter"
)

func exportHandlerGin(c *gin.Context) {
    ctx := c.Request.Context()
    
    // Prepare data
    data := [][]interface{}{
        {"Name", "Age", "Email"},
        {"John Doe", 30, "john@example.com"},
        {"Jane Smith", 25, "jane@example.com"},
    }
    
    // Create exporter
    exp := exporter.NewExporter(data, excel.WithHeadings(true))
    
    // Set headers so the browser treats the response as a file download
    c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    c.Header("Content-Disposition", "attachment; filename=export.xlsx")
    
    // Stream the export directly to the response writer
    if err := exp.ExportToWriter(ctx, c.Writer); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
}
```

**Standard Library (`net/http`) Example:**

```go
import (
    "net/http"
    "github.com/bonarizki-dat/go-excel/excel"
    "github.com/bonarizki-dat/go-excel/excel/exporter"
)

func exportHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // Prepare data
    data := [][]interface{}{
        {"Name", "Age", "Email"},
        {"John Doe", 30, "john@example.com"},
        {"Jane Smith", 25, "jane@example.com"},
    }
    
    // Create exporter
    exp := exporter.NewExporter(data, excel.WithHeadings(true))
    
    // Headers are required for the browser to trigger a download
    w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    w.Header().Set("Content-Disposition", "attachment; filename=export.xlsx")
    
    // Stream the export directly to the response writer
    if err := exp.ExportToWriter(ctx, w); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}
```

**How it works:**
- `ExportToFile()` writes the file to disk on the server (CLI tools, background jobs)
- `ExportToWriter()` streams the file directly to any `io.Writer`, such as an HTTP response, so the browser downloads it without an intermediate file on disk

**📥 Download complete example:** [examples/web_export/main.go](examples/web_export/main.go)

### Basic Import

**Copy and run this example:**

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/bonarizki-dat/go-excel/excel"
    "github.com/bonarizki-dat/go-excel/excel/importer"
)

func main() {
    ctx := context.Background()
    
    // Create importer with headings enabled
    imp := importer.NewImporter(excel.WithHeadings(true))
    
    // Import from file
    err := imp.ImportFromFile(ctx, "input.xlsx")
    if err != nil {
        log.Fatal(err)
    }
    
    // Get imported data
    data := imp.GetRows()
    headings := imp.GetHeadings()
    
    // Display headings
    fmt.Printf("Columns: %v\n", headings)
    
    // Process data
    for i, row := range data {
        fmt.Printf("Row %d: %v\n", i+1, row)
    }
}
```

**Run it:**
```bash
go run main.go
```

**📥 Download complete example:** [examples/basic_import/main.go](examples/basic_import/main.go)

## 🚀 Current Status

**Version**: v0.3.0
**Status**: 🚧 Pre-release, API frozen going into `1.0`; only bugfixes until a 2-4 week soak (or public beta) completes

See [CHANGELOG.md](CHANGELOG.md) for release notes.

## ✨ Features

- ✅ **XLSX & CSV Formats** - Format is derived from the filename extension (`ExportToFile`/`ImportFromFile`) or set explicitly with `WithFormat` for writer/buffer targets; CSV drops XLSX-only features (styles, column widths, multiple sheets) with a one-time warning instead of failing - see [docs/API_REFERENCE.md](docs/API_REFERENCE.md#option)
- ✅ **Basic Export/Import** - Simple API for small to medium files
- ✅ **Streaming Support** - Row-at-a-time export/import via a worker pool, so caller code never holds every row at once; not constant memory — peak memory still scales with dataset size and is still dominated by excelize internally (see [Memory Requirements](#-memory-requirements))
- ✅ **Multi-Sheet Support** - Export/import multiple sheets in one file, with styles and column widths applied to every sheet - see [docs/API_REFERENCE.md](docs/API_REFERENCE.md#exporttosheets)
- ✅ **Struct Mapping** - Import directly to Go structs, with `min:`/`max:`/`pattern:` tag validation
- ✅ **Event System** - Hooks for BeforeExport, AfterExport, BeforeRowExport, AfterRowExport, BeforeRowImport, AfterRowImport
- ✅ **Styling & Formatting** - Fonts, colors, borders, column widths, number formats
- ✅ **Merge Cells** - Access excelize directly via `GetFile()` for merge operations
- ✅ **Memory Pooling** - `sync.Pool` reuse of row buffers within the stream package
- ✅ **Worker Pools** - Parallel processing utilizing all CPU cores
- ✅ **Context Support** - Cancellation and timeout support
- ✅ **Progress Callbacks** - Monitor long-running operations

## 📚 Documentation

### User Guides:
- **[docs/API_REFERENCE.md](docs/API_REFERENCE.md)** - API documentation, including a [Style section](docs/API_REFERENCE.md#style) with styling examples, best practices, and limitations
- **[docs/PERFORMANCE_GUIDE.md](docs/PERFORMANCE_GUIDE.md)** - Performance and memory guidance

### For Developers:
- **[docs/CODING_STANDARDS.md](docs/CODING_STANDARDS.md)** - Coding guidelines
- **[docs/FEATURE_CHECKLIST.md](docs/FEATURE_CHECKLIST.md)** - Test coverage figures, benchmark inventory, and feature completeness tracking
- **[CHANGELOG.md](CHANGELOG.md)** - Release notes and change history

### Key Rules:
1. **English-only comments** that explain intent, invariants, or trade-offs - not narration of the next line
2. **Every exported identifier** has a godoc comment
3. **Every new bug fix or behavior change** ships with a test that would have failed before the fix
4. **`go test -race ./...`** must pass before merging

## 🏗️ Architecture

```
go-excel/
├── excel/            # Shared types: options, events, styling, struct-tag mapping
│   ├── config/       # Config, defaults, CSVOptions, FileFormat, ConfigError
│   ├── csvio/        # Shared CSV Reader/Writer, used by both basic and stream paths
│   ├── exporter/     # BasicExporter
│   └── importer/     # BasicImporter
├── stream/           # StreamExporter/StreamImporter for row-at-a-time processing
└── internal/
    └── pipeline/     # Multi-stage concurrent pipeline (not public API)
```

## 💡 Design Philosophy

- **Framework-agnostic** - Operates on `io.Writer`/`io.Reader`, no dependency on any web framework
- **Explicit over implicit** - Row/column ranges and config fields are documented as absolute, 0-indexed values with clearly stated bounds
- **Context-aware** - Every long-running operation accepts `context.Context`, and `Config.Timeout` wraps it with a deadline when set
- **Benchmark-backed changes** - Performance-sensitive changes (e.g. style caching, the ordered stream-export pipeline) are verified with `go test -bench` before and after

## 🧪 Testing Strategy

- **Unit tests**: table-driven where practical; per-package coverage is reproducible via `go test -cover ./...`, and the whole-module figure (97.0%, via `-coverpkg`) is documented in [docs/FEATURE_CHECKLIST.md](docs/FEATURE_CHECKLIST.md#test-coverage)
- **Race detection**: `go test -race ./...` is required to pass, enforced in CI (see [.github/workflows/ci.yml](.github/workflows/ci.yml))
- **Benchmarks**: dedicated `_bench_test.go` files for performance-critical paths (exporter styling, stream export/import)

## 🛠️ Tech Stack

- **Language**: Go 1.26+
- **Excel Library**: [excelize/v2](https://github.com/xuri/excelize)
- **Testing**: Go testing + benchmarks
- **Linting**: golangci-lint

## 📖 Examples

See [examples/](examples/) directory for complete examples:

- **[examples/basic_export/](examples/basic_export/)** - Basic export example
- **[examples/basic_import/](examples/basic_import/)** - Basic import example
- **[examples/web_export/](examples/web_export/)** - Export to an `http.ResponseWriter` for browser download
- **[examples/merge_cells/](examples/merge_cells/)** - Merged cells via `GetFile()`
- **[examples/multisheet/](examples/multisheet/)** - Multi-sheet export/import
- **[examples/struct_mapping/](examples/struct_mapping/)** - Import to structs
- **[examples/events/](examples/events/)** - Event system hooks
- **[examples/styling/](examples/styling/)** - Styling and formatting
- **[examples/csv_format/](examples/csv_format/)** - CSV export/import options (delimiter, BOM, line ending)
- **[examples/stream_export/](examples/stream_export/)** - Row-at-a-time export of a large dataset with a
  concurrent row transformer, progress reporting, and correct `Close()` usage
- **[examples/stream_import/](examples/stream_import/)** - Row-at-a-time import with per-row error handling,
  context cancellation, and correct `Close()` usage

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for setup
instructions, coding conventions, and the pull request checklist. This
project follows the [Code of Conduct](CODE_OF_CONDUCT.md).

## 📄 License

MIT License - See [LICENSE](LICENSE) file

## 🙏 Acknowledgments

Inspired by [Maatwebsite/Laravel-Excel](https://github.com/SpartnerNL/Laravel-Excel)

## 📝 Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history and release notes.

## 🐛 Issues & Support

- **GitHub Issues**: [Report issues](https://github.com/bonarizki-dat/go-excel/issues)
- **Security vulnerabilities**: Do not open a public issue — see [SECURITY.md](SECURITY.md) for private reporting instructions
- **Documentation**: See [docs/](docs/) directory
- **Examples**: See [examples/](examples/) directory

---

For questions or discussions, see documentation files above.
