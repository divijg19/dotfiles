package tool

import (
	"context"
	"debug/buildinfo"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"testing"
)

func makeBenchTool(name, pkg, version string) Tool {
	return NewTool(name, "/gobin/"+name, &buildinfo.BuildInfo{
		Path: pkg + "/cmd/" + name,
		Main: debug.Module{Path: pkg, Version: version},
	})
}

// benchGobin builds one real Go binary into a temp directory so Load exercises
// the actual discovery + buildinfo inspection path against a real filesystem.
func benchGobin(b *testing.B) string {
	b.Helper()
	dir := b.TempDir()
	src := filepath.Join("..", "..", "testdata", "fixtures", "binaries", "hello")
	cmd := exec.Command("go", "build", "-o", filepath.Join(dir, "hello"), ".")
	cmd.Dir = src
	if out, err := cmd.CombinedOutput(); err != nil {
		b.Fatalf("go build fixture failed: %v\n%s", err, out)
	}
	return dir
}

func BenchmarkLoad(b *testing.B) {
	dir := benchGobin(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Load(dir)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	tools := []Tool{
		makeBenchTool("tool1", "example.com/tool1", "v1.0.0"),
		makeBenchTool("tool2", "example.com/tool2", "v2.0.0"),
		makeBenchTool("tool3", "example.com/tool3", "v3.0.0"),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Verify(tools)
	}
}

func BenchmarkPlan(b *testing.B) {
	tools := []Tool{
		makeBenchTool("tool1", "example.com/tool1", "v1.0.0"),
		makeBenchTool("tool2", "example.com/tool2", "v2.0.0"),
		makeBenchTool("tool3", "example.com/tool3", "(devel)"),
	}
	loadRes := LoadResult{Tools: tools}
	filter := []string{"tool1", "tool2"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := Plan(loadRes, filter); len(got.ToUpdate) != 2 {
			b.Fatalf("expected 2 to update, got %d", len(got.ToUpdate))
		}
	}
}

func BenchmarkCheckOutdated(b *testing.B) {
	ctx := context.Background()
	tools := []Tool{
		makeBenchTool("tool1", "example.com/tool1", "v1.0.0"),
		makeBenchTool("tool2", "example.com/tool2", "v2.0.0"),
	}
	runner := mockRunner{output: `{"Path":"example.com/tool1","Version":"v1.1.0"}`}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if got := CheckOutdated(ctx, tools, runner); len(got) != 2 {
			b.Fatalf("expected 2 results, got %d", len(got))
		}
	}
}
