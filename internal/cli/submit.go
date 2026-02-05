package cli

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brianndofor/prq/internal/diff"
	"github.com/brianndofor/prq/internal/github"
	"github.com/brianndofor/prq/internal/provider"
	"github.com/spf13/cobra"
)

type issuePosition struct {
	Issue    provider.Issue
	Path     string
	Line     int
	Position int
	Mapped   bool
}

func NewSubmitCmd() *cobra.Command {
	var yes bool
	var dryRun bool
	var eventOverride string

	cmd := &cobra.Command{
		Use:   "submit <pr-url|OWNER/REPO#123>",
		Short: "Submit the latest draft review to GitHub",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd.Context())
			if err != nil {
				return err
			}
			repo, number, err := github.ParsePR(args[0])
			if err != nil {
				return err
			}
			fullRef := fmt.Sprintf("%s#%d", repo, number)

			draft, err := app.Store.GetDraftReview(fullRef)
			if err != nil {
				if err == sql.ErrNoRows {
					return fmt.Errorf("no saved draft for %s; run `prq draft %s` or `prq review %s` first", fullRef, fullRef, fullRef)
				}
				return err
			}

			var payload DraftReviewPayload
			if err := json.Unmarshal([]byte(draft.PayloadJSON), &payload); err != nil {
				return fmt.Errorf("failed to decode saved draft payload: %w", err)
			}

			ctx := cmd.Context()
			view, err := app.GH.PRView(ctx, fullRef)
			if err != nil {
				return err
			}
			diffText, err := app.GH.PRDiff(ctx, fullRef)
			if err != nil {
				return err
			}
			files, err := diff.ParseUnified(diffText)
			if err != nil {
				return err
			}
			posMap, err := diff.BuildPositionMap(files)
			if err != nil {
				return err
			}

			var event string
			var eventErr error
			if eventOverride != "" {
				// Use the override if provided
				event = strings.ToUpper(eventOverride)
				if event != "APPROVE" && event != "COMMENT" && event != "REQUEST_CHANGES" {
					return fmt.Errorf("invalid --event value %q; must be one of: approve, comment, request_changes", eventOverride)
				}
			} else {
				event, eventErr = decisionToGitHubEvent(payload.Plan.Decision)
				if eventErr != nil {
					event = "COMMENT"
				}
			}
			issuePositions := mapIssuesToPositions(posMap, payload.Plan.Issues)
			comments, unmapped := buildReviewComments(issuePositions)
			body := buildReviewBody(payload.Plan.DraftReviewBody, payload.Plan.Summary, unmapped)

			preview := renderSubmitPreview(fullRef, payload, view.HeadRefOid, event, body, issuePositions, dryRun, eventErr)
			fmt.Fprint(cmd.OutOrStdout(), preview)

			if dryRun {
				return nil
			}
			if !yes {
				ok, err := confirm(cmd, fmt.Sprintf("Post this review to %s? [y/N]: ", fullRef))
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Aborted.")
					return nil
				}
			}

			resp, err := app.GH.CreateReview(ctx, repo, number, github.CreateReviewRequest{Body: body, Event: event, Comments: comments})
			if err != nil {
				return err
			}
			if err := app.Store.UpsertPR(fullRef, view.Repository.NameWithOwner, view.Number, view.HeadRefOid); err != nil {
				return err
			}
			if err := app.Store.MarkSubmitted(fullRef); err != nil {
				return err
			}
			if err := app.Store.DeleteDraftReview(fullRef); err != nil {
				return err
			}
			if strings.TrimSpace(resp.HTMLURL) != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Submitted review: %s\n", resp.HTMLURL)
			} else if resp.ID != 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Submitted review id: %d\n", resp.ID)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Submitted review.")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview only; do not post")
	cmd.Flags().StringVar(&eventOverride, "event", "", "Override review event (approve, comment, request_changes)")
	return cmd
}

func mapIssuesToPositions(posMap diff.PositionMap, issues []provider.Issue) []issuePosition {
	positions := make([]issuePosition, 0, len(issues))
	for _, issue := range issues {
		pos, line, ok, path := positionForIssue(posMap, issue)
		positions = append(positions, issuePosition{Issue: issue, Path: path, Position: pos, Line: line, Mapped: ok})
	}
	return positions
}

func positionForIssue(posMap diff.PositionMap, issue provider.Issue) (pos int, line int, ok bool, path string) {
	file := strings.TrimSpace(issue.File)
	if file == "" {
		return 0, 0, false, ""
	}
	candidates := []string{file}
	if strings.HasPrefix(file, "./") {
		candidates = append(candidates, strings.TrimPrefix(file, "./"))
	}
	if strings.HasPrefix(file, "a/") || strings.HasPrefix(file, "b/") {
		candidates = append(candidates, file[2:])
	}

	for _, candidate := range candidates {
		if issue.StartLine > 0 {
			if pos, ok := posMap.PositionForNewLine(candidate, issue.StartLine); ok {
				return pos, issue.StartLine, true, candidate
			}
		}
		if issue.EndLine > 0 && issue.EndLine != issue.StartLine {
			if pos, ok := posMap.PositionForNewLine(candidate, issue.EndLine); ok {
				return pos, issue.EndLine, true, candidate
			}
		}
	}
	return 0, 0, false, file
}

func buildReviewComments(positions []issuePosition) ([]github.ReviewComment, []provider.Issue) {
	comments := []github.ReviewComment{}
	unmapped := []provider.Issue{}
	for _, item := range positions {
		if !item.Mapped {
			unmapped = append(unmapped, item.Issue)
			continue
		}
		comments = append(comments, github.ReviewComment{Path: item.Path, Position: item.Position, Body: renderIssueCommentBody(item.Issue)})
	}
	return comments, unmapped
}

func buildReviewBody(base string, summary string, unmapped []provider.Issue) string {
	body := strings.TrimSpace(base)
	if body == "" {
		body = strings.TrimSpace(summary)
	}
	if len(unmapped) == 0 {
		return body
	}
	var b strings.Builder
	if body != "" {
		b.WriteString(body)
		b.WriteString("\n\n")
	}
	b.WriteString("Additional notes (could not map to diff positions):\n")
	for _, issue := range unmapped {
		b.WriteString("- ")
		b.WriteString(renderIssueSummary(issue))
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func renderSubmitPreview(fullRef string, payload DraftReviewPayload, currentHeadSHA string, event string, body string, issuePositions []issuePosition, dryRun bool, decisionErr error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PR: %s\n", fullRef)
	if strings.TrimSpace(payload.HeadSHA) != "" {
		fmt.Fprintf(&b, "Draft head SHA: %s\n", payload.HeadSHA)
	}
	if strings.TrimSpace(currentHeadSHA) != "" {
		fmt.Fprintf(&b, "Current head SHA: %s\n", currentHeadSHA)
	}
	if strings.TrimSpace(payload.HeadSHA) != "" && strings.TrimSpace(currentHeadSHA) != "" && payload.HeadSHA != currentHeadSHA {
		b.WriteString("WARNING: draft was generated for a different head SHA; inline comment mapping may be incomplete.\n")
	}
	if decisionErr != nil {
		fmt.Fprintf(&b, "WARNING: unknown decision %q; defaulting to COMMENT.\n", payload.Plan.Decision)
	}
	fmt.Fprintf(&b, "Event: %s\n", event)
	if dryRun {
		b.WriteString("DRY RUN: not posting to GitHub.\n")
	}

	b.WriteString("\nReview body:\n")
	b.WriteString(strings.TrimSpace(body))
	b.WriteString("\n\n")

	mappedCount := 0
	for _, item := range issuePositions {
		if item.Mapped {
			mappedCount++
		}
	}
	fmt.Fprintf(&b, "Inline comments: %d mapped, %d unmapped\n", mappedCount, len(issuePositions)-mappedCount)
	for _, item := range issuePositions {
		if item.Mapped {
			fmt.Fprintf(&b, "- %s:%d => position %d\n", item.Path, item.Line, item.Position)
		} else {
			fmt.Fprintf(&b, "- %s (unmapped)\n", renderIssueSummary(item.Issue))
		}
	}
	b.WriteString("\n")
	return b.String()
}
