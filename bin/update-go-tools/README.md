# `update-go-tools`

`update-go-tools` is a lightweight utility designed to inspect, inventory, verify, and update Go developer binaries managed via `go install`. It leverages embedded module build metadata through `debug/buildinfo`.

## Command Usage

```
update-go-tools [tool...]
update-go-tools --list
update-go-tools --info <tool>
update-go-tools --verify
update-go-tools --outdated
update-go-tools --check / --dry-run
update-go-tools --json
update-go-tools --help
update-go-tools --version
```

Without arguments, updates all discovered Go tools.
With one or more tool names, updates only those specified tools.

## Flags and Options

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
    <td>Display inventory of installed Go tools with versions and modules</td>
  </tr>
  <tr>
    <td><code>--info &lt;tool&gt;</code></td>
    <td>Show detailed metadata for a specific tool</td>
  </tr>
  <tr>
    <td><code>--verify</code></td>
    <td>Verify integrity of installed Go tools without updating</td>
  </tr>
  <tr>
    <td><code>--outdated</code></td>
    <td>Check upstream releases for installed tools and report which are out of date</td>
  </tr>
  <tr>
    <td><code>--check</code></td>
    <td>Preview updates without executing changes</td>
  </tr>
  <tr>
    <td><code>--dry-run</code></td>
    <td>Alias for --check; plans updates without executing</td>
  </tr>
  <tr>
    <td><code>--json</code></td>
    <td>Emit machine-readable JSON output for the selected command</td>
  </tr>
</table>

## Examples

List the inventory of Go-managed tools:

```
$ update-go-tools --list
NAME                 VERSION         PACKAGE PATH
----                 -------         ------------
air                  v1.67.4         github.com/air-verse/air
golangci-lint        v1.64.8         github.com/golangci/golangci-lint/cmd/golangci-lint
```

Check which installed tools have newer upstream releases:

```
$ update-go-tools --outdated
Checking upstream versions...

NAME                 CURRENT         LATEST
----                 -------         ------
golangci-lint        v1.64.8         v1.65.0
gopls                v0.23.0         up-to-date
```

Preview what an update would do without changing anything:

```
$ update-go-tools --check
Go: go1.26.5

Discovering Go tools... 14 found.

Would update golangci-lint       -> github.com/golangci/golangci-lint/cmd/golangci-lint
Would update gopls               -> golang.org/x/tools/gopls

Would update: 2
```

Update a single tool:

```
$ update-go-tools golangci-lint
Updating golangci-lint... ✓

Updated:  1
```

Emit JSON for scripting:

```
$ update-go-tools --list --json
{
  "tools": [
    {
      "name": "air",
      "version": "v1.67.4",
      "package_path": "github.com/air-verse/air",
      "module_path": "github.com/air-verse/air"
    }
  ]
}
```

## JSON Output

The `--json` flag makes the selected command emit machine-readable output.
Responses use stable field names and nesting and never mix presentation
formatting with data.

<table>
  <tr>
    <th>Command</th>
    <th>JSON Schema</th>
  </tr>
  <tr>
    <td><code>--list --json</code></td>
    <td><code>{"tools": [ToolReport...]}</code></td>
  </tr>
  <tr>
    <td><code>--info &lt;tool&gt; --json</code></td>
    <td><code>ToolReport</code></td>
  </tr>
  <tr>
    <td><code>--verify --json</code></td>
    <td><code>{"results": [VerifyResultReport...]}</code></td>
  </tr>
  <tr>
    <td><code>--outdated --json</code></td>
    <td><code>{"results": [OutdatedItemReport...]}</code></td>
  </tr>
  <tr>
    <td><code>default/--check --json</code></td>
    <td><code>{"updated": [], "notes": [], "skipped": [], "failed": [], "check_only": bool}</code></td>
  </tr>
</table>

Reports:

* `ToolReport`: `name`, `version`, `package_path`, `module_path`
* `VerifyResultReport`: `name`, `healthy`, `error` (omitted when healthy)
* `OutdatedItemReport`: `name`, `current`, `latest` (omitted on error), `outdated`, `error` (omitted on success)

Empty list fields are emitted as `[]`, never `null`.

## Exit Codes

* 0 - Success
* 1 - Update or verification failure
* 2 - Usage error
* 3 - Environment error

## API Stability

The CLI contract for the 1.x series is frozen.

All flags, exit codes, JSON response schemas, and terminal output formats are guaranteed stable.
Terminal UX may evolve incrementally. No breaking changes within the 1.x series.

The version string is injected at build time and defaults to `dev`:

```
go build -ldflags "-X main.version=v1.4.0" ./cmd/update-go-tools
```

Build metadata (commit hash and build date) can also be injected:

```
go build -ldflags "-X main.version=v1.4.0 -X main.commitHash=abc1234 -X main.buildDate=2026-08-04" ./cmd/update-go-tools
```

`--version` then prints:

```
update-go-tools v1.4.0
Commit    abc1234
Built     2026-08-04
```

## Testing

The test suite follows the project's release contract (see
`.opencode/bin/update-go-tools/CONTRACT.md`). It includes unit tests,
integration tests against a temporary GOBIN populated with real fixture
binaries served from an offline file proxy, and CLI golden tests.

```
go test ./...
```

Regenerate golden files after an intentional output change:

```
go test ./cmd/update-go-tools -update
```

The test environment is hermetic: it never touches your real GOBIN, module
cache, or network.
