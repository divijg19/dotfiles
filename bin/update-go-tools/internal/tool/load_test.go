package tool

import (
	"testing"
	"update-go-tools/internal/testutil"
)

func TestLoad_RealGOBIN(t *testing.T) {
	f := testutil.NewFixture(t)
	t.Setenv("GOBIN", f.GobinDir)

	loadRes, err := Load(f.GobinDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loadRes.Tools) != 3 {
		names := make([]string, len(loadRes.Tools))
		for i, tool := range loadRes.Tools {
			names[i] = tool.Name()
		}
		t.Errorf("expected 3 valid tools (hello, world, localdev), got %d: %v", len(loadRes.Tools), names)
	}

	if len(loadRes.Invalid) != 1 {
		invalids := make([]string, len(loadRes.Invalid))
		for i, inv := range loadRes.Invalid {
			invalids[i] = inv.Path
		}
		t.Errorf("expected 1 invalid (notgo), got %d: %v", len(loadRes.Invalid), invalids)
	}
}

func TestLoad_SortedAlphabetically(t *testing.T) {
	f := testutil.NewFixture(t)

	loadRes, err := Load(f.GobinDir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	for i := 1; i < len(loadRes.Tools); i++ {
		if loadRes.Tools[i-1].Name() > loadRes.Tools[i].Name() {
			t.Errorf("tools not sorted: %s > %s", loadRes.Tools[i-1].Name(), loadRes.Tools[i].Name())
		}
	}
}
