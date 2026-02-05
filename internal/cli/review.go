package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brianndofor/prq/internal/github"
	"github.com/brianndofor/prq/internal/provider"
	"github.com/spf13/cobra"
)

func NewReviewCmd() *cobra.Command {
	var format string
	var maxIssues int
	var runTests bool

	cmd := &cobra.Command{
		Use:   "review <pr-url|OWNER/REPO#123>",
		Short: "Generate a review plan using Claude Code",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd.Context())
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			run, err := generateReviewPlan(ctx, app, args[0], maxIssues, runTests)
			if err != nil {
				return err
			}

			if err := app.Store.UpsertPR(run.FullRef, run.View.Repository.NameWithOwner, run.View.Number, run.View.HeadRefOid); err != nil {
				return err
			}
			if err := app.Store.MarkReviewed(run.FullRef, run.View.HeadRefOid); err != nil {
				return err
			}
			payload := DraftReviewPayload{
				Repo:    run.View.Repository.NameWithOwner,
				Number:  run.View.Number,
				BaseSHA: run.View.BaseRefOid,
				HeadSHA: run.View.HeadRefOid,
				Plan:    run.Plan,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return err
			}
			if err := app.Store.UpsertDraftReview(run.FullRef, string(payloadJSON), renderDraftPreview(payload)); err != nil {
				return err
			}
			if format != "json" {
				fmt.Fprintf(cmd.OutOrStdout(), "Draft saved. Run `prq submit %s` to post to GitHub.\n\n", run.FullRef)
			}

			switch format {
			case "json":
				return printReviewJSON(cmd, run.Plan, run.Raw)
			case "md":
				return printReviewMarkdown(cmd, run.Plan)
			default:
				return printReviewText(cmd, run.Plan)
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "text|json|md")
	cmd.Flags().IntVar(&maxIssues, "max-issues", 0, "Limit issues count")
	cmd.Flags().BoolVar(&runTests, "run-tests", false, "Run repo tests before review")
	return cmd
}

func renderFileList(files []github.PRFile) string {
	if len(files) == 0 {
		return "No files"
	}
	var b strings.Builder
	for _, file := range files {
		fmt.Fprintf(&b, "%s (+%d/-%d)\n", file.Path, file.Additions, file.Deletions)
	}
	return strings.TrimSpace(b.String())
}

func printReviewJSON(cmd *cobra.Command, plan provider.ReviewPlan, raw string) error {
	if raw != "" {
		_, err := cmd.OutOrStdout().Write([]byte(raw))
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write([]byte("\n"))
		return err
	}
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(data)
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write([]byte("\n"))
	return err
}

func printReviewText(cmd *cobra.Command, plan provider.ReviewPlan) error {
	writeReview(cmd, plan, false)
	return nil
}

func printReviewMarkdown(cmd *cobra.Command, plan provider.ReviewPlan) error {
	writeReview(cmd, plan, true)
	return nil
}

// ANSI color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

func riskColor(risk string) string {
	switch risk {
	case "high":
		return colorRed
	case "medium":
		return colorYellow
	case "low":
		return colorGreen
	default:
		return colorReset
	}
}

func decisionColor(decision string) string {
	switch decision {
	case "approve":
		return colorGreen
	case "request_changes":
		return colorRed
	case "comment":
		return colorYellow
	default:
		return colorReset
	}
}

func severityColor(severity string) string {
	switch severity {
	case "blocker":
		return colorRed
	case "major":
		return colorYellow
	case "minor":
		return colorCyan
	default:
		return colorReset
	}
}

func writeReview(cmd *cobra.Command, plan provider.ReviewPlan, markdown bool) {
	out := cmd.OutOrStdout()

	header := func(title string) {
		if markdown {
			fmt.Fprintf(out, "## %s\n", title)
		} else {
			fmt.Fprintf(out, "\n%s%s══ %s ══%s\n\n", colorBold, colorBlue, title, colorReset)
		}
	}

	line := func(text string) {
		fmt.Fprintln(out, text)
	}

	// Summary
	header("Summary")
	line(plan.Summary)

	// Risk and Decision - highlighted box
	header("Risk & Decision")
	if markdown {
		line(fmt.Sprintf("- **Risk:** %s", plan.RiskLevel))
		line(fmt.Sprintf("- **Decision:** %s", plan.Decision))
	} else {
		riskC := riskColor(plan.RiskLevel)
		decC := decisionColor(plan.Decision)
		line(fmt.Sprintf("  %s●%s Risk: %s%s%s%s", riskC, colorReset, colorBold, riskC, plan.RiskLevel, colorReset))
		line(fmt.Sprintf("  %s●%s Decision: %s%s%s%s", decC, colorReset, colorBold, decC, plan.Decision, colorReset))
	}

	// Key Changes
	header("Key Changes")
	if len(plan.KeyChanges) == 0 {
		line(fmt.Sprintf("  %s(none)%s", colorDim, colorReset))
	} else {
		for _, change := range plan.KeyChanges {
			if markdown {
				line("- " + change)
			} else {
				line(fmt.Sprintf("  %s•%s %s", colorCyan, colorReset, change))
			}
		}
	}

	// Issues
	header("Issues")
	issuesByFile := map[string][]provider.Issue{}
	order := []string{}
	for _, issue := range plan.Issues {
		if _, ok := issuesByFile[issue.File]; !ok {
			order = append(order, issue.File)
		}
		issuesByFile[issue.File] = append(issuesByFile[issue.File], issue)
	}
	if len(order) == 0 {
		line(fmt.Sprintf("  %s✓ No issues found%s", colorGreen, colorReset))
	} else {
		for _, file := range order {
			if markdown {
				line(fmt.Sprintf("### `%s`", file))
			} else {
				line(fmt.Sprintf("  %s%s📄 %s%s", colorBold, colorCyan, file, colorReset))
			}
			for _, issue := range issuesByFile[file] {
				sevC := severityColor(issue.Severity)
				if markdown {
					line(fmt.Sprintf("- **[%s/%s]** %s (L%d-%d)", issue.Severity, issue.Category, issue.Message, issue.StartLine, issue.EndLine))
				} else {
					line(fmt.Sprintf("     %s[%s]%s %s%s%s", sevC, issue.Severity, colorReset, colorDim, issue.Category, colorReset))
					line(fmt.Sprintf("     %s", issue.Message))
					line(fmt.Sprintf("     %sLines %d-%d%s", colorDim, issue.StartLine, issue.EndLine, colorReset))
				}
				if issue.SuggestionPatch != "" {
					if markdown {
						line("  ```diff")
						line(issue.SuggestionPatch)
						line("  ```")
					} else {
						line(fmt.Sprintf("     %s── Suggested fix ──%s", colorDim, colorReset))
						for _, patchLine := range strings.Split(issue.SuggestionPatch, "\n") {
							if strings.HasPrefix(patchLine, "+") {
								line(fmt.Sprintf("     %s%s%s", colorGreen, patchLine, colorReset))
							} else if strings.HasPrefix(patchLine, "-") {
								line(fmt.Sprintf("     %s%s%s", colorRed, patchLine, colorReset))
							} else {
								line(fmt.Sprintf("     %s", patchLine))
							}
						}
					}
				}
				if !markdown {
					line("") // spacing between issues
				}
			}
		}
	}

	// Questions
	header("Questions")
	if len(plan.Questions) == 0 {
		line(fmt.Sprintf("  %s(none)%s", colorDim, colorReset))
	} else {
		for i, q := range plan.Questions {
			if markdown {
				line(fmt.Sprintf("%d. %s", i+1, q))
			} else {
				line(fmt.Sprintf("  %s%d.%s %s", colorYellow, i+1, colorReset, q))
			}
		}
	}

	// Praise
	header("Praise")
	if len(plan.Praise) == 0 {
		line(fmt.Sprintf("  %s(none)%s", colorDim, colorReset))
	} else {
		for _, p := range plan.Praise {
			if markdown {
				line("- " + p)
			} else {
				line(fmt.Sprintf("  %s✓%s %s", colorGreen, colorReset, p))
			}
		}
	}

	// Draft Review Body
	header("Draft Review Body")
	if strings.TrimSpace(plan.DraftReviewBody) == "" {
		line(fmt.Sprintf("  %s(none)%s", colorDim, colorReset))
	} else {
		if markdown {
			line(plan.DraftReviewBody)
		} else {
			// Indent the draft body for visual separation
			for _, bodyLine := range strings.Split(plan.DraftReviewBody, "\n") {
				line(fmt.Sprintf("  %s", bodyLine))
			}
		}
	}
	line("") // final newline
}
