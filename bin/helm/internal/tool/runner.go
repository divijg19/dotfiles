package tool

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"os/exec"
)

type Command struct {
	Name   string
	Args   []string
	Dir    string
	Env    []string
	OnLine func(string)
}

type Runner interface {
	Run(ctx context.Context, cmd Command) (string, error)
}

type DefaultRunner struct{}

func (DefaultRunner) Run(ctx context.Context, c Command) (string, error) {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Dir = c.Dir
	if len(c.Env) > 0 {
		cmd.Env = append(cmd.Environ(), c.Env...)
	}

	if c.OnLine == nil {
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		err := cmd.Run()
		return buf.String(), err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}

	var fullOutput bytes.Buffer
	r := io.MultiReader(stdoutPipe, stderrPipe)

	if err := cmd.Start(); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		fullOutput.WriteString(line + "\n")
		c.OnLine(line)
	}

	err = cmd.Wait()
	return fullOutput.String(), err
}
