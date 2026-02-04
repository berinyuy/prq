package cli

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/brianndofor/prq/internal/github"
	"github.com/spf13/cobra"
)

func NewFollowupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "followup <pr-url|OWNER/REPO#123>",
		Short: "Show open review threads and changes since last review",
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
			ctx := cmd.Context()

			view, err := app.GH.PRView(ctx, fullRef)
			if err != nil {
				return err
			}
			if err := app.Store.UpsertPR(fullRef, view.Repository.NameWithOwner, view.Number, view.HeadRefOid); err != nil {
				return err
			}
			state, err := app.Store.GetPR(fullRef)
			if err != nil && err != sql.ErrNoRows {
				return err
			}

			threads, err := app.GH.ReviewThreads(ctx, repo, number)
			if err != nil {
				return err
			}
			openThreads := []github.ReviewThread{}
			for _, thread := range threads {
				if !thread.IsResolved {
					openThreads = append(openThreads, thread)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Follow-up for %s\n", fullRef)
			fmt.Fprintf(cmd.OutOrStdout(), "Current head: %s\n", view.HeadRefOid)
			if state.LastReviewedHeadSHA.Valid {
				fmt.Fprintf(cmd.OutOrStdout(), "Last reviewed head: %s\n", state.LastReviewedHeadSHA.String)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No previous review recorded.")
			}

			if state.LastReviewedHeadSHA.Valid {
				if state.LastReviewedHeadSHA.String == view.HeadRefOid {
					fmt.Fprintln(cmd.OutOrStdout(), "No new commits since last review.")
				} else {
					compare, err := app.GH.CompareCommits(ctx, repo, state.LastReviewedHeadSHA.String, view.HeadRefOid)
					if err != nil {
						return err
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Changes since last review: %d commits\n", compare.TotalCommits)
					for _, file := range compare.Files {
						fmt.Fprintf(cmd.OutOrStdout(), "- %s (%s +%d/-%d)\n", file.Filename, file.Status, file.Additions, file.Deletions)
					}
				}
			}

			if len(openThreads) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No open review threads.")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Open review threads: %d\n", len(openThreads))
			for _, thread := range openThreads {
				lineLabel := "?"
				if thread.Line != nil {
					lineLabel = fmt.Sprintf("%d", *thread.Line)
				}
				path := thread.Path
				if strings.TrimSpace(path) == "" {
					path = "(unknown)"
				}
				count := thread.Count
				if count == 0 {
					count = len(thread.Comments)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "- %s:%s (%d comments)", path, lineLabel, count)
				if thread.IsOutdated {
					fmt.Fprint(cmd.OutOrStdout(), " [outdated]")
				}
				fmt.Fprintln(cmd.OutOrStdout())
				if len(thread.Comments) > 0 {
					last := thread.Comments[len(thread.Comments)-1]
					fmt.Fprintf(cmd.OutOrStdout(), "  Last: %s at %s\n", last.Author, last.CreatedAt)
				}
			}
			return nil
		},
	}

	return cmd
}
