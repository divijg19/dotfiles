package tool

import (
	"context"
	"debug/buildinfo"
	"errors"
	"runtime/debug"
	"testing"
)

type mockRunner struct {
	output string
	err    error
}

func (m mockRunner) Run(ctx context.Context, c Command) (string, error) {
	return m.output, m.err
}

func TestUpdateRunner(t *testing.T) {
	ctx := context.Background()
	bi := &buildinfo.BuildInfo{
		Path: "example.com/tool/cmd/tool",
		Main: debug.Module{
			Path:    "example.com/tool",
			Version: "v1.0.0",
		},
	}
	tool := Tool{
		name: "dummy",
		path: "/fake/path",
		info: bi,
	}

	// Test successful run with note
	runner := mockRunner{output: "deprecated warning\n"}
	results, _, diagnostics := Update(ctx, []Tool{tool}, nil, false, runner, nil)
	if len(results) != 1 {
		Fatalf(t, "Expected 1 result, got %d", len(results))
	}
	res := results[0]

	if !res.Success {
		t.Errorf("Expected success, got failure")
	}
	if len(res.Notes) != 1 || res.Notes[0] != "deprecated warning" {
		t.Errorf("Unexpected notes: %v", res.Notes)
	}
	if len(diagnostics) != 1 || diagnostics[0].Category != "Deprecation" {
		t.Errorf("Expected deprecation diagnostic, got %v", diagnostics)
	}

	// Test failed run
	failRunner := mockRunner{err: errors.New("network error")}
	resultsFail, _, _ := Update(ctx, []Tool{tool}, nil, false, failRunner, nil)
	if len(resultsFail) != 1 {
		Fatalf(t, "Expected 1 result, got %d", len(resultsFail))
	}
	resFail := resultsFail[0]

	if resFail.Success {
		t.Errorf("Expected failure, got success")
	}
}

func Fatalf(t *testing.T, format string, args ...any) {
	t.Fatalf(format, args...)
}
