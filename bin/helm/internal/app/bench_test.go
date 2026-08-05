package app

import (
	"os"
	"testing"
)

// silenceStdout redirects os.Stdout so renderer benchmarks measure pure
// rendering cost without polluting benchmark output.
func silenceStdout() func() {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		panic(err)
	}
	old := os.Stdout
	os.Stdout = devNull
	return func() {
		os.Stdout = old
		devNull.Close()
	}
}

func BenchmarkJSONUpdateRendering(b *testing.B) {
	r := JSONRenderer{}
	report := UpdateReport{
		Updated: []string{"tool1"},
		Failed:  []string{"tool2"},
		Skipped: []string{"tool3"},
	}
	restore := silenceStdout()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Update(report)
	}
	restore()
}

func BenchmarkTerminalUpdateRendering(b *testing.B) {
	r := TerminalRenderer{}
	report := UpdateReport{
		Updated: []string{"tool1"},
		Failed:  []string{"tool2"},
		Skipped: []string{"tool3"},
	}
	restore := silenceStdout()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Update(report)
	}
	restore()
}
