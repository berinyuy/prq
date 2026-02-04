package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func NewDraftCmd() *cobra.Command {
	var maxIssues int
	var runTests bool

	cmd := &cobra.Command{
		Use:   "draft <pr-url|OWNER/REPO#123>",
		Short: "Generate and save a draft review (not posted)",
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
			preview := renderDraftPreview(payload)
			if err := app.Store.UpsertDraftReview(run.FullRef, string(payloadJSON), preview); err != nil {
				return err
			}

			fmt.Fprint(cmd.OutOrStdout(), preview)
			fmt.Fprintf(cmd.OutOrStdout(), "\nSaved locally. To post to GitHub, run: prq submit %s\n", run.FullRef)
			return nil
		},
	}

	cmd.Flags().IntVar(&maxIssues, "max-issues", 0, "Limit issues count")
	cmd.Flags().BoolVar(&runTests, "run-tests", false, "Run repo tests before drafting")
	return cmd
}
