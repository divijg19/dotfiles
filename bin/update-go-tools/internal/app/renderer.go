package app

import (
	"time"
	"update-go-tools/internal/tool"
)

type Renderer interface {
	Inventory(report InventoryReport) error
	Verify(report VerifyReport) error
	Outdated(report OutdatedReport) error
	Update(report UpdateReport) error
	DryRun(report DryRunReport) error
	Info(loadRes tool.LoadResult, target string) error
}

type ListReport struct {
	Tools []ToolReport `json:"tools"`
}

type InventoryReport struct {
	Tools   []ToolReport     `json:"tools"`
	Invalid []InvalidReport  `json:"invalid,omitempty"`
	Summary tool.LoadSummary `json:"-"`
}

type ToolReport struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	PackagePath string `json:"package_path"`
	ModulePath  string `json:"module_path"`
}

type InvalidReport struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type VerifyReport struct {
	Results []VerifyResultReport `json:"results"`
	Invalid []InvalidReport      `json:"-"`
	Summary VerifySummary        `json:"-"`
}

type VerifyResultReport struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	PackagePath string `json:"package_path,omitempty"`
	Healthy     bool   `json:"healthy"`
	Error       string `json:"error,omitempty"`
}

type VerifySummary struct {
	Healthy   int `json:"healthy"`
	Local     int `json:"local"`
	Invalid   int `json:"invalid"`
	Unhealthy int `json:"unhealthy"`
}

type OutdatedReport struct {
	Results []OutdatedItemReport `json:"results"`
	Summary OutdatedSummary      `json:"-"`
}

type OutdatedItemReport struct {
	Name     string `json:"name"`
	Current  string `json:"current"`
	Latest   string `json:"latest,omitempty"`
	Outdated bool   `json:"outdated"`
	Error    string `json:"error,omitempty"`
}

type OutdatedSummary struct {
	Outdated int `json:"outdated"`
	UpToDate int `json:"up_to_date"`
}

type UpdateReport struct {
	Updated      []string          `json:"updated"`
	Notes        []string          `json:"notes,omitempty"`
	Skipped      []string          `json:"skipped"`
	Failed       []string          `json:"failed"`
	CheckOnly    bool              `json:"check_only"`
	Duration     time.Duration     `json:"-"`
	Diagnostics  []tool.Diagnostic `json:"-"`
	CheckTargets []CheckTarget     `json:"-"`
}

type CheckTarget struct {
	Name          string
	InstallTarget string
}

type DryRunReport struct {
	ToUpdate []DryRunItem `json:"would_update"`
	Skipped  []DryRunItem `json:"skipped"`
}

type DryRunItem struct {
	Name          string `json:"name"`
	PackagePath   string `json:"package_path,omitempty"`
	InstallTarget string `json:"install_target,omitempty"`
}
