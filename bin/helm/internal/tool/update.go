package tool

import (
	"context"
	"strings"
	"time"
)

type Diagnostic struct {
	ToolName string
	Category string
	Message  string
}

type ToolUpdateResult struct {
	Tool    Tool
	Status  Status
	Success bool
	Notes   []string
	Error   error
}

type Progress struct {
	Current int
	Total   int
	Tool    Tool
	Action  string // "Start", "Output", "Complete", "Skipped"
	Line    string
	Status  Status
	Success bool
	Notes   []string
	Error   error
}

func Update(ctx context.Context, tools []Tool, filter []string, dryRun bool, runner Runner, onProgress func(Progress)) ([]ToolUpdateResult, time.Duration, []Diagnostic) {
	start := time.Now()
	if runner == nil {
		runner = DefaultRunner{}
	}

	set := nameSet(filter)

	var total int
	for _, t := range tools {
		if !selected(t.Name(), set) {
			continue
		}
		total++
	}

	var results []ToolUpdateResult
	var diagnostics []Diagnostic
	var current int

	for _, t := range tools {
		if !selected(t.Name(), set) {
			continue
		}
		current++

		if !t.CanUpdate() {
			prog := Progress{
				Current: current,
				Total:   total,
				Tool:    t,
				Action:  "Skipped",
				Status:  StatusSkippedLocal,
			}
			if onProgress != nil {
				onProgress(prog)
			}
			results = append(results, ToolUpdateResult{
				Tool:   t,
				Status: StatusSkippedLocal,
			})
			continue
		}

		if dryRun {
			prog := Progress{
				Current: current,
				Total:   total,
				Tool:    t,
				Action:  "Complete",
				Status:  StatusUpdated,
				Success: true,
			}
			if onProgress != nil {
				onProgress(prog)
			}
			results = append(results, ToolUpdateResult{
				Tool:    t,
				Status:  StatusUpdated,
				Success: true,
			})
			continue
		}

		if onProgress != nil {
			onProgress(Progress{
				Current: current,
				Total:   total,
				Tool:    t,
				Action:  "Start",
			})
		}

		output, err := runner.Run(ctx, Command{
			Name: "go",
			Args: []string{"install", InstallRef(t.InstallTarget())},
			OnLine: func(line string) {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					return
				}
				if strings.HasPrefix(trimmed, "go: downloading") || strings.HasPrefix(trimmed, "go: extracting") {
					return
				}
				if onProgress != nil {
					onProgress(Progress{
						Current: current,
						Total:   total,
						Tool:    t,
						Action:  "Output",
						Line:    trimmed,
					})
				}
			},
		})

		success := err == nil
		status := StatusUpdated
		if !success {
			status = StatusFailed
		}

		var notes []string
		for _, line := range strings.Split(output, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "go: downloading") || strings.HasPrefix(line, "go: extracting") {
				continue
			}
			notes = append(notes, line)

			lower := strings.ToLower(line)
			if strings.Contains(lower, "deprecated") || strings.Contains(lower, "deprecation") {
				diagnostics = append(diagnostics, Diagnostic{
					ToolName: t.Name(),
					Category: "Deprecation",
					Message:  line,
				})
			} else if strings.Contains(lower, "warning") || strings.Contains(lower, "warn") {
				diagnostics = append(diagnostics, Diagnostic{
					ToolName: t.Name(),
					Category: "Warning",
					Message:  line,
				})
			}
		}

		prog := Progress{
			Current: current,
			Total:   total,
			Tool:    t,
			Action:  "Complete",
			Status:  status,
			Success: success,
			Notes:   notes,
			Error:   err,
		}
		if onProgress != nil {
			onProgress(prog)
		}

		results = append(results, ToolUpdateResult{
			Tool:    t,
			Status:  status,
			Success: success,
			Notes:   notes,
			Error:   err,
		})
	}

	return results, time.Since(start), diagnostics
}
