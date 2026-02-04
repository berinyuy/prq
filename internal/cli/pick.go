package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type pickOptions struct {
	limit  int
	repo   string
	owner  string
	label  string
	checks string
	draft  string
	sortBy string
}

func NewPickCmd() *cobra.Command {
	var opts pickOptions

	cmd := &cobra.Command{
		Use:   "pick",
		Short: "Interactive picker for PRs requested for review",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd.Context())
			if err != nil {
				return err
			}
			return runPicker(cmd, app, opts)
		},
	}

	cmd.Flags().IntVar(&opts.limit, "limit", 0, "Max results")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "Filter by repo OWNER/REPO")
	cmd.Flags().StringVar(&opts.owner, "owner", "", "Filter by org/owner")
	cmd.Flags().StringVar(&opts.label, "label", "", "Filter by label")
	cmd.Flags().StringVar(&opts.checks, "checks", "any", "Filter by checks: failure|pending|success|any")
	cmd.Flags().StringVar(&opts.draft, "draft", "any", "Filter by draft: true|false|any")
	cmd.Flags().StringVar(&opts.sortBy, "sort", "", "Sort: oldest|updated|ci|size")

	return cmd
}

func runPicker(cmd *cobra.Command, app *App, opts pickOptions) error {
	if !app.Config.TUI.Enabled {
		return fmt.Errorf("tui picker is disabled in config")
	}
	if opts.limit == 0 {
		opts.limit = app.Config.Queue.DefaultLimit
	}
	if opts.sortBy == "" {
		opts.sortBy = app.Config.Queue.DefaultSort
	}

	query := buildQueueQuery(opts.repo, opts.owner, opts.label, opts.draft)
	ghSort, order := mapSort(opts.sortBy)
	items, err := app.GH.SearchPRs(cmd.Context(), query, opts.limit, ghSort, order)
	if err != nil {
		return err
	}
	queue := buildQueueItems(items)
	if opts.checks != "any" || opts.sortBy == "ci" {
		queue, err = applyChecks(cmd.Context(), app.GH, queue, opts.checks)
		if err != nil {
			return err
		}
	}
	if opts.sortBy == "size" {
		queue, err = applySizes(cmd.Context(), app.GH, queue)
		if err != nil {
			return err
		}
	}
	if opts.checks != "any" {
		queue = filterByChecks(queue, opts.checks)
	}
	sortQueue(queue, opts.sortBy)

	if len(queue) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No PRs found.")
		return nil
	}

	result, err := runPickTUI(queue)
	if err != nil {
		return err
	}
	if result.Action == "" {
		return nil
	}
	return runPickAction(cmd, app, result.Item, result.Action)
}

func runPickAction(cmd *cobra.Command, app *App, item QueueItem, action string) error {
	fullRef := fmt.Sprintf("%s#%d", item.Repo, item.Number)
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "q", "quit":
		return nil
	case "o", "open":
		_, err := app.Exec.Run(cmd.Context(), "", "gh", "pr", "view", fullRef, "--web")
		return err
	case "r", "review":
		return runSubcommand(cmd, NewReviewCmd(), fullRef)
	case "d", "draft":
		return runSubcommand(cmd, NewDraftCmd(), fullRef)
	case "s", "submit":
		return runSubcommand(cmd, NewSubmitCmd(), fullRef)
	case "f", "followup":
		return runSubcommand(cmd, NewFollowupCmd(), fullRef)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func runSubcommand(parent *cobra.Command, sub *cobra.Command, args ...string) error {
	sub.SetContext(parent.Context())
	sub.SetOut(parent.OutOrStdout())
	sub.SetErr(parent.ErrOrStderr())
	sub.SetArgs(args)
	return sub.Execute()
}
