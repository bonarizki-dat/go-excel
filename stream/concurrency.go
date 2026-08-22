package stream

import "github.com/bonarizki-dat/go-excel/internal/concurrency"

// WorkerStats is a type alias for the worker pool's statistics type.
// It stays exported here — the only one of its former neighbors that
// does — because StreamExporter.GetStats and StreamImporter.GetStats
// return it as part of their public signature. WorkerPool, PoolManager,
// Task, Result, ProcessFunc, and WorkerError moved to
// internal/concurrency: they are implementation details of how rows
// are processed concurrently, not something the happy-path API
// (NewStreamExporter/NewStreamImporter plus With* options) ever
// requires a caller to name directly. See CHANGELOG.md.
type WorkerStats = concurrency.WorkerStats
