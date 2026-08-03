package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"update-go-tools/internal/tool"
)

const Version = "1.2.0"

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
	if len(args) > 0 {
		switch args[0] {
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
		case "--info":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "Error: Option --info requires a tool name.")
				os.Exit(ExitUsage)
			}
			if err := runInfo(gobin, args[1]); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(ExitUsage)
			}
			os.Exit(ExitSuccess)
		default:
			if strings.HasPrefix(args[0], "-") {
				fmt.Fprintf(os.Stderr, "Error: Unknown option: %s. Run 'update-go-tools --help' for usage.\n", args[0])
				os.Exit(ExitUsage)
			}
		}
	}

	if goVer, err := getGoVersion(); err == nil && goVer != "" {
		fmt.Printf("Go: %s\n\n", goVer)
	}

	results, err := tool.UpdateTools(gobin, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error updating tools:", err)
		os.Exit(ExitFailure)
	}

	updated := 0
	failed := 0
	var failedList []string

	for _, res := range results {
		fmt.Printf("Updating %-20s ", res.Tool.Name()+"...")
		if res.Success {
			fmt.Println("✓")
			updated++
		} else {
			fmt.Println("✗")
			failed++
			failedList = append(failedList, "- "+res.Tool.Name())
		}
	}

	tools, _ := tool.Load(gobin)
	totalFiles := 0
	if entries, err := os.ReadDir(gobin); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				totalFiles++
			}
		}
	}
	skipped := totalFiles - len(tools)
	for _, t := range tools {
		if !t.CanUpdate() {
			skipped++
		}
	}

	fmt.Println()
	fmt.Printf("Updated:  %d\n", updated)
	if skipped > 0 {
		fmt.Printf("Skipped:  %d\n", skipped)
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
	tools, err := tool.Load(gobin)
	if err != nil {
		return err
	}

	fmt.Printf("%-20s %-15s %s\n", "NAME", "VERSION", "PACKAGE PATH")
	fmt.Printf("%-20s %-15s %s\n", "----", "-------", "------------")
	for _, t := range tools {
		fmt.Printf("%-20s %-15s %s\n", t.Name(), t.Version(), t.PackagePath())
	}
	return nil
}

func runInfo(gobin, target string) error {
	tools, err := tool.Load(gobin)
	if err != nil {
		return err
	}

	for _, t := range tools {
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
	results, err := tool.VerifyAll(gobin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error verifying tools:", err)
		return err
	}

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
	fmt.Printf("\n%d tools verified (%d unhealthy).\n", count, unhealthy)
	if unhealthy > 0 {
		return fmt.Errorf("%d unhealthy tools found", unhealthy)
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
    update-go-tools --help
    update-go-tools --version

Options:
    --help       Display this help message
    --version    Display version information
    --list       List inventory of all Go-managed tools with versions and modules
    --info       Show detailed metadata for a specific tool
    --verify     Verify integrity of installed Go tools without updating

Without arguments, updates all discovered Go tools.
With one or more tool names, updates only those specified tools.
`, Version)
}
