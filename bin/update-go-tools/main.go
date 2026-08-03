package main

import (
	"debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const Version = "1.2.0"

const (
	ExitSuccess = 0
	ExitFailure = 1
	ExitUsage   = 2
	ExitEnv     = 3
)

type Tool struct {
	Name        string
	PackagePath string
	Path        string
	Version     string
	BuildInfo   *buildinfo.BuildInfo
}

func main() {
	// Verify go exists in PATH
	if _, err := exec.LookPath("go"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: 'go' is not installed or not in PATH.")
		os.Exit(ExitEnv)
	}

	gobin, err := getGobin()
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
			if err := printInventory(gobin); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(ExitFailure)
			}
			os.Exit(ExitSuccess)
		case "--verify":
			if err := verifyTools(gobin); err != nil {
				os.Exit(ExitFailure)
			}
			os.Exit(ExitSuccess)
		case "--info":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "Error: Option --info requires a tool name.")
				os.Exit(ExitUsage)
			}
			if err := printToolInfo(gobin, args[1]); err != nil {
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

	// Default: update tools (with optional positional filtering)
	if err := updateTools(gobin, args); err != nil {
		os.Exit(ExitFailure)
	}
}

func getGobin() (string, error) {
	out, err := exec.Command("go", "env", "GOBIN").Output()
	if err == nil {
		gobin := strings.TrimSpace(string(out))
		if gobin != "" {
			return gobin, nil
		}
	}

	out, err = exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return "", fmt.Errorf("failed to determine GOPATH: %w", err)
	}
	gopath := strings.TrimSpace(string(out))
	if gopath == "" {
		return "", fmt.Errorf("GOPATH is not set and GOBIN is empty")
	}
	return filepath.Join(gopath, "bin"), nil
}

func discoverTools(gobin string) ([]Tool, error) {
	entries, err := os.ReadDir(gobin)
	if err != nil {
		return nil, err
	}

	var tools []Tool
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		toolPath := filepath.Join(gobin, entry.Name())

		info, err := entry.Info()
		if err != nil || info.Mode()&0111 == 0 {
			continue
		}

		bi, err := buildinfo.ReadFile(toolPath)
		if err != nil {
			continue
		}

		pkgPath := bi.Main.Path
		if pkgPath == "" || pkgPath == "(devel)" {
			pkgPath = bi.Path
		}
		if pkgPath == "" {
			continue
		}

		version := bi.Main.Version
		if version == "" {
			version = "unknown"
		}

		tools = append(tools, Tool{
			Name:        entry.Name(),
			PackagePath: pkgPath,
			Path:        toolPath,
			Version:     version,
			BuildInfo:   bi,
		})
	}

	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	return tools, nil
}

func printInventory(gobin string) error {
	tools, err := discoverTools(gobin)
	if err != nil {
		return err
	}

	fmt.Printf("%-20s %-15s %s\n", "NAME", "VERSION", "MODULE")
	fmt.Printf("%-20s %-15s %s\n", "----", "-------", "------")
	for _, t := range tools {
		fmt.Printf("%-20s %-15s %s\n", t.Name, t.Version, t.PackagePath)
	}
	return nil
}

func printToolInfo(gobin, target string) error {
	tools, err := discoverTools(gobin)
	if err != nil {
		return err
	}

	for _, t := range tools {
		if t.Name == target {
			goVersion := t.BuildInfo.GoVersion
			if goVersion == "" {
				goVersion = "unknown"
			}

			fmt.Printf("Binary\n\n  %s\n\n", t.Name)
			fmt.Printf("Module\n\n  %s\n\n", t.PackagePath)
			fmt.Printf("Version\n\n  %s\n\n", t.Version)
			fmt.Printf("Go (Built with)\n\n  %s\n\n", goVersion)
			fmt.Printf("Location\n\n  %s\n", t.Path)
			return nil
		}
	}
	return fmt.Errorf("tool '%s' not found or has no module metadata", target)
}

func verifyTools(gobin string) error {
	tools, err := discoverTools(gobin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error discovering tools:", err)
		return err
	}

	count := 0
	for _, t := range tools {
		if _, err := os.Stat(t.Path); err == nil {
			fmt.Printf("✓ %-15s (%s)\n", t.Name, t.Version)
			count++
		} else {
			fmt.Fprintf(os.Stderr, "✗ %-15s (not accessible)\n", t.Name)
		}
	}
	fmt.Printf("\n%d tools verified.\n", count)
	return nil
}

func updateTools(gobin string, filter []string) error {
	out, err := exec.Command("go", "env", "GOVERSION").Output()
	if err == nil {
		fmt.Printf("Go: %s\n\n", strings.TrimSpace(string(out)))
	}

	tools, err := discoverTools(gobin)
	if err != nil {
		return err
	}

	entries, _ := os.ReadDir(gobin)
	totalFiles := 0
	for _, e := range entries {
		if !e.IsDir() {
			totalFiles++
		}
	}

	updated := 0
	failed := 0
	var failedList []string

	filterMap := make(map[string]bool)
	for _, f := range filter {
		filterMap[f] = true
	}

	for _, t := range tools {
		if len(filterMap) > 0 && !filterMap[t.Name] {
			continue
		}

		fmt.Printf("Updating %-20s ", t.Name+"...")

		cmd := exec.Command("go", "install", t.PackagePath+"@latest")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err == nil {
			fmt.Println("✓")
			updated++
		} else {
			fmt.Println("✗")
			failed++
			failedList = append(failedList, "- "+t.Name)
		}
	}

	skipped := totalFiles - len(tools)

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
		return fmt.Errorf("%d updates failed", failed)
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
