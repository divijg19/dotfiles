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
//	go build -ldflags "-X main.version=v1.2.0"
var version = "v1.3.0"

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

func main() {
	ctx := context.Background()
	args := os.Args[1:]
	checkOnly := false
	jsonOutput := false

	var remainingArgs []string
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "--check", "--dry-run":
			checkOnly = true
		default:
			remainingArgs = append(remainingArgs, arg)
		}
	}

	var renderer app.Renderer = app.TerminalRenderer{}
	if jsonOutput {
		renderer = app.JSONRenderer{}
	}

	application, err := app.NewApp(renderer, tool.DefaultRunner{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(ExitEnv)
	}

	if len(remainingArgs) > 0 {
		switch remainingArgs[0] {
		case "--help", "-h":
			printHelp()
			os.Exit(ExitSuccess)
		case "--version", "-v":
			fmt.Printf("update-go-tools %s\n", version)
			if commitHash != "" {
				fmt.Printf("Commit    %s\n", commitHash)
			}
			if buildDate != "" {
				fmt.Printf("Built     %s\n", buildDate)
			}
			fmt.Println()
			os.Exit(ExitSuccess)
		case "--list":
			if err := application.RunInventory(); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(ExitFailure)
			}
			os.Exit(ExitSuccess)
		case "--verify":
			if err := application.RunVerify(); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(ExitFailure)
			}
			os.Exit(ExitSuccess)
		case "--outdated":
			if err := application.RunOutdated(ctx); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(ExitFailure)
			}
			os.Exit(ExitSuccess)
		case "--info":
			if len(remainingArgs) < 2 {
				fmt.Fprintln(os.Stderr, "Error: Option --info requires a tool name.")
				os.Exit(ExitUsage)
			}
			if err := application.RunInfo(remainingArgs[1]); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(ExitUsage)
			}
			os.Exit(ExitSuccess)
		}
	}

	var filteredArgs []string
	for _, arg := range remainingArgs {
		if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr, "Error: Unknown option: %s. Run 'update-go-tools --help' for usage.\n", arg)
			os.Exit(ExitUsage)
		}
		filteredArgs = append(filteredArgs, arg)
	}

	loadRes, err := application.LoadTools()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading tools:", err)
		os.Exit(ExitFailure)
	}

	if !jsonOutput {
		if goVer, err := getGoVersion(ctx, application.Runner); err == nil && goVer != "" {
			fmt.Printf("Go: %s\n\n", goVer)
		}
		fmt.Printf("Scanning %s\n\nDiscovered\n  Executables : %d\n  Updatable   : %d\n  Local       : %d\n  Invalid     : %d\n\n",
			application.Gobin,
			loadRes.Summary.Executables,
			loadRes.Summary.Updatable,
			loadRes.Summary.Local,
			loadRes.Summary.Invalid,
		)
		if loadRes.Summary.Local > 0 {
			fmt.Println("Skipping local development binaries:")
			for _, t := range loadRes.Tools {
				if !t.CanUpdate() {
					fmt.Printf("  • %s\n", t.Name())
				}
			}
			fmt.Println()
		}
	}

	if err := application.RunUpdate(ctx, filteredArgs, checkOnly, loadRes); err != nil {
		os.Exit(ExitFailure)
	}
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

func printHelp() {
	fmt.Printf(`update-go-tools %s - Inspect, inventory, and update Go-managed tools

Usage:
    update-go-tools [tool...]
    update-go-tools --list
    update-go-tools --info <tool>
    update-go-tools --verify
    update-go-tools --outdated
    update-go-tools --check / --dry-run
    update-go-tools --json
    update-go-tools --help
    update-go-tools --version

Options:
    --help       Display this help message
    --version    Display version information
    --list       List inventory of all Go-managed tools with versions and modules
    --info       Show detailed metadata for a specific tool
    --verify     Verify integrity of installed Go tools without updating
    --outdated   Check upstream releases for installed tools
    --check      Show what would be updated without executing changes
    --dry-run    Alias for --check
    --json       Emit machine-readable JSON output for commands

Without arguments, updates all discovered Go tools.
With one or more tool names, updates only those specified tools.
`, version)
}
