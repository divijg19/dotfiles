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

func (TerminalRenderer) Inventory(report InventoryReport) error {
	fmt.Printf("%-20s %-15s %s\n", "NAME", "VERSION", "PACKAGE PATH")
	fmt.Printf("%-20s %-15s %s\n", "----", "-------", "------------")
	for _, t := range report.Tools {
		fmt.Printf("%-20s %-15s %s\n", t.Name, t.Version, t.PackagePath)
	}
	if report.Summary.Invalid > 0 {
		fmt.Printf("\nInvalid / Uninspectable binaries (%d):\n", report.Summary.Invalid)
		for _, inv := range report.Invalid {
			fmt.Printf("  - %s (%s)\n", inv.Path, inv.Message)
		}
	}
	return nil
}

func (TerminalRenderer) Verify(report VerifyReport) error {
	maxVerLen := 7
	for _, r := range report.Results {
		if r.Healthy {
			ver := r.Version
			if ver == "" {
				ver = r.Name
			}
			if len(ver) > maxVerLen {
				maxVerLen = len(ver)
			}
		}
	}
	for _, inv := range report.Invalid {
		if len(inv.Message) > maxVerLen {
			maxVerLen = len(inv.Message)
		}
	}

	healthy := 0
	localCount := 0
	for _, r := range report.Results {
		if r.Healthy {
			status := "✓"
			extra := r.Version
			if extra == "" {
				extra = r.Name
			}
			format := fmt.Sprintf("%%s %%-15s %%-%ds  %%s\n", maxVerLen)
			fmt.Printf(format, status, r.Name, extra, r.PackagePath)
			healthy++
			if r.PackagePath == "" || r.PackagePath == "(devel)" {
				localCount++
			}
		} else {
			fmt.Fprintf(os.Stderr, "✗ %-15s (%s)\n", r.Name, r.Error)
		}
	}

	for _, inv := range report.Invalid {
		fmt.Fprintf(os.Stderr, "✗ %-15s (%s)\n", inv.Path, inv.Message)
	}

	fmt.Printf("\nHealthy    %d\n", report.Summary.Healthy)
	fmt.Printf("Local      %d\n", report.Summary.Local)
	fmt.Printf("Invalid    %d\n", report.Summary.Invalid)
	fmt.Printf("Unhealthy  %d\n", report.Summary.Unhealthy)
	if report.Summary.Unhealthy > 0 {
		return fmt.Errorf("%d unhealthy binaries found", report.Summary.Unhealthy)
	}
	return nil
}

func (TerminalRenderer) Outdated(report OutdatedReport) error {
	maxNameLen := 4
	maxCurrLen := 7
	for _, o := range report.Results {
		if len(o.Name) > maxNameLen {
			maxNameLen = len(o.Name)
		}
		if len(o.Current) > maxCurrLen {
			maxCurrLen = len(o.Current)
		}
	}

	format := fmt.Sprintf("%%-%ds   %%-%ds   %%s\n", maxNameLen, maxCurrLen)
	fmt.Printf(format, "NAME", "CURRENT", "STATUS")

	outdatedCount := 0
	upToDateCount := 0

	for _, o := range report.Results {
		status := "✓"
		if o.Error != "" {
			status = "error (" + o.Error + ")"
		} else if o.Outdated {
			status = "↑ " + o.Latest
			outdatedCount++
		} else {
			upToDateCount++
		}
		fmt.Printf(format, o.Name, o.Current, status)
	}
	fmt.Println()
	fmt.Printf("%d tools checked\n\n", len(report.Results))
	fmt.Printf("%d outdated\n", report.Summary.Outdated)
	fmt.Printf("%d up-to-date\n", report.Summary.UpToDate)
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
	}
}

func (r TerminalRenderer) Update(report UpdateReport) error {
	if len(report.Skipped) > 0 {
		fmt.Println()
		fmt.Println("Skipped")
		fmt.Println()
		for _, name := range report.Skipped {
			fmt.Printf("• %s\n", name)
		}
		fmt.Println()
	}

	if len(report.Diagnostics) > 0 {
		fmt.Println()
		fmt.Println("Diagnostics")
		fmt.Println()
		for _, d := range report.Diagnostics {
			fmt.Printf("• %s\n", d.ToolName)
			fmt.Printf("  Category : %s\n", d.Category)
			fmt.Printf("  Message  : %s\n\n", d.Message)
		}
	}

	fmt.Println()
	fmt.Println("Summary")
	fmt.Println()
	fmt.Printf("Updated       %d\n", len(report.Updated))
	fmt.Printf("Skipped       %d\n", len(report.Skipped))
	fmt.Printf("Failed        %d\n", len(report.Failed))
	fmt.Printf("Duration      %s\n", formatDuration(report.Duration))

	if len(report.Failed) > 0 {
		fmt.Println()
		fmt.Println("Failed tools:")
		for _, f := range report.Failed {
			fmt.Printf("- %s\n", f)
		}
		return fmt.Errorf("%d updates failed", len(report.Failed))
	}

	return nil
}

func (TerminalRenderer) Check(report CheckReport) error {
	for _, ct := range report.CheckTargets {
		fmt.Printf("Would update %-16s -> %s\n", ct.Name, ct.InstallTarget)
	}
	fmt.Println()
	fmt.Printf("Would update: %d\n", len(report.CheckTargets))
	return nil
}

func (TerminalRenderer) DryRun(report DryRunReport) error {
	fmt.Println("Update plan")
	fmt.Println()
	for _, item := range report.ToUpdate {
		fmt.Printf("  %s\n", item.Name)
		fmt.Printf("    Package : %s\n", item.PackagePath)
		fmt.Printf("    Command : %s\n\n", item.Command)
	}
	if len(report.Skipped) > 0 {
		fmt.Println("Skipped")
		fmt.Println()
		for _, item := range report.Skipped {
			fmt.Printf("  • %s\n", item.Name)
		}
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