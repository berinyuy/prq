package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Snapshot struct {
	Repo          string
	PRNumber      int
	Title         string
	Description   string
	BaseSHA       string
	HeadSHA       string
	CISummary     string
	TestResults   string
	FileListStats string
	DiffChunks    string
}

func LoadTemplate() (string, error) {
	path := os.Getenv("PRQ_PROMPT_PATH")
	if path == "" {
		path = filepath.Join("prompts", "code-reviewer.txt")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt template: %w", err)
	}
	return string(content), nil
}

func Render(template string, userRules []string, repoRules []string, snap Snapshot) string {
	userRulesBlock := renderRules(userRules)
	repoRulesBlock := renderRules(repoRules)

	out := template
	out = strings.ReplaceAll(out, "{USER_RULES}", userRulesBlock)
	out = strings.ReplaceAll(out, "{REPO_RULES}", repoRulesBlock)
	out = strings.ReplaceAll(out, "{REPO}", snap.Repo)
	out = strings.ReplaceAll(out, "{PR_NUMBER}", fmt.Sprintf("%d", snap.PRNumber))
	out = strings.ReplaceAll(out, "{TITLE}", snap.Title)
	out = strings.ReplaceAll(out, "{DESCRIPTION}", snap.Description)
	out = strings.ReplaceAll(out, "{BASE_SHA}", snap.BaseSHA)
	out = strings.ReplaceAll(out, "{HEAD_SHA}", snap.HeadSHA)
	out = strings.ReplaceAll(out, "{CI_SUMMARY}", snap.CISummary)
	out = strings.ReplaceAll(out, "{TEST_RESULTS}", snap.TestResults)
	out = strings.ReplaceAll(out, "{FILE_LIST_WITH_STATS}", snap.FileListStats)
	out = strings.ReplaceAll(out, "{DIFF_CHUNKS}", snap.DiffChunks)

	return out
}

func renderRules(rules []string) string {
	if len(rules) == 0 {
		return "None"
	}
	var b strings.Builder
	for _, rule := range rules {
		b.WriteString("- ")
		b.WriteString(rule)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func DefaultSchemaPath() string {
	path := os.Getenv("PRQ_SCHEMA_PATH")
	if path != "" {
		return path
	}
	return filepath.Join("schemas", "review_plan.schema.json")
}
