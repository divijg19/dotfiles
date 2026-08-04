// Package testutil provides helpers for building hermetic, offline test
// environments: a temporary GOBIN populated with real Go binaries whose build
// metadata is sourced from a file-based module proxy.
package testutil

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"golang.org/x/mod/module"
	"golang.org/x/mod/zip"
)

// Fixture is a hermetic environment backed by a temporary GOBIN and a
// file-based module proxy.
type Fixture struct {
	GobinDir  string
	ProxyDir  string
	moduleSrc string
}

// NewFixture creates the environment and installs the fixture binaries:
//
//	hello   v1.0.0 installed; v1.0.0 latest on proxy  (up to date)
//	world   v1.2.0 installed; v1.3.0 latest on proxy  (outdated)
//	localdev                                      (devel)  (not updatable)
//	notgo   executable shell script                (invalid)
//
// Binaries named after the last path element of the module path (hello, world)
// and a (devel) binary built from a local module directory.
func NewFixture(t *testing.T) *Fixture {
	t.Helper()

	tmp := t.TempDir()
	f := &Fixture{
		GobinDir:  filepath.Join(tmp, "gobin"),
		ProxyDir:  filepath.Join(tmp, "proxy"),
		moduleSrc: moduleRoot(t),
	}
	for _, dir := range []string{f.GobinDir, f.ProxyDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Install hello@v1.0.0 (up to date) and world@v1.2.0 (outdated).
	f.publish(t, "example.com/hello", "v1.0.0")
	f.publish(t, "example.com/world", "v1.2.0")
	f.publish(t, "example.com/world", "v1.3.0")
	f.install(t, "example.com/hello", "v1.0.0")
	f.install(t, "example.com/world", "v1.2.0")

	// Local (devel) binary built from a source module directory.
	f.buildDevel(t, "localdev")

	// Invalid executable (not a Go binary).
	f.writeInvalid(t)

	return f
}

// Env returns the environment variables the tool subprocesses must inherit to
// resolve GOBIN and reach the file proxy.
func (f *Fixture) Env() []string {
	return []string{
		"GOBIN=" + f.GobinDir,
		"GOPROXY=file://" + f.ProxyDir,
		"GOSUMDB=off",
		"GONOSUMDB=*",
	}
}

// Gobin returns the full path to a named binary in the fixture GOBIN.
func (f *Fixture) Gobin(name string) string {
	return filepath.Join(f.GobinDir, name)
}

// Command returns an exec.Cmd configured with the fixture environment.
func (f *Fixture) Command(t *testing.T, name string, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), f.Env()...)
	return cmd
}

// moduleRoot returns the update-go-tools module root.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root from " + dir)
		}
		dir = parent
	}
}

// publish writes the module zip and metadata into the file proxy layout.
func (f *Fixture) publish(t *testing.T, modPath, ver string) {
	t.Helper()
	src := filepath.Join(f.moduleSrc, "testdata", "fixtures", "binaries", filepath.Base(modPath))

	proxyMod := filepath.Join(f.ProxyDir, modPath, "@v")
	if err := os.MkdirAll(proxyMod, 0o755); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(proxyMod, ver+".zip")
	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := zip.CreateFromDir(zf, module.Version{Path: modPath, Version: ver}, src); err != nil {
		t.Fatal(err)
	}
	if err := zf.Close(); err != nil {
		t.Fatal(err)
	}

	info, _ := json.Marshal(struct {
		Version string `json:"Version"`
	}{Version: ver})
	if err := os.WriteFile(filepath.Join(proxyMod, ver+".info"), info, 0o644); err != nil {
		t.Fatal(err)
	}

	mod, err := os.ReadFile(filepath.Join(src, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proxyMod, ver+".mod"), mod, 0o644); err != nil {
		t.Fatal(err)
	}

	listPath := filepath.Join(proxyMod, "list")
	list, _ := os.ReadFile(listPath)
	list = append(list, ver+"\n"...)
	if err := os.WriteFile(listPath, list, 0o644); err != nil {
		t.Fatal(err)
	}
}

// install runs `go install mod@ver` against the file proxy into the GOBIN.
func (f *Fixture) install(t *testing.T, modPath, ver string) {
	t.Helper()
	scratch := t.TempDir()
	cmd := f.Command(t, "go", "install", modPath+"@"+ver)
	cmd.Dir = scratch
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go install %s@%s failed: %v\n%s", modPath, ver, err, out)
	}
}

// buildDevel builds a binary from a local module directory, producing
// Main.Version == "(devel)" so it is reported as not updatable.
func (f *Fixture) buildDevel(t *testing.T, name string) {
	t.Helper()
	src := filepath.Join(f.moduleSrc, "testdata", "fixtures", "binaries", name)
	out := filepath.Join(f.GobinDir, name)
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", out, ".")
	cmd.Dir = src
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build %s failed: %v\n%s", name, err, out)
	}
}

// writeInvalid places an executable shell script in GOBIN that carries no Go
// build metadata, so it is discovered but reported invalid.
func (f *Fixture) writeInvalid(t *testing.T) {
	t.Helper()
	src := filepath.Join(f.moduleSrc, "testdata", "fixtures", "invalid", "notgo")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(f.GobinDir, "notgo")
	if err := os.WriteFile(out, data, 0o755); err != nil {
		t.Fatal(err)
	}
}
