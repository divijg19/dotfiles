package app

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"update-go-tools/internal/tool"
)

const (
	symCheck    = "✓"
	symBullet   = "•"
	symFail     = "✗"
	symOutdated = "↑"
	symNote     = "ⓘ"
)

type TerminalRenderer struct {
	verbose bool
}

func (r TerminalRenderer) Header(hdr HeaderInfo) error {
	if hdr.GoVersion != "" {
		fmt.Printf("Go: %s\n\n", hdr.GoVersion)
	}
	fmt.Println("Discovery")
	fmt.Println()
	fmt.Printf("  %-11s : %s\n", "Gobin", hdr.Gobin)
	fmt.Printf("  %-11s : %d\n", "Executables", hdr.LoadRes.Summary.Executables)
	fmt.Printf("  %-11s : %d\n", "Updatable", hdr.LoadRes.Summary.Updatable)
	fmt.Printf("  %-11s : %d\n", "Local", hdr.LoadRes.Summary.Local)
	fmt.Printf("  %-11s : %d\n", "Invalid", hdr.LoadRes.Summary.Invalid)
	fmt.Println()
	if hdr.LoadRes.Summary.Local > 0 {
		fmt.Println("Skipping local development binaries:")
		for _, t := range hdr.LoadRes.Tools {
			if !t.CanUpdate() {
				fmt.Printf("  %s %s\n", symBullet, t.Name())
			}
		}
		fmt.Println()
	}
	return nil
}

func (r TerminalRenderer) Inventory(report InventoryReport) error {
	if len(report.Tools) == 0 && len(report.Invalid) == 0 {
		fmt.Println("No Go tools found.")
		return nil
	}

	maxNameLen := 4
	maxVerLen := 7
	maxStatusLen := 7
	for _, t := range report.Tools {
		if len(t.Name) > maxNameLen {
			maxNameLen = len(t.Name)
		}
		if len(t.Version) > maxVerLen {
			maxVerLen = len(t.Version)
		}
		if len(t.Status) > maxStatusLen {
			maxStatusLen = len(t.Status)
		}
	}

	format := fmt.Sprintf("%%-%ds   %%-%ds   %%-%ds   %%s\n", maxNameLen, maxVerLen, maxStatusLen)
	fmt.Printf(format, "NAME", "VERSION", "STATUS", "PACKAGE")

	for _, t := range report.Tools {
		if t.Name == "" {
			continue
		}
		fmt.Printf(format, t.Name, t.Version, t.Status, t.PackagePath)
		if t.Error != "" {
			fmt.Fprintf(os.Stderr, "  ↳ %s\n", t.Error)
		}
	}

	fmt.Println()
	fmt.Println("Invalid / Uninspectable binaries")
	fmt.Println()
	if len(report.Invalid) > 0 {
		for _, inv := range report.Invalid {
			fmt.Printf("  %s %s (%s)\n", symBullet, inv.Path, inv.Message)
		}
	} else {
		fmt.Println("  none")
	}

	fmt.Println()
	printSummaryBlock([][2]string{
		{"Healthy", itoa(report.Summary.Healthy)},
		{"Local", itoa(report.Summary.Local)},
		{"Invalid", itoa(report.Summary.Invalid)},
		{"Unhealthy", itoa(report.Summary.Unhealthy)},
	})

	if report.Summary.Unhealthy > 0 || report.Summary.Invalid > 0 {
		return fmt.Errorf("%d issues found during inventory check", report.Summary.Unhealthy+report.Summary.Invalid)
	}
	return nil
}

func (r TerminalRenderer) Plan(report PlanReport) error {
	if len(report.WouldUpdate) > 0 {
		fmt.Println("Would update")
		fmt.Println()
		for _, item := range report.WouldUpdate {
			if r.verbose {
				fmt.Printf("  %s\n", item.Name)
				fmt.Printf("    Package : %s\n", item.PackagePath)
				fmt.Printf("    Command : %s\n\n", item.Command)
			} else {
				fmt.Printf("  %s\n", item.Name)
			}
		}
		fmt.Println()
	}

	if len(report.Skipped) > 0 {
		fmt.Println("Skipped")
		fmt.Println()
		for _, item := range report.Skipped {
			fmt.Printf("  %s %s\n", symBullet, item.Name)
		}
		fmt.Println()
	}

	printSummaryBlock([][2]string{
		{"Would update", itoa(len(report.WouldUpdate))},
		{"Skipped", itoa(len(report.Skipped))},
	})
	return nil
}

func (r TerminalRenderer) Outdated(report OutdatedReport) error {
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
		status := symCheck
		if o.Error != "" {
			status = "error (" + o.Error + ")"
		} else if o.Outdated {
			status = symOutdated + " " + o.Latest
			outdatedCount++
		} else {
			upToDateCount++
		}
		fmt.Printf(format, o.Name, o.Current, status)
	}

	fmt.Println()
	printSummaryBlock([][2]string{
		{"Checked", itoa(len(report.Results))},
		{"Outdated", itoa(report.Summary.Outdated)},
		{"Up-to-date", itoa(report.Summary.UpToDate)},
	})
	return nil
}

func (r TerminalRenderer) OnProgress(p tool.Progress) {
	switch p.Action {
	case "Start":
		fmt.Printf("[%02d/%02d] %-18s", p.Current, p.Total, p.Tool.Name())
	case "Output":
		fmt.Printf("  %s\n", p.Line)
	case "Complete":
		if p.Success && len(p.Notes) == 0 {
			fmt.Printf("           %s\n", symCheck)
		} else if p.Success && len(p.Notes) > 0 {
			fmt.Printf("           %s\n", symNote)
			fmt.Printf("  Package    %s\n", p.Tool.InstallTarget())
			for _, note := range p.Notes {
				fmt.Printf("  %s\n", note)
			}
		} else {
			fmt.Printf("           %s\n", symFail)
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
			fmt.Printf("%s %s\n", symBullet, name)
		}
		fmt.Println()
	}

	if len(report.Diagnostics) > 0 {
		fmt.Println()
		fmt.Println("Diagnostics")
		fmt.Println()
		for _, d := range report.Diagnostics {
			fmt.Printf("%s %s\n", symBullet, d.ToolName)
			fmt.Printf("  Category : %s\n", d.Category)
			fmt.Printf("  Message  : %s\n\n", d.Message)
		}
	}

	printSummaryBlock([][2]string{
		{"Updated", itoa(len(report.Updated))},
		{"Skipped", itoa(len(report.Skipped))},
		{"Failed", itoa(len(report.Failed))},
		{"Duration", formatDuration(report.Duration)},
	})

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

// summaryLabelWidth aligns every summary block across all commands so each
// renderer ends with the same visual rhythm: "Summary" then aligned values.
const summaryLabelWidth = 14

func printSummaryLine(label, value string) {
	fmt.Printf("%-*s%s\n", summaryLabelWidth, label, value)
}

func printSummaryBlock(rows [][2]string) {
	fmt.Println("Summary")
	fmt.Println()
	for _, row := range rows {
		printSummaryLine(row[0], row[1])
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	mins := int(d.Minutes())
	secs := d.Seconds() - float64(mins*60)
	return fmt.Sprintf("%dm%.1fs", mins, secs)
}

func (r TerminalRenderer) Info(loadRes tool.LoadResult, target string) error {
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
