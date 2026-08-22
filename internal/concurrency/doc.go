// Package concurrency holds the worker-pool and object-pool primitives
// that stream.StreamExporter and stream.StreamImporter use internally
// to process rows concurrently. It moved here from the stream package
// because WorkerPool, PoolManager, Task, Result, ProcessFunc, and
// WorkerError are implementation details of that concurrency, not part
// of the module's public contract: nothing in the happy-path API
// (NewStreamExporter/NewStreamImporter plus the With* options) requires
// a caller to name any of these types directly.
//
// stream.WorkerStats is the one exception, kept as a type alias to
// concurrency.WorkerStats, because StreamExporter.GetStats and
// StreamImporter.GetStats return it as part of their public signature.
//
// Being an internal package, everything exported here is only
// reachable from within this module (see
// https://go.dev/cmd/go/#hdr-Internal_Directories); that is the
// mechanism enforcing "not part of the public contract" above, not a
// promise that these types are simple or stable.
package concurrency
