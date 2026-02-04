package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brianndofor/prq/internal/diff"
	"github.com/brianndofor/prq/internal/github"
	"github.com/brianndofor/prq/internal/prompt"
	"github.com/brianndofor/prq/internal/provider"
	"github.com/brianndofor/prq/internal/redact"
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
			if runTests {
				return fmt.Errorf("--run-tests is not implemented yet")
			}
			prRef := args[0]
			repo, number, err := github.ParsePR(prRef)
			if err != nil {
				return err
			}
			fullRef := fmt.Sprintf("%s#%d", repo, number)
			ctx := context.Background()
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
			chunks, err := diff.BuildChunks(files, app.RepoConfig.Diff.Ignore, app.RepoConfig.Diff.MaxFiles, app.RepoConfig.Diff.MaxChunkChars)
			if err != nil {
				return err
			}
			fileList := renderFileList(view.Files)
			diffChunks := strings.Join(chunks, "\n\n")

			redactedTitle := redact.RedactOptional(view.Title, app.Config.Redaction.Enabled)
			redactedBody := redact.RedactOptional(view.Body, app.Config.Redaction.Enabled)
			redactedDiff := redact.RedactOptional(diffChunks, app.Config.Redaction.Enabled)
			redactedFiles := redact.RedactOptional(fileList, app.Config.Redaction.Enabled)
			redactedUserRules := redact.RedactRuleList(app.Config.UserRules, app.Config.Redaction.Enabled)
			redactedRepoRules := redact.RedactRuleList(app.RepoConfig.RepoRules, app.Config.Redaction.Enabled)

			snap := prompt.Snapshot{
				Repo:          view.Repository.NameWithOwner,
				PRNumber:      view.Number,
				Title:         redactedTitle,
				Description:   redactedBody,
				BaseSHA:       view.BaseRefOid,
				HeadSHA:       view.HeadRefOid,
				CISummary:     "Not fetched",
				TestResults:   "Not run",
				FileListStats: redactedFiles,
				DiffChunks:    redactedDiff,
			}

			template, err := prompt.LoadTemplate()
			if err != nil {
				return err
			}
			promptText := prompt.Render(template, redactedUserRules, redactedRepoRules, snap)
			promptText = redact.RedactPromptBlock(promptText, app.Config.Redaction.Enabled)

			schemaPath := prompt.DefaultSchemaPath()
			plan, raw, err := app.Provider.RunReview(ctx, promptText, schemaPath)
			if err != nil {
				return err
			}
			if maxIssues > 0 && len(plan.Issues) > maxIssues {
				plan.Issues = plan.Issues[:maxIssues]
			}

			if err := app.Store.UpsertPR(fullRef, view.Repository.NameWithOwner, view.Number, view.HeadRefOid); err != nil {
				return err
			}

			switch format {
			case "json":
				return printReviewJSON(cmd, plan, raw)
			case "md":
				return printReviewMarkdown(cmd, plan)
			default:
				return printReviewText(cmd, plan)
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
	for _, change := range plan.KeyChanges {
		line("- " + change)
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

	header("Questions")
	for _, q := range plan.Questions {
		line("- " + q)
	}

	header("Praise")
	for _, p := range plan.Praise {
		line("- " + p)
	}

	header("Draft Review Body")
	line(plan.DraftReviewBody)
}
