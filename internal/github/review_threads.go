package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ReviewThread struct {
	IsResolved bool
	IsOutdated bool
	Path       string
	Line       *int
	Comments   []ReviewThreadComment
	Count      int
}

type ReviewThreadComment struct {
	Author    string
	Body      string
	CreatedAt string
}

type reviewThreadsResponse struct {
	Data struct {
		Repository struct {
			PullRequest struct {
				ReviewThreads struct {
					Nodes    []reviewThreadNode `json:"nodes"`
					PageInfo pageInfo           `json:"pageInfo"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
}

type reviewThreadNode struct {
	IsResolved bool `json:"isResolved"`
	IsOutdated bool `json:"isOutdated"`
	Path       string
	Line       *int `json:"line"`
	Comments   struct {
		Nodes      []reviewThreadCommentNode `json:"nodes"`
		TotalCount int                       `json:"totalCount"`
	} `json:"comments"`
}

type reviewThreadCommentNode struct {
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

type pageInfo struct {
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   *string `json:"endCursor"`
}

func (c *Client) ReviewThreads(ctx context.Context, repo string, number int) ([]ReviewThread, error) {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return nil, err
	}

	query := `query($owner: String!, $name: String!, $number: Int!, $after: String) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 100, after: $after) {
        nodes {
          isResolved
          isOutdated
          path
          line
          comments(last: 1) {
            nodes {
              author { login }
              body
              createdAt
            }
            totalCount
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}`

	var threads []ReviewThread
	var after string
	for {
		args := []string{"api", "graphql", "-f", "query=" + query, "-f", "owner=" + owner, "-f", "name=" + name, "-F", fmt.Sprintf("number=%d", number)}
		if after != "" {
			args = append(args, "-f", "after="+after)
		}
		output, err := c.Runner.Run(ctx, args, nil)
		if err != nil {
			return nil, err
		}
		var resp reviewThreadsResponse
		if err := json.Unmarshal(output, &resp); err != nil {
			return nil, fmt.Errorf("failed to decode review threads: %w", err)
		}

		for _, node := range resp.Data.Repository.PullRequest.ReviewThreads.Nodes {
			thread := ReviewThread{
				IsResolved: node.IsResolved,
				IsOutdated: node.IsOutdated,
				Path:       node.Path,
				Line:       node.Line,
				Count:      node.Comments.TotalCount,
			}
			for _, comment := range node.Comments.Nodes {
				thread.Comments = append(thread.Comments, ReviewThreadComment{
					Author:    comment.Author.Login,
					Body:      comment.Body,
					CreatedAt: comment.CreatedAt,
				})
			}
			threads = append(threads, thread)
		}

		info := resp.Data.Repository.PullRequest.ReviewThreads.PageInfo
		if !info.HasNextPage || info.EndCursor == nil || strings.TrimSpace(*info.EndCursor) == "" {
			break
		}
		after = *info.EndCursor
	}

	return threads, nil
}

func splitRepo(repo string) (string, string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo: %s", repo)
	}
	return parts[0], parts[1], nil
}
