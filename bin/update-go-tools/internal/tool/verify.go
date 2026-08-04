package tool

import (
	"os"
)

type VerificationResult struct {
	Tool    Tool
	Healthy bool
	Error   string
}

func Verify(tools []Tool) []VerificationResult {
	var results []VerificationResult
	for _, t := range tools {
		res := VerificationResult{Tool: t}

		info, statErr := os.Stat(t.Path())
		if statErr != nil || info.Mode()&0111 == 0 {
			res.Healthy = false
			res.Error = "file is not accessible or not executable"
			results = append(results, res)
			continue
		}

		if t.PackagePath() == "" {
			res.Healthy = false
			res.Error = "missing main package path"
			results = append(results, res)
			continue
		}

		if t.Version() == "" {
			res.Healthy = false
			res.Error = "missing version metadata"
			results = append(results, res)
			continue
		}

		res.Healthy = true
		results = append(results, res)
	}

	return results
}
