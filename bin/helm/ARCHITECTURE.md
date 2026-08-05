# Architecture

This document records the rationale behind Helm's structure. It is
updated alongside code in accordance with the release contract
(`.opencode/bin/helm/CONTRACT.md`): behavior, architecture, tests,
and documentation evolve together.

## Package Layout

```
cmd/helm/            CLI entrypoint: flag parsing, renderer selection, exit codes.
cmd/update-go-tools/ Alias entrypoint: delegates to cmd/helm.
internal/app/          Orchestration (Run* methods) and output rendering.
internal/tool/         Domain: discovery, metadata parsing, update, outdated, verify.
internal/testutil/     Hermetic fixture builder for tests.
```

## Layers

```
Discovery
    ↓
Inventory
    ↓
Operation
    ↓
Renderer
```

Operations:

```
Update
List
Plan
Outdated
```

Renderers:

```
Terminal
JSON
Quiet
CI
```

Business logic must not know about formatting. Renderers must not contain
business logic.

### cmd

The `main` package is intentionally thin. It:

* parses flags (`--json`, `--quiet`/`-q`, `--ci`, `--verbose`, `--check`,
  `--dry-run`, and positional tool names),
* selects a `Renderer` via `app.NewRenderer(mode, verbose)`,
* constructs the `App`,
* renders the discovery `Header` through the selected renderer, skipping the
  `go env GOVERSION` subprocess when the mode's renderer no-ops the header,
* maps `Runner` errors to exit codes.

It contains no domain logic. Business rules live in `internal/app` and
`internal/tool`.

`--check` and `--dry-run` are both routed to the same `RunPlan` operation; they
are aliases with a single execution path. `--verbose` only affects how the
planning renderer presents the report.

### internal/app

`App` orchestrates a command from start to finish: it loads the inventory once
and hands immutable report structures to a `Renderer`. It never touches the
terminal directly.

The `Renderer` interface exists because output is a public API (see contract
§5/§6). The four concrete implementations are `TerminalRenderer`,
`JSONRenderer`, `QuietRenderer`, and `CIRenderer` — each is a distinct,
well-defined output mode, selected by `RenderMode`. The interface boundary
isolates presentation and keeps JSON, quiet, and CI output free of incidental
formatting.

`Renderer` methods:

* `Header(HeaderInfo)` — the `Go`/`Discovery` header. JSON and quiet
  renderers no-op it; the header therefore belongs to the renderer, not the
  CLI, so every operation shares the same `Go → Discovery → Body → Summary`
  shape.
* `Inventory(InventoryReport)` — `--list`.
* `Plan(PlanReport)` — `--check` / `--dry-run`.
* `Outdated(OutdatedReport)` — `--outdated`.
* `Update(UpdateReport)` — default update.
* `Info(...)` — `--info <tool>`.

The optional `ProgressSink` interface (`OnProgress`) is implemented only by the
interactive `TerminalRenderer`; business logic probes for it via type
assertion so non-interactive modes stay deterministic.

### internal/tool

The domain package owns an immutable `Tool` model backed by `debug.BuildInfo`,
and pure operations over tools: discovery, metadata parsing, planning
(`Plan`), updating, and outdated checks. Planning has a single owner: `Plan`
returns the `PlanResult` (to-update, skipped, invalid) that renderers consume,
and `InstallRef`/`InstallCommand` are the single source of truth for what
`go install <target>@latest` means. The app layer and renderers never
re-derive which tools would update or how the install command is formed.

`Runner` abstracts subprocess execution so commands can be injected in tests
and carry a `context.Context` for cancellation and timeout. The concrete
`DefaultRunner` wraps `exec.CommandContext`. All external operations—`go
install`, `go list`—go through `Runner`, which also ensures GOBIN and proxy
resolution are deterministic.

`GetGobin` intentionally uses the `go` toolchain to avoid reimplementing
GOPATH/GOBIN resolution and to honor user overrides.

## Renderers

* `TerminalRenderer` — human-oriented. Unicode symbol set
  (`✓` healthy, `•` skipped, `✗` failed, `↑` outdated, `ⓘ` note), dynamic
  column widths with consistent spacing, streaming progress during updates.
* `JSONRenderer` — pure data. Stable schema, arrays never `null`, deterministic
  ordering, no presentation formatting. Every report carries an
  `OperationEnvelope` (`operation`, `success`) set by the application layer;
  the renderer only serializes it and never branches on it. Human renderers
  ignore the envelope entirely.
* `QuietRenderer` — scripting mode. Same visual language as Terminal but
  suppresses the header, discovery summary, progress, and per-tool status;
  update emits only the summary plus diagnostics and failures.
* `CIRenderer` — deterministic machine-oriented terminal output. ASCII-only,
  line-oriented, no progress renderer, no cursor movement, reproducible across
  environments. Summary keys never collide with per-record keys: CI update
  records are `updated: <name>` while totals are `updated-count: <n>`, and
  every operation separates records from summary counts with a blank line.
  `--verbose` selects the detailed planning view for human renderers.

## Determinism

* Discovery sorts candidates alphabetically.
* JSON relies on `MarshalIndent` with stable, explicit struct field order.
* Empty list fields marshal to `[]`, never `null`.
* Version ordering uses `golang.org/x/mod/semver` for deterministic comparison.
* Renderers receive immutable report structures and perform presentation only.
* Discovery happens once per invocation and is cached by `App`.
* CI mode performs no streaming or timing-dependent rendering; output is
  line-oriented and stable.

## Why interfaces exist

* `Runner`: second implementation (a mock) exists in tests; it also gives
  context-aware execution.
* `Renderer`: four concrete implementations exist, and it isolates presentation
  as a stable contract.

Interfaces without a second consumer were avoided. Orchestration is not spread
across packages: the CLI does not re-scan GOBIN after loading it once.

## Error handling

Every external operation specifies its possible failure and the exit code it
maps to (`ExitEnv` for environment, `ExitUsage` for misuse, `ExitFailure` for
update/inventory failures). Renderers propagate marshal errors rather than
swallowing them.
