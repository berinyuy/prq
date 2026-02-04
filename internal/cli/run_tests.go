package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/brianndofor/prq/internal/github"
)

func runTestsForPR(ctx context.Context, app *App, view github.PRView) (string, error) {
	commands := make([]string, 0, len(app.RepoConfig.Tests.Commands))
	for _, cmd := range app.RepoConfig.Tests.Commands {
		if strings.TrimSpace(cmd) != "" {
			commands = append(commands, cmd)
		}
	}
	if len(commands) == 0 {
		return "No test commands configured in prq.yaml.", nil
	}

	workDir, err := os.MkdirTemp("", "prq-tests-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(workDir)
	}()

	repoDir := filepath.Join(workDir, "repo")
	if _, err := app.Exec.Run(ctx, "", "gh", "repo", "clone", view.Repository.NameWithOwner, repoDir); err != nil {
		return "", fmt.Errorf("failed to clone repo: %w", err)
	}
	if _, err := app.Exec.Run(ctx, repoDir, "gh", "pr", "checkout", strconv.Itoa(view.Number)); err != nil {
		return "", fmt.Errorf("failed to checkout PR: %w", err)
	}

	var output strings.Builder
	for _, cmd := range commands {
		fmt.Fprintf(&output, "$ %s\n", cmd)
		cmdOutput, err := app.Exec.Run(ctx, repoDir, "sh", "-lc", cmd)
		if cmdOutput != "" {
			output.WriteString(cmdOutput)
			if !strings.HasSuffix(cmdOutput, "\n") {
				output.WriteString("\n")
			}
		}
		if err != nil {
			fmt.Fprintf(&output, "Command failed: %v\n", err)
		}
	}

	return strings.TrimSpace(output.String()), nil
}
