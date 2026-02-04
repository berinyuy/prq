package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FixtureRunner struct {
	Root string
}

func NewFixtureRunner(root string) FixtureRunner {
	return FixtureRunner{Root: root}
}

func (f FixtureRunner) Run(ctx context.Context, args []string, stdin []byte) ([]byte, error) {
	_ = ctx
	_ = stdin
	key := strings.Join(args, " ")
	var file string
	if strings.Contains(key, "search prs") {
		file = "queue.json"
	} else if strings.Contains(key, "pr view") {
		file = "pr_view.json"
	} else if strings.Contains(key, "pr diff") {
		file = "pr_diff.txt"
	} else if strings.Contains(key, "check-runs") {
		file = "check_runs.json"
	} else if strings.Contains(key, "auth status") {
		return []byte("logged in"), nil
	} else {
		return nil, fmt.Errorf("no fixture for gh args: %s", key)
	}
	path := filepath.Join(f.Root, file)
	return os.ReadFile(path)
}
