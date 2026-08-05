package app

import (
	"time"

	"helm/internal/tool"
)

// RenderMode selects the concrete renderer in main via NewRenderer.
type RenderMode int

const (
	// ModeTerminal is the default human-oriented renderer (Unicode, progress).
	ModeTerminal RenderMode = iota
	// ModeQuiet suppresses informational output for scripting (Terminal output,
	// banner + progress hidden).
	ModeQuiet
	// ModeCI produces deterministic, ASCII-only, line-oriented terminal output.
	ModeCI
	// ModeJSON emits machine-readable JSON only.
	ModeJSON
)

// Renderer is the single output abstraction for every operation.
// Business logic never knows how output is formatted.
type Renderer interface {
	// Header renders the discovery header. Renderers that fold header
	// information into a machine format (JSON) or hide it (quiet) no-op.
	Header(hdr HeaderInfo) error
	Inventory(report InventoryReport) error
	Plan(report PlanReport) error
	Outdated(report OutdatedReport) error
	Update(report UpdateReport) error
	Info(loadRes tool.LoadResult, target string) error
}

// ProgressSink is implemented by renderers that stream per-tool progress
// while an update runs. Business logic uses it only when present (type probe).
type ProgressSink interface {
	OnProgress(p tool.Progress)
}

// NewRenderer returns the renderer matching the requested mode. verbose
// selects the detailed planning view (packages + commands) for human
// renderers; it is ignored by JSON and quiet modes.
func NewRenderer(mode RenderMode, verbose bool) Renderer {
	switch mode {
	case ModeQuiet:
		return QuietRenderer{}
	case ModeCI:
		return CIRenderer{verbose: verbose}
	case ModeJSON:
		return JSONRenderer{}
	default:
		return TerminalRenderer{verbose: verbose}
	}
}

// HeaderInfo carries the discovery context shown by human renderers.
type HeaderInfo struct {
	Gobin     string
	GoVersion string // may be empty when go env GOVERSION cannot be read
	LoadRes   tool.LoadResult
}

// Operation identifiers carried by every operation report. These values are
// frozen for the 1.x series; the JSON renderer emits them, human renderers
// ignore them. --dry-run reports the same operation as --check.
const (
	OperationList     = "list"
	OperationCheck    = "check"
	OperationUpdate   = "update"
	OperationOutdated = "outdated"
)

// OperationEnvelope is the machine-readable prefix every operation report
// carries. Human renderers never read it; the JSON renderer serializes it.
type OperationEnvelope struct {
	Operation string `json:"operation"`
	Success   bool   `json:"success"`
}

type ListReport struct {
	OperationEnvelope
	Tools []ToolReport `json:"tools"`
}

type InventoryReport struct {
	OperationEnvelope
	Tools   []ToolInventoryItem `json:"tools"`
	Invalid []InvalidReport     `json:"invalid,omitempty"`
	Summary InventorySummary    `json:"summary"`
}

type ToolInventoryItem struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	PackagePath string `json:"package_path"`
	ModulePath  string `json:"module_path,omitempty"`
	Status      string `json:"status"` // "Healthy", "Local", "Unhealthy", "Invalid"
	Error       string `json:"error,omitempty"`
}

type InventorySummary struct {
	Healthy   int `json:"healthy"`
	Local     int `json:"local"`
	Invalid   int `json:"invalid"`
	Unhealthy int `json:"unhealthy"`
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

type OutdatedReport struct {
	OperationEnvelope
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
	OperationEnvelope
	Updated     []string          `json:"updated"`
	Notes       []string          `json:"notes,omitempty"`
	Skipped     []string          `json:"skipped"`
	Failed      []string          `json:"failed"`
	Duration    time.Duration     `json:"-"`
	Diagnostics []tool.Diagnostic `json:"-"`
}

// PlanReport is the single unified planning operation report produced by
// --check and --dry-run (aliases). Verbosity only affects human rendering; the
// JSON shape is identical regardless of --verbose.
type PlanReport struct {
	OperationEnvelope
	WouldUpdate []PlanItem `json:"would_update"`
	Skipped     []PlanItem `json:"skipped"`
}

type PlanItem struct {
	Name          string `json:"name"`
	PackagePath   string `json:"package_path,omitempty"`
	InstallTarget string `json:"install_target,omitempty"`
	Command       string `json:"command,omitempty"`
}
