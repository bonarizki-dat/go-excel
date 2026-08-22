/*
Package pipeline implements a generic multi-stage concurrent processing
primitive: data flows through a sequence of named stages, each with its
own concurrency level, connected by backpressured channels.

It lives under internal/ rather than in the public stream package
because nothing in this module's public API uses it: StreamExporter and
StreamImporter (in the stream package) are built directly on
internal/concurrency.WorkerPool, not on Pipeline. Exporting a type with
no production call path invites external callers to depend on
concurrency, shutdown, and backpressure behavior that is validated only
by this package's own tests. Moving it under internal/ keeps it
available for future use inside this module without it being part of
the module's compatibility guarantees.

# Review

This package remains unused by any production call path as of the
v0.3.0 harden pass (see CHANGELOG.md). Revisit 30 days after the
v0.3.0 tag is created: if still nothing in this module calls New
outside this package's own tests, delete it rather than carry an
untested-in-production primitive indefinitely. File a tracking
issue against this review date once v0.3.0 is tagged (deferred here
because `gh` was unavailable in the environment that wrote this note).

# Usage

	p := pipeline.New(ctx)
	p.AddStage("parse", parseFunc, 4)
	p.AddStage("validate", validateFunc, 2)
	p.Start()
	defer p.Stop()

	go func() {
		for i := 0; i < 1000; i++ {
			p.Input() <- fmt.Sprintf("data-%d", i)
		}
		close(p.Input())
	}()

	for result := range p.Output() {
		log.Printf("Result: %v", result)
	}
*/
package pipeline
