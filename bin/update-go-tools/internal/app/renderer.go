package app

import (
	"time"
	"update-go-tools/internal/tool"
)

type Renderer interface {
	Inventory(loadRes tool.LoadResult) error
	Verify(loadRes tool.LoadResult) error
	Outdated(outdatedRes []tool.OutdatedResult) error
	Update(results []tool.ToolUpdateResult, loadRes tool.LoadResult, duration time.Duration, diagnostics []tool.Diagnostic, checkOnly bool) error
	Info(loadRes tool.LoadResult, target string) error
}

// JSON Response Models
type ListReport struct {
	Tools []ToolReport `json:"tools"`
}

type ToolReport struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	PackagePath string `json:"package_path"`
	ModulePath  string `json:"module_path"`
}

type VerifyReport struct {
	Results []VerifyResultReport `json:"results"`
}

type VerifyResultReport struct {
	Name    string `json:"name"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

type OutdatedReport struct {
	Results []OutdatedItemReport `json:"results"`
}

type OutdatedItemReport struct {
	Name     string `json:"name"`
	Current  string `json:"current"`
	Latest   string `json:"latest,omitempty"`
	Outdated bool   `json:"outdated"`
	Error    string `json:"error,omitempty"`
}

type UpdateReport struct {
	Updated   []string `json:"updated"`
	Notes     []string `json:"notes,omitempty"`
	Skipped   []string `json:"skipped"`
	Failed    []string `json:"failed"`
	CheckOnly bool     `json:"check_only"`
}
