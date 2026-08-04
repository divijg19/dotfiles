package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"update-go-tools/internal/testutil"
)

var (
	binaryPath string
	testDir    string
	updateFlag = flag.Bool("update", false, "update golden files")
)

func TestMain(m *testing.M) {
	flag.Parse()
	tmp, err := os.MkdirTemp("", "update-go-tools-cli-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	testDir = tmp

	binaryPath = filepath.Join(tmp, "update-go-tools")
	build := exec.Command("go", "build", "-ldflags=-X=main.version=v1.2.0-test -X=main.commitHash=abc1234 -X=main.buildDate=2026-08-04", "-o", binaryPath, ".")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("go build failed: " + err.Error())
	}

	os.Exit(m.Run())
}

type cliResult struct {
	stdout string
	stderr string
	code   int
}

func runCLI(t *testing.T, fixtureEnv []string, args ...string) cliResult {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = append(os.Environ(), fixtureEnv...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	code := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run CLI: %v", err)
		}
	}

	return cliResult{
		stdout: stdout.String(),
		stderr: stderr.String(),
		code:   code,
	}
}

var goVersionRe = regexp.MustCompile(`go1\.\d+(\.\d+)?(-[A-Za-z0-9:\.]+)?`)
var durationRe = regexp.MustCompile(`\d+\.\d+s`)

func normalizeOutput(t *testing.T, gobinDir, s string) string {
	t.Helper()
	s = goVersionRe.ReplaceAllString(s, "goVERSION")
	s = durationRe.ReplaceAllString(s, "0.0s")
	parent := filepath.Dir(gobinDir)
	escaped := regexp.QuoteMeta(parent)
	s = regexp.MustCompile(escaped).ReplaceAllString(s, "<TMP>")
	return strings.TrimSpace(s)
}

func checkGolden(t *testing.T, gobinDir, name string, got, gotErr string, goldenPath, goldenErrPath string) {
	t.Helper()
	if *updateFlag {
		writeGolden(t, goldenPath, got)
		if goldenErrPath != "" {
			writeGolden(t, goldenErrPath, gotErr)
		}
		return
	}
	golden := readGolden(t, goldenPath)
	if got != golden {
		t.Errorf("stdout mismatch:\ngot:\n%s\nwant:\n%s", got, golden)
	}
	if goldenErrPath != "" {
		goldenErr := readGolden(t, goldenErrPath)
		if gotErr != goldenErr {
			t.Errorf("stderr mismatch:\ngot:\n%s\nwant:\n%s", gotErr, goldenErr)
		}
	}
}

func goldenPath(name string) string {
	return filepath.Join("..", "..", "testdata", "golden", name+".txt")
}

func jsonGoldenPath(name string) string {
	return filepath.Join("..", "..", "testdata", "json", name+".json")
}

func readGolden(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read golden %s: %v", path, err)
	}
	return strings.TrimSpace(string(data))
}

func writeGolden(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestHelp(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--help")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "help", got, "", goldenPath("help"), "")
}

func TestVersion(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--version")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "version", got, "", goldenPath("version"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestList(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--list")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "list", got, "", goldenPath("list"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestVerify(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--verify")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	gotErr := normalizeOutput(t, f.GobinDir, result.stderr)
	checkGolden(t, f.GobinDir, "verify", got, gotErr, goldenPath("verify"), goldenPath("verify-stderr"))
	if result.code != 1 {
		t.Errorf("exit code: expected 1 (unhealthy), got %d", result.code)
	}
}

func TestOutdated(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--outdated")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "outdated", got, "", goldenPath("outdated"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestInfoTool(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--info", "hello")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "info", got, "", goldenPath("info"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestDefaultUpdate(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env())
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "update", got, "", goldenPath("update"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestCheckDefaultUpdate(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--check")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "check", got, "", goldenPath("check"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestListJSON(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--list", "--json")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "list-json", got, "", jsonGoldenPath("list"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestVerifyJSON(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--verify", "--json")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "verify-json", got, "", jsonGoldenPath("verify"), "")
	if result.code != 1 {
		t.Errorf("exit code: expected 1 (unhealthy), got %d", result.code)
	}
}

func TestOutdatedJSON(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--outdated", "--json")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "outdated-json", got, "", jsonGoldenPath("outdated"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestUpdateJSON(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--json")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "update-json", got, "", jsonGoldenPath("update"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestUnknownOption(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--unknown")
	if result.code != 2 {
		t.Errorf("exit code: expected 2, got %d", result.code)
	}
}

func TestInfoMissing(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--info", "nonexistent")
	if result.code != 2 {
		t.Errorf("exit code: expected 2, got %d", result.code)
	}
}
