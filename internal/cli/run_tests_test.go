package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/brianndofor/prq/internal/config"
	"github.com/brianndofor/prq/internal/github"
)

func TestRunTestsNoCommands(t *testing.T) {
	app := &App{
		RepoConfig: config.RepoConfig{Tests: config.TestsConfig{Commands: []string{}}},
		Exec:       FakeExecRunner{},
	}
	view := github.PRView{Repository: github.RepoRef{NameWithOwner: "acme/app"}, Number: 42}
	output, err := runTestsForPR(context.Background(), app, view)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "No test commands configured in prq.yaml." {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRunTestsExecutesCommands(t *testing.T) {
	app := &App{
		RepoConfig: config.RepoConfig{Tests: config.TestsConfig{Commands: []string{"echo ok"}}},
		Exec:       FakeExecRunner{},
	}
	view := github.PRView{Repository: github.RepoRef{NameWithOwner: "acme/app"}, Number: 42}
	output, err := runTestsForPR(context.Background(), app, view)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "$ echo ok") {
		t.Fatalf("expected output to include command, got: %q", output)
	}
	if !strings.Contains(output, "mock command output") {
		t.Fatalf("expected mock output, got: %q", output)
	}
}
