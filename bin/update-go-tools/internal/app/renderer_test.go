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
	results := []tool.ToolUpdateResult{
		{Tool: makeTool("good", "example.com/good", "v1.0.0"), Status: tool.StatusUpdated, Success: true},
		{Tool: makeTool("skipped", "example.com/skipped", "(devel)"), Status: tool.StatusSkippedLocal, Success: false},
		{Tool: makeTool("failed", "example.com/failed", "v1.0.0"), Status: tool.StatusFailed, Success: false, Notes: []string{"err"}},
	}

	out, err := captureOutput(t, func() error {
		return r.Update(results, tool.LoadResult{}, 0, nil, false)
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
	results := []tool.ToolUpdateResult{
		{Tool: makeTool("hello", "example.com/hello", "v1.0.0"), Status: tool.StatusUpdated, Success: true},
		{Tool: makeTool("localdev", "example.com/localdev", "(devel)"), Status: tool.StatusSkippedLocal, Success: false},
	}

	out, err := captureOutput(t, func() error {
		return r.Update(results, tool.LoadResult{}, 0, nil, false)
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
	results := []tool.ToolUpdateResult{
		{Tool: makeTool("hello", "example.com/hello", "v1.0.0"), Status: tool.StatusUpdated, Success: true},
	}
	out, err := captureOutput(t, func() error {
		return r.Update(results, tool.LoadResult{}, 0, nil, true)
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
	loadRes := tool.LoadResult{
		Invalid: []tool.InvalidBinary{
			{Path: "/gobin/notgo", Error: tool.ErrMissingBuildInfo},
		},
	}
	err := r.Verify(loadRes)
	if err == nil {
		t.Error("expected error for unhealthy verify")
	}
}
