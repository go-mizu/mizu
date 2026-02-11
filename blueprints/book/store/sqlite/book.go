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

// BookStore implements store.BookStore backed by SQLite.
type BookStore struct {
	db *sql.DB
}

func (s *BookStore) Create(ctx context.Context, book *types.Book) error {
	subj, _ := json.Marshal(book.Subjects)
	book.SubjectsJSON = string(subj)
	rdist, _ := json.Marshal(book.RatingDist)
	book.RatingDistJSON = string(rdist)
	now := time.Now()
	book.CreatedAt = now
	book.UpdatedAt = now

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO books (ol_key, google_id, title, subtitle, description, author_names,
			cover_url, cover_id, isbn10, isbn13, publisher, publish_date, publish_year,
			page_count, language, format, subjects_json, average_rating, ratings_count,
			created_at, updated_at, goodreads_id, asin, series, reviews_count,
			currently_reading, want_to_read, rating_dist, first_published)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?)`,
		book.OLKey, book.GoogleID, book.Title, book.Subtitle, book.Description, book.AuthorNames,
		book.CoverURL, book.CoverID, book.ISBN10, book.ISBN13, book.Publisher, book.PublishDate,
		book.PublishYear, book.PageCount, book.Language, book.Format, book.SubjectsJSON,
		book.AverageRating, book.RatingsCount, now, now,
		book.GoodreadsID, book.ASIN, book.Series, book.ReviewsCount,
		book.CurrentlyReading, book.WantToRead, book.RatingDistJSON, book.FirstPublished)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	book.ID = id

	// Update FTS index
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO books_fts(rowid, title, author_names, description, subjects_json) VALUES (?, ?, ?, ?, ?)`,
		id, book.Title, book.AuthorNames, book.Description, book.SubjectsJSON)
	return err
}

func (s *BookStore) Get(ctx context.Context, id int64) (*types.Book, error) {
	return s.scanBook(s.db.QueryRowContext(ctx, `SELECT * FROM books WHERE id = ?`, id))
}

func (s *BookStore) GetByISBN(ctx context.Context, isbn string) (*types.Book, error) {
	isbn = strings.ReplaceAll(isbn, "-", "")
	if len(isbn) == 13 {
		return s.scanBook(s.db.QueryRowContext(ctx, `SELECT * FROM books WHERE isbn13 = ?`, isbn))
	}
	return s.scanBook(s.db.QueryRowContext(ctx, `SELECT * FROM books WHERE isbn10 = ?`, isbn))
}

func (s *BookStore) GetByOLKey(ctx context.Context, olKey string) (*types.Book, error) {
	return s.scanBook(s.db.QueryRowContext(ctx, `SELECT * FROM books WHERE ol_key = ?`, olKey))
}

func (s *BookStore) Search(ctx context.Context, query string, page, limit int) (*types.SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Empty query: return all books
	if strings.TrimSpace(query) == "" {
		var total int
		s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books`).Scan(&total)
		rows, err := s.db.QueryContext(ctx,
			`SELECT * FROM books ORDER BY ratings_count DESC LIMIT ? OFFSET ?`, limit, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		books, err := s.scanBooks(rows)
		if err != nil {
			return nil, err
		}
		return &types.SearchResult{Books: books, TotalCount: total, Page: page, PageSize: limit}, nil
	}

	// Try FTS search first
	var total int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books_fts WHERE books_fts MATCH ?`, query).Scan(&total)
	if err != nil {
		// Fallback to LIKE search
		likeQ := "%" + query + "%"
		err = s.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM books WHERE title LIKE ? OR author_names LIKE ?`, likeQ, likeQ).Scan(&total)
		if err != nil {
			return nil, err
		}

		rows, err := s.db.QueryContext(ctx,
			`SELECT * FROM books WHERE title LIKE ? OR author_names LIKE ? ORDER BY ratings_count DESC LIMIT ? OFFSET ?`,
			likeQ, likeQ, limit, offset)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		books, err := s.scanBooks(rows)
		if err != nil {
			return nil, err
		}
		return &types.SearchResult{Books: books, TotalCount: total, Page: page, PageSize: limit}, nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT b.* FROM books b
		JOIN books_fts f ON b.id = f.rowid
		WHERE books_fts MATCH ?
		ORDER BY rank
		LIMIT ? OFFSET ?`, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books, err := s.scanBooks(rows)
	if err != nil {
		return nil, err
	}
	return &types.SearchResult{Books: books, TotalCount: total, Page: page, PageSize: limit}, nil
}

func (s *BookStore) Update(ctx context.Context, book *types.Book) error {
	subj, _ := json.Marshal(book.Subjects)
	book.SubjectsJSON = string(subj)
	rdist, _ := json.Marshal(book.RatingDist)
	book.RatingDistJSON = string(rdist)
	book.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		UPDATE books SET ol_key=?, google_id=?, title=?, subtitle=?, description=?,
			author_names=?, cover_url=?, cover_id=?, isbn10=?, isbn13=?, publisher=?,
			publish_date=?, publish_year=?, page_count=?, language=?, format=?,
			subjects_json=?, average_rating=?, ratings_count=?, updated_at=?,
			goodreads_id=?, asin=?, series=?, reviews_count=?,
			currently_reading=?, want_to_read=?, rating_dist=?, first_published=?
		WHERE id=?`,
		book.OLKey, book.GoogleID, book.Title, book.Subtitle, book.Description,
		book.AuthorNames, book.CoverURL, book.CoverID, book.ISBN10, book.ISBN13,
		book.Publisher, book.PublishDate, book.PublishYear, book.PageCount, book.Language,
		book.Format, book.SubjectsJSON, book.AverageRating, book.RatingsCount,
		book.UpdatedAt,
		book.GoodreadsID, book.ASIN, book.Series, book.ReviewsCount,
		book.CurrentlyReading, book.WantToRead, book.RatingDistJSON, book.FirstPublished,
		book.ID)
	if err != nil {
		return err
	}

	// Update FTS
	_, _ = s.db.ExecContext(ctx, `DELETE FROM books_fts WHERE rowid = ?`, book.ID)
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO books_fts(rowid, title, author_names, description, subjects_json) VALUES (?, ?, ?, ?, ?)`,
		book.ID, book.Title, book.AuthorNames, book.Description, book.SubjectsJSON)
	return err
}

func (s *BookStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM books WHERE id = ?`, id)
	if err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `DELETE FROM books_fts WHERE rowid = ?`, id)
	return nil
}

func (s *BookStore) GetByGenre(ctx context.Context, genre string, page, limit int) (*types.SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	likeG := "%" + genre + "%"

	var total int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books WHERE subjects_json LIKE ?`, likeG).Scan(&total)

	rows, err := s.db.QueryContext(ctx,
		`SELECT * FROM books WHERE subjects_json LIKE ? ORDER BY ratings_count DESC LIMIT ? OFFSET ?`,
		likeG, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books, err := s.scanBooks(rows)
	if err != nil {
		return nil, err
	}
	return &types.SearchResult{Books: books, TotalCount: total, Page: page, PageSize: limit}, nil
}

func (s *BookStore) GetTrending(ctx context.Context, limit int) ([]types.Book, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT * FROM books ORDER BY updated_at DESC, ratings_count DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanBooks(rows)
}

func (s *BookStore) GetNewReleases(ctx context.Context, limit int) ([]types.Book, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT * FROM books ORDER BY publish_year DESC, created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanBooks(rows)
}

func (s *BookStore) GetPopular(ctx context.Context, limit int) ([]types.Book, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT * FROM books ORDER BY ratings_count DESC, average_rating DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanBooks(rows)
}

func (s *BookStore) GetSimilar(ctx context.Context, bookID int64, limit int) ([]types.Book, error) {
	if limit <= 0 {
		limit = 10
	}
	book, err := s.Get(ctx, bookID)
	if err != nil {
		return nil, err
	}
	if book == nil || len(book.Subjects) == 0 {
		return []types.Book{}, nil
	}

	// Try matching on multiple subjects for better results
	var conditions []string
	var args []any
	for i, subj := range book.Subjects {
		if i >= 3 {
			break
		}
		conditions = append(conditions, "subjects_json LIKE ?")
		args = append(args, "%"+subj+"%")
	}
	args = append(args, bookID, limit)

	query := fmt.Sprintf(
		`SELECT * FROM books WHERE (%s) AND id != ? ORDER BY ratings_count DESC LIMIT ?`,
		strings.Join(conditions, " OR "))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanBooks(rows)
}

func (s *BookStore) ListGenres(ctx context.Context) ([]types.Genre, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT subjects_json FROM books WHERE subjects_json != '[]'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var subj string
		if err := rows.Scan(&subj); err != nil {
			continue
		}
		var subjects []string
		if err := json.Unmarshal([]byte(subj), &subjects); err != nil {
			continue
		}
		for _, s := range subjects {
			counts[s]++
		}
	}

	var genres []types.Genre
	for name, count := range counts {
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		genres = append(genres, types.Genre{Name: name, Slug: slug, BookCount: count})
	}
	return genres, nil
}

func (s *BookStore) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books`).Scan(&count)
	return count, err
}

func (s *BookStore) GetByGoodreadsID(ctx context.Context, grID string) (*types.Book, error) {
	return s.scanBook(s.db.QueryRowContext(ctx, `SELECT * FROM books WHERE goodreads_id = ?`, grID))
}

func scanFields(b *types.Book) []any {
	return []any{
		&b.ID, &b.OLKey, &b.GoogleID, &b.Title, &b.Subtitle, &b.Description,
		&b.AuthorNames, &b.CoverURL, &b.CoverID, &b.ISBN10, &b.ISBN13, &b.Publisher,
		&b.PublishDate, &b.PublishYear, &b.PageCount, &b.Language, &b.Format,
		&b.SubjectsJSON, &b.AverageRating, &b.RatingsCount, &b.CreatedAt, &b.UpdatedAt,
		&b.GoodreadsID, &b.ASIN, &b.Series, &b.ReviewsCount,
		&b.CurrentlyReading, &b.WantToRead, &b.RatingDistJSON, &b.FirstPublished,
	}
}

func hydrateBook(b *types.Book) {
	json.Unmarshal([]byte(b.SubjectsJSON), &b.Subjects)
	json.Unmarshal([]byte(b.RatingDistJSON), &b.RatingDist)
	if b.AuthorNames != "" {
		for _, name := range strings.Split(b.AuthorNames, ", ") {
			b.Authors = append(b.Authors, types.Author{Name: name})
		}
	}
}

func (s *BookStore) scanBook(row *sql.Row) (*types.Book, error) {
	var b types.Book
	err := row.Scan(scanFields(&b)...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	hydrateBook(&b)
	return &b, nil
}

func (s *BookStore) scanBooks(rows *sql.Rows) ([]types.Book, error) {
	var books []types.Book
	for rows.Next() {
		var b types.Book
		err := rows.Scan(scanFields(&b)...)
		if err != nil {
			return nil, fmt.Errorf("scan book: %w", err)
		}
		hydrateBook(&b)
		books = append(books, b)
	}
	if books == nil {
		books = []types.Book{}
	}
	return books, nil
}
