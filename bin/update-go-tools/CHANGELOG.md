# Changelog

All notable changes to update-go-tools are documented here.

## [Unreleased]

### Changed

* v1.4.0 architectural consolidation:
  * Renderers now consume immutable report structures instead of deriving statistics independently.
  * `Renderer` interface updated with `Inventory`, `Verify`, `Outdated`, `Update`, and `DryRun` report types.
  * `App` caches discovery results per invocation, eliminating redundant `Load()` calls.
  * `Update()` pipeline separates planning from execution internally.
  * `--dry-run` is a planning-only command, distinct from `--check`.
  * Progress renderer dynamically expands only when subprocess output exists.
  * JSON output uses stable lowercase field names with `json` tags on all report structs.
  * All duplicated counting logic eliminated; `LoadSummary` is the single source of truth.

## [v1.4.0]

### Changed

* CLI version bumped to v1.4.0.

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
* Release contract at `.opencode/bin/update-go-tools/CONTRACT.md`.

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