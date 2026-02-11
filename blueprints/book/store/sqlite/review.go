package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// ReviewStore implements store.ReviewStore backed by SQLite.
type ReviewStore struct {
	db *sql.DB
}

func (s *ReviewStore) Create(ctx context.Context, review *types.Review) error {
	now := time.Now()
	review.CreatedAt = now
	review.UpdatedAt = now
	if review.Source == "" {
		review.Source = "user"
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO reviews (book_id, rating, text, is_spoiler, likes_count, started_at, finished_at, created_at, updated_at, reviewer_name, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		review.BookID, review.Rating, review.Text, boolToInt(review.IsSpoiler),
		review.LikesCount, review.StartedAt, review.FinishedAt, now, now,
		review.ReviewerName, review.Source)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	review.ID = id

	return s.updateBookRating(ctx, review.BookID)
}

func (s *ReviewStore) Get(ctx context.Context, id int64) (*types.Review, error) {
	return s.scanReview(s.db.QueryRowContext(ctx, `
		SELECT id, book_id, rating, text, is_spoiler, likes_count, started_at, finished_at, created_at, updated_at, reviewer_name, source
		FROM reviews WHERE id = ?`, id))
}

func (s *ReviewStore) GetByBook(ctx context.Context, bookID int64, page, limit int) ([]types.Review, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reviews WHERE book_id = ?`, bookID).Scan(&total)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, book_id, rating, text, is_spoiler, likes_count, started_at, finished_at, created_at, updated_at, reviewer_name, source
		FROM reviews WHERE book_id = ? ORDER BY likes_count DESC, created_at DESC LIMIT ? OFFSET ?`,
		bookID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	reviews, err := s.scanReviews(rows)
	if err != nil {
		return nil, 0, err
	}
	return reviews, total, nil
}

func (s *ReviewStore) Update(ctx context.Context, review *types.Review) error {
	review.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE reviews SET rating=?, text=?, is_spoiler=?, started_at=?, finished_at=?, updated_at=?
		WHERE id=?`,
		review.Rating, review.Text, boolToInt(review.IsSpoiler),
		review.StartedAt, review.FinishedAt, review.UpdatedAt, review.ID)
	if err != nil {
		return err
	}
	return s.updateBookRating(ctx, review.BookID)
}

func (s *ReviewStore) Delete(ctx context.Context, id int64) error {
	// Get book_id before deleting
	var bookID int64
	err := s.db.QueryRowContext(ctx, `SELECT book_id FROM reviews WHERE id = ?`, id).Scan(&bookID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM reviews WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return s.updateBookRating(ctx, bookID)
}

// GetUserReview returns the single user review for a book (single-user app).
func (s *ReviewStore) GetUserReview(ctx context.Context, bookID int64) (*types.Review, error) {
	return s.scanReview(s.db.QueryRowContext(ctx, `
		SELECT id, book_id, rating, text, is_spoiler, likes_count, started_at, finished_at, created_at, updated_at, reviewer_name, source
		FROM reviews WHERE book_id = ? AND source = 'user' ORDER BY created_at DESC LIMIT 1`, bookID))
}

// updateBookRating recalculates the average rating and ratings count for a book.
func (s *ReviewStore) updateBookRating(ctx context.Context, bookID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE books SET
			average_rating = COALESCE((SELECT AVG(CAST(rating AS REAL)) FROM reviews WHERE book_id = ? AND rating > 0), 0),
			ratings_count = (SELECT COUNT(*) FROM reviews WHERE book_id = ? AND rating > 0)
		WHERE id = ?`, bookID, bookID, bookID)
	return err
}

func (s *ReviewStore) scanReview(row *sql.Row) (*types.Review, error) {
	var r types.Review
	var isSpoiler int
	err := row.Scan(&r.ID, &r.BookID, &r.Rating, &r.Text, &isSpoiler,
		&r.LikesCount, &r.StartedAt, &r.FinishedAt, &r.CreatedAt, &r.UpdatedAt,
		&r.ReviewerName, &r.Source)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	r.IsSpoiler = isSpoiler != 0
	return &r, nil
}

func (s *ReviewStore) scanReviews(rows *sql.Rows) ([]types.Review, error) {
	var reviews []types.Review
	for rows.Next() {
		var r types.Review
		var isSpoiler int
		err := rows.Scan(&r.ID, &r.BookID, &r.Rating, &r.Text, &isSpoiler,
			&r.LikesCount, &r.StartedAt, &r.FinishedAt, &r.CreatedAt, &r.UpdatedAt,
			&r.ReviewerName, &r.Source)
		if err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		r.IsSpoiler = isSpoiler != 0
		reviews = append(reviews, r)
	}
	if reviews == nil {
		reviews = []types.Review{}
	}
	return reviews, nil
}
