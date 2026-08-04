package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"update-go-tools/internal/tool"
)

const Version = "v1.0.0"

const (
	ExitSuccess = 0
	ExitFailure = 1
	ExitUsage   = 2
	ExitEnv     = 3
)

func main() {
	gobin, err := tool.GetGobin()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(ExitEnv)
	}

	args := os.Args[1:]
	checkOnly := false

	var filteredArgs []string
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			printHelp()
			os.Exit(ExitSuccess)
		case "--version", "-v":
			fmt.Printf("update-go-tools %s\n", Version)
			os.Exit(ExitSuccess)
		case "--list":
			if err := runInventory(gobin); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(ExitFailure)
			}
			os.Exit(ExitSuccess)
		case "--verify":
			if err := runVerify(gobin); err != nil {
				os.Exit(ExitFailure)
			}
			os.Exit(ExitSuccess)
		case "--check":
			checkOnly = true
		default:
			if strings.HasPrefix(arg, "--info") {
				filteredArgs = append(filteredArgs, arg)
			} else if strings.HasPrefix(arg, "-") {
				fmt.Fprintf(os.Stderr, "Error: Unknown option: %s. Run 'update-go-tools --help' for usage.\n", arg)
				os.Exit(ExitUsage)
			} else {
				filteredArgs = append(filteredArgs, arg)
			}
		}
	}

	for i, arg := range args {
		if arg == "--info" {
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: Option --info requires a tool name.")
				os.Exit(ExitUsage)
			}
			if err := runInfo(gobin, args[i+1]); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(ExitUsage)
			}
			os.Exit(ExitSuccess)
		}
	}

	if goVer, err := getGoVersion(); err == nil && goVer != "" {
		fmt.Printf("Go: %s\n\n", goVer)
	}

	fmt.Printf("Discovering Go tools...")
	loadRes, err := tool.Load(gobin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nError loading tools:", err)
		os.Exit(ExitFailure)
	}
	fmt.Printf(" %d found.\n\n", len(loadRes.Tools)+len(loadRes.Invalid))

	resultsChan := tool.Update(loadRes.Tools, filteredArgs, checkOnly)

	updated := 0
	notesCount := 0
	skippedLocal := 0
	failed := 0
	var failedList []string

	for res := range resultsChan {
		if res.Status == tool.StatusSkippedLocal {
			skippedLocal++
			if len(filteredArgs) > 0 {
				fmt.Printf("Skipping %-20s (local/devel build)\n", res.Tool.Name()+"...")
			}
			continue
		}

		if checkOnly {
			fmt.Printf("Would update %-16s -> %s\n", res.Tool.Name(), res.Tool.InstallTarget())
			updated++
			continue
		}

		fmt.Printf("Updating %-20s ", res.Tool.Name()+"...")
		if res.Success {
			if len(res.Notes) > 0 {
				fmt.Println("ⓘ")
				for _, note := range res.Notes {
					fmt.Printf("  %s\n", note)
				}
				notesCount++
			} else {
				fmt.Println("✓")
			}
			updated++
		} else {
			fmt.Println("✗")
			for _, note := range res.Notes {
				fmt.Printf("  %s\n", note)
			}
			failed++
			failedList = append(failedList, "- "+res.Tool.Name())
		}
	}

	invalidCount := len(loadRes.Invalid)
	totalSkipped := skippedLocal + invalidCount

	fmt.Println()
	fmt.Printf("Updated:  %d\n", updated)
	if notesCount > 0 {
		fmt.Printf("Notes:    %d\n", notesCount)
	}
	if totalSkipped > 0 {
		fmt.Printf("Skipped:  %d (Local: %d, Invalid: %d)\n", totalSkipped, skippedLocal, invalidCount)
	}
	if failed > 0 {
		fmt.Printf("Failed:   %d\n\n", failed)
		fmt.Println("Failed tools:")
		for _, f := range failedList {
			fmt.Println(f)
		}
		os.Exit(ExitFailure)
	}
}

func getGoVersion() (string, error) {
	out, err := exec.Command("go", "env", "GOVERSION").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func runInventory(gobin string) error {
	loadRes, err := tool.Load(gobin)
	if err != nil {
		return err
	}

	fmt.Printf("%-20s %-15s %s\n", "NAME", "VERSION", "PACKAGE PATH")
	fmt.Printf("%-20s %-15s %s\n", "----", "-------", "------------")
	for _, t := range loadRes.Tools {
		fmt.Printf("%-20s %-15s %s\n", t.Name(), t.Version(), t.PackagePath())
	}
	if len(loadRes.Invalid) > 0 {
		fmt.Printf("\nInvalid / Uninspectable binaries (%d):\n", len(loadRes.Invalid))
		for _, inv := range loadRes.Invalid {
			fmt.Printf("  - %s (%v)\n", inv.Path, inv.Error)
		}
	}
	return nil
}

func runInfo(gobin, target string) error {
	loadRes, err := tool.Load(gobin)
	if err != nil {
		return err
	}

	for _, t := range loadRes.Tools {
		if t.Name() == target {
			fmt.Printf("Binary\n\n  %s\n\n", t.Name())
			fmt.Printf("Main Package Path\n\n  %s\n\n", t.PackagePath())
			fmt.Printf("Module Path\n\n  %s\n\n", t.ModulePath())
			fmt.Printf("Version\n\n  %s\n\n", t.Version())
			fmt.Printf("Go (Built with)\n\n  %s\n\n", t.GoVersion())
			fmt.Printf("Location\n\n  %s\n\n", t.Path())
			fmt.Printf("Can Update\n\n  %t\n", t.CanUpdate())
			return nil
		}
	}
	return fmt.Errorf("tool '%s' not found or has no module metadata", target)
}

func runVerify(gobin string) error {
	loadRes, err := tool.Load(gobin)
	if err != nil {
		return err
	}

	results := tool.Verify(loadRes.Tools)

	count := 0
	unhealthy := 0
	for _, r := range results {
		if r.Healthy {
			status := "✓"
			extra := ""
			if !r.Tool.CanUpdate() {
				status = "•"
				extra = " (local/devel)"
			}
			fmt.Printf("%s %-15s (%s)%s -> %s\n", status, r.Tool.Name(), r.Tool.Version(), extra, r.Tool.PackagePath())
			count++
		} else {
			fmt.Fprintf(os.Stderr, "✗ %-15s (%s)\n", r.Tool.Name(), r.Error)
			unhealthy++
		}
	}

	if len(loadRes.Invalid) > 0 {
		for _, inv := range loadRes.Invalid {
			fmt.Fprintf(os.Stderr, "✗ %-15s (%v)\n", inv.Path, inv.Error)
			unhealthy++
		}
	}

	totalChecked := count + unhealthy
	fmt.Printf("\n%d binaries verified (%d unhealthy).\n", totalChecked, unhealthy)
	if unhealthy > 0 {
		return fmt.Errorf("%d unhealthy binaries found", unhealthy)
	}
	return nil
}

func printHelp() {
	fmt.Printf(`update-go-tools %s - Inspect, inventory, and update Go-managed tools

Usage:
    update-go-tools [tool...]
    update-go-tools --list
    update-go-tools --info <tool>
    update-go-tools --verify
    update-go-tools --check
    update-go-tools --help
    update-go-tools --version

Options:
    --help       Display this help message
    --version    Display version information
    --list       List inventory of all Go-managed tools with versions and modules
    --info       Show detailed metadata for a specific tool
    --verify     Verify integrity of installed Go tools without updating
    --check      Show what would be updated without executing changes

Without arguments, updates all discovered Go tools.
With one or more tool names, updates only those specified tools.
`, Version)
}
