package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"update-go-tools/internal/app"
	"update-go-tools/internal/tool"
)

// version is injected at build time via:
//
//	go build -ldflags "-X main.version=v1.5.0"
var version = "v1.5.0"

// commitHash is injected at build time via -ldflags.
var commitHash = ""

// buildDate is injected at build time via -ldflags (YYYY-MM-DD).
var buildDate = ""

const (
	ExitSuccess = 0
	ExitFailure = 1
	ExitUsage   = 2
	ExitEnv     = 3
)

type cliOptions struct {
	jsonOutput bool
	quiet      bool
	ci         bool
	verbose    bool
	plan       bool
	positional []string
}

func main() {
	ctx := context.Background()
	args := os.Args[1:]
	opts := parseFlags(args)

	mode := app.ModeTerminal
	switch {
	case opts.jsonOutput:
		mode = app.ModeJSON
	case opts.ci:
		mode = app.ModeCI
	case opts.quiet:
		mode = app.ModeQuiet
	}

	renderer := app.NewRenderer(mode, opts.verbose)
	application, err := app.NewApp(renderer, tool.DefaultRunner{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(ExitEnv)
	}

	operation, toolArgs := splitOperation(opts.positional)

	switch operation {
	case "--help", "-h":
		printHelp()
		os.Exit(ExitSuccess)
	case "--version", "-v":
		printVersion()
		os.Exit(ExitSuccess)
	}

	if operation != "" {
		loadRes, err := application.LoadTools()
		if err != nil {
			fail("Error loading tools:", err)
		}
		renderHeader(ctx, application, renderer, loadRes)

		switch operation {
		case "--list":
			if err := application.RunInventory(); err != nil {
				fail("Error:", err)
			}
		case "--outdated":
			if err := application.RunOutdated(ctx); err != nil {
				fail("Error:", err)
			}
		case "--info":
			if len(toolArgs) == 0 {
				fmt.Fprintln(os.Stderr, "Error: Option --info requires a tool name.")
				os.Exit(ExitUsage)
			}
			if err := application.RunInfo(toolArgs[0]); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(ExitUsage)
			}
		case "--check", "--dry-run":
			if err := application.RunPlan(ctx, toolArgs); err != nil {
				fail("Error:", err)
			}
		}
		os.Exit(ExitSuccess)
	}

	loadRes, err := application.LoadTools()
	if err != nil {
		fail("Error loading tools:", err)
	}
	renderHeader(ctx, application, renderer, loadRes)

	if opts.plan {
		if err := application.RunPlan(ctx, toolArgs); err != nil {
			fail("Error:", err)
		}
		os.Exit(ExitSuccess)
	}

	if err := application.RunUpdate(ctx, toolArgs); err != nil {
		fail("Error:", err)
	}
}

// parseFlags collects the known global flags and returns the remaining
// positional arguments plus each flag's value. Unknown options are reported as
// usage errors so the CLI surface stays explicit.
func parseFlags(args []string) cliOptions {
	opts := cliOptions{}
	var positional []string
	for _, arg := range args {
		switch arg {
		case "--json":
			opts.jsonOutput = true
		case "--quiet", "-q":
			opts.quiet = true
		case "--ci":
			opts.ci = true
		case "--verbose", "-V":
			opts.verbose = true
		case "--check", "--dry-run":
			opts.plan = true
		case "--help", "-h", "--version", "-v", "--list", "--outdated", "--info":
			positional = append(positional, arg)
		default:
			if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "Error: Unknown option: %s. Run 'update-go-tools --help' for usage.\n", arg)
				os.Exit(ExitUsage)
			}
			positional = append(positional, arg)
		}
	}
	opts.positional = positional
	return opts
}

func splitOperation(positional []string) (string, []string) {
	if len(positional) == 0 {
		return "", nil
	}
	switch positional[0] {
	case "--list", "--outdated", "--check", "--dry-run", "--help", "-h", "--version", "-v", "--info":
		return positional[0], positional[1:]
	}
	return "", positional
}

func renderHeader(ctx context.Context, application *app.App, renderer app.Renderer, loadRes tool.LoadResult) {
	hdr := app.HeaderInfo{Gobin: application.Gobin, LoadRes: loadRes}
	if !isNoopHeader(renderer) {
		if goVer, err := getGoVersion(ctx, application.Runner); err == nil && goVer != "" {
			hdr.GoVersion = goVer
		}
	}
	_ = renderer.Header(hdr)
}

func isNoopHeader(renderer app.Renderer) bool {
	switch renderer.(type) {
	case app.JSONRenderer, app.QuietRenderer:
		return true
	default:
		return false
	}
}

func fail(prefix string, err error) {
	fmt.Fprintln(os.Stderr, prefix, err)
	os.Exit(ExitFailure)
}

func getGoVersion(ctx context.Context, runner tool.Runner) (string, error) {
	output, err := runner.Run(ctx, tool.Command{
		Name: "go",
		Args: []string{"env", "GOVERSION"},
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func printVersion() {
	fmt.Printf("update-go-tools %s\n", version)
	if commitHash != "" {
		fmt.Printf("Commit    %s\n", commitHash)
	}
	if buildDate != "" {
		fmt.Printf("Built     %s\n", buildDate)
	}
	fmt.Println()
}

func printHelp() {
	fmt.Printf(`update-go-tools %s - Inspect, inventory, and update Go-managed tools

Usage:
    update-go-tools [tool...]
    update-go-tools --list [--json|--ci|--quiet]
    update-go-tools --check [--verbose|-V]
    update-go-tools --dry-run [--verbose|-V]
    update-go-tools --outdated [--json|--ci]
    update-go-tools --info <tool>
    update-go-tools --json
    update-go-tools --quiet | -q
    update-go-tools --ci
    update-go-tools --help
    update-go-tools --version

Operations:
    --list       List all Go-managed tools with versions, health status, and packages
    --check      Summarize pending updates without executing them
    --dry-run    Alias for --check; shows the execution plan without running it
    --outdated   Check upstream releases for installed tools
    --info       Show detailed metadata for a specific tool

Output modifiers:
    --json       Emit machine-readable JSON for any operation
    --quiet, -q  Suppress informational output; emit only the update summary
    --ci         Deterministic, ASCII-only, line-oriented terminal output
    --verbose, -V  Detailed planning view (packages and install commands)

Utility:
    --help       Display this help message
    --version    Display version information

Without arguments, updates all discovered Go tools.
With one or more tool names, updates only those specified tools.
`, version)
}
