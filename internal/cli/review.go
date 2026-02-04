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

func writeReview(cmd *cobra.Command, plan provider.ReviewPlan, markdown bool) {
	header := func(title string) {
		if markdown {
			fmt.Fprintf(cmd.OutOrStdout(), "## %s\n", title)
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", title)
	}
	line := func(text string) {
		fmt.Fprintln(cmd.OutOrStdout(), text)
	}

	header("Summary")
	line(plan.Summary)
	header("Risk and Decision")
	line(fmt.Sprintf("Risk: %s", plan.RiskLevel))
	line(fmt.Sprintf("Decision: %s", plan.Decision))

	header("Key Changes")
	if len(plan.KeyChanges) == 0 {
		line("No key changes listed.")
	} else {
		for _, change := range plan.KeyChanges {
			line("- " + change)
		}
	}

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
		line("No issues found.")
	} else {
		for _, file := range order {
			line(file)
			for _, issue := range issuesByFile[file] {
				line(fmt.Sprintf("  - [%s/%s] %s (%d-%d)", issue.Severity, issue.Category, issue.Message, issue.StartLine, issue.EndLine))
				if issue.SuggestionPatch != "" {
					line("    Suggested patch:")
					if markdown {
						line("```diff")
						line(issue.SuggestionPatch)
						line("```")
					} else {
						line(issue.SuggestionPatch)
					}
				}
			}
		}
	}

	header("Questions")
	if len(plan.Questions) == 0 {
		line("No questions.")
	} else {
		for _, q := range plan.Questions {
			line("- " + q)
		}
	}

	header("Praise")
	if len(plan.Praise) == 0 {
		line("No praise.")
	} else {
		for _, p := range plan.Praise {
			line("- " + p)
		}
	}

	header("Draft Review Body")
	if strings.TrimSpace(plan.DraftReviewBody) == "" {
		line("No draft review body.")
	} else {
		line(plan.DraftReviewBody)
	}
}
