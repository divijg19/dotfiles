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

	"helm/internal/testutil"
)

var (
	binaryPath string
	testDir    string
	updateFlag = flag.Bool("update", false, "update golden files")
)

func TestMain(m *testing.M) {
	flag.Parse()
	tmp, err := os.MkdirTemp("", "helm-cli-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	testDir = tmp

	binaryPath = filepath.Join(tmp, "helm")
	build := exec.Command("go", "build", "-ldflags=-X=helm/internal/cli.version=v1.6.0-test -X=helm/internal/cli.commitHash=abc1234 -X=helm/internal/cli.buildDate=2026-08-05", "-o", binaryPath, ".")
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

var (
	goVersionRe = regexp.MustCompile(`go1\.\d+(\.\d+)?(-[A-Za-z0-9:\.]+)?`)
	durationRe  = regexp.MustCompile(`\d+\.\d+s`)
)

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
	if result.code != 1 {
		t.Errorf("exit code: expected 1 (issues found), got %d", result.code)
	}
}

func TestListCI(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--list", "--ci")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "list-ci", got, "", goldenPath("list-ci"), "")
	if result.code != 1 {
		t.Errorf("exit code: expected 1 (issues found), got %d", result.code)
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

func TestPlanCheck(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--check")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "check", got, "", goldenPath("check"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestPlanDryRunAlias(t *testing.T) {
	f := testutil.NewFixture(t)
	check := runCLI(t, f.Env(), "--check")
	dry := runCLI(t, f.Env(), "--dry-run")
	if check.stdout != dry.stdout {
		t.Errorf("--dry-run must be an alias of --check:\n--check:\n%s\n--dry-run:\n%s", check.stdout, dry.stdout)
	}
	if check.code != dry.code {
		t.Errorf("exit code mismatch: --check=%d --dry-run=%d", check.code, dry.code)
	}
}

func TestPlanVerbose(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--check", "--verbose")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "check-verbose", got, "", goldenPath("check-verbose"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestPlanVerboseShortFlag(t *testing.T) {
	f := testutil.NewFixture(t)
	long := runCLI(t, f.Env(), "--check", "--verbose")
	short := runCLI(t, f.Env(), "--check", "-V")
	gotLong := normalizeOutput(t, f.GobinDir, long.stdout)
	gotShort := normalizeOutput(t, f.GobinDir, short.stdout)
	if gotLong != gotShort {
		t.Errorf("-V must match --verbose:\n--verbose:\n%s\n-V:\n%s", gotLong, gotShort)
	}
}

func TestQuietUpdate(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "-q")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "quiet", got, "", goldenPath("quiet"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestQuietLongFlag(t *testing.T) {
	f := testutil.NewFixture(t)
	short := runCLI(t, f.Env(), "-q")
	long := runCLI(t, f.Env(), "--quiet")
	gotShort := normalizeOutput(t, f.GobinDir, short.stdout)
	gotLong := normalizeOutput(t, f.GobinDir, long.stdout)
	if gotShort != gotLong {
		t.Errorf("--quiet must match -q output:\n-q:\n%s\n--quiet:\n%s", gotShort, gotLong)
	}
}

func TestQuietListSuppressesHeader(t *testing.T) {
	f := testutil.NewFixture(t)
	quiet := runCLI(t, f.Env(), "--list", "-q")
	normal := runCLI(t, f.Env(), "--list")
	if strings.Contains(quiet.stdout, "Discovery") || strings.Contains(quiet.stdout, "Go:") {
		t.Errorf("quiet --list must suppress the discovery header:\n%s", quiet.stdout)
	}
	if !strings.Contains(quiet.stdout, "NAME") || !strings.Contains(quiet.stdout, "Summary") {
		t.Errorf("quiet --list must still emit the table and summary:\n%s", quiet.stdout)
	}
	if !strings.Contains(normal.stdout, "Discovery") {
		t.Errorf("non-quiet --list must include the discovery header (sanity):\n%s", normal.stdout)
	}
}

func TestQuietOutdatedSuppressesHeader(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--outdated", "-q")
	if strings.Contains(result.stdout, "Discovery") || strings.Contains(result.stdout, "Go:") {
		t.Errorf("quiet --outdated must suppress the discovery header:\n%s", result.stdout)
	}
	if !strings.Contains(result.stdout, "NAME") || !strings.Contains(result.stdout, "Summary") {
		t.Errorf("quiet --outdated must still emit the table and summary:\n%s", result.stdout)
	}
}

func TestUpdateCI(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--ci")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "update-ci", got, "", goldenPath("update-ci"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
}

func TestListJSON(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--list", "--json")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "list-json", got, "", jsonGoldenPath("list"), "")
	if result.code != 1 {
		t.Errorf("exit code: expected 1 (issues found), got %d", result.code)
	}
}

func TestPlanJSON(t *testing.T) {
	f := testutil.NewFixture(t)
	check := runCLI(t, f.Env(), "--check", "--json")
	dry := runCLI(t, f.Env(), "--dry-run", "--json")
	if check.stdout != dry.stdout {
		t.Errorf("--dry-run --json must match --check --json:\n--check:\n%s\n--dry-run:\n%s", check.stdout, dry.stdout)
	}
	got := normalizeOutput(t, f.GobinDir, check.stdout)
	checkGolden(t, f.GobinDir, "plan-json", got, "", jsonGoldenPath("plan"), "")
	if check.code != 0 {
		t.Errorf("exit code: expected 0, got %d", check.code)
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

func TestInfoJSON(t *testing.T) {
	f := testutil.NewFixture(t)
	result := runCLI(t, f.Env(), "--info", "hello", "--json")
	got := normalizeOutput(t, f.GobinDir, result.stdout)
	checkGolden(t, f.GobinDir, "info-json", got, "", jsonGoldenPath("info"), "")
	if result.code != 0 {
		t.Errorf("exit code: expected 0, got %d", result.code)
	}
	if strings.Contains(got, `"operation"`) || strings.Contains(got, `"success"`) {
		t.Errorf("--info --json must stay a bare ToolReport without the operation envelope:\n%s", got)
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
