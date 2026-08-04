package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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

var pseudoVersionRe = regexp.MustCompile(`-\d{14}-[0-9a-f]{12}$`)

func isPseudoVersion(v string) bool {
	return pseudoVersionRe.MatchString(v)
}

func pseudoVersionBase(v string) string {
	idx := strings.Index(v, "-20")
	if idx != -1 {
		base := v[:idx]
		base = strings.TrimSuffix(base, "-0")
		return base
	}
	return v
}

func isRetracted(version string, retracted []string) bool {
	for _, r := range retracted {
		if r == version {
			return true
		}
		if strings.HasPrefix(r, "[") && strings.HasSuffix(r, "]") {
			parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(r, "["), "]"), ",")
			if len(parts) == 2 {
				low := strings.TrimSpace(parts[0])
				high := strings.TrimSpace(parts[1])
				if semver.IsValid(low) && semver.IsValid(high) && semver.IsValid(version) {
					if semver.Compare(version, low) >= 0 && semver.Compare(version, high) <= 0 {
						return true
					}
				}
			}
		}
	}
	return false
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

		if latest == "" {
			results = append(results, OutdatedResult{
				Tool:    t,
				Current: current,
				Error:   fmt.Errorf("unable to resolve latest version"),
			})
			continue
		}

		if isRetracted(latest, mod.Retracted) {
			results = append(results, OutdatedResult{
				Tool:    t,
				Current: current,
				Latest:  latest,
				Error:   fmt.Errorf("latest version %s is retracted", latest),
			})
			continue
		}

		// Normalize versions for semver comparison (ensure leading 'v')
		normCurrent := current
		if isPseudoVersion(normCurrent) {
			normCurrent = pseudoVersionBase(normCurrent)
		}
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