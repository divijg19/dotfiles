package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"helm/internal/app"
	"helm/internal/tool"
)

var (
	version    = "v1.6.0"
	commitHash = ""
	buildDate  = ""
)

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

func Run(args []string) int {
	ctx := context.Background()
	opts, code := parseFlags(args)
	if code != 0 {
		return code
	}

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
		return fail("Error:", err)
	}

	operation, toolArgs := splitOperation(opts.positional)

	switch operation {
	case "--help", "-h":
		printHelp()
		return ExitSuccess
	case "--version", "-v":
		printVersion()
		return ExitSuccess
	}

	if operation != "" {
		loadRes, err := application.LoadTools()
		if err != nil {
			return fail("Error loading tools:", err)
		}
		renderHeader(ctx, application, renderer, loadRes, mode)

		switch operation {
		case "--list":
			if err := application.RunInventory(); err != nil {
				return fail("Error:", err)
			}
		case "--outdated":
			if err := application.RunOutdated(ctx); err != nil {
				return fail("Error:", err)
			}
		case "--info":
			if len(toolArgs) == 0 {
				fmt.Fprintln(os.Stderr, "Error: Option --info requires a tool name.")
				return ExitUsage
			}
			if err := application.RunInfo(toolArgs[0]); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				return ExitUsage
			}
		case "--check", "--dry-run":
			if err := application.RunPlan(ctx, toolArgs); err != nil {
				return fail("Error:", err)
			}
		}
		return ExitSuccess
	}

	loadRes, err := application.LoadTools()
	if err != nil {
		return fail("Error loading tools:", err)
	}
	renderHeader(ctx, application, renderer, loadRes, mode)

	if opts.plan {
		if err := application.RunPlan(ctx, toolArgs); err != nil {
			return fail("Error:", err)
		}
		return ExitSuccess
	}

	if err := application.RunUpdate(ctx, toolArgs); err != nil {
		return fail("Error:", err)
	}
	return ExitSuccess
}

func parseFlags(args []string) (cliOptions, int) {
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
				fmt.Fprintf(os.Stderr, "Error: Unknown option: %s. Run 'helm --help' for usage.\n", arg)
				return cliOptions{}, ExitUsage
			}
			positional = append(positional, arg)
		}
	}
	opts.positional = positional
	return opts, 0
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

func renderHeader(ctx context.Context, application *app.App, renderer app.Renderer, loadRes tool.LoadResult, mode app.RenderMode) {
	hdr := app.HeaderInfo{Gobin: application.Gobin, LoadRes: loadRes}
	if mode != app.ModeJSON && mode != app.ModeQuiet {
		if goVer, err := getGoVersion(ctx, application.Runner); err == nil && goVer != "" {
			hdr.GoVersion = goVer
		}
	}
	_ = renderer.Header(hdr)
}

func fail(prefix string, err error) int {
	fmt.Fprintln(os.Stderr, prefix, err)
	return ExitFailure
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
	fmt.Printf("Helm %s\n", version)
	if commitHash != "" {
		fmt.Printf("Commit    %s\n", commitHash)
	}
	if buildDate != "" {
		fmt.Printf("Built     %s\n", buildDate)
	}
	fmt.Println()
}

func printHelp() {
	fmt.Printf(`Helm %s - Inspect, inventory, and update Go-managed tools

Usage:
    helm [tool...]
    helm --list [--json|--ci|--quiet]
    helm --check/--dry-run [--verbose|-V]
    helm --outdated [--json|--ci]
    helm --info <tool>
    helm --json
    helm --quiet | -q
    helm --ci
    helm --help
    helm --version

Operations:
    --list             List all Go-managed tools with versions, health status, and packages
    --check/--dry-run  Summarize pending updates without executing them (aliases)
    --outdated         Check upstream releases for installed tools
    --info             Show detailed metadata for a specific tool

Output modifiers:
    --json       Emit machine-readable JSON for any operation
    --quiet, -q  Suppress the discovery header and chatter; emit only the requested data and summary
    --ci         Deterministic, ASCII-only, line-oriented terminal output
    --verbose, -V  Detailed planning view (packages and install commands)

Utility:
    --help       Display this help message
    --version    Display version information

Without arguments, updates all discovered Go tools.
With one or more tool names, updates only those specified tools.
`, version)
}
