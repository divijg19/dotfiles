package tool

import (
	"bytes"
	"os/exec"
	"strings"
)

type ToolUpdateResult struct {
	Tool    Tool
	Status  Status
	Success bool
	Notes   []string
	Error   error
}

func Update(tools []Tool, filter []string, dryRun bool) <-chan ToolUpdateResult {
	ch := make(chan ToolUpdateResult)

	go func() {
		defer close(ch)

		filterMap := make(map[string]bool)
		for _, f := range filter {
			filterMap[f] = true
		}

		for _, t := range tools {
			if len(filterMap) > 0 && !filterMap[t.Name()] {
				continue
			}

			if !t.CanUpdate() {
				ch <- ToolUpdateResult{
					Tool:   t,
					Status: StatusSkippedLocal,
				}
				continue
			}

			if dryRun {
				ch <- ToolUpdateResult{
					Tool:    t,
					Status:  StatusUpdated,
					Success: true,
				}
				continue
			}

			cmd := exec.Command("go", "install", t.InstallTarget()+"@latest")
			var buf bytes.Buffer
			cmd.Stdout = &buf
			cmd.Stderr = &buf

			err := cmd.Run()
			success := err == nil
			status := StatusUpdated
			if !success {
				status = StatusFailed
			}

			var notes []string
			output := buf.String()
			for _, line := range strings.Split(output, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, "go: downloading") || strings.HasPrefix(line, "go: extracting") {
					continue
				}
				notes = append(notes, line)
			}

			ch <- ToolUpdateResult{
				Tool:    t,
				Status:  status,
				Success: success,
				Notes:   notes,
				Error:   err,
			}
		}
	}()

	return ch
}
