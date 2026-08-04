package app

import (
	"fmt"
	"os"
	"time"
	"update-go-tools/internal/tool"
)

type TerminalRenderer struct {
	Gobin string
}

func (TerminalRenderer) Inventory(loadRes tool.LoadResult) error {
	fmt.Printf("%-20s %-15s %s\n", "NAME", "VERSION", "PACKAGE PATH")
	fmt.Printf("%-20s %-15s %s\n", "----", "-------", "------------")
	for _, t := range loadRes.Tools {
		fmt.Printf("%-20s %-15s %s\n", t.Name(), t.Version(), t.PackagePath())
	}
	if len(loadRes.Invalid) > 0 {
		fmt.Printf("\nInvalid / Uninspectable binaries (%d):\n", len(loadRes.Invalid))
		for _, inv := range loadRes.Invalid {
			fmt.Printf("  - %s (%s)\n", inv.Path, inv.Message())
		}
	}
	return nil
}

func (TerminalRenderer) Verify(loadRes tool.LoadResult) error {
	results := tool.Verify(loadRes.Tools)

	count := 0
	unhealthy := 0
	for _, r := range results {
		if r.Healthy {
			status := "✓"
			extra := r.Tool.Version()
			if !r.Tool.CanUpdate() {
				status = "•"
				extra = "(devel)"
			}
			fmt.Printf("%s %-15s %-15s %s\n", status, r.Tool.Name(), extra, r.Tool.PackagePath())
			count++
		} else {
			fmt.Fprintf(os.Stderr, "✗ %-15s (%s)\n", r.Tool.Name(), r.Error)
			unhealthy++
		}
	}

	if len(loadRes.Invalid) > 0 {
		for _, inv := range loadRes.Invalid {
			fmt.Fprintf(os.Stderr, "✗ %-15s (%s)\n", inv.Path, inv.Message())
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

func (TerminalRenderer) Outdated(outdatedRes []tool.OutdatedResult) error {
	fmt.Printf("%-15s %-12s %s\n", "NAME", "CURRENT", "STATUS")
	outdatedCount := 0
	upToDateCount := 0

	for _, o := range outdatedRes {
		status := "✓"
		if o.Error != nil {
			status = "error (" + o.Error.Error() + ")"
		} else if o.Outdated {
			status = "↑ " + o.Latest
			outdatedCount++
		} else {
			upToDateCount++
		}
		fmt.Printf("%-15s %-12s %s\n", o.Tool.Name(), o.Current, status)
	}
	fmt.Println()
	fmt.Printf("%d tools checked\n\n", len(outdatedRes))
	fmt.Printf("%d outdated\n", outdatedCount)
	fmt.Printf("%d up-to-date\n", upToDateCount)
	return nil
}

func (TerminalRenderer) OnProgress(p tool.Progress) {
	switch p.Action {
	case "Start":
		fmt.Printf("[%02d/%02d] %-18s", p.Current, p.Total, p.Tool.Name())
	case "Output":
		fmt.Printf("  %s\n", p.Line)
	case "Complete":
		if p.Success && len(p.Notes) == 0 {
			fmt.Printf("           ✓\n")
		} else if p.Success && len(p.Notes) > 0 {
			fmt.Printf("           ⓘ\n")
			fmt.Printf("  Package    %s\n", p.Tool.InstallTarget())
			for _, note := range p.Notes {
				fmt.Printf("  %s\n", note)
			}
		} else {
			fmt.Printf("           ✗\n")
			fmt.Printf("  Error\n")
			if p.Error != nil {
				fmt.Printf("    %v\n", p.Error)
			}
			for _, note := range p.Notes {
				fmt.Printf("    %s\n", note)
			}
		}
	case "Skipped":
		// Handled in summary/skip block or concise
		fmt.Printf("[%02d/%02d] %-18s • (skipped)\n", p.Current, p.Total, p.Tool.Name())
	}
}

func (r TerminalRenderer) Update(results []tool.ToolUpdateResult, loadRes tool.LoadResult, duration time.Duration, diagnostics []tool.Diagnostic, checkOnly bool) error {
	updated := 0
	failed := 0
	var failedList []string

	for _, res := range results {
		if res.Status == tool.StatusSkippedLocal {
			continue
		}
		if checkOnly {
			fmt.Printf("Would update %-16s -> %s\n", res.Tool.Name(), res.Tool.InstallTarget())
			updated++
			continue
		}
		if res.Success {
			updated++
		} else {
			failed++
			failedList = append(failedList, res.Tool.Name())
		}
	}

	if checkOnly {
		fmt.Println()
		fmt.Printf("Would update: %d\n", updated)
		return nil
	}

	type skipItem struct {
		name   string
		reason string
		path   string
	}
	var skippedItems []skipItem
	for _, res := range results {
		if res.Status == tool.StatusSkippedLocal {
			skippedItems = append(skippedItems, skipItem{
				name:   res.Tool.Name(),
				reason: res.Tool.SkipReason(),
				path:   res.Tool.Path(),
			})
		}
	}
	for _, inv := range loadRes.Invalid {
		skippedItems = append(skippedItems, skipItem{
			name:   inv.Path,
			reason: inv.Message(),
			path:   inv.Path,
		})
	}

	if len(skippedItems) > 0 {
		fmt.Println()
		fmt.Println("Skipped")
		fmt.Println()
		for _, item := range skippedItems {
			fmt.Printf("• %s\n", item.name)
			fmt.Printf("  Reason  : %s\n", item.reason)
			fmt.Printf("  Path    : %s\n\n", item.path)
		}
	}

	if len(diagnostics) > 0 {
		fmt.Println()
		fmt.Println("Diagnostics")
		fmt.Println()
		for _, d := range diagnostics {
			fmt.Printf("• %s\n", d.ToolName)
			fmt.Printf("  Category : %s\n", d.Category)
			fmt.Printf("  Message  : %s\n\n", d.Message)
		}
	}

	fmt.Println()
	fmt.Println("Summary")
	fmt.Println()
	fmt.Printf("Updated       %d\n", updated)
	fmt.Printf("Skipped       %d\n", len(skippedItems))
	fmt.Printf("Failed        %d\n", failed)
	fmt.Printf("Duration      %s\n", formatDuration(duration))

	if failed > 0 {
		fmt.Println()
		fmt.Println("Failed tools:")
		for _, f := range failedList {
			fmt.Printf("- %s\n", f)
		}
		return fmt.Errorf("%d updates failed", failed)
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	mins := int(d.Minutes())
	secs := d.Seconds() - float64(mins*60)
	return fmt.Sprintf("%dm%.1fs", mins, secs)
}

func (TerminalRenderer) Info(loadRes tool.LoadResult, target string) error {
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
