package diff

import (
	"fmt"
	"path/filepath"
	"strings"
)

type FileDiff struct {
	Path string
	Text string
}

func ParseUnified(input string) ([]FileDiff, error) {
	lines := strings.Split(input, "\n")
	var files []FileDiff
	var current *FileDiff
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			if current != nil {
				files = append(files, *current)
			}
			path := parsePath(line)
			current = &FileDiff{Path: path, Text: line + "\n"}
			continue
		}
		if current == nil {
			continue
		}
		current.Text += line + "\n"
	}
	if current != nil {
		files = append(files, *current)
	}
	return files, nil
}

func parsePath(line string) string {
	parts := strings.Split(line, " ")
	if len(parts) < 4 {
		return ""
	}
	path := strings.TrimPrefix(parts[3], "b/")
	return path
}

func BuildChunks(files []FileDiff, ignoreGlobs []string, maxFiles int, maxChunkChars int) ([]string, error) {
	if maxFiles <= 0 {
		return nil, fmt.Errorf("maxFiles must be > 0")
	}
	if maxChunkChars <= 0 {
		return nil, fmt.Errorf("maxChunkChars must be > 0")
	}
	chunks := []string{}
	count := 0
	for _, file := range files {
		if count >= maxFiles {
			break
		}
		if file.Path == "" {
			continue
		}
		if isIgnored(file.Path, ignoreGlobs) {
			continue
		}
		for _, chunk := range splitChunk(file.Path, file.Text, maxChunkChars) {
			chunks = append(chunks, chunk)
		}
		count++
	}
	return chunks, nil
}

func isIgnored(path string, globs []string) bool {
	for _, glob := range globs {
		match, err := filepath.Match(glob, path)
		if err == nil && match {
			return true
		}
	}
	return false
}

func splitChunk(path string, text string, maxChunkChars int) []string {
	if len(text) <= maxChunkChars {
		return []string{formatChunk(path, text)}
	}
	var chunks []string
	remaining := text
	for len(remaining) > 0 {
		limit := maxChunkChars
		if len(remaining) < limit {
			limit = len(remaining)
		}
		piece := remaining[:limit]
		chunks = append(chunks, formatChunk(path, piece))
		remaining = remaining[limit:]
	}
	return chunks
}

func formatChunk(path string, text string) string {
	return fmt.Sprintf("File: %s\n%s", path, text)
}
