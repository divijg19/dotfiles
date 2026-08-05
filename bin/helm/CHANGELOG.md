# Changelog

All notable changes to Helm are documented here.

## [Unreleased]

## [v1.6.0]

### Changed

* Renamed and rebranded from `update-go-tools` to `Helm`.
  The binary is now `helm`; `update-go-tools` is preserved as a
  first-class alias that delegates to the same entrypoint.
* Module path changed from `update-go-tools` to `helm`.
* Import paths updated from `update-go-tools/internal/...` to
  `helm/internal/...`.
* Primary CLI entrypoint moved to `cmd/helm`; `cmd/update-go-tools`
  is the alias entrypoint.
* App name in output changed from `update-go-tools` to `Helm`.
* Version bumped to v1.6.0.

## [v1.5.0]

### Changed

* CLI consolidation, UX consistency & interface finalization:
  * `--list` is now the single canonical inventory command and absorbs the
    health reporting formerly exposed by `--verify`. It reports tool, version,
    health status (`Healthy`/`Local`/`Unhealthy`/`Invalid`), and package.
  * `--verify` is removed as an independent implementation; `--list` covers
    verification.
  * `--check` and `--dry-run` are aliases of a single planning operation with
    one execution path. A new `--verbose`/`-V` output flag switches between the
    concise summary (default) and the detailed execution plan
    (packages + commands).
  * New `--quiet`/`-q` renderer for scripting: suppresses banner, discovery
    summary, progress, and per-tool status; emits only
    `Updated`/`Skipped`/`Failed`/`Duration` plus diagnostics and failures.
  * New `--ci` renderer: deterministic, ASCII-only, line-oriented terminal
    output with no ANSI, no Unicode, no progress renderer, and no cursor
    movement.
  * `--json` is now a pure output renderer available on every operation with a
    single stable schema; arrays are never `null` and ordering is deterministic.
    Every response carries a JSON envelope with frozen `operation` values
    (`list`, `check`, `update`, `outdated`; `--dry-run` reports `check`) and a
    `success` boolean. No CLI version or timestamps are embedded.
  * The discovery header (`Go:`, `Discovery`) moved from the CLI into renderers
    so every operation follows `Go → Discovery → Body → Summary`.
  * `Renderer` interface replaced `Verify`/`Check`/`DryRun` with a unified
    `Plan` report and `Header`; concrete renderers are `Terminal`, `Quiet`,
    `CI`, and `JSON`.
  * Unified symbol set (`✓`, `•`, `✗`, `↑`, `ⓘ`) and summary-block formatting
    across all human output; CI uses an ASCII equivalent.
  * Final presentation polish: every command follows the canonical
    `Go → Discovery → Body → Summary` rhythm, the `Scanning` action label was
    replaced with a `Discovery` header, all summary blocks are visually
    identical (aligned 14-character labels), and the plan command now ends
    with the same `Summary` block instead of a duplicated `Would update: N`
    count.
  * `--quiet`/`-q` works on every operation (not just update), suppressing the
    discovery header while keeping the requested data and summary.
  * Help output groups output flags under `Output modifiers`.

### Polish

* Release-polish pass (engineering certification) before the v1.5.x freeze:
  * Planning is now owned by the domain layer (`tool.Plan`); the app layer no
    longer re-derives the update plan, and update/plan filtering share one
    implementation.
  * `tool.InstallCommand`/`tool.InstallRef` are the single source of truth for
    the `go install <target>@latest` command, so the plan always displays
    exactly what execution would run.
  * `TerminalRenderer.Outdated` no longer keeps unused local counters; it
    consumes the report summary like every other renderer.
  * Update skipped/diagnostic bullets are indented consistently with every
    other section (`  • ...`), and an empty inventory now still ends with the
    canonical `Summary` block.
  * CI mode: the plan summary is preceded by the same blank line as the other
    operations, and update summary keys are unambiguous
    (`updated-count`/`skipped-count`/`failed-count`) instead of colliding with
    per-tool `updated:`/`skipped:`/`failed:` records.
  * The discovery header no longer runs `go env GOVERSION` in JSON or quiet
    modes via a renderer type-switch; the mode is decided once in `cmd`.
  * `go.mod` dependency metadata corrected (`golang.org/x/mod` is a direct
    requirement, not `indirect`).
  * Benchmarks fixed to measure the real code paths hermetically (discovery,
    planning, outdated) and recorded in the certification report.
  * New regression tests: plan ownership, install-command agreement,
    update skipped indentation, summary-from-report counting, CI count keys,
    `--info --json` envelope absence, and empty-inventory summary.

## [v1.4.0]

### Changed

* CLI version bumped to v1.4.0.
* v1.4.0 architectural consolidation:
  * Renderers now consume immutable report structures instead of deriving statistics independently.
  * `Renderer` interface updated with `Inventory`, `Verify`, `Outdated`, `Update`, and `DryRun` report types.
  * `App` caches discovery results per invocation, eliminating redundant `Load()` calls.
  * `Update()` pipeline separates planning from execution internally.
  * `--dry-run` is a planning-only command, distinct from `--check`.
  * Progress renderer dynamically expands only when subprocess output exists.
  * JSON output uses stable lowercase field names with `json` tags on all report structs.
  * All duplicated counting logic eliminated; `LoadSummary` is the single source of truth.

## [v1.3.0]

### Added

* `--outdated` flag to compare installed versions against upstream releases
  using semantic version ordering.
* `--json` global flag that emits stable machine-readable JSON for `--list`,
  `--info`, `--verify`, `--outdated`, and update output.
* `internal/app` orchestration layer separating CLI from domain logic.
* `Renderer` interface with `TerminalRenderer` and `JSONRenderer`
  implementations.
* `Runner` interface for injectable, context-aware subprocess execution.
* `testdata/` golden and JSON fixtures, plus a hermetic fixture builder.

## Changed

* Version is now injected at build time via `-ldflags "-X main.version=..."`
  instead of being hardcoded; it defaults to `dev`.
* GOBIN discovery is performed once per invocation; the update path no longer
  re-scans the filesystem after listing.
* JSON responses emit `[]` rather than `null` for empty lists, and honor
  `--check` via the `check_only` field.
* Local/devel tools are reported as skipped in JSON output rather than
  misclassified as failed.
* Invalid-binary diagnostics render a stable, categorized message instead of
  raw toolchain text.

### Fixed

* JSON update output previously classified skipped local/devel tools as
  failures.
* JSON update output previously discarded the skipped-tool list and the
  `--check` flag.
* The CLI loaded the tool inventory twice on the default update path.

### Tested

* Unit tests for `CanUpdate`, `InstallTarget`, `Update`, `CheckOutdated`, and
  `Verify`, including error paths.
* Integration tests exercising discovery, metadata parsing, and verification
  against a temporary GOBIN of real fixture binaries.
* Golden CLI tests covering stdout, stderr, and exit codes for every command.