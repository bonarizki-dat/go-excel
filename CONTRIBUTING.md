# Contributing

Thanks for considering a contribution to `github.com/bonarizki-dat/go-excel`.

This project is pre-1.0 (`v0.x`), so breaking changes are sometimes acceptable
when they fix a genuine design problem — but they still need a clear
migration note in [CHANGELOG.md](CHANGELOG.md).

## Before you start

- For anything beyond a small fix (typo, doc drift, obvious bug), open an
  issue first to discuss the approach. This avoids wasted work on pull
  requests that don't fit the project's direction.
- Check [docs/CODING_STANDARDS.md](docs/CODING_STANDARDS.md)
  for naming, error-handling, comment, and testing conventions used
  throughout this codebase. Reviews will ask for changes that don't follow
  it.
- Check [SECURITY.md](SECURITY.md) instead of opening an issue if you're
  reporting a vulnerability rather than a bug.

## Development setup

Requires Go 1.26 or later (see [go.mod](go.mod)).

```bash
git clone https://github.com/bonarizki-dat/go-excel.git
cd go-excel
make install   # go mod download && go mod tidy
```

## Making a change

1. Create a branch off `main`.
2. Make your change, keeping it focused — separate unrelated fixes into
   separate pull requests.
3. Add or update tests. New behavior needs new tests; bug fixes need a
   regression test that fails before the fix and passes after.
4. Run the full check suite locally before opening a pull request:

   ```bash
   make check       # gofmt, go vet, golangci-lint, go test -race
   make vulncheck    # govulncheck against the module and its dependencies
   ```

   CI runs the same checks (see [.github/workflows/ci.yml](.github/workflows/ci.yml))
   on Go 1.26 across Linux/macOS/Windows, plus a coverage gate (80%
   minimum) — a pull request that doesn't pass `make check` locally won't
   pass CI either.
5. If you touched exported API, update
   [docs/API_REFERENCE.md](docs/API_REFERENCE.md) and add or update the
   relevant `Example_` function so drift becomes a build failure, not a
   future audit finding.
6. If your change is user-visible (new option, bug fix, behavior change),
   add an entry under an "Unreleased" heading in
   [CHANGELOG.md](CHANGELOG.md).

## Code style

- Run `gofmt -s` (via `make fmt`) before committing.
- Follow [docs/CODING_STANDARDS.md](docs/CODING_STANDARDS.md):
  concise godoc comments (prose, not `Parameters:`/`Returns:` blocks), no
  narration or aspirational comments ("for now", "TODO", "simplify later"),
  `any` instead of `interface{}`, sentinel errors checked with `errors.Is`,
  typed errors checked with `errors.As`.
- `golangci-lint run` (via `make lint`) must be clean; it is a required CI
  check and enforces `godot`, `revive`, and the rest of the ruleset in
  [.golangci.yml](.golangci.yml).

## Commit messages and pull requests

- Write commit messages that explain *why*, not just *what* — the diff
  already shows what changed.
- Keep pull requests reviewable: prefer several small, focused PRs over one
  large one when the change has independent pieces.
- Link the issue the pull request addresses, if any.

## Reporting bugs and requesting features

Use the issue templates under `.github/ISSUE_TEMPLATE/`. For bugs, include a
minimal reproduction (a small `main.go` or test is ideal) and your Go and
`excelize` versions.
