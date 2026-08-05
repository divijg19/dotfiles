# Development

## Building

```bash
go build -o helm ./cmd/helm
```

The version string is injected at build time:

```bash
go build -ldflags "-X helm/internal/cli.version=v1.6.0" ./cmd/helm
```

Build metadata (commit hash and build date) can also be injected:

```bash
go build -ldflags "-X helm/internal/cli.version=v1.6.0 -X helm/internal/cli.commitHash=abc1234 -X helm/internal/cli.buildDate=2026-08-05" ./cmd/helm
```

`--version` then prints:

```
Helm v1.6.0
Commit    abc1234
Built     2026-08-05
```

## Testing

The test suite includes unit tests, integration tests against a temporary GOBIN populated with real fixture binaries served from an offline file proxy, and CLI golden tests for every command, renderer, and exit code.

```bash
go test ./...
```

Regenerate golden files after an intentional output change:

```bash
go test ./cmd/helm -update
```

The test environment is hermetic: it never touches your real GOBIN, module cache, or network.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Update or inventory/health failure |
| 2 | Usage error (unknown option, `--info` without a tool, unknown tool) |
| 3 | Environment error |

## API stability

The CLI contract for the 1.x series is frozen. All flags, exit codes, JSON response schemas, and terminal output formats are guaranteed stable. Terminal UX may evolve incrementally. No breaking changes within the 1.x series.
