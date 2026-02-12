package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// AuthorStore implements store.AuthorStore backed by SQLite.
type AuthorStore struct {
	db *sql.DB
}

func (s *AuthorStore) Create(ctx context.Context, author *types.Author) error {
	if author.CreatedAt.IsZero() {
		author.CreatedAt = time.Now()
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO authors (ol_key, name, bio, photo_url, birth_date, death_date, works_count, created_at,
			goodreads_id, followers, genres, influences, website)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		author.OLKey, author.Name, author.Bio, author.PhotoURL,
		author.BirthDate, author.DeathDate, author.WorksCount, author.CreatedAt,
		author.GoodreadsID, author.Followers, author.Genres, author.Influences, author.Website)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	author.ID = id
	return nil
}

func (s *AuthorStore) Get(ctx context.Context, id int64) (*types.Author, error) {
	return s.scanAuthor(s.db.QueryRowContext(ctx, `SELECT * FROM authors WHERE id = ?`, id))
}

func (s *AuthorStore) GetByOLKey(ctx context.Context, olKey string) (*types.Author, error) {
	return s.scanAuthor(s.db.QueryRowContext(ctx, `SELECT * FROM authors WHERE ol_key = ?`, olKey))
}

func (s *AuthorStore) Search(ctx context.Context, query string, limit int) ([]types.Author, error) {
	if limit <= 0 {
		limit = 20
	}
	likeQ := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx,
		`SELECT * FROM authors WHERE name LIKE ? ORDER BY works_count DESC LIMIT ?`, likeQ, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return s.scanAuthors(rows)
}

func (s *AuthorStore) GetBooks(ctx context.Context, authorID int64, page, limit int) (*types.SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Get the author's name to match against book author_names
	author, err := s.Get(ctx, authorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return &types.SearchResult{Books: []types.Book{}, Page: page, PageSize: limit}, nil
	}

	likeA := "%" + author.Name + "%"

	var total int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM books WHERE author_names LIKE ?`, likeA).Scan(&total)

	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf(`SELECT %s FROM books WHERE author_names LIKE ? ORDER BY publish_year DESC LIMIT ? OFFSET ?`, bookColumns("")),
		likeA, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bs := &BookStore{db: s.db}
	books, err := bs.scanBooks(rows)
	if err != nil {
		return nil, err
	}
	return &types.SearchResult{Books: books, TotalCount: total, Page: page, PageSize: limit}, nil
}

func (s *AuthorStore) GetByGoodreadsID(ctx context.Context, grID string) (*types.Author, error) {
	return s.scanAuthor(s.db.QueryRowContext(ctx, `SELECT * FROM authors WHERE goodreads_id = ?`, grID))
}

func (s *AuthorStore) Update(ctx context.Context, author *types.Author) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE authors SET name=?, bio=?, photo_url=?, birth_date=?, death_date=?,
			works_count=?, goodreads_id=?, followers=?, genres=?, influences=?, website=?
		WHERE id=?`,
		author.Name, author.Bio, author.PhotoURL, author.BirthDate, author.DeathDate,
		author.WorksCount, author.GoodreadsID, author.Followers, author.Genres, author.Influences, author.Website,
		author.ID)
	return err
}

func (s *AuthorStore) scanAuthor(row *sql.Row) (*types.Author, error) {
	var a types.Author
	err := row.Scan(&a.ID, &a.OLKey, &a.Name, &a.Bio, &a.PhotoURL,
		&a.BirthDate, &a.DeathDate, &a.WorksCount, &a.CreatedAt,
		&a.GoodreadsID, &a.Followers, &a.Genres, &a.Influences, &a.Website)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (s *AuthorStore) scanAuthors(rows *sql.Rows) ([]types.Author, error) {
	var authors []types.Author
	for rows.Next() {
		var a types.Author
		err := rows.Scan(&a.ID, &a.OLKey, &a.Name, &a.Bio, &a.PhotoURL,
			&a.BirthDate, &a.DeathDate, &a.WorksCount, &a.CreatedAt,
			&a.GoodreadsID, &a.Followers, &a.Genres, &a.Influences, &a.Website)
		if err != nil {
			return nil, fmt.Errorf("scan author: %w", err)
		}
		authors = append(authors, a)
	}
	if authors == nil {
		authors = []types.Author{}
	}
	return authors, nil
}
