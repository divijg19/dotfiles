package tool

import (
	"debug/buildinfo"
	"runtime/debug"
	"testing"
)

func planTool(name string, devel bool) Tool {
	version := "v1.0.0"
	if devel {
		version = "(devel)"
	}
	return NewTool(name, "/gobin/"+name, &buildinfo.BuildInfo{
		Path: "example.com/" + name + "/cmd/" + name,
		Main: debug.Module{Path: "example.com/" + name, Version: version},
	})
}

func TestPlan_AllTools(t *testing.T) {
	tools := []Tool{
		planTool("hello", false),
		planTool("world", false),
		planTool("localdev", true),
	}
	result := Plan(LoadResult{Tools: tools}, nil)
	if len(result.ToUpdate) != 2 {
		t.Errorf("expected 2 updatable tools, got %d", len(result.ToUpdate))
	}
	if len(result.Skipped) != 1 || result.Skipped[0].Name() != "localdev" {
		t.Errorf("expected localdev skipped, got %v", result.Skipped)
	}
}

func TestPlan_Filter(t *testing.T) {
	tools := []Tool{
		planTool("hello", false),
		planTool("world", false),
		planTool("localdev", true),
	}
	result := Plan(LoadResult{Tools: tools}, []string{"world"})
	if len(result.ToUpdate) != 1 || result.ToUpdate[0].Name() != "world" {
		t.Errorf("expected only world to update, got %v", result.ToUpdate)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("filtered-out tools must not appear in skipped, got %v", result.Skipped)
	}
}

func TestPlan_FilterMatchesLocalOnly(t *testing.T) {
	tools := []Tool{
		planTool("localdev", true),
	}
	result := Plan(LoadResult{Tools: tools}, []string{"localdev"})
	if len(result.ToUpdate) != 0 {
		t.Errorf("local tool must never be an update candidate, got %v", result.ToUpdate)
	}
	if len(result.Skipped) != 1 || result.Skipped[0].Name() != "localdev" {
		t.Errorf("expected localdev in skipped, got %v", result.Skipped)
	}
}

func TestPlan_InvalidBinariesAlwaysSkipped(t *testing.T) {
	loadRes := LoadResult{
		Tools: []Tool{planTool("hello", false)},
		Invalid: []InvalidBinary{
			{Path: "/gobin/notgo", Error: ErrMissingBuildInfo},
		},
	}
	result := Plan(loadRes, []string{"hello"})
	if len(result.Invalid) != 1 || result.Invalid[0].Path != "/gobin/notgo" {
		t.Errorf("invalid binaries must always be reported, got %v", result.Invalid)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("no local tools expected, got %v", result.Skipped)
	}
}

func TestInstallRef(t *testing.T) {
	if got := InstallRef("example.com/hello"); got != "example.com/hello@latest" {
		t.Errorf("InstallRef = %q, want example.com/hello@latest", got)
	}
}

func TestInstallCommand(t *testing.T) {
	got := InstallCommand("example.com/hello")
	want := "go install example.com/hello@latest"
	if got != want {
		t.Errorf("InstallCommand = %q, want %q", got, want)
	}
}

// InstallCommand must always agree with the reference Update actually runs,
// so the plan can never display a command that differs from execution.
func TestInstallCommandAgreesWithUpdate(t *testing.T) {
	target := "example.com/hello"
	if got := InstallCommand(target); got != "go install "+InstallRef(target) {
		t.Errorf("command %q does not wrap ref %q", got, InstallRef(target))
	}
}
