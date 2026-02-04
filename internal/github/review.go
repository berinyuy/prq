package github

import (
	"context"
	"encoding/json"
	"fmt"
)

type ReviewComment struct {
	Path     string `json:"path"`
	Position int    `json:"position"`
	Body     string `json:"body"`
}

type CreateReviewRequest struct {
	Body     string          `json:"body,omitempty"`
	Event    string          `json:"event,omitempty"`
	Comments []ReviewComment `json:"comments,omitempty"`
}

type CreateReviewResponse struct {
	ID      int64  `json:"id"`
	HTMLURL string `json:"html_url"`
}

func (c *Client) CreateReview(ctx context.Context, repo string, number int, req CreateReviewRequest) (CreateReviewResponse, error) {
	endpoint := fmt.Sprintf("repos/%s/pulls/%d/reviews", repo, number)
	payload, err := json.Marshal(req)
	if err != nil {
		return CreateReviewResponse{}, fmt.Errorf("marshal create review request: %w", err)
	}
	args := []string{"api", "-X", "POST", endpoint, "--input", "-"}
	output, err := c.Runner.Run(ctx, args, payload)
	if err != nil {
		return CreateReviewResponse{}, err
	}
	if len(output) == 0 {
		return CreateReviewResponse{}, nil
	}
	var resp CreateReviewResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return CreateReviewResponse{}, fmt.Errorf("decode create review output: %w", err)
	}
	return resp, nil
}
