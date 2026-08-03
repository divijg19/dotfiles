package tool

import (
	"fmt"
	"os"
)

type VerificationResult struct {
	Tool    Tool
	Healthy bool
	Error   string
}

func VerifyAll(gobin string) ([]VerificationResult, error) {
	candidates, err := discover(gobin)
	if err != nil {
		return nil, err
	}

	var results []VerificationResult
	for _, c := range candidates {
		res := VerificationResult{}
		t, err := inspect(c)
		if err != nil {
			res.Tool = Tool{name: c.name, path: c.path}
			res.Healthy = false
			res.Error = fmt.Sprintf("cannot read build info: %v", err)
			results = append(results, res)
			continue
		}
		res.Tool = t

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

	return results, nil
}
