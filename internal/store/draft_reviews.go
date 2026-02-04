package store

import (
	"database/sql"
	"fmt"
	"time"
)

type DraftReview struct {
	ID              string
	PRID            string
	CreatedAt       time.Time
	PayloadJSON     string
	RenderedPreview string
}

func (s *Store) UpsertDraftReview(prID string, payloadJSON string, renderedPreview string) error {
	if prID == "" {
		return fmt.Errorf("prID is required")
	}
	if payloadJSON == "" {
		return fmt.Errorf("payloadJSON is required")
	}
	if renderedPreview == "" {
		return fmt.Errorf("renderedPreview is required")
	}
	_, err := s.db.Exec(`
		INSERT INTO draft_reviews (id, pr_id, created_at, payload_json, rendered_preview)
		VALUES (?, ?, datetime('now'), ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			pr_id = excluded.pr_id,
			created_at = excluded.created_at,
			payload_json = excluded.payload_json,
			rendered_preview = excluded.rendered_preview
	`, prID, prID, payloadJSON, renderedPreview)
	if err != nil {
		return fmt.Errorf("failed to upsert draft review: %w", err)
	}
	return nil
}

func (s *Store) GetDraftReview(prID string) (DraftReview, error) {
	row := s.db.QueryRow(`
		SELECT id, pr_id, created_at, payload_json, rendered_preview
		FROM draft_reviews
		WHERE id = ?
	`, prID)
	if dr, err := scanDraftReview(row); err == nil {
		return dr, nil
	} else if err != sql.ErrNoRows {
		return DraftReview{}, err
	}

	row = s.db.QueryRow(`
		SELECT id, pr_id, created_at, payload_json, rendered_preview
		FROM draft_reviews
		WHERE pr_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, prID)
	return scanDraftReview(row)
}

func (s *Store) DeleteDraftReview(prID string) error {
	_, err := s.db.Exec(`DELETE FROM draft_reviews WHERE id = ? OR pr_id = ?`, prID, prID)
	if err != nil {
		return fmt.Errorf("failed to delete draft review: %w", err)
	}
	return nil
}

func scanDraftReview(row *sql.Row) (DraftReview, error) {
	var dr DraftReview
	var createdAtStr string
	if err := row.Scan(&dr.ID, &dr.PRID, &createdAtStr, &dr.PayloadJSON, &dr.RenderedPreview); err != nil {
		if err == sql.ErrNoRows {
			return DraftReview{}, err
		}
		return DraftReview{}, fmt.Errorf("failed to read draft review: %w", err)
	}
	parsed, err := time.Parse("2006-01-02 15:04:05", createdAtStr)
	if err == nil {
		dr.CreatedAt = parsed
	}
	return dr, nil
}
