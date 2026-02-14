package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// ChallengeStore implements store.ChallengeStore backed by SQLite.
type ChallengeStore struct {
	db *sql.DB
}

// Set creates or updates a reading challenge for the given year.
func (s *ChallengeStore) Set(ctx context.Context, ch *types.ReadingChallenge) error {
	if ch.CreatedAt.IsZero() {
		ch.CreatedAt = time.Now()
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO reading_challenges (year, goal, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(year) DO UPDATE SET goal = excluded.goal`,
		ch.Year, ch.Goal, ch.CreatedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if ch.ID == 0 {
		ch.ID = id
	}
	return nil
}

func (s *ChallengeStore) Get(ctx context.Context, year int) (*types.ReadingChallenge, error) {
	var ch types.ReadingChallenge
	err := s.db.QueryRowContext(ctx, `SELECT id, year, goal, created_at FROM reading_challenges WHERE year = ?`, year).
		Scan(&ch.ID, &ch.Year, &ch.Goal, &ch.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	// Compute progress
	progress, err := s.GetProgress(ctx, year)
	if err != nil {
		return nil, fmt.Errorf("get progress: %w", err)
	}
	ch.Progress = progress
	return &ch, nil
}

// GetProgress counts books in the "Read" shelf that have a review with finished_at
// in the given year.
func (s *ChallengeStore) GetProgress(ctx context.Context, year int) (int, error) {
	startDate := fmt.Sprintf("%d-01-01", year)
	endDate := fmt.Sprintf("%d-01-01", year+1)

	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT r.book_id) FROM reviews r
		JOIN shelf_books sb ON sb.book_id = r.book_id
		JOIN shelves s ON s.id = sb.shelf_id AND s.slug = 'read'
		WHERE r.finished_at >= ? AND r.finished_at < ?`,
		startDate, endDate).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
