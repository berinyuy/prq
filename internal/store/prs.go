package store

import (
	"database/sql"
	"fmt"
)

func (s *Store) GetPR(id string) (PRState, error) {
	row := s.db.QueryRow(`
		SELECT id, repo, number, last_seen_head_sha, last_reviewed_head_sha, last_reviewed_at, last_submitted_at, snoozed_until, notes
		FROM prs
		WHERE id = ?
	`, id)
	var st PRState
	if err := row.Scan(
		&st.ID,
		&st.Repo,
		&st.Number,
		&st.LastSeenHeadSHA,
		&st.LastReviewedHeadSHA,
		&st.LastReviewedAt,
		&st.LastSubmittedAt,
		&st.SnoozedUntil,
		&st.Notes,
	); err != nil {
		if err == sql.ErrNoRows {
			return PRState{}, err
		}
		return PRState{}, fmt.Errorf("failed to read pr: %w", err)
	}
	return st, nil
}

func (s *Store) MarkReviewed(id string, headSHA string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	if headSHA == "" {
		return fmt.Errorf("headSHA is required")
	}
	res, err := s.db.Exec(`
		UPDATE prs
		SET last_reviewed_head_sha = ?, last_reviewed_at = datetime('now')
		WHERE id = ?
	`, headSHA, id)
	if err != nil {
		return fmt.Errorf("failed to mark reviewed: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) MarkSubmitted(id string) error {
	if id == "" {
		return fmt.Errorf("id is required")
	}
	res, err := s.db.Exec(`
		UPDATE prs
		SET last_submitted_at = datetime('now')
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("failed to mark submitted: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
