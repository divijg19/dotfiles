package tool

import (
	"debug/buildinfo"
	"runtime/debug"
	"testing"
)

func TestToolCanUpdate(t *testing.T) {
	tests := []struct {
		name string
		bi   *buildinfo.BuildInfo
		want bool
	}{
		{
			name: "valid remote tool",
			bi: &buildinfo.BuildInfo{
				Path: "example.com/tool/cmd/tool",
				Main: debug.Module{
					Path:    "example.com/tool",
					Version: "v1.0.0",
				},
			},
			want: true,
		},
		{
			name: "local devel tool",
			bi: &buildinfo.BuildInfo{
				Path: "example.com/tool",
				Main: debug.Module{
					Path:    "example.com/tool",
					Version: "(devel)",
				},
			},
			want: false,
		},
		{
			name: "missing package path",
			bi: &buildinfo.BuildInfo{
				Path: "",
				Main: debug.Module{
					Path:    "example.com/tool",
					Version: "v1.0.0",
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := Tool{
				name: "test",
				path: "/fake/path",
				info: tt.bi,
			}
			if got := tool.CanUpdate(); got != tt.want {
				t.Errorf("Tool.CanUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToolInstallTarget(t *testing.T) {
	bi := &buildinfo.BuildInfo{
		Path: "example.com/tool/cmd/tool",
		Main: debug.Module{
			Path:    "example.com/tool",
			Version: "v1.0.0",
		},
	}
	tool := Tool{
		name: "test",
		path: "/fake/path",
		info: bi,
	}
	if got := tool.InstallTarget(); got != "example.com/tool/cmd/tool" {
		t.Errorf("Tool.InstallTarget() = %v, want %v", got, "example.com/tool/cmd/tool")
	}
}
