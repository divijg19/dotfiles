package tool

import (
	"os"
	"path/filepath"
	"sort"
)

func discover(gobin string) ([]candidate, error) {
	entries, err := os.ReadDir(gobin)
	if err != nil {
		return nil, err
	}

	var candidates []candidate
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		toolPath := filepath.Join(gobin, entry.Name())

		info, err := entry.Info()
		if err != nil || info.Mode()&0o111 == 0 {
			continue
		}

		candidates = append(candidates, candidate{
			name: entry.Name(),
			path: toolPath,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].name < candidates[j].name
	})

	return candidates, nil
}
