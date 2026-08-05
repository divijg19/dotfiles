package app

import (
	"bytes"
	"debug/buildinfo"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"testing"

	"update-go-tools/internal/tool"
)

func captureOutput(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	runErr := fn()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	r.Close()
	return buf.String(), runErr
}

func makeTool(name, pkg, version string) tool.Tool {
	return tool.NewTool(name, "/gobin/"+name, &buildinfo.BuildInfo{
		Path: pkg + "/cmd/" + name,
		Main: debug.Module{Path: pkg, Version: version},
	})
}

func TestTerminalUpdate_ClassesStatusCorrectly(t *testing.T) {
	r := TerminalRenderer{}
	report := UpdateReport{
		Updated: []string{"good"},
		Failed:  []string{"failed"},
		Skipped: []string{"localdev"},
	}

	out, err := captureOutput(t, func() error {
		return r.Update(report)
	})
	if err == nil {
		t.Error("expected error because one update failed")
	}
	if !bytes.Contains([]byte(out), []byte("Updated")) {
		t.Errorf("expected Updated, got:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("Failed")) {
		t.Errorf("expected Failed, got:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("Skipped")) {
		t.Errorf("expected Skipped, got:\n%s", out)
	}
}

func TestJSONRenderer_SkippedNotInFailed(t *testing.T) {
	r := JSONRenderer{}
	report := UpdateReport{
		Updated: []string{"hello"},
		Skipped: []string{"localdev"},
		Failed:  make([]string, 0),
	}

	out, err := captureOutput(t, func() error {
		return r.Update(report)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Contains([]byte(out), []byte(`"failed": []`)) {
		t.Errorf("local/devel tool must not appear in failed list (regression):\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte(`"skipped": [`)) {
		t.Errorf("expected skipped list populated:\n%s", out)
	}
}

func TestJSONRenderer_InventoryReportsIssues(t *testing.T) {
	r := JSONRenderer{}
	report := InventoryReport{
		Tools: []ToolInventoryItem{
			{Name: "hello", Version: "v1.0.0", PackagePath: "example.com/hello", Status: "Healthy"},
		},
		Invalid: []InvalidReport{
			{Path: "/gobin/notgo", Message: "missing or unreadable build info"},
		},
		Summary: InventorySummary{Healthy: 1, Invalid: 1, Unhealthy: 1},
	}
	_, err := captureOutput(t, func() error {
		return r.Inventory(report)
	})
	if err == nil {
		t.Error("expected error when inventory has issues")
	}
}

func TestJSONRenderer_PlanSchemaStable(t *testing.T) {
	r := JSONRenderer{}
	report := PlanReport{
		OperationEnvelope: OperationEnvelope{Operation: OperationCheck, Success: true},
		WouldUpdate: []PlanItem{
			{Name: "hello", PackagePath: "example.com/hello", InstallTarget: "example.com/hello", Command: "go install example.com/hello@latest"},
		},
		Skipped: []PlanItem{{Name: "localdev"}},
	}
	out, err := captureOutput(t, func() error {
		return r.Plan(report)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{`"operation": "check"`, `"success": true`, `"would_update"`, `"skipped"`, `"command"`, `"install_target"`} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("expected %s in plan JSON:\n%s", want, out)
		}
	}
}

func TestJSONEnvelope_OperationAndSuccess(t *testing.T) {
	r := JSONRenderer{}

	plan, _ := captureOutput(t, func() error {
		return r.Plan(PlanReport{OperationEnvelope: OperationEnvelope{Operation: OperationCheck, Success: true}})
	})
	if !bytes.Contains([]byte(plan), []byte(`"operation": "check"`)) || !bytes.Contains([]byte(plan), []byte(`"success": true`)) {
		t.Errorf("plan envelope mismatch:\n%s", plan)
	}

	update, _ := captureOutput(t, func() error {
		return r.Update(UpdateReport{
			OperationEnvelope: OperationEnvelope{Operation: OperationUpdate, Success: false},
			Updated:           []string{"hello"},
			Skipped:           []string{},
			Failed:            []string{"world"},
		})
	})
	if !bytes.Contains([]byte(update), []byte(`"operation": "update"`)) || !bytes.Contains([]byte(update), []byte(`"success": false`)) {
		t.Errorf("update envelope mismatch:\n%s", update)
	}
}

func TestJSONRenderer_InfoNotFound(t *testing.T) {
	r := JSONRenderer{}
	loadRes := tool.LoadResult{
		Tools: []tool.Tool{makeTool("hello", "example.com/hello", "v1.0.0")},
	}
	err := r.Info(loadRes, "missing")
	if err == nil {
		t.Error("expected error for missing tool")
	}
}

func TestJSONRenderer_InfoFound(t *testing.T) {
	r := JSONRenderer{}
	loadRes := tool.LoadResult{
		Tools: []tool.Tool{makeTool("hello", "example.com/hello", "v1.0.0")},
	}
	out, err := captureOutput(t, func() error {
		return r.Info(loadRes, "hello")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte(`"name": "hello"`)) {
		t.Errorf("expected tool in JSON output:\n%s", out)
	}
}

func TestTerminalRenderer_InventorySummary(t *testing.T) {
	r := TerminalRenderer{}
	report := InventoryReport{
		Tools: []ToolInventoryItem{
			{Name: "hello", Version: "v1.0.0", PackagePath: "example.com/hello", Status: "Healthy"},
			{Name: "localdev", Version: "(devel)", PackagePath: "example.com/localdev", Status: "Local"},
		},
		Invalid: []InvalidReport{
			{Path: "/gobin/notgo", Message: "missing or unreadable build info"},
		},
		Summary: InventorySummary{Healthy: 2, Local: 1, Invalid: 1, Unhealthy: 1},
	}
	out, err := captureOutput(t, func() error {
		return r.Inventory(report)
	})
	if err == nil {
		t.Error("expected error for unhealthy inventory")
	}
	if !bytes.Contains([]byte(out), []byte("NAME")) {
		t.Errorf("expected table header in output:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("Healthy")) {
		t.Errorf("expected Healthy in output:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("Local         1")) {
		t.Errorf("expected Local count in output:\n%s", out)
	}
}

func TestTerminalRenderer_PlanConciseAndVerbose(t *testing.T) {
	report := PlanReport{
		WouldUpdate: []PlanItem{
			{Name: "hello", PackagePath: "example.com/hello", InstallTarget: "example.com/hello", Command: "go install example.com/hello@latest"},
		},
		Skipped: []PlanItem{{Name: "localdev"}},
	}

	concise, err := captureOutput(t, func() error {
		return TerminalRenderer{}.Plan(report)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(concise, "go install") {
		t.Errorf("concise plan must not show commands:\n%s", concise)
	}

	verbose, err := captureOutput(t, func() error {
		return TerminalRenderer{verbose: true}.Plan(report)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(verbose, "go install example.com/hello@latest") {
		t.Errorf("verbose plan must show install commands:\n%s", verbose)
	}
}

func TestQuietRenderer_UpdateSummaryOnly(t *testing.T) {
	r := QuietRenderer{}
	report := UpdateReport{
		Updated: []string{"hello"},
		Failed:  make([]string, 0),
		Skipped: []string{"localdev"},
	}
	out, err := captureOutput(t, func() error {
		return r.Update(report)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"Updated", "Skipped", "Failed", "Duration"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("expected %s in quiet summary:\n%s", want, out)
		}
	}
	if bytes.Contains([]byte(out), []byte("hello")) {
		t.Errorf("quiet mode must not print per-tool names:\n%s", out)
	}
}

func TestQuietRenderer_HeaderNoop(t *testing.T) {
	out, err := captureOutput(t, func() error {
		return QuietRenderer{}.Header(HeaderInfo{Gobin: "/gobin"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("quiet header must emit nothing, got:\n%s", out)
	}
}

func TestCIRenderer_ASCIIOnly(t *testing.T) {
	r := CIRenderer{}
	report := InventoryReport{
		Tools: []ToolInventoryItem{
			{Name: "hello", Version: "v1.0.0", PackagePath: "example.com/hello", Status: "Healthy"},
		},
		Summary: InventorySummary{Healthy: 1},
	}
	out, err := captureOutput(t, func() error {
		return r.Inventory(report)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "✓") || strings.Contains(out, "✗") || strings.Contains(out, "•") {
		t.Errorf("CI renderer must not emit Unicode symbols:\n%s", out)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("expected ASCII status OK in CI output:\n%s", out)
	}
}

func TestCIRenderer_DeterministicPlan(t *testing.T) {
	r := CIRenderer{}
	report := PlanReport{
		WouldUpdate: []PlanItem{
			{Name: "world", PackagePath: "example.com/world", InstallTarget: "example.com/world", Command: "go install example.com/world@latest"},
		},
	}
	out, err := captureOutput(t, func() error {
		return r.Plan(report)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "update: world") {
		t.Errorf("expected line-oriented update entry:\n%s", out)
	}
	if strings.Contains(out, "would-update: 0") {
		t.Errorf("expected would-update count to reflect plan:\n%s", out)
	}
}
