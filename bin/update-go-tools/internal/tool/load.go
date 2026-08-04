package tool

import (
	"fmt"
)

type InvalidBinary struct {
	Path  string
	Error error
}

type LoadResult struct {
	Tools   []Tool
	Invalid []InvalidBinary
}

func Load(gobin string) (LoadResult, error) {
	candidates, err := discover(gobin)
	if err != nil {
		return LoadResult{}, err
	}

	var tools []Tool
	var invalids []InvalidBinary

	for _, c := range candidates {
		t, err := inspect(c)
		if err != nil {
			invalids = append(invalids, InvalidBinary{
				Path:  c.path,
				Error: fmt.Errorf("%w: %v", ErrMissingBuildInfo, err),
			})
			continue
		}
		if !t.IsValid() {
			invalids = append(invalids, InvalidBinary{
				Path:  c.path,
				Error: ErrMissingPackagePath,
			})
			continue
		}
		tools = append(tools, t)
	}

	return LoadResult{
		Tools:   tools,
		Invalid: invalids,
	}, nil
}
