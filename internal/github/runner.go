package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type Runner interface {
	Run(ctx context.Context, args []string, stdin []byte) ([]byte, error)
}

type RealRunner struct{}

func (r RealRunner) Run(ctx context.Context, args []string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gh %v failed: %w\n%s", args, err, string(output))
	}
	return output, nil
}
