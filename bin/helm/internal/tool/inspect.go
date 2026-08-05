package tool

import (
	"debug/buildinfo"
	"fmt"
)

func inspect(c candidate) (Tool, error) {
	bi, err := buildinfo.ReadFile(c.path)
	if err != nil {
		return Tool{}, fmt.Errorf("failed to read buildinfo for %s: %w", c.name, err)
	}

	return NewTool(c.name, c.path, bi), nil
}
