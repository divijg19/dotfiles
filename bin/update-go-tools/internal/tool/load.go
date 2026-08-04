package tool

import (
	"errors"
	"fmt"
)

type InvalidBinary struct {
	Path  string
	Error error
}

// Message returns a stable, user-visible reason for why the binary could not
// be inspected. The wrapped error in Error may carry toolchain-specific
// detail; Message is what renderers should present to users.
func (inv InvalidBinary) Message() string {
	switch {
	case errors.Is(inv.Error, ErrMissingBuildInfo):
		return ErrMissingBuildInfo.Error()
	case errors.Is(inv.Error, ErrMissingPackagePath):
		return ErrMissingPackagePath.Error()
	case errors.Is(inv.Error, ErrInvalidMetadata):
		return ErrInvalidMetadata.Error()
	default:
		return "unable to inspect binary"
	}
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
