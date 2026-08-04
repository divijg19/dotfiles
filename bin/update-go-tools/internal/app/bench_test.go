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
	report := UpdateReport{
		Updated: []string{"tool1"},
		Failed:  []string{"tool2"},
		Skipped: []string{"tool3"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Update(report)
	}
}

func BenchmarkTerminalUpdateRendering(b *testing.B) {
	r := TerminalRenderer{}
	report := UpdateReport{
		Updated: []string{"tool1"},
		Failed:  []string{"tool2"},
		Skipped: []string{"tool3"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Update(report)
	}
}
