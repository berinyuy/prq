package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type Client struct {
	Runner Runner
}

func NewClient(runner Runner) *Client {
	return &Client{Runner: runner}
}

func (c *Client) CheckInstalled() error {
	_, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("gh CLI not found in PATH")
	}
	return nil
}

func (c *Client) AuthStatus(ctx context.Context) error {
	_, err := c.Runner.Run(ctx, []string{"auth", "status"}, nil)
	return err
}

type SearchPRsResponse []SearchPRItem

type SearchPRItem struct {
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	URL       string  `json:"url"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
	IsDraft   bool    `json:"isDraft"`
	Author    UserRef `json:"author"`
	Labels    []Label `json:"labels"`
	Repo      RepoRef `json:"repository"`
}

type RepoRef struct {
	NameWithOwner string `json:"nameWithOwner"`
}

type UserRef struct {
	Login string `json:"login"`
}

type Label struct {
	Name string `json:"name"`
}

// SearchMode defines the type of PR search to perform
type SearchMode string

const (
	// SearchModeReviewRequested searches for PRs where user is requested as reviewer
	SearchModeReviewRequested SearchMode = "review"
	// SearchModeMine searches for PRs authored by the user
	SearchModeMine SearchMode = "mine"
)

func (c *Client) SearchPRs(ctx context.Context, query string, limit int, sort string, order string, mode SearchMode) ([]SearchPRItem, error) {
	args := []string{"search", "prs", "--state", "open", "--limit", strconv.Itoa(limit), "--sort", sort, "--order", order, "--json", "number,title,url,repository,author,createdAt,updatedAt,isDraft,labels"}

	// Add mode-specific filter
	switch mode {
	case SearchModeMine:
		args = append(args, "--author=@me")
	default:
		args = append(args, "--review-requested=@me")
	}

	if strings.TrimSpace(query) != "" {
		args = append(args, query)
	}
	output, err := c.Runner.Run(ctx, args, nil)
	if err != nil {
		return nil, err
	}
	var items []SearchPRItem
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("failed to decode gh search output: %w", err)
	}
	return items, nil
}

type PRView struct {
	Number     int      `json:"number"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	URL        string   `json:"url"`
	Author     UserRef  `json:"author"`
	Labels     []Label  `json:"labels"`
	BaseRefOid string   `json:"baseRefOid"`
	HeadRefOid string   `json:"headRefOid"`
	Files      []PRFile `json:"files"`
	Repository RepoRef  `json:"headRepository"`
}

type PRFile struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// prRefToArgs converts a PR reference (owner/repo#N or URL) to gh CLI args
func prRefToArgs(ref string) []string {
	// Try to parse as owner/repo#number
	repo, number, err := ParsePR(ref)
	if err == nil && repo != "" {
		return []string{"-R", repo, strconv.Itoa(number)}
	}
	// Fall back to passing the ref directly (e.g., just a number)
	return []string{ref}
}

func (c *Client) PRView(ctx context.Context, pr string) (PRView, error) {
	prArgs := prRefToArgs(pr)
	args := append([]string{"pr", "view"}, prArgs...)
	args = append(args, "--json", "number,title,body,url,author,labels,baseRefOid,headRefOid,files,headRepository")
	output, err := c.Runner.Run(ctx, args, nil)
	if err != nil {
		return PRView{}, err
	}
	var view PRView
	if err := json.Unmarshal(output, &view); err != nil {
		return PRView{}, fmt.Errorf("failed to decode gh pr view output: %w", err)
	}
	return view, nil
}

func (c *Client) PRDiff(ctx context.Context, pr string) (string, error) {
	prArgs := prRefToArgs(pr)
	args := append([]string{"pr", "diff"}, prArgs...)
	output, err := c.Runner.Run(ctx, args, nil)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

type CheckRunsResponse struct {
	Total int        `json:"total"`
	Runs  []CheckRun `json:"check_runs"`
}

type CheckRun struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

func (c *Client) CheckRuns(ctx context.Context, repo string, sha string) (CheckRunsResponse, error) {
	args := []string{"api", fmt.Sprintf("repos/%s/commits/%s/check-runs", repo, sha)}
	output, err := c.Runner.Run(ctx, args, nil)
	if err != nil {
		return CheckRunsResponse{}, err
	}
	var resp CheckRunsResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return CheckRunsResponse{}, fmt.Errorf("failed to decode check-runs output: %w", err)
	}
	return resp, nil
}

var prRefRe = regexp.MustCompile(`^([^/]+/[^#]+)#([0-9]+)$`)

func ParsePR(ref string) (repo string, number int, err error) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		parsed, parseErr := url.Parse(ref)
		if parseErr != nil {
			return "", 0, fmt.Errorf("invalid PR URL")
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) < 4 || parts[2] != "pull" {
			return "", 0, fmt.Errorf("invalid PR URL")
		}
		repo = fmt.Sprintf("%s/%s", parts[0], parts[1])
		number, err = strconv.Atoi(parts[3])
		if err != nil {
			return "", 0, fmt.Errorf("invalid PR URL")
		}
		return repo, number, nil
	}

	matches := prRefRe.FindStringSubmatch(ref)
	if len(matches) != 3 {
		return "", 0, fmt.Errorf("invalid PR reference")
	}

	number, err = strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, fmt.Errorf("invalid PR reference")
	}
	return matches[1], number, nil
}
