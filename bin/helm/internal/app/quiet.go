package app

import (
	"fmt"
	"os"

	"helm/internal/tool"
)

// QuietRenderer is the shell-scripting mode. It suppresses the banner,
// discovery summary, progress renderer, and per-tool status, emitting only the
// final update summary (Updated / Skipped / Failed / Duration) plus
// diagnostics and failures.
type QuietRenderer struct{}

func (QuietRenderer) Header(HeaderInfo) error { return nil }

func (QuietRenderer) Inventory(report InventoryReport) error {
	// Inventory output is the requested data, not chatter; still print it but
	// without the discovery header.
	return TerminalRenderer{}.Inventory(report)
}

func (QuietRenderer) Plan(report PlanReport) error {
	return TerminalRenderer{}.Plan(report)
}

func (QuietRenderer) Outdated(report OutdatedReport) error {
	return TerminalRenderer{}.Outdated(report)
}

func (QuietRenderer) Update(report UpdateReport) error {
	printSummaryLine("Updated", itoa(len(report.Updated)))
	printSummaryLine("Skipped", itoa(len(report.Skipped)))
	printSummaryLine("Failed", itoa(len(report.Failed)))
	printSummaryLine("Duration", formatDuration(report.Duration))

	if len(report.Diagnostics) > 0 {
		fmt.Println()
		for _, d := range report.Diagnostics {
			fmt.Fprintf(os.Stderr, "diagnostic: %s: %s\n", d.ToolName, d.Message)
		}
	}

	if len(report.Failed) > 0 {
		fmt.Println()
		for _, f := range report.Failed {
			fmt.Fprintf(os.Stderr, "failed: %s\n", f)
		}
		return fmt.Errorf("%d updates failed", len(report.Failed))
	}
	return nil
}

func (QuietRenderer) Info(loadRes tool.LoadResult, target string) error {
	return TerminalRenderer{}.Info(loadRes, target)
}
