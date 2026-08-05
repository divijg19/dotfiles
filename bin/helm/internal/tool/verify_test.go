package tool

import (
	"debug/buildinfo"
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
)

func validBuildInfo() *buildinfo.BuildInfo {
	return &buildinfo.BuildInfo{
		Path: "example.com/test/cmd/tool",
		Main: debug.Module{Path: "example.com/test", Version: "v1.0.0"},
	}
}

func TestVerify_Executable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "good")
	if err := os.WriteFile(path, []byte("dummy"), 0o755); err != nil {
		t.Fatal(err)
	}
	tools := []Tool{
		{name: "good", path: path, info: validBuildInfo()},
	}
	results := Verify(tools)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Healthy {
		t.Errorf("expected healthy, got: %s", results[0].Error)
	}
}

func TestVerify_NotExecutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "readonly")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	tools := []Tool{
		{name: "readonly", path: path, info: validBuildInfo()},
	}
	results := Verify(tools)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Healthy {
		t.Error("expected unhealthy for non-executable file")
	}
}

func TestVerify_MissingFile(t *testing.T) {
	tools := []Tool{
		{name: "missing", path: "/nonexistent/gobin/missing", info: validBuildInfo()},
	}
	results := Verify(tools)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Healthy {
		t.Error("expected unhealthy for missing file")
	}
}

func TestVerify_EmptyPackagePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "emptypkg")
	if err := os.WriteFile(path, []byte("dummy"), 0o755); err != nil {
		t.Fatal(err)
	}
	tools := []Tool{
		{name: "emptypkg", path: path, info: &buildinfo.BuildInfo{
			Path: "",
			Main: debug.Module{Path: "example.com/emptypkg", Version: "v1.0.0"},
		}},
	}
	results := Verify(tools)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Healthy {
		t.Error("expected unhealthy for empty package path")
	}
}

func TestVerify_EmptyVersionPassesVerify(t *testing.T) {
	path := filepath.Join(t.TempDir(), "noversion")
	if err := os.WriteFile(path, []byte("dummy"), 0o755); err != nil {
		t.Fatal(err)
	}
	tools := []Tool{
		{name: "noversion", path: path, info: &buildinfo.BuildInfo{
			Path: "example.com/noversion",
			Main: debug.Module{Path: "example.com/noversion", Version: ""},
		}},
	}
	results := Verify(tools)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Tool.Version() returns "unknown" for empty metadata, which is != "",
	// so Verify's `t.Version() == ""` check never triggers. This test
	// documents the existing behavior; if Version() is changed to return ""
	// for empty, this test should flip to expect unhealthy.
	if !results[0].Healthy {
		t.Error("existing behavior: empty-version tool passes verify (Version() returns 'unknown')")
	}
}
