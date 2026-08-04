# Architecture

This document records the rationale behind update-go-tools' structure. It is
updated alongside code in accordance with the release contract
(`.opencode/bin/update-go-tools/CONTRACT.md`): behavior, architecture, tests,
and documentation evolve together.

## Package Layout

```
cmd/update-go-tools/   CLI entrypoint: flag parsing, exit codes, output preamble.
internal/app/          Orchestration (Run* methods) and output rendering.
internal/tool/         Domain: discovery, metadata parsing, update, verify, outdated.
internal/testutil/     Hermetic fixture builder for tests.
```

## Layers

### cmd

The `main` package is intentionally thin. It:

* parses flags (`--json`, `--check`, and positional tool names),
* selects a `Renderer`,
* constructs the `App`,
* maps `Runner` errors to exit codes,
* prints the Go-version and tool-count preamble for the default update path.

It contains no domain logic. Business rules live in `internal/app` and
`internal/tool`.

### internal/app

`App` orchestrates a command from start to finish: it loads the inventory once
and hands it to a `Renderer`. It never touches the terminal directly.

The `Renderer` interface exists because output is a public API (see contract
§5/§6). `TerminalRenderer` and `JSONRenderer` are two real implementations, not
an abstraction awaiting a second consumer. The interface boundary is justified:
it isolates the presentation of each command and keeps JSON free of formatting.

### internal/tool

The domain package owns an immutable `Tool` model backed by `debug.BuildInfo`,
and pure operations over tools.

`Runner` abstracts subprocess execution so commands can be injected in tests
and carry a `context.Context` for cancellation and timeout. The concrete
`DefaultRunner` wraps `exec.CommandContext`. All external operations—`go
install`, `go list`—go through `Runner`, which also ensures GOBIN and proxy
resolution are deterministic.

`GetGobin` intentionally uses the `go` toolchain to avoid reimplementing
GOPATH/GOBIN resolution and to honor user overrides.

## Determinism

* Discovery sorts candidates alphabetically.
* JSON relies on `MarshalIndent` with stable, explicit struct field order.
* Empty list fields marshal to `[]`, never `null`.
* Version ordering uses `golang.org/x/mod/semver` for deterministic comparison.

## Why interfaces exist

* `Runner`: second implementation (a mock) exists in tests; it also gives
  context-aware execution.
* `Renderer`: two concrete implementations exist, and it isolates presentation
  as a stable contract.

Interfaces without a second consumer were avoided. Orchestration is not spread
across packages: the CLI does not re-scan GOBIN after loading it once.

## Error handling

Every external operation specifies its possible failure and the exit code it
maps to (`ExitEnv` for environment, `ExitUsage` for misuse, `ExitFailure` for
update/verify failures). Renderers propagate marshal errors rather than
swallowing them.