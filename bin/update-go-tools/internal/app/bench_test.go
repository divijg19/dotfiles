package app

import (
	"debug/buildinfo"
	"runtime/debug"
	"testing"

	"update-go-tools/internal/tool"
)

func makeBenchTool(name, pkg, version string) tool.Tool {
	return tool.NewTool(name, "/gobin/"+name, &buildinfo.BuildInfo{
		Path: pkg + "/cmd/" + name,
		Main: debug.Module{Path: pkg, Version: version},
	})
}

func BenchmarkJSONUpdateRendering(b *testing.B) {
	r := JSONRenderer{}
	results := []tool.ToolUpdateResult{
		{Tool: makeBenchTool("tool1", "example.com/tool1", "v1.0.0"), Status: tool.StatusUpdated, Success: true},
		{Tool: makeBenchTool("tool2", "example.com/tool2", "v2.0.0"), Status: tool.StatusFailed, Success: false},
	}
	loadRes := tool.LoadResult{
		Tools: []tool.Tool{
			makeBenchTool("tool1", "example.com/tool1", "v1.0.0"),
			makeBenchTool("tool2", "example.com/tool2", "v2.0.0"),
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Update(results, loadRes, 0, nil, false)
	}
}

func BenchmarkTerminalUpdateRendering(b *testing.B) {
	r := TerminalRenderer{}
	results := []tool.ToolUpdateResult{
		{Tool: makeBenchTool("tool1", "example.com/tool1", "v1.0.0"), Status: tool.StatusUpdated, Success: true},
		{Tool: makeBenchTool("tool2", "example.com/tool2", "v2.0.0"), Status: tool.StatusFailed, Success: false},
	}
	loadRes := tool.LoadResult{
		Tools: []tool.Tool{
			makeBenchTool("tool1", "example.com/tool1", "v1.0.0"),
			makeBenchTool("tool2", "example.com/tool2", "v2.0.0"),
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Update(results, loadRes, 0, nil, false)
	}
}
