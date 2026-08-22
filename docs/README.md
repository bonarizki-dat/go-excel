# Documentation Index

This folder contains documentation for go-excel, kept flat on purpose —
each file has one job.

---

## 📁 Files

- **[API_REFERENCE.md](API_REFERENCE.md)** - Full API reference: exporters,
  importers, config/options, events, styling (including best practices,
  limitations, and merge cells), multi-sheet, struct mapping, interfaces,
  error types
- **[PERFORMANCE_GUIDE.md](PERFORMANCE_GUIDE.md)** - Memory profile summary,
  decision tree (Basic vs. Stream), and optimization tips
- **[CODING_STANDARDS.md](CODING_STANDARDS.md)** - Coding conventions:
  naming, error handling, options pattern, context support, testing
- **[FEATURE_CHECKLIST.md](FEATURE_CHECKLIST.md)** - Feature tracker. Part 1
  is verified against the code and dated; Part 2 lists unimplemented
  designs; Part 3 is a historical log kept only to record corrections
- **[../README.md](../README.md)** - Project overview, quick start, and
  installation
- **[../CHANGELOG.md](../CHANGELOG.md)** - Release notes and change history

### Performance numbers

There is no separate benchmark-results file. Six historical
profiling/roadmap documents that used to live in `docs/performance/` were
deleted: their numbers predated the streaming-export race fix, their
roadmaps proposed files that were never built (`perf/pool_manager.go`,
`perf/simd_amd64.go`), and their targets (`<500 MB per 10M rows`, "true
O(1) streaming") were later proven unreachable through excelize.

Current sources instead:

- **[../README.md](../README.md#-performance-benchmarks)** - measured, re-verified benchmark table
- **[PERFORMANCE_GUIDE.md](PERFORMANCE_GUIDE.md)** - memory profile summary, decision tree, optimization tips
- `go test -bench . -benchmem -run '^$' ./...` - the only authoritative number for your machine

---

## 🔗 External References

- **Go Documentation**: https://go.dev/doc/
- **Excelize Library**: https://github.com/xuri/excelize
- **Laravel Excel**: https://github.com/SpartnerNL/Laravel-Excel (inspiration)
