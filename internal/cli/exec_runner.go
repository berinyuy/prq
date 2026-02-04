package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ExecRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (string, error)
}

type RealExecRunner struct{}

func (r RealExecRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %s %s: %w", name, strings.Join(args, " "), err)
	}
	return string(output), nil
}

type FakeExecRunner struct{}

func (f FakeExecRunner) Run(ctx context.Context, dir string, name string, args ...string) (string, error) {
	_ = ctx
	_ = dir
	if name == "gh" && len(args) >= 2 && args[0] == "repo" && args[1] == "clone" {
		if len(args) >= 4 {
			if err := os.MkdirAll(filepath.Clean(args[3]), 0o755); err != nil {
				return "", err
			}
		}
		return "cloned", nil
	}
	if name == "gh" && len(args) >= 2 && args[0] == "pr" && args[1] == "checkout" {
		return "checked out", nil
	}
	return "mock command output", nil
}
