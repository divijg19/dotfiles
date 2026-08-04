package testutil_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"update-go-tools/internal/testutil"
)

func TestFixture(t *testing.T) {
	f := testutil.NewFixture(t)
	for _, name := range []string{"hello", "world", "localdev", "notgo"} {
		if _, err := os.Stat(f.Gobin(name)); os.IsNotExist(err) {
			t.Errorf("missing fixture binary %s", name)
		}
	}
	cmd := exec.Command("go", "version", "-m", f.Gobin("hello"))
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "mod\texample.com/hello") {
		t.Errorf("hello: expected module path, got:\n%s", out)
	}
	cmd = exec.Command("go", "version", "-m", f.Gobin("localdev"))
	out, _ = cmd.CombinedOutput()
	if !strings.Contains(string(out), "(devel)") {
		t.Errorf("localdev: expected (devel), got:\n%s", out)
	}
}
