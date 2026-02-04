package cli

import "github.com/brianndofor/prq/internal/provider"

type DraftReviewPayload struct {
	Repo    string              `json:"repo"`
	Number  int                 `json:"number"`
	BaseSHA string              `json:"base_sha"`
	HeadSHA string              `json:"head_sha"`
	Plan    provider.ReviewPlan `json:"plan"`
}
