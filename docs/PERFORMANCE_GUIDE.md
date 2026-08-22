# Performance Guide - Go Excel Library

> Understanding memory usage, benchmark interpretations, and optimization tips

**Caveat:** most of the numbers in this guide (everything above roughly
100K rows) were measured against a build of the stream exporter that had
the data-loss/race bug described in [CHANGELOG.md](../CHANGELOG.md) —
see [README.md](../README.md#-performance-benchmarks) for details. They
are kept here as historical context for the shape of the memory/time
curves, not as current guarantees. Re-run the benchmarks in
`excel/exporter`, `excel/importer`, and `stream` for current numbers
before relying on any figure below. The README table is the only
re-verified set of figures in this repository.

**Hard limit:** the `.xlsx` format caps every worksheet at
`excelize.TotalRows` = 1,048,576 rows (2^20); `excelize` returns
`ErrMaxRows` if you try to exceed it. Any figure below for "5M rows" or
similar refers to a *dataset* split across multiple sheets or files, not
a single sheet — a single sheet with 5M rows cannot exist. See
[README.md](../README.md#-performance-benchmarks) for the current,
verified 1M and max-single-sheet (1,048,575 row) numbers.

---

## Table of Contents

1. [Understanding Memory Usage](#understanding-memory-usage)
2. [Memory Profiling Summary](#memory-profiling-summary)
3. [Benchmark Interpretations](#benchmark-interpretations)
4. [Choosing the Right Tool](#choosing-the-right-tool)
5. [Optimization Tips](#optimization-tips)

---

## Understanding Memory Usage

### Excelize Library Limitation

**Key Finding**: 98% of memory usage comes from the underlying `excelize` library, not our code.

| Component | Memory (100K rows) | % of Total |
|-----------|-------------------|------------|
| Excelize internal | ~800 MB | 98% |
| Our code (with pooling) | ~14 MB | 2% |

**Implication**: Further optimization has < 2% potential gain. Our library is already optimized within excelize constraints.

### Why BasicImporter is More Efficient for Medium Files

**Counter-intuitive Discovery**: BasicImporter uses LESS memory than StreamImporter for datasets < 500K rows.

| Dataset | BasicImporter | StreamImporter | Winner |
|---------|---------------|----------------|--------|
| 10K rows | ~45 MB | ~88 MB | **BasicImporter** (-49% memory) |
| 100K rows | **551 MB** | **872 MB** | **BasicImporter** (-37% memory) |
| 1M rows | ~5 GB (may OOM) | ~8.5 GB | **StreamImporter** (prevents crash) |

**Reason:**
- `BasicImporter` uses `file.GetRows()` which pre-allocates efficiently
- `StreamImporter` uses `file.Rows()` iterator with per-row allocation overhead
- Iterator overhead > batch allocation for medium datasets

**Recommendation**: Use `BasicImporter` for datasets < 500K rows

### When to Use StreamImporter

Use `StreamImporter` when:
- **Dataset > 1M rows** - Prevents out-of-memory errors
- **Memory-constrained environment** - Even if dataset is smaller
- **Processing in chunks** - Can process very large files incrementally

**Memory Characteristics:**
- StreamImporter: ~8.5 GB per 1M rows
- BasicImporter: ~5 GB per 1M rows (but may OOM)

**Trade-off**: StreamImporter uses more memory but prevents crashes.

---

## Memory Profiling Summary

### Key Findings from Profiling

1. **Bottleneck Identification**
   - `rows.Columns()` method: 1.54 GB = 41% of total memory
   - Excelize internal buffers: ~800 MB = 98% of allocations
   - Our code: ~14 MB = 2% of allocations

2. **Pool Optimization Results**
   - Pool-based row buffer allocation: -1.6% memory improvement
   - ~14 MB saved for 100K rows
   - Limited improvement due to excelize constraints

3. **True O(1) Memory Not Possible**
   - Excelize API requires loading row data into memory
   - `Rows()` iterator still allocates per-row buffers
   - Streaming architecture minimizes growth but cannot achieve true constant memory

### Memory Usage by Dataset Size

| Rows | BasicExporter | BasicImporter | StreamExporter | StreamImporter |
|------|---------------|---------------|----------------|----------------|
| 1K | ~5 MB | ~5 MB | ~8 MB | ~10 MB |
| 10K | ~45 MB | ~45 MB | ~80 MB | ~88 MB |
| 100K | ~200 MB | **551 MB** | ~250 MB | **872 MB** |
| 500K | ~1 GB | ~2.8 GB | ~1.2 GB | ~4.4 GB |
| 1M | ~2 GB (risk OOM) | ~5 GB (risk OOM) | **~2.5 GB** ✅ | **~8.5 GB** ✅ |
| 1,048,576 (max per sheet) | N/A | N/A | max single-sheet size; see README | max single-sheet size; see README |
| 5M (across ~5 sheets) | N/A | N/A | ~12 GB (est.) | ~42 GB (est.) |

**Note**: Memory usage depends on data complexity (formulas, styles, etc.). Rows beyond 1,048,576 require multiple sheets or files — see the hard limit note above.

### System Requirements

| Dataset Size | Minimum RAM | Recommended RAM | Approach |
|--------------|-------------|-----------------|----------|
| < 100K rows | 1 GB | 2 GB | BasicImporter/Exporter |
| 100K - 500K | 4 GB | 8 GB | BasicImporter/Exporter |
| 500K - 1,048,576 (max per sheet) | 8 GB | 16 GB | StreamImporter/Exporter |
| Multi-sheet totals beyond that | 16 GB+ | 32 GB+ | Split across sheets/files with `ExportToSheets` |

---

## Benchmark Interpretations

### How to Read Benchmark Results

**Example Output:**
```
BenchmarkExporter1M-8   1  9,909,924,417 ns/op  2,477,342,088 B/op  31,002,993 allocs/op
```

**Breaking it down:**
- `BenchmarkExporter1M-8` - Benchmark name, 8 = number of CPU cores
- `1` - Number of iterations (1 because it's slow)
- `9,909,924,417 ns/op` - Nanoseconds per operation (9.9 seconds)
- `2,477,342,088 B/op` - Bytes allocated per operation (2.47 GB)
- `31,002,993 allocs/op` - Number of allocations per operation (31M)

### What the Numbers Mean

**Time Performance:**
- **ns/op**: Nanoseconds per operation
  - 1,000,000,000 ns = 1 second
  - 9,909,924,417 ns = 9.9 seconds
- **Lower is better** - Faster execution

**Memory Performance:**
- **B/op**: Bytes allocated per operation
  - 1,024 B = 1 KB
  - 1,048,576 B = 1 MB
  - 2,477,342,088 B = 2.47 GB
- **Lower is better** - Less memory usage

**Allocation Count:**
- **allocs/op**: Number of allocations per operation
  - Higher = more GC pressure
  - Lower = better GC performance
- **Lower is better** - Fewer allocations

### Performance Targets vs Actual

| Dataset | Export Target | Export Actual | Import Target | Import Actual | Status |
|---------|---------------|---------------|---------------|---------------|--------|
| 1K | < 100ms | 11.5ms | < 150ms | 7.6ms | ✅ **8.7x faster** |
| 10K | < 500ms | 86.8ms | < 750ms | 91.4ms | ✅ **5.8x faster** |
| 100K | < 3s | 1.88s | < 5s | 1.43s | ✅ **1.6x faster** |
| 1M | < 30s | 9.9s | < 45s | 16.4s | ✅ **3.0x faster** |

**All targets exceeded by 1.6-8.7x!**

### Time Scaling

| Dataset | Export Time | Import Time | Export Time/Row | Import Time/Row |
|---------|-------------|-------------|-----------------|-----------------|
| 1K | 11.5ms | 7.6ms | 11.5µs | 7.6µs |
| 10K | 86.8ms | 91.4ms | 8.7µs | 9.1µs |
| 100K | 1.88s | 1.43s | 18.8µs | 14.3µs |
| 1M | 9.9s | 16.4s | 9.9µs | 16.4µs |

**Analysis:**
- Export: 8.7-18.8µs per row (near-constant with slight variation)
- Import: 7.6-16.4µs per row (near-constant with slight variation)
- Both operations scale near-linearly with dataset size ✅

---

## Choosing the Right Tool

### Decision Tree

```
How large is your dataset?
│
├─ < 100K rows
│  └─ Use BasicImporter + BasicExporter
│     ✅ Best memory efficiency
│     ✅ Fastest performance
│     Memory: ~200-550 MB
│
├─ 100K - 500K rows
│  └─ Use BasicImporter + BasicExporter
│     ✅ Still more efficient than streaming
│     Memory: ~550 MB - 2.8 GB
│
├─ 500K - 1M rows
│  └─ Use StreamImporter + StreamExporter
│     ⚠️  Higher memory usage
│     Memory: ~2.5-8.5 GB
│
├─ 1M - 1,048,576 rows (max per sheet)
│  └─ Use StreamImporter + StreamExporter
│     ✅ Prevents out-of-memory errors
│     Memory: ~8-10 GB per 1M rows
│
└─ > 1,048,576 rows total
   └─ Split across multiple sheets (ExportToSheets) or files —
      a single .xlsx sheet cannot hold more than 1,048,576 rows
```

### Comparison Table

| Feature | BasicExporter | StreamExporter | BasicImporter | StreamImporter |
|---------|---------------|----------------|---------------|----------------|
| **Best For** | < 500K rows | > 1M rows | < 500K rows | > 1M rows |
| **Memory (1M)** | ~2 GB | ~2.5 GB | ~5 GB | ~8.5 GB |
| **Speed (1M)** | 9.9s | 10s | 12s | 16s |
| **Memory Efficiency** | ✅ Best | Good | ✅ Best | Good |
| **OOM Prevention** | ⚠️ Risk | ✅ Safe | ⚠️ Risk | ✅ Safe |
| **Use Case** | Medium files | Large files | Medium files | Large files |

### When to Use Each

**BasicExporter:**
- Files < 500K rows
- When memory is not a concern
- Fastest performance for medium files

**StreamExporter:**
- Files > 1M rows
- Memory-constrained environments
- When OOM prevention is critical

**BasicImporter:**
- Files < 500K rows
- When you need all data in memory
- Fastest performance for medium files

**StreamImporter:**
- Files > 1M rows
- When processing row-by-row
- When OOM prevention is critical

---

## Optimization Tips

### 1. Choose the right importer/exporter for the dataset size

```go
// < 500K rows: BasicImporter (see Memory Usage table above)
imp := importer.NewImporter(excel.WithHeadings(true))
if err := imp.ImportFromFile(ctx, "medium_file.xlsx"); err != nil {
    log.Fatal(err)
}
data := imp.GetRows()
```

```go
// > 1M rows or memory-constrained: StreamImporter
f, _ := os.Open("large_file.xlsx")
defer f.Close()

si, err := stream.NewStreamImporter(f, stream.WithImportWorkers(4))
if err != nil {
    log.Fatal(err)
}
defer si.Close()

err = si.ImportStream(ctx, func(ctx context.Context, rowNum int, row []any) ([]any, error) {
    // process row
    return row, nil
})
```

### 2. Use `WithRows` to process a subset of rows

Both `excel.WithRows` (basic importer) and `stream.WithImportRows`
(stream importer) accept a 1-based, inclusive row range instead of
loading the whole sheet, useful for splitting very large files into
passes:

```go
imp := importer.NewImporter(excel.WithRows(1, 500000))
```

### 3. Tune worker count for `RowTransformer`-based stream exports

`stream.WithExportWorkers` only matters when a `RowTransformer` is set via
`stream.WithRowTransform`; without a transformer, rows still write
through a single ordered writer goroutine.

```go
exp := stream.NewStreamExporter(f,
    stream.WithRowTransform(transform),
    stream.WithExportWorkers(runtime.NumCPU()),
)
```

### 4. Use progress callbacks for long-running operations

```go
imp := importer.NewImporter(excel.WithProgress(func(processed, total int64) {
    log.Printf("imported %d/%d rows", processed, total)
}))
```

```go
exp := stream.NewStreamExporter(f,
    stream.WithStreamProgress(func(count uint64, msg string) {
        log.Printf("[%d] %s", count, msg)
    }, 1000), // report every 1000 rows
)
```

### 5. Pre-allocate data slices when the row count is known

```go
rowCount := 100000
data := make([][]any, 0, rowCount)
```

### 6. Always use context timeouts for long-running operations

`excel.WithTimeout` wraps the operation's context for you; alternatively
pass an already-bounded context directly:

```go
imp := importer.NewImporter(excel.WithTimeout(5 * time.Minute))

// or:
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
if err := imp.ImportFromFile(ctx, "large.xlsx"); err != nil {
    log.Fatal(err)
}
```

### 7. Monitor memory usage during development

```go
import "runtime"

func logMemStats(label string) {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    log.Printf("%s - Alloc: %d MB, TotalAlloc: %d MB, Sys: %d MB, NumGC: %d",
        label, m.Alloc/1024/1024, m.TotalAlloc/1024/1024, m.Sys/1024/1024, m.NumGC)
}
```

---

## Additional Resources

- [README.md](../README.md#-performance-benchmarks) - the re-verified benchmark table
- [docs/FEATURE_CHECKLIST.md](FEATURE_CHECKLIST.md) - which performance goals were met, revised, or rejected

Re-measure rather than quoting any figure above:

```bash
go test -bench . -benchmem -run '^$' ./excel/exporter ./excel/importer ./stream
```
