package tool

import (
	"os/exec"
)

type ToolUpdateResult struct {
	Tool    Tool
	Success bool
	Error   error
}

func UpdateTools(gobin string, filter []string) ([]ToolUpdateResult, error) {
	tools, err := Load(gobin)
	if err != nil {
		return nil, err
	}

	filterMap := make(map[string]bool)
	for _, f := range filter {
		filterMap[f] = true
	}

	var results []ToolUpdateResult

	for _, t := range tools {
		if len(filterMap) > 0 && !filterMap[t.Name()] {
			continue
		}

		if !t.CanUpdate() {
			continue
		}

		cmd := exec.Command("go", "install", t.InstallTarget()+"@latest")
		err := cmd.Run()

		results = append(results, ToolUpdateResult{
			Tool:    t,
			Success: err == nil,
			Error:   err,
		})
	}

	return results, nil
}
