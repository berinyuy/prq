package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestDraftReviewCRUDAndPRState(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "prq.db"))
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	prID := "acme/app#1"
	if err := st.UpsertPR(prID, "acme/app", 1, "headsha1"); err != nil {
		t.Fatalf("upsert pr: %v", err)
	}
	if err := st.MarkReviewed(prID, "headsha1"); err != nil {
		t.Fatalf("mark reviewed: %v", err)
	}
	if err := st.MarkSubmitted(prID); err != nil {
		t.Fatalf("mark submitted: %v", err)
	}
	pr, err := st.GetPR(prID)
	if err != nil {
		t.Fatalf("get pr: %v", err)
	}
	if !pr.LastReviewedHeadSHA.Valid || pr.LastReviewedHeadSHA.String != "headsha1" {
		t.Fatalf("unexpected last_reviewed_head_sha: %#v", pr.LastReviewedHeadSHA)
	}
	if !pr.LastReviewedAt.Valid {
		t.Fatalf("expected last_reviewed_at to be set")
	}
	if !pr.LastSubmittedAt.Valid {
		t.Fatalf("expected last_submitted_at to be set")
	}

	if err := st.UpsertDraftReview(prID, `{"ok":true}`, "preview"); err != nil {
		t.Fatalf("upsert draft: %v", err)
	}
	dr, err := st.GetDraftReview(prID)
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}
	if dr.PRID != prID {
		t.Fatalf("unexpected pr_id: %q", dr.PRID)
	}
	if dr.PayloadJSON != `{"ok":true}` {
		t.Fatalf("unexpected payload: %q", dr.PayloadJSON)
	}
	if dr.RenderedPreview != "preview" {
		t.Fatalf("unexpected preview: %q", dr.RenderedPreview)
	}

	if err := st.DeleteDraftReview(prID); err != nil {
		t.Fatalf("delete draft: %v", err)
	}
	_, err = st.GetDraftReview(prID)
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}
