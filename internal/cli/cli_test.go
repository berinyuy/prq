package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
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
