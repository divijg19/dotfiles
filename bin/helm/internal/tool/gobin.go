package tool

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func GetGobin() (string, error) {
	out, err := exec.Command("go", "env", "GOBIN").Output()
	if err == nil {
		gobin := strings.TrimSpace(string(out))
		if gobin != "" {
			return gobin, nil
		}
	}

	out, err = exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return "", fmt.Errorf("failed to determine GOPATH: %w", err)
	}
	gopath := strings.TrimSpace(string(out))
	if gopath == "" {
		return "", fmt.Errorf("GOPATH is not set and GOBIN is empty")
	}
	return filepath.Join(gopath, "bin"), nil
}
