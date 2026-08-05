package app

import (
	"fmt"
	"os"
	"update-go-tools/internal/tool"
)

// CIRenderer produces deterministic, machine-oriented terminal output: no ANSI,
// no Unicode, no progress renderer, no cursor movement. Output is stable,
// line-oriented, and reproducible across environments. Exit codes are still
// surfaced by the caller.
type CIRenderer struct {
	verbose bool
}

func (r CIRenderer) Header(hdr HeaderInfo) error {
	fmt.Printf("gobin: %s\n", hdr.Gobin)
	if hdr.GoVersion != "" {
		fmt.Printf("go-version: %s\n", hdr.GoVersion)
	}
	fmt.Printf("executables: %d\n", hdr.LoadRes.Summary.Executables)
	fmt.Printf("updatable: %d\n", hdr.LoadRes.Summary.Updatable)
	fmt.Printf("local: %d\n", hdr.LoadRes.Summary.Local)
	fmt.Printf("invalid: %d\n", hdr.LoadRes.Summary.Invalid)
	fmt.Println()
	return nil
}

func (r CIRenderer) Inventory(report InventoryReport) error {
	for _, t := range report.Tools {
		status := stringsToASCII(t.Status)
		fmt.Printf("%-20s  %-16s  %-9s  %s\n", t.Name, t.Version, status, t.PackagePath)
		if t.Error != "" {
			fmt.Fprintf(os.Stderr, "error: %s: %s\n", t.Name, t.Error)
		}
	}
	for _, inv := range report.Invalid {
		fmt.Printf("invalid: %s (%s)\n", inv.Path, inv.Message)
	}
	fmt.Println()
	fmt.Printf("healthy: %d\n", report.Summary.Healthy)
	fmt.Printf("local: %d\n", report.Summary.Local)
	fmt.Printf("invalid: %d\n", report.Summary.Invalid)
	fmt.Printf("unhealthy: %d\n", report.Summary.Unhealthy)
	if report.Summary.Unhealthy > 0 || report.Summary.Invalid > 0 {
		return fmt.Errorf("%d issues found during inventory check", report.Summary.Unhealthy+report.Summary.Invalid)
	}
	return nil
}

func (r CIRenderer) Plan(report PlanReport) error {
	for _, item := range report.WouldUpdate {
		if r.verbose {
			fmt.Printf("update: %s\n", item.Name)
			fmt.Printf("  package: %s\n", item.PackagePath)
			fmt.Printf("  command: %s\n", item.Command)
		} else {
			fmt.Printf("update: %s\n", item.Name)
		}
	}
	for _, item := range report.Skipped {
		fmt.Printf("skip: %s\n", item.Name)
	}
	fmt.Printf("would-update: %d\n", len(report.WouldUpdate))
	return nil
}

func (r CIRenderer) Outdated(report OutdatedReport) error {
	for _, o := range report.Results {
		switch {
		case o.Error != "":
			fmt.Printf("%-20s  error: %s\n", o.Name, o.Error)
		case o.Outdated:
			fmt.Printf("%-20s  outdated: %s -> %s\n", o.Name, o.Current, o.Latest)
		default:
			fmt.Printf("%-20s  up-to-date: %s\n", o.Name, o.Current)
		}
	}
	fmt.Println()
	fmt.Printf("checked: %d\n", len(report.Results))
	fmt.Printf("outdated: %d\n", report.Summary.Outdated)
	fmt.Printf("up-to-date: %d\n", report.Summary.UpToDate)
	return nil
}

func (r CIRenderer) Update(report UpdateReport) error {
	for _, name := range report.Updated {
		fmt.Printf("updated: %s\n", name)
	}
	for _, name := range report.Failed {
		fmt.Printf("failed: %s\n", name)
	}
	for _, name := range report.Skipped {
		fmt.Printf("skipped: %s\n", name)
	}
	for _, name := range report.Notes {
		fmt.Printf("note: %s\n", name)
	}
	fmt.Println()
	fmt.Printf("updated: %d\n", len(report.Updated))
	fmt.Printf("skipped: %d\n", len(report.Skipped))
	fmt.Printf("failed: %d\n", len(report.Failed))
	if len(report.Failed) > 0 {
		return fmt.Errorf("%d updates failed", len(report.Failed))
	}
	return nil
}

func (r CIRenderer) Info(loadRes tool.LoadResult, target string) error {
	for _, t := range loadRes.Tools {
		if t.Name() == target {
			fmt.Printf("name: %s\n", t.Name())
			fmt.Printf("package: %s\n", t.PackagePath())
			fmt.Printf("module: %s\n", t.ModulePath())
			fmt.Printf("version: %s\n", t.Version())
			fmt.Printf("go-version: %s\n", t.GoVersion())
			fmt.Printf("path: %s\n", t.Path())
			fmt.Printf("can-update: %t\n", t.CanUpdate())
			return nil
		}
	}
	return fmt.Errorf("tool '%s' not found or has no module metadata", target)
}

func stringsToASCII(s string) string {
	switch s {
	case "Healthy":
		return "OK"
	case "Local":
		return "LOCAL"
	case "Invalid":
		return "INVALID"
	case "Unhealthy":
		return "ERROR"
	default:
		return s
	}
}
