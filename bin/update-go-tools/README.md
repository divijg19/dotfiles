# `update-go-tools`

`update-go-tools` is a lightweight utility designed to discover, inspect, and
maintain Go developer binaries installed via `go install`. It reads embedded
module build metadata through `debug/buildinfo`.

The utility performs one responsibility:

> Discover, inspect and maintain Go tools installed via `go install`.

Everything else is a different way of interacting with that single
responsibility. The CLI exposes a small number of orthogonal, flag-driven
operating modes. There are no subcommands.

## Command Usage

```
update-go-tools [tool...]
update-go-tools --list [--json|--ci]
update-go-tools --check [--verbose|-V]
update-go-tools --dry-run [--verbose|-V]
update-go-tools --outdated [--json|--ci]
update-go-tools --info <tool>
update-go-tools --json
update-go-tools --quiet | -q
update-go-tools --ci
update-go-tools --help
update-go-tools --version
```

Without arguments, updates all discovered Go tools.
With one or more tool names, updates only those specified tools.

## Operations and Flags

<table>
  <tr>
    <th>Flag</th>
    <th>Description</th>
  </tr>
  <tr>
    <td><code>--help</code>, <code>-h</code></td>
    <td>Display help documentation</td>
  </tr>
  <tr>
    <td><code>--version</code>, <code>-v</code></td>
    <td>Display version information</td>
  </tr>
  <tr>
    <td><code>--list</code></td>
    <td>Canonical inventory: tool, version, health status, and package for every Go-managed binary</td>
  </tr>
  <tr>
    <td><code>--check</code></td>
    <td>Summarize pending updates without executing them</td>
  </tr>
  <tr>
    <td><code>--dry-run</code></td>
    <td>Alias for <code>--check</code>; same planning operation, no execution</td>
  </tr>
  <tr>
    <td><code>--outdated</code></td>
    <td>Check upstream releases for installed tools and report which are out of date</td>
  </tr>
  <tr>
    <td><code>--info &lt;tool&gt;</code></td>
    <td>Show detailed metadata for a specific tool</td>
  </tr>
  <tr>
    <td><code>--json</code></td>
    <td>Output renderer: emit machine-readable JSON for any operation</td>
  </tr>
  <tr>
    <td><code>--quiet</code>, <code>-q</code></td>
    <td>Suppress the discovery header and chatter; emit only the requested data and summary (for scripting)</td>
  </tr>
  <tr>
    <td><code>--ci</code></td>
    <td>Deterministic, ASCII-only, line-oriented terminal output (no ANSI, no Unicode, no progress)</td>
  </tr>
  <tr>
    <td><code>--verbose</code>, <code>-V</code></td>
    <td>Detailed planning view: packages and install commands</td>
  </tr>
</table>

`--json`, `--quiet`/`-q`, `--ci`, and `--verbose`/`-V` are output modifiers,
not operations. They can be combined with any operation.

## Output Structure

Every operation follows the same canonical rhythm:

```
Go

Discovery

Body

Summary
```

Every summary block uses the identical layout — a `Summary` heading followed
by aligned label/value rows — so every command ends exactly the same way:

```
Summary

Updated       2
Skipped       1
Failed        0
Duration      3.2s
```

Human output uses a canonical symbol set: `✓` healthy, `•` skipped, `✗`
failed, `↑` outdated, `ⓘ` note. CI mode substitutes plain ASCII
(`OK`, `-`, `FAIL`).

## Examples

Inventory with health status (`--list`):

```
$ update-go-tools --list
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
update-go-tools (devel)             Local       update-go-tools/cmd/update-go-tools

Summary

Healthy       2
Local         1
Invalid       0
Unhealthy     0
```

Concise plan (`--check`):

```
$ update-go-tools --check
Would update

  air
  templ

Skipped

  • update-go-tools

Summary

Would update  2
Skipped       1
```

Detailed plan (`--check --verbose`, same as `--dry-run --verbose`):

```
$ update-go-tools --check --verbose
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

Outdated check (`--outdated`):

```
$ update-go-tools --outdated
NAME            CURRENT         STATUS
golangci-lint   v1.64.8         ↑ v1.65.0
gopls           v0.23.0         ✓

Summary

Checked       2
Outdated      1
Up-to-date    1
```

Quiet update for scripting (`-q`):

```
$ update-go-tools -q
Updated       2
Skipped       0
Failed        0
Duration      3.2s
```

Deterministic CI output (`--ci`):

```
$ update-go-tools --list --ci
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

Update a single tool:

```
$ update-go-tools golangci-lint
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

## JSON Output

`--json` is a pure output renderer available on every operation. The schema is
stable: field names, nesting, and ordering never change between releases, and
human formatting never affects the JSON shape. Empty list fields are always
emitted as `[]`, never `null`.

Every operation response carries a JSON envelope:

```json
{
  "operation": "list",
  "success": true,
  "tools": []
}
```

* `operation` — the logical operation. Values are frozen:
  `list`, `check`, `update`, `outdated`. There are no aliases: `--dry-run`
  emits `"operation": "check"` because it invokes the same planning operation.
* `success` — `true` unless the operation finished with issues (for example an
  update with failures, or an inventory with unhealthy/invalid binaries).
  Scripts can gate on `.success` instead of parsing the payload.

<table>
  <tr>
    <th>Command</th>
    <th>JSON Schema</th>
  </tr>
  <tr>
    <td><code>--list --json</code></td>
    <td><code>{"operation": "list", "success": bool, "tools": [ToolReport...]}</code></td>
  </tr>
  <tr>
    <td><code>--info &lt;tool&gt; --json</code></td>
    <td><code>ToolReport</code></td>
  </tr>
  <tr>
    <td><code>--check --json</code> / <code>--dry-run --json</code></td>
    <td><code>{"operation": "check", "success": bool, "would_update": [PlanItem...], "skipped": [PlanItem...]}</code></td>
  </tr>
  <tr>
    <td><code>--outdated --json</code></td>
    <td><code>{"operation": "outdated", "success": bool, "results": [OutdatedItemReport...]}</code></td>
  </tr>
  <tr>
    <td><code>default --json</code></td>
    <td><code>{"operation": "update", "success": bool, "updated": [], "skipped": [], "failed": [], "notes": []}</code></td>
  </tr>
</table>

Reports:

* `ToolReport`: `name`, `version`, `package_path`, `module_path`
* `PlanItem`: `name`, `package_path`, `install_target`, `command`
* `OutdatedItemReport`: `name`, `current`, `latest`, `outdated`, `error` (omitted on success)

### JSON Compatibility

Within the 1.x series:

* existing fields will not be renamed or removed;
* field meanings will not change;
* new fields may only be added in a backward-compatible manner;
* arrays are never `null`;
* `operation` values are stable (`list`, `check`, `update`, `outdated`).

## Exit Codes

* 0 - Success
* 1 - Update or inventory/health failure
* 2 - Usage error (unknown option, `--info` without a tool, unknown tool)
* 3 - Environment error

## API Stability

The CLI contract for the 1.x series is frozen.

All flags, exit codes, JSON response schemas, and terminal output formats are
guaranteed stable. Terminal UX may evolve incrementally. No breaking changes
within the 1.x series.

The version string is injected at build time:

```
go build -ldflags "-X main.version=v1.5.0" ./cmd/update-go-tools
```

Build metadata (commit hash and build date) can also be injected:

```
go build -ldflags "-X main.version=v1.5.0 -X main.commitHash=abc1234 -X main.buildDate=2026-08-05" ./cmd/update-go-tools
```

`--version` then prints:

```
update-go-tools v1.5.0
Commit    abc1234
Built     2026-08-05
```

## Testing

The test suite follows the project's release contract (see
`.opencode/bin/update-go-tools/CONTRACT.md`). It includes unit tests,
integration tests against a temporary GOBIN populated with real fixture
binaries served from an offline file proxy, and CLI golden tests for every
command, renderer, and exit code.

```
go test ./...
```

Regenerate golden files after an intentional output change:

```
go test ./cmd/update-go-tools -update
```

The test environment is hermetic: it never touches your real GOBIN, module
cache, or network.
