package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/brianndofor/prq/internal/github"
	"github.com/spf13/cobra"
)

type QueueItem struct {
	Repo        string   `json:"repo"`
	Number      int      `json:"number"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Author      string   `json:"author"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	IsDraft     bool     `json:"is_draft"`
	Labels      []string `json:"labels"`
	AgeDays     int      `json:"age_days"`
	UpdatedDays int      `json:"updated_days"`
	Checks      string   `json:"checks"`
	HeadSHA     string   `json:"head_sha"`
	Size        int      `json:"size"`
}

func NewQueueCmd() *cobra.Command {
	var limit int
	var repo string
	var owner string
	var label string
	var checks string
	var draft string
	var sortBy string
	var jsonOut bool
	var tui bool

	cmd := &cobra.Command{
		Use:   "queue",
		Short: "List PRs requested for review",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := getApp(cmd.Context())
			if err != nil {
				return err
			}
			if tui {
				if jsonOut {
					return fmt.Errorf("--json is not supported with --tui")
				}
				return runPicker(cmd, app, pickOptions{limit: limit, repo: repo, owner: owner, label: label, checks: checks, draft: draft, sortBy: sortBy})
			}

			if limit == 0 {
				limit = app.Config.Queue.DefaultLimit
			}
			if sortBy == "" {
				sortBy = app.Config.Queue.DefaultSort
			}

			query := buildQueueQuery(repo, owner, label, draft)
			ghSort, order := mapSort(sortBy)
			items, err := app.GH.SearchPRs(context.Background(), query, limit, ghSort, order)
			if err != nil {
				return err
			}
			queue := buildQueueItems(items)

			if checks != "any" || sortBy == "ci" {
				queue, err = applyChecks(context.Background(), app.GH, queue, checks)
				if err != nil {
					return err
				}
			}

			if sortBy == "size" {
				queue, err = applySizes(context.Background(), app.GH, queue)
				if err != nil {
					return err
				}
			}

			if checks != "any" {
				queue = filterByChecks(queue, checks)
			}

			sortQueue(queue, sortBy)
			if jsonOut {
				return printQueueJSON(cmd, queue, limit)
			}
			return printQueueText(cmd, queue, limit)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Max results")
	cmd.Flags().StringVar(&repo, "repo", "", "Filter by repo OWNER/REPO")
	cmd.Flags().StringVar(&owner, "owner", "", "Filter by org/owner")
	cmd.Flags().StringVar(&label, "label", "", "Filter by label")
	cmd.Flags().StringVar(&checks, "checks", "any", "Filter by checks: failure|pending|success|any")
	cmd.Flags().StringVar(&draft, "draft", "any", "Filter by draft: true|false|any")
	cmd.Flags().StringVar(&sortBy, "sort", "", "Sort: oldest|updated|ci|size")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output JSON")
	cmd.Flags().BoolVar(&tui, "tui", false, "Open TUI picker")
	return cmd
}

func buildQueueQuery(repo, owner, label, draft string) string {
	query := []string{}
	if repo != "" {
		query = append(query, fmt.Sprintf("repo:%s", repo))
	}
	if owner != "" {
		query = append(query, fmt.Sprintf("org:%s", owner))
	}
	if label != "" {
		query = append(query, fmt.Sprintf("label:%s", label))
	}
	if draft == "true" {
		query = append(query, "draft:true")
	} else if draft == "false" {
		query = append(query, "draft:false")
	}
	return strings.Join(query, " ")
}

func mapSort(sortBy string) (string, string) {
	switch sortBy {
	case "updated":
		return "updated", "asc"
	case "ci":
		return "created", "asc"
	case "size":
		return "created", "asc"
	default:
		return "created", "asc"
	}
}

func buildQueueItems(items []github.SearchPRItem) []QueueItem {
	queue := make([]QueueItem, 0, len(items))
	now := queueNow()
	for _, item := range items {
		created := parseTime(item.CreatedAt)
		updated := parseTime(item.UpdatedAt)
		labels := make([]string, 0, len(item.Labels))
		for _, label := range item.Labels {
			labels = append(labels, label.Name)
		}
		queue = append(queue, QueueItem{
			Repo:        item.Repo.NameWithOwner,
			Number:      item.Number,
			Title:       item.Title,
			URL:         item.URL,
			Author:      item.Author.Login,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
			IsDraft:     item.IsDraft,
			Labels:      labels,
			AgeDays:     int(now.Sub(created).Hours() / 24),
			UpdatedDays: int(now.Sub(updated).Hours() / 24),
			Checks:      "unknown",
		})
	}
	return queue
}

func queueNow() time.Time {
	if value := os.Getenv("PRQ_NOW"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return parsed
		}
	}
	return time.Now()
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now()
	}
	return parsed
}

func applyChecks(ctx context.Context, gh *github.Client, queue []QueueItem, filter string) ([]QueueItem, error) {
	for i, item := range queue {
		if item.Repo == "" || item.Number == 0 {
			queue[i].Checks = "unknown"
			continue
		}
		headSHA := item.HeadSHA
		if headSHA == "" {
			ref := fmt.Sprintf("%s#%d", item.Repo, item.Number)
			view, err := gh.PRView(ctx, ref)
			if err != nil {
				return nil, err
			}
			headSHA = view.HeadRefOid
			queue[i].HeadSHA = headSHA
		}
		if headSHA == "" {
			queue[i].Checks = "unknown"
			continue
		}
		resp, err := gh.CheckRuns(ctx, item.Repo, headSHA)
		if err != nil {
			return nil, err
		}
		queue[i].Checks = summarizeChecks(resp)
	}
	return queue, nil
}

func summarizeChecks(resp github.CheckRunsResponse) string {
	if resp.Total == 0 {
		return "none"
	}
	status := "success"
	for _, run := range resp.Runs {
		if run.Conclusion == "failure" || run.Conclusion == "cancelled" || run.Conclusion == "timed_out" {
			return "failure"
		}
		if run.Status != "completed" {
			status = "pending"
		}
	}
	return status
}

func filterByChecks(queue []QueueItem, filter string) []QueueItem {
	if filter == "any" {
		return queue
	}
	filtered := []QueueItem{}
	for _, item := range queue {
		if item.Checks == filter {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func sortQueue(queue []QueueItem, sortBy string) {
	switch sortBy {
	case "updated":
		sort.Slice(queue, func(i, j int) bool {
			return queue[i].UpdatedAt < queue[j].UpdatedAt
		})
	case "ci":
		sort.Slice(queue, func(i, j int) bool {
			return queue[i].Checks < queue[j].Checks
		})
	case "size":
		sort.Slice(queue, func(i, j int) bool {
			return queue[i].Size > queue[j].Size
		})
	default:
		sort.Slice(queue, func(i, j int) bool {
			return queue[i].CreatedAt < queue[j].CreatedAt
		})
	}
}

func applySizes(ctx context.Context, gh *github.Client, queue []QueueItem) ([]QueueItem, error) {
	for i, item := range queue {
		if item.Repo == "" || item.Number == 0 {
			continue
		}
		ref := fmt.Sprintf("%s#%d", item.Repo, item.Number)
		view, err := gh.PRView(ctx, ref)
		if err != nil {
			return nil, err
		}
		queue[i].Size = sumSize(view.Files)
		if queue[i].HeadSHA == "" {
			queue[i].HeadSHA = view.HeadRefOid
		}
	}
	return queue, nil
}

func sumSize(files []github.PRFile) int {
	size := 0
	for _, file := range files {
		size += file.Additions + file.Deletions
	}
	return size
}

func printQueueText(cmd *cobra.Command, queue []QueueItem, limit int) error {
	if len(queue) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No PRs found.")
		return nil
	}
	for _, item := range queue {
		fmt.Fprintf(cmd.OutOrStdout(), "%s#%d %s\n", item.Repo, item.Number, item.Title)
		fmt.Fprintf(cmd.OutOrStdout(), "  Author: %s  Age: %dd  Updated: %dd  Draft: %v  Checks: %s\n", item.Author, item.AgeDays, item.UpdatedDays, item.IsDraft, item.Checks)
		fmt.Fprintf(cmd.OutOrStdout(), "  URL: %s\n", item.URL)
	}
	if len(queue) >= limit {
		fmt.Fprintf(cmd.OutOrStdout(), "Showing first %d results. Refine with filters or raise --limit.\n", limit)
	}
	return nil
}

func printQueueJSON(cmd *cobra.Command, queue []QueueItem, limit int) error {
	payload := map[string]any{
		"items": queue,
		"limit": limit,
	}
	if len(queue) == 0 {
		payload["message"] = "No PRs found."
	}
	data, err := json.MarshalIndent(payload, "", "  ")
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
