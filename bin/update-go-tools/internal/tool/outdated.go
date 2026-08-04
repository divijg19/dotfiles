package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"
)

type OutdatedResult struct {
	Tool     Tool
	Current  string
	Latest   string
	Outdated bool
	Error    error
}

type goListModule struct {
	Path      string   `json:"Path"`
	Version   string   `json:"Version"`
	Retracted []string `json:"Retracted"`
}

func CheckOutdated(ctx context.Context, tools []Tool, runner Runner) []OutdatedResult {
	if runner == nil {
		runner = DefaultRunner{}
	}

	var results []OutdatedResult
	for _, t := range tools {
		if !t.CanUpdate() {
			continue
		}

		modPath := t.ModulePath()
		if modPath == "" {
			modPath = t.PackagePath()
		}

		output, err := runner.Run(ctx, Command{
			Name: "go",
			Args: []string{"list", "-m", "-json", modPath + "@latest"},
		})
		if err != nil {
			results = append(results, OutdatedResult{
				Tool:    t,
				Current: t.Version(),
				Error:   err,
			})
			continue
		}

		var mod goListModule
		decoder := json.NewDecoder(strings.NewReader(output))
		if err := decoder.Decode(&mod); err != nil {
			results = append(results, OutdatedResult{
				Tool:    t,
				Current: t.Version(),
				Error:   fmt.Errorf("failed to parse go list output: %w", err),
			})
			continue
		}

		latest := mod.Version
		current := t.Version()

		// Normalize versions for semver comparison (ensure leading 'v')
		normCurrent := current
		if !strings.HasPrefix(normCurrent, "v") && semver.IsValid("v"+normCurrent) {
			normCurrent = "v" + normCurrent
		}
		normLatest := latest
		if !strings.HasPrefix(normLatest, "v") && semver.IsValid("v"+normLatest) {
			normLatest = "v" + normLatest
		}

		outdated := false
		if semver.IsValid(normCurrent) && semver.IsValid(normLatest) {
			outdated = semver.Compare(normLatest, normCurrent) > 0
		} else {
			outdated = current != latest && latest != ""
		}

		results = append(results, OutdatedResult{
			Tool:     t,
			Current:  current,
			Latest:   latest,
			Outdated: outdated,
		})
	}

	return results
}
