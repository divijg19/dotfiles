package app

import (
	"bytes"
	"debug/buildinfo"
	"io"
	"os"
	"runtime/debug"
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
	if !bytes.Contains([]byte(out), []byte(`"check_only": false`)) {
		t.Errorf("expected check_only=false in JSON:\n%s", out)
	}
}

func TestJSONRenderer_CheckOnly(t *testing.T) {
	r := JSONRenderer{}
	report := UpdateReport{
		Updated:   []string{"hello"},
		CheckOnly: true,
	}
	out, err := captureOutput(t, func() error {
		return r.Update(report)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte(`"check_only": true`)) {
		t.Errorf("expected check_only=true in JSON:\n%s", out)
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

func TestJSONRenderer_VerifyReportsUnhealthyOnInvalid(t *testing.T) {
	r := JSONRenderer{}
	report := VerifyReport{
		Results: []VerifyResultReport{
			{Name: "/gobin/notgo", Healthy: false, Error: "missing or unreadable build info"},
		},
		Summary: VerifySummary{Unhealthy: 1},
	}
	err := r.Verify(report)
	if err == nil {
		t.Error("expected error for unhealthy verify")
	}
}

func TestTerminalRenderer_VerifyOutput(t *testing.T) {
	r := TerminalRenderer{}
	report := VerifyReport{
		Results: []VerifyResultReport{
			{Name: "hello", Version: "v1.0.0", PackagePath: "example.com/hello", Healthy: true},
			{Name: "localdev", Version: "(devel)", PackagePath: "(devel)", Healthy: true},
		},
		Invalid: []InvalidReport{
			{Path: "/gobin/notgo", Message: "missing or unreadable build info"},
		},
		Summary: VerifySummary{Healthy: 2, Local: 1, Invalid: 1, Unhealthy: 1},
	}
	out, err := captureOutput(t, func() error {
		return r.Verify(report)
	})
	if err == nil {
		t.Error("expected error for unhealthy verify")
	}
	if !bytes.Contains([]byte(out), []byte("✓ hello")) {
		t.Errorf("expected hello in output:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("Local      1")) {
		t.Errorf("expected Local count in output:\n%s", out)
	}
}

func TestTerminalRenderer_DryRun(t *testing.T) {
	r := TerminalRenderer{}
	report := DryRunReport{
		ToUpdate: []DryRunItem{
			{Name: "hello", PackagePath: "example.com/hello", InstallTarget: "example.com/hello", Command: "go install example.com/hello@latest"},
		},
		Skipped: []DryRunItem{
			{Name: "localdev"},
		},
	}
	out, err := captureOutput(t, func() error {
		return r.DryRun(report)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte("Update plan")) {
		t.Errorf("expected Update plan in output:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("hello")) {
		t.Errorf("expected hello in output:\n%s", out)
	}
}
