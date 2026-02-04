package cli

import (
	"fmt"
	"strings"

	"github.com/brianndofor/prq/internal/provider"
)

func decisionToGitHubEvent(decision string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(decision)) {
	case "approve":
		return "APPROVE", nil
	case "request_changes":
		return "REQUEST_CHANGES", nil
	case "comment", "":
		return "COMMENT", nil
	default:
		return "COMMENT", fmt.Errorf("unknown decision %q", decision)
	}
}

func renderDraftPreview(payload DraftReviewPayload) string {
	event, _ := decisionToGitHubEvent(payload.Plan.Decision)
	var b strings.Builder
	fmt.Fprintf(&b, "PR: %s#%d\n", payload.Repo, payload.Number)
	fmt.Fprintf(&b, "Head SHA: %s\n", payload.HeadSHA)
	fmt.Fprintf(&b, "Event: %s\n\n", event)
	b.WriteString("Review body:\n")
	body := strings.TrimSpace(payload.Plan.DraftReviewBody)
	if body == "" {
		body = strings.TrimSpace(payload.Plan.Summary)
	}
	b.WriteString(body)
	b.WriteString("\n\n")

	issues := payload.Plan.Issues
	if len(issues) == 0 {
		b.WriteString("Inline comments: none\n")
		return b.String()
	}
	b.WriteString(fmt.Sprintf("Inline comments (%d):\n", len(issues)))
	for _, issue := range issues {
		b.WriteString("- ")
		b.WriteString(renderIssueSummary(issue))
		b.WriteString("\n")
	}
	return b.String()
}

func renderIssueSummary(issue provider.Issue) string {
	loc := issue.File
	if issue.StartLine > 0 {
		loc = fmt.Sprintf("%s:%d", issue.File, issue.StartLine)
	}
	if issue.EndLine > 0 && issue.EndLine != issue.StartLine {
		loc = fmt.Sprintf("%s:%d-%d", issue.File, issue.StartLine, issue.EndLine)
	}
	return fmt.Sprintf("%s [%s/%s] %s", loc, issue.Severity, issue.Category, issue.Message)
}

func renderIssueCommentBody(issue provider.Issue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s/%s] %s", issue.Severity, issue.Category, issue.Message)
	if strings.TrimSpace(issue.SuggestionPatch) != "" {
		b.WriteString("\n\nSuggested patch:\n```diff\n")
		b.WriteString(strings.TrimSpace(issue.SuggestionPatch))
		b.WriteString("\n```\n")
	}
	return b.String()
}
