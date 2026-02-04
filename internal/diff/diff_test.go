package diff

import "testing"

const sampleDiff = "diff --git a/file.txt b/file.txt\nindex 123..456 100644\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,2 @@\n-hello\n+hello world\n"

func TestParseUnified(t *testing.T) {
	files, err := ParseUnified(sampleDiff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "file.txt" {
		t.Fatalf("unexpected path: %s", files[0].Path)
	}
}

func TestBuildChunks(t *testing.T) {
	files, _ := ParseUnified(sampleDiff)
	chunks, err := BuildChunks(files, nil, 10, 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
}
