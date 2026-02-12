package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// ReviewStore implements store.ReviewStore backed by SQLite.
type ReviewStore struct {
	db *sql.DB
}

func (s *ReviewStore) Create(ctx context.Context, review *types.Review) error {
	now := time.Now()
	if review.CreatedAt.IsZero() {
		review.CreatedAt = now
	}
	if review.UpdatedAt.IsZero() {
		review.UpdatedAt = review.CreatedAt
	}
	if review.Source == "" {
		review.Source = "user"
	}
	if review.Rating < 0 {
		review.Rating = 0
	}
	if review.Rating > 5 {
		review.Rating = 5
	}

	// Single-user app behavior: keep exactly one local review per book.
	if review.Source == "user" {
		existing, err := s.GetUserReview(ctx, review.BookID)
		if err != nil {
			return err
		}
		if existing != nil {
			review.ID = existing.ID
			return s.Update(ctx, review)
		}
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO reviews (book_id, rating, text, is_spoiler, likes_count, started_at, finished_at, created_at, updated_at, reviewer_name, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		review.BookID, review.Rating, review.Text, boolToInt(review.IsSpoiler),
		review.LikesCount, review.StartedAt, review.FinishedAt, review.CreatedAt, review.UpdatedAt,
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
		SELECT id, book_id, rating, text, is_spoiler, likes_count, comments_count, started_at, finished_at, created_at, updated_at, reviewer_name, source
		FROM reviews WHERE id = ?`, id))
}

func (s *ReviewStore) GetByBook(ctx context.Context, bookID int64, page, limit int) ([]types.Review, int, error) {
	return s.GetByBookFiltered(ctx, bookID, types.ReviewQuery{
		Page:  page,
		Limit: limit,
		Sort:  "popular",
	})
}

func (s *ReviewStore) GetByBookFiltered(ctx context.Context, bookID int64, q types.ReviewQuery) ([]types.Review, int, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	offset := (q.Page - 1) * q.Limit

	where := []string{"book_id = ?"}
	args := []any{bookID}

	if q.Rating >= 1 && q.Rating <= 5 {
		where = append(where, "rating = ?")
		args = append(args, q.Rating)
	}
	if q.Source == "user" || q.Source == "goodreads" {
		where = append(where, "source = ?")
		args = append(args, q.Source)
	}
	if q.HasText != nil {
		if *q.HasText {
			where = append(where, "LENGTH(TRIM(text)) > 0")
		} else {
			where = append(where, "LENGTH(TRIM(text)) = 0")
		}
	}
	if !q.IncludeSpoilers {
		where = append(where, "is_spoiler = 0")
	}
	if strings.TrimSpace(q.Query) != "" {
		pat := "%" + strings.TrimSpace(q.Query) + "%"
		where = append(where, "(text LIKE ? OR reviewer_name LIKE ?)")
		args = append(args, pat, pat)
	}

	orderBy := "likes_count DESC, created_at DESC"
	switch q.Sort {
	case "newest":
		orderBy = "created_at DESC"
	case "oldest":
		orderBy = "created_at ASC"
	case "rating_desc":
		orderBy = "rating DESC, created_at DESC"
	case "rating_asc":
		orderBy = "rating ASC, created_at DESC"
	}

	whereSQL := strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM reviews WHERE %s`, whereSQL), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(args, q.Limit, offset)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, book_id, rating, text, is_spoiler, likes_count, comments_count, started_at, finished_at, created_at, updated_at, reviewer_name, source
		FROM reviews WHERE %s ORDER BY %s LIMIT ? OFFSET ?`, whereSQL, orderBy), listArgs...)
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
	if review.Rating < 0 {
		review.Rating = 0
	}
	if review.Rating > 5 {
		review.Rating = 5
	}
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
		SELECT id, book_id, rating, text, is_spoiler, likes_count, comments_count, started_at, finished_at, created_at, updated_at, reviewer_name, source
		FROM reviews WHERE book_id = ? AND source = 'user' ORDER BY created_at DESC LIMIT 1`, bookID))
}

// updateBookRating recalculates the average rating and ratings count for a book.
func (s *ReviewStore) updateBookRating(ctx context.Context, bookID int64) error {
	var avgRating float64
	var ratingsCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(AVG(CAST(rating AS REAL)), 0), COUNT(*)
		FROM reviews WHERE book_id = ? AND rating > 0`, bookID).Scan(&avgRating, &ratingsCount); err != nil {
		return err
	}

	var reviewsCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reviews WHERE book_id = ?`, bookID).Scan(&reviewsCount); err != nil {
		return err
	}

	dist := [5]int{}
	rows, err := s.db.QueryContext(ctx, `
		SELECT rating, COUNT(*) FROM reviews WHERE book_id = ? AND rating > 0 GROUP BY rating`, bookID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var rating, count int
		if err := rows.Scan(&rating, &count); err != nil {
			return err
		}
		if rating >= 1 && rating <= 5 {
			dist[5-rating] = count
		}
	}
	rdistJSON, _ := json.Marshal(dist)

	_, err = s.db.ExecContext(ctx, `
		UPDATE books SET
			average_rating = ?,
			ratings_count = ?,
			reviews_count = ?,
			rating_dist = ?
		WHERE id = ?`, avgRating, ratingsCount, reviewsCount, string(rdistJSON), bookID)
	return err
}

func (s *ReviewStore) scanReview(row *sql.Row) (*types.Review, error) {
	var r types.Review
	var isSpoiler int
	err := row.Scan(&r.ID, &r.BookID, &r.Rating, &r.Text, &isSpoiler,
		&r.LikesCount, &r.CommentsCount, &r.StartedAt, &r.FinishedAt, &r.CreatedAt, &r.UpdatedAt,
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
			&r.LikesCount, &r.CommentsCount, &r.StartedAt, &r.FinishedAt, &r.CreatedAt, &r.UpdatedAt,
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

func (s *ReviewStore) AddLike(ctx context.Context, reviewID int64) (int, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE reviews SET likes_count = likes_count + 1 WHERE id = ?`, reviewID)
	if err != nil {
		return 0, err
	}
	var likes int
	if err := s.db.QueryRowContext(ctx, `SELECT likes_count FROM reviews WHERE id = ?`, reviewID).Scan(&likes); err != nil {
		return 0, err
	}
	return likes, nil
}
