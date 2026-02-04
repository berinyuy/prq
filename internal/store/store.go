package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create store dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS prs (
			id TEXT PRIMARY KEY,
			repo TEXT NOT NULL,
			number INTEGER NOT NULL,
			last_seen_head_sha TEXT NOT NULL,
			last_reviewed_head_sha TEXT,
			last_reviewed_at DATETIME,
			last_submitted_at DATETIME,
			snoozed_until DATETIME,
			notes TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS draft_reviews (
			id TEXT PRIMARY KEY,
			pr_id TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			payload_json TEXT NOT NULL,
			rendered_preview TEXT NOT NULL
		);`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to migrate db: %w", err)
		}
	}
	return nil
}

type PRState struct {
	ID                  string
	Repo                string
	Number              int
	LastSeenHeadSHA     string
	LastReviewedHeadSHA sql.NullString
	LastReviewedAt      sql.NullTime
	LastSubmittedAt     sql.NullTime
	SnoozedUntil        sql.NullTime
	Notes               sql.NullString
}

func (s *Store) UpsertPR(id, repo string, number int, headSHA string) error {
	_, err := s.db.Exec(`
		INSERT INTO prs (id, repo, number, last_seen_head_sha)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			repo = excluded.repo,
			number = excluded.number,
			last_seen_head_sha = excluded.last_seen_head_sha
	`, id, repo, number, headSHA)
	if err != nil {
		return fmt.Errorf("failed to upsert pr: %w", err)
	}
	return nil
}
