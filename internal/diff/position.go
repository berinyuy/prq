package diff

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderRE = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

type PositionMap struct {
	NewLineToPosition map[string]map[int]int
	OldLineToPosition map[string]map[int]int
}

func BuildPositionMap(files []FileDiff) (PositionMap, error) {
	pm := PositionMap{
		NewLineToPosition: map[string]map[int]int{},
		OldLineToPosition: map[string]map[int]int{},
	}
	for _, file := range files {
		newMap, oldMap, err := buildFilePositionMaps(file.Text)
		if err != nil {
			return PositionMap{}, fmt.Errorf("build position map for %q: %w", file.Path, err)
		}
		if len(newMap) > 0 {
			pm.NewLineToPosition[file.Path] = newMap
		}
		if len(oldMap) > 0 {
			pm.OldLineToPosition[file.Path] = oldMap
		}
	}
	return pm, nil
}

func (p PositionMap) PositionForNewLine(path string, line int) (int, bool) {
	fileMap, ok := p.NewLineToPosition[path]
	if !ok {
		return 0, false
	}
	pos, ok := fileMap[line]
	return pos, ok
}

func buildFilePositionMaps(unifiedFileDiff string) (newLineToPos map[int]int, oldLineToPos map[int]int, err error) {
	newLineToPos = map[int]int{}
	oldLineToPos = map[int]int{}

	pos := 0
	oldLine := 0
	newLine := 0
	inPatch := false

	lines := strings.Split(unifiedFileDiff, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		matches := hunkHeaderRE.FindStringSubmatch(line)
		if len(matches) > 0 {
			oldStart, parseErr := strconv.Atoi(matches[1])
			if parseErr != nil {
				return nil, nil, fmt.Errorf("invalid hunk header: %q", line)
			}
			newStart, parseErr := strconv.Atoi(matches[3])
			if parseErr != nil {
				return nil, nil, fmt.Errorf("invalid hunk header: %q", line)
			}
			oldLine = oldStart
			newLine = newStart
			inPatch = true
			pos++
			continue
		}
		if !inPatch {
			continue
		}

		pos++
		switch {
		case strings.HasPrefix(line, " "):
			oldLineToPos[oldLine] = pos
			newLineToPos[newLine] = pos
			oldLine++
			newLine++
		case strings.HasPrefix(line, "+"):
			newLineToPos[newLine] = pos
			newLine++
		case strings.HasPrefix(line, "-"):
			oldLineToPos[oldLine] = pos
			oldLine++
		case strings.HasPrefix(line, "\\"):
			// "\\ No newline at end of file" applies to the previous line.
		default:
			// Unknown line prefix; treat as non-mappable.
		}
	}

	return newLineToPos, oldLineToPos, nil
}
