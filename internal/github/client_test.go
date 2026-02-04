package github

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

type fakeRunner struct {
	Output []byte
}

func (f fakeRunner) Run(ctx context.Context, args []string, stdin []byte) ([]byte, error) {
	return f.Output, nil
}

func TestSearchPRs(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "gh", "queue.json"))
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	client := NewClient(fakeRunner{Output: data})
	items, err := client.SearchPRs(context.Background(), "", 10, "created", "asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}
