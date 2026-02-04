package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/brianndofor/prq/internal/diff"
	"github.com/brianndofor/prq/internal/github"
	"github.com/brianndofor/prq/internal/prompt"
	"github.com/brianndofor/prq/internal/provider"
	"github.com/brianndofor/prq/internal/redact"
)

type ReviewRun struct {
	FullRef  string
	View     github.PRView
	Plan     provider.ReviewPlan
	Raw      string
	DiffText string
}

func generateReviewPlan(ctx context.Context, app *App, prRef string, maxIssues int, runTests bool) (ReviewRun, error) {
	repo, number, err := github.ParsePR(prRef)
	if err != nil {
		return ReviewRun{}, err
	}
	fullRef := fmt.Sprintf("%s#%d", repo, number)

	view, err := app.GH.PRView(ctx, fullRef)
	if err != nil {
		return ReviewRun{}, err
	}
	diffText, err := app.GH.PRDiff(ctx, fullRef)
	if err != nil {
		return ReviewRun{}, err
	}

	files, err := diff.ParseUnified(diffText)
	if err != nil {
		return ReviewRun{}, err
	}
	chunks, err := diff.BuildChunks(files, app.RepoConfig.Diff.Ignore, app.RepoConfig.Diff.MaxFiles, app.RepoConfig.Diff.MaxChunkChars)
	if err != nil {
		return ReviewRun{}, err
	}
	fileList := renderFileList(view.Files)
	diffChunks := strings.Join(chunks, "\n\n")

	testResults := "Not run"
	if runTests {
		output, err := runTestsForPR(ctx, app, view)
		if err != nil {
			return ReviewRun{}, err
		}
		testResults = output
	}

	redactedTitle := redact.RedactOptional(view.Title, app.Config.Redaction.Enabled)
	redactedBody := redact.RedactOptional(view.Body, app.Config.Redaction.Enabled)
	redactedDiff := redact.RedactOptional(diffChunks, app.Config.Redaction.Enabled)
	redactedFiles := redact.RedactOptional(fileList, app.Config.Redaction.Enabled)
	redactedUserRules := redact.RedactRuleList(app.Config.UserRules, app.Config.Redaction.Enabled)
	redactedRepoRules := redact.RedactRuleList(app.RepoConfig.RepoRules, app.Config.Redaction.Enabled)
	redactedTests := redact.RedactOptional(testResults, app.Config.Redaction.Enabled)

	snap := prompt.Snapshot{
		Repo:          view.Repository.NameWithOwner,
		PRNumber:      view.Number,
		Title:         redactedTitle,
		Description:   redactedBody,
		BaseSHA:       view.BaseRefOid,
		HeadSHA:       view.HeadRefOid,
		CISummary:     "Not fetched",
		TestResults:   redactedTests,
		FileListStats: redactedFiles,
		DiffChunks:    redactedDiff,
	}

	template, err := prompt.LoadTemplate()
	if err != nil {
		return ReviewRun{}, err
	}
	promptText := prompt.Render(template, redactedUserRules, redactedRepoRules, snap)
	promptText = redact.RedactPromptBlock(promptText, app.Config.Redaction.Enabled)

	schemaPath := prompt.DefaultSchemaPath()
	plan, raw, err := app.Provider.RunReview(ctx, promptText, schemaPath)
	if err != nil {
		return ReviewRun{}, err
	}
	if maxIssues > 0 && len(plan.Issues) > maxIssues {
		plan.Issues = plan.Issues[:maxIssues]
	}

	return ReviewRun{FullRef: fullRef, View: view, Plan: plan, Raw: raw, DiffText: diffText}, nil
}
