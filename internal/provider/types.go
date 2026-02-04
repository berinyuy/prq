package provider

type ReviewPlan struct {
	Summary         string   `json:"summary"`
	RiskLevel       string   `json:"risk_level"`
	Decision        string   `json:"decision"`
	KeyChanges      []string `json:"key_changes"`
	Issues          []Issue  `json:"issues"`
	Questions       []string `json:"questions"`
	Praise          []string `json:"praise"`
	DraftReviewBody string   `json:"draft_review_body"`
}

type Issue struct {
	Severity        string  `json:"severity"`
	Category        string  `json:"category"`
	File            string  `json:"file"`
	StartLine       int     `json:"start_line"`
	EndLine         int     `json:"end_line"`
	Message         string  `json:"message"`
	SuggestionPatch string  `json:"suggestion_patch,omitempty"`
	Confidence      float64 `json:"confidence,omitempty"`
}
