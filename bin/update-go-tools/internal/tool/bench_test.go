package tool

import (
	"context"
	"debug/buildinfo"
	"runtime/debug"
	"testing"
)

func makeBenchTool(name, pkg, version string) Tool {
	return NewTool(name, "/gobin/"+name, &buildinfo.BuildInfo{
		Path: pkg + "/cmd/" + name,
		Main: debug.Module{Path: pkg, Version: version},
	})
}

func BenchmarkLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Load("")
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

func BenchmarkUpdatePlanning(b *testing.B) {
	tools := []Tool{
		makeBenchTool("tool1", "example.com/tool1", "v1.0.0"),
		makeBenchTool("tool2", "example.com/tool2", "v2.0.0"),
		makeBenchTool("tool3", "example.com/tool3", "(devel)"),
	}
	filter := []string{"tool1", "tool2"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var filtered []Tool
		for _, t := range tools {
			for _, f := range filter {
				if t.Name() == f {
					filtered = append(filtered, t)
					break
				}
			}
		}
		_ = filtered
	}
}

func BenchmarkCheckOutdated(b *testing.B) {
	ctx := context.Background()
	tools := []Tool{
		makeBenchTool("tool1", "example.com/tool1", "v1.0.0"),
		makeBenchTool("tool2", "example.com/tool2", "v2.0.0"),
	}
	runner := DefaultRunner{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckOutdated(ctx, tools, runner)
	}
}
