package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func goldenPath(t *testing.T, name string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return filepath.Join(root, "testdata", "golden", name)
}

func readGolden(t *testing.T, name string) string {
	path := goldenPath(t, name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}
	return string(data)
}

func TestQueueGolden(t *testing.T) {
	cleanup := withMockEnv(t)
	defer cleanup()
	_ = os.Setenv("PRQ_NOW", "2026-02-04T00:00:00Z")
	defer func() { _ = os.Unsetenv("PRQ_NOW") }()

	output := runRoot(t, "queue", "--checks", "success")
	expected := readGolden(t, "queue.txt")
	if output != expected {
		t.Fatalf("queue output mismatch\n--- expected\n%s\n--- got\n%s", expected, output)
	}
}

func TestReviewGolden(t *testing.T) {
	cleanup := withMockEnv(t)
	defer cleanup()
	output := runRoot(t, "review", "acme/app#42")
	expected := readGolden(t, "review.txt")
	if output != expected {
		t.Fatalf("review output mismatch\n--- expected\n%s\n--- got\n%s", expected, output)
	}
}
