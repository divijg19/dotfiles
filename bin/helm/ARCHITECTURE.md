# Architecture

This document records the rationale behind Helm's structure. It is
updated alongside code: behavior, architecture, tests,
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

The `Renderer` interface exists because output is a public API. The four concrete implementations are `TerminalRenderer`,
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

## Output format

Every operation follows the same canonical rhythm:

```
Discovery

Body

Summary
```

Every summary block uses the identical layout — a `Summary` heading followed by aligned label/value rows — so every command ends exactly the same way:

```
Summary

Updated       2
Skipped       1
Failed        0
Duration      3.2s
```

### Human output

Human output uses a canonical symbol set:

| Symbol | Meaning |
|---|---|
| `✓` | Healthy |
| `•` | Skipped |
| `✗` | Failed |
| `↑` | Outdated |
| `ⓘ` | Note |

CI mode substitutes plain ASCII (`OK`, `-`, `FAIL`).

### Examples

#### Inventory with health status (`--list`)

```
$ helm --list
Go: go1.26.5

Discovery

  Gobin       : /home/dev/go/bin
  Executables : 3
  Updatable   : 2
  Local       : 1
  Invalid     : 0

NAME            VERSION             STATUS      PACKAGE
air             v1.67.4             Healthy     github.com/air-verse/air
templ           v0.3.1020           Healthy     github.com/a-h/templ/cmd/templ
Helm (devel)             Local       helm/cmd/helm

Summary

Healthy       2
Local         1
Invalid       0
Unhealthy     0
```

#### Concise plan (`--check`)

```
helm --check
Would update

  air
  templ

Skipped

  • Helm

Summary

Would update  2
Skipped       1
```

#### Detailed plan (`--check --verbose`, same as `--dry-run --verbose`)

```
helm --check --verbose
Would update

  air
    Package : github.com/air-verse/air
    Command : go install github.com/air-verse/air@latest

  templ
    Package : github.com/a-h/templ/cmd/templ
    Command : go install github.com/a-h/templ/cmd/templ@latest

Summary

Would update  2
Skipped       1
```

#### Outdated check (`--outdated`)

```
$ helm --outdated
NAME            CURRENT         STATUS
golangci-lint   v1.64.8         ↑ v1.65.0
gopls           v0.23.0         ✓

Summary

Checked       2
Outdated      1
Up-to-date    1
```

#### Quiet update for scripting (`-q`)

```
helm -q
Updated       2
Skipped       0
Failed        0
Duration      3.2s
```

#### Deterministic CI output (`--ci`)

```
$ helm --list --ci
gobin: /home/dev/go/bin
go-version: go1.26.5
executables: 3
updatable: 2
local: 1
invalid: 0

air                 v1.67.4            OK         github.com/air-verse/air
templ               v0.3.1020          OK         github.com/a-h/templ/cmd/templ

healthy: 2
local: 1
invalid: 0
unhealthy: 0
```

#### CI update (`--ci`)

CI update distinguishes per-tool records from the summary counts so scripts can parse both unambiguously:

```
$ helm --ci
gobin: /home/dev/go/bin
go-version: go1.26.5
executables: 3
updatable: 2
local: 1
invalid: 0

updated: air
updated: templ

updated-count: 2
skipped-count: 0
failed-count: 0
```

#### Update a single tool

```
helm golangci-lint
Go: go1.26.5

Discovery

  Gobin       : /home/dev/go/bin
  Executables : 3
  Updatable   : 2
  Local       : 1
  Invalid     : 0

[01/02] golangci-lint              ✓

Summary

Updated       1
Skipped       0
Failed        0
Duration      1.4s
```

## JSON schema

`--json` is a pure output renderer available on every operation. The schema is stable: field names, nesting, and ordering never change between releases, and human formatting never affects the JSON shape. Empty list fields are always emitted as `[]`, never `null`.

### Envelope

Every operation response carries a JSON envelope:

```json
{
  "operation": "list",
  "success": true,
  "tools": []
}
```

* `operation` — the logical operation. Values are frozen: `list`, `check`, `update`, `outdated`. There are no aliases: `--dry-run` emits `"operation": "check"` because it invokes the same planning operation.
* `success` — `true` unless the operation finished with issues (for example an update with failures, or an inventory with unhealthy/invalid binaries). Scripts can gate on `.success` instead of parsing the payload.

### Schemas by command

| Command | JSON Schema |
|---|---|
| `--list --json` | `{"operation": "list", "success": bool, "tools": [ToolReport...]}` |
| `--info <tool> --json` | `ToolReport` |
| `--check/--dry-run --json` | `{"operation": "check", "success": bool, "would_update": [PlanItem...], "skipped": [PlanItem...]}` |
| `--outdated --json` | `{"operation": "outdated", "success": bool, "results": [OutdatedItemReport...]}` |
| default `--json` | `{"operation": "update", "success": bool, "updated": [], "skipped": [], "failed": [], "notes": []}` |

### Report types

* `ToolReport`: `name`, `version`, `package_path`, `module_path`
* `PlanItem`: `name`, `package_path`, `install_target`, `command`
* `OutdatedItemReport`: `name`, `current`, `latest`, `outdated`, `error` (omitted on success)

### Compatibility

Within the 1.x series:

* existing fields will not be renamed or removed;
* field meanings will not change;
* new fields may only be added in a backward-compatible manner;
* arrays are never `null`;
* `operation` values are stable (`list`, `check`, `update`, `outdated`).
