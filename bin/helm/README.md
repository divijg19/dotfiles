# Helm
> alias: update-go-tools

A lightweight utility to discover, inspect, and maintain Go developer binaries installed via `go install`. Reads embedded module metadata through `debug/buildinfo`.

```
helm [tool...]          # update specific tools (or all if omitted)
helm --list             # inventory with health status
helm --check            # plan updates without executing
helm --outdated         # check which tools have newer releases
helm --info <tool>      # detailed metadata for a single tool
helm --json             # machine-readable JSON output (any operation)
helm --ci               # deterministic, script-friendly output
helm --quiet            # suppress headers; emit only data
helm --help / --version
```

There are no subcommands — all interactions are flag-driven operating modes on a single responsibility.

## Quick start

```bash
go build -o helm ./cmd/helm
./helm --list
```

## Flags

| Flag | Description |
|---|---|
| `--help`, `-h` | Display help documentation |
| `--version`, `-v` | Display version information |
| `--list` | Canonical inventory: tool, version, health status, and package for every Go-managed binary |
| `--check` / `--dry-run` | Summarize pending updates without executing |
| `--outdated` | Check upstream releases for installed tools and report which are out of date |
| `--info <tool>` | Show detailed metadata for a specific tool |
| `--json` | Output renderer: emit machine-readable JSON for any operation |
| `--quiet`, `-q` | Suppress the discovery header; emit only data (for scripting) |
| `--ci` | Deterministic, ASCII-only output (no ANSI, no Unicode, no progress) |
| `--verbose`, `-V` | Detailed planning view: packages and install commands |

`--json`, `--quiet`/`-q`, `--ci`, and `--verbose`/`-V` are output modifiers, not operations. They can be combined with any operation.

## Key features

- **Single responsibility** — one job: keep `go install` tools up to date.
- **Multiple output modes** — terminal, JSON, quiet, and CI (deterministic, ASCII-only).
- **Stable contract** — flags, exit codes, and JSON schema are guaranteed stable within the 1.x series.
- **Hermetic testing** — tests never touch your real GOBIN, module cache, or network.

## Documentation

| Doc | Covers |
|---|---|
| [Architecture](ARCHITECTURE.md) | Design rationale, layers, renderers, determinism |
| [Development](DEVELOPMENT.md) | Building, testing, versioning, and exit codes |
| [Changelog](CHANGELOG.md) | Version history |

Project structure: `cmd/` for entrypoints, `internal/app/` for orchestration, `internal/tool/` for domain logic, `internal/cli/` for the CLI package, and `testdata/` for hermetic fixtures.
