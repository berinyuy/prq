package cli

import (
	"bytes"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/brianndofor/prq/internal/store"
)

func runRoot(t *testing.T, args ...string) string {
	t.Helper()
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	return buf.String()
}

func withMockEnv(t *testing.T) func() {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	_ = os.Setenv("PRQ_MOCK", "1")
	_ = os.Setenv("PRQ_MOCK_DIR", filepath.Join(root, "testdata", "gh"))
	_ = os.Setenv("PRQ_PROVIDER_FIXTURE", filepath.Join(root, "testdata", "provider", "review.json"))
	_ = os.Setenv("PRQ_DB_PATH", filepath.Join(t.TempDir(), "prq.db"))
	_ = os.Setenv("PRQ_PROMPT_PATH", filepath.Join(root, "prompts", "code-reviewer.txt"))
	_ = os.Setenv("PRQ_SCHEMA_PATH", filepath.Join(root, "schemas", "review_plan.schema.json"))
	return func() {
		_ = os.Unsetenv("PRQ_MOCK")
		_ = os.Unsetenv("PRQ_MOCK_DIR")
		_ = os.Unsetenv("PRQ_PROVIDER_FIXTURE")
		_ = os.Unsetenv("PRQ_DB_PATH")
		_ = os.Unsetenv("PRQ_PROMPT_PATH")
		_ = os.Unsetenv("PRQ_SCHEMA_PATH")
	}
}

func TestQueueCommand(t *testing.T) {
	cleanup := withMockEnv(t)
	defer cleanup()
	output := runRoot(t, "queue")
	if output == "" {
		t.Fatalf("expected output")
	}
}

func TestReviewCommand(t *testing.T) {
	cleanup := withMockEnv(t)
	defer cleanup()
	output := runRoot(t, "review", "acme/app#42", "--format", "json")
	if output == "" {
		t.Fatalf("expected output")
	}
}

func TestDraftAndSubmitWorkflow(t *testing.T) {
	cleanup := withMockEnv(t)
	defer cleanup()
	dbPath := os.Getenv("PRQ_DB_PATH")

	output := runRoot(t, "draft", "acme/app#42")
	if !strings.Contains(output, "Saved locally") {
		t.Fatalf("expected draft output to mention saving")
	}
	output = runRoot(t, "submit", "acme/app#42", "--yes")
	if !strings.Contains(output, "Submitted review") {
		t.Fatalf("expected submit output to include submission")
	}

	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_, err = st.GetDraftReview("acme/app#42")
	if err != sql.ErrNoRows {
		t.Fatalf("expected draft to be deleted after submit, got: %v", err)
	}
	pr, err := st.GetPR("acme/app#42")
	if err != nil {
		t.Fatalf("get pr: %v", err)
	}
	if !pr.LastSubmittedAt.Valid {
		t.Fatalf("expected last_submitted_at to be set")
	}
}

func TestFollowupCommand(t *testing.T) {
	cleanup := withMockEnv(t)
	defer cleanup()

	dbPath := os.Getenv("PRQ_DB_PATH")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := st.UpsertPR("acme/app#42", "acme/app", 42, "head5678"); err != nil {
		t.Fatalf("upsert pr: %v", err)
	}
	if err := st.MarkReviewed("acme/app#42", "old1234"); err != nil {
		t.Fatalf("mark reviewed: %v", err)
	}

	output := runRoot(t, "followup", "acme/app#42")
	if !strings.Contains(output, "Open review threads") {
		t.Fatalf("expected followup output to list review threads")
	}
	if !strings.Contains(output, "Changes since last review") {
		t.Fatalf("expected followup output to include changes")
	}
}

func TestPickCommandQuit(t *testing.T) {
	t.Skip("TUI picker requires an interactive terminal")

	cleanup := withMockEnv(t)
	defer cleanup()

	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetIn(strings.NewReader("\nq\n"))
	cmd.SetArgs([]string{"pick"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("pick command failed: %v", err)
	}
}
