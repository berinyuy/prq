package diff

import "testing"

func TestBuildPositionMap_SingleHunk(t *testing.T) {
	input := "" +
		"diff --git a/a.txt b/a.txt\n" +
		"index 111..222 100644\n" +
		"--- a/a.txt\n" +
		"+++ b/a.txt\n" +
		"@@ -1,3 +1,4 @@\n" +
		" line1\n" +
		"-line2\n" +
		"+line2b\n" +
		"+line3\n" +
		" line4\n"

	files, err := ParseUnified(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pm, err := BuildPositionMap(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pos, ok := pm.PositionForNewLine("a.txt", 1)
	if !ok || pos != 2 {
		t.Fatalf("expected new line 1 => pos 2, got %d (ok=%v)", pos, ok)
	}
	pos, ok = pm.PositionForNewLine("a.txt", 2)
	if !ok || pos != 4 {
		t.Fatalf("expected new line 2 => pos 4, got %d (ok=%v)", pos, ok)
	}
	pos, ok = pm.PositionForNewLine("a.txt", 3)
	if !ok || pos != 5 {
		t.Fatalf("expected new line 3 => pos 5, got %d (ok=%v)", pos, ok)
	}
	pos, ok = pm.PositionForNewLine("a.txt", 4)
	if !ok || pos != 6 {
		t.Fatalf("expected new line 4 => pos 6, got %d (ok=%v)", pos, ok)
	}
}

func TestBuildPositionMap_MultiHunk(t *testing.T) {
	input := "" +
		"diff --git a/a.txt b/a.txt\n" +
		"index 111..222 100644\n" +
		"--- a/a.txt\n" +
		"+++ b/a.txt\n" +
		"@@ -1,2 +1,2 @@\n" +
		" line1\n" +
		" line2\n" +
		"@@ -10,0 +11,2 @@\n" +
		"+new11\n" +
		"+new12\n"

	files, err := ParseUnified(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pm, err := BuildPositionMap(files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pos, ok := pm.PositionForNewLine("a.txt", 1)
	if !ok || pos != 2 {
		t.Fatalf("expected new line 1 => pos 2, got %d (ok=%v)", pos, ok)
	}
	pos, ok = pm.PositionForNewLine("a.txt", 2)
	if !ok || pos != 3 {
		t.Fatalf("expected new line 2 => pos 3, got %d (ok=%v)", pos, ok)
	}
	pos, ok = pm.PositionForNewLine("a.txt", 11)
	if !ok || pos != 5 {
		t.Fatalf("expected new line 11 => pos 5, got %d (ok=%v)", pos, ok)
	}
	pos, ok = pm.PositionForNewLine("a.txt", 12)
	if !ok || pos != 6 {
		t.Fatalf("expected new line 12 => pos 6, got %d (ok=%v)", pos, ok)
	}
}
