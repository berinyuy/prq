package github

import (
	"context"
	"encoding/json"
	"fmt"
)

type CompareResponse struct {
	AheadBy      int           `json:"ahead_by"`
	TotalCommits int           `json:"total_commits"`
	Files        []CompareFile `json:"files"`
}

type CompareFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
}

func (c *Client) CompareCommits(ctx context.Context, repo, base, head string) (CompareResponse, error) {
	args := []string{"api", fmt.Sprintf("repos/%s/compare/%s...%s", repo, base, head)}
	output, err := c.Runner.Run(ctx, args, nil)
	if err != nil {
		return CompareResponse{}, err
	}
	var resp CompareResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return CompareResponse{}, fmt.Errorf("failed to decode compare response: %w", err)
	}
	return resp, nil
}
