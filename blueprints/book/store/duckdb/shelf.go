package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// ShelfStore implements store.ShelfStore backed by SQLite.
type ShelfStore struct {
	db *sql.DB
}

func (s *ShelfStore) Create(ctx context.Context, shelf *types.Shelf) error {
	if shelf.CreatedAt.IsZero() {
		shelf.CreatedAt = time.Now()
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO shelves (name, slug, is_exclusive, is_default, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		shelf.Name, shelf.Slug, boolToInt(shelf.IsExclusive), boolToInt(shelf.IsDefault),
		shelf.SortOrder, shelf.CreatedAt)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	shelf.ID = id
	return nil
}

func (s *ShelfStore) Get(ctx context.Context, id int64) (*types.Shelf, error) {
	return s.scanShelf(s.db.QueryRowContext(ctx, `
		SELECT s.*, COALESCE(c.cnt, 0) FROM shelves s
		LEFT JOIN (SELECT shelf_id, COUNT(*) as cnt FROM shelf_books GROUP BY shelf_id) c ON c.shelf_id = s.id
		WHERE s.id = ?`, id))
}

func (s *ShelfStore) GetBySlug(ctx context.Context, slug string) (*types.Shelf, error) {
	return s.scanShelf(s.db.QueryRowContext(ctx, `
		SELECT s.*, COALESCE(c.cnt, 0) FROM shelves s
		LEFT JOIN (SELECT shelf_id, COUNT(*) as cnt FROM shelf_books GROUP BY shelf_id) c ON c.shelf_id = s.id
		WHERE s.slug = ?`, slug))
}

func (s *ShelfStore) List(ctx context.Context) ([]types.Shelf, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.*, COALESCE(c.cnt, 0) FROM shelves s
		LEFT JOIN (SELECT shelf_id, COUNT(*) as cnt FROM shelf_books GROUP BY shelf_id) c ON c.shelf_id = s.id
		ORDER BY s.sort_order, s.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shelves []types.Shelf
	for rows.Next() {
		var sh types.Shelf
		var isExcl, isDef int
		err := rows.Scan(&sh.ID, &sh.Name, &sh.Slug, &isExcl, &isDef,
			&sh.SortOrder, &sh.CreatedAt, &sh.BookCount)
		if err != nil {
			return nil, fmt.Errorf("scan shelf: %w", err)
		}
		sh.IsExclusive = isExcl != 0
		sh.IsDefault = isDef != 0
		shelves = append(shelves, sh)
	}
	if shelves == nil {
		shelves = []types.Shelf{}
	}
	return shelves, nil
}

func (s *ShelfStore) Update(ctx context.Context, shelf *types.Shelf) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE shelves SET name=?, slug=?, is_exclusive=?, is_default=?, sort_order=?
		WHERE id=?`,
		shelf.Name, shelf.Slug, boolToInt(shelf.IsExclusive), boolToInt(shelf.IsDefault),
		shelf.SortOrder, shelf.ID)
	return err
}

func (s *ShelfStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM shelves WHERE id = ? AND is_default = 0`, id)
	return err
}

// AddBook adds a book to a shelf. If the shelf is exclusive, the book is first
// removed from all other exclusive shelves.
func (s *ShelfStore) AddBook(ctx context.Context, shelfID, bookID int64) error {
	// Check if target shelf is exclusive
	var isExcl int
	err := s.db.QueryRowContext(ctx, `SELECT is_exclusive FROM shelves WHERE id = ?`, shelfID).Scan(&isExcl)
	if err != nil {
		return err
	}

	if isExcl != 0 {
		// Remove from all other exclusive shelves
		_, err = s.db.ExecContext(ctx, `
			DELETE FROM shelf_books WHERE book_id = ? AND shelf_id IN (
				SELECT id FROM shelves WHERE is_exclusive = 1
			)`, bookID)
		if err != nil {
			return err
		}
	}

	// Determine next position
	var maxPos int
	s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(position), 0) FROM shelf_books WHERE shelf_id = ?`, shelfID).Scan(&maxPos)

	now := time.Now()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO shelf_books (shelf_id, book_id, date_added, position)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (shelf_id, book_id) DO UPDATE SET date_added = EXCLUDED.date_added, position = EXCLUDED.position`,
		shelfID, bookID, now, maxPos+1)
	return err
}

func (s *ShelfStore) RemoveBook(ctx context.Context, shelfID, bookID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM shelf_books WHERE shelf_id = ? AND book_id = ?`, shelfID, bookID)
	return err
}

func (s *ShelfStore) GetBooks(ctx context.Context, shelfID int64, sort string, page, limit int) ([]types.ShelfBook, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM shelf_books WHERE shelf_id = ?`, shelfID).Scan(&total)

	orderBy := "sb.date_added DESC"
	switch sort {
	case "title":
		orderBy = "b.title ASC"
	case "author":
		orderBy = "b.author_names ASC"
	case "rating":
		orderBy = "b.average_rating DESC"
	case "date_added":
		orderBy = "sb.date_added DESC"
	case "position":
		orderBy = "sb.position ASC"
	case "date_read":
		orderBy = "sb.date_read DESC NULLS LAST"
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT sb.id, sb.shelf_id, sb.book_id, sb.date_added, sb.position,
			sb.date_started, sb.date_read, sb.read_count,
			%s
		FROM shelf_books sb
		JOIN books b ON b.id = sb.book_id
		WHERE sb.shelf_id = ?
		ORDER BY %s
		LIMIT ? OFFSET ?`, bookColumns("b"), orderBy), shelfID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []types.ShelfBook
	for rows.Next() {
		var sb types.ShelfBook
		var b types.Book
		fields := append([]any{&sb.ID, &sb.ShelfID, &sb.BookID, &sb.DateAdded, &sb.Position,
			&sb.DateStarted, &sb.DateRead, &sb.ReadCount}, scanFields(&b)...)
		err := rows.Scan(fields...)
		if err != nil {
			return nil, 0, fmt.Errorf("scan shelf book: %w", err)
		}
		hydrateBook(&b)
		sb.Book = &b
		items = append(items, sb)
	}
	if items == nil {
		items = []types.ShelfBook{}
	}
	return items, total, nil
}

func (s *ShelfStore) UpdateShelfBook(ctx context.Context, shelfID, bookID int64, dateStarted, dateRead *string, readCount *int) error {
	sets := []string{}
	args := []any{}
	if dateStarted != nil {
		if *dateStarted == "" {
			sets = append(sets, "date_started = NULL")
		} else {
			sets = append(sets, "date_started = ?")
			args = append(args, *dateStarted)
		}
	}
	if dateRead != nil {
		if *dateRead == "" {
			sets = append(sets, "date_read = NULL")
		} else {
			sets = append(sets, "date_read = ?")
			args = append(args, *dateRead)
		}
	}
	if readCount != nil {
		sets = append(sets, "read_count = ?")
		args = append(args, *readCount)
	}
	if len(sets) == 0 {
		return nil
	}
	query := fmt.Sprintf("UPDATE shelf_books SET %s WHERE shelf_id = ? AND book_id = ?",
		joinStrings(sets, ", "))
	args = append(args, shelfID, bookID)
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func joinStrings(s []string, sep string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += sep
		}
		result += v
	}
	return result
}

func (s *ShelfStore) GetBookShelves(ctx context.Context, bookID int64) ([]types.Shelf, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.*, 0 FROM shelves s
		JOIN shelf_books sb ON sb.shelf_id = s.id
		WHERE sb.book_id = ?
		ORDER BY s.sort_order`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shelves []types.Shelf
	for rows.Next() {
		var sh types.Shelf
		var isExcl, isDef int
		err := rows.Scan(&sh.ID, &sh.Name, &sh.Slug, &isExcl, &isDef,
			&sh.SortOrder, &sh.CreatedAt, &sh.BookCount)
		if err != nil {
			return nil, fmt.Errorf("scan shelf: %w", err)
		}
		sh.IsExclusive = isExcl != 0
		sh.IsDefault = isDef != 0
		shelves = append(shelves, sh)
	}
	if shelves == nil {
		shelves = []types.Shelf{}
	}
	return shelves, nil
}

// SeedDefaults creates the three default exclusive shelves if they do not exist.
func (s *ShelfStore) SeedDefaults(ctx context.Context) error {
	defaults := []struct {
		name      string
		slug      string
		sortOrder int
	}{
		{"Read", "read", 1},
		{"Currently Reading", "currently-reading", 2},
		{"Want to Read", "want-to-read", 3},
	}

	for _, d := range defaults {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO shelves (name, slug, is_exclusive, is_default, sort_order)
			VALUES (?, ?, 1, 1, ?)
			ON CONFLICT(slug) DO NOTHING`, d.name, d.slug, d.sortOrder)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ShelfStore) scanShelf(row *sql.Row) (*types.Shelf, error) {
	var sh types.Shelf
	var isExcl, isDef int
	err := row.Scan(&sh.ID, &sh.Name, &sh.Slug, &isExcl, &isDef,
		&sh.SortOrder, &sh.CreatedAt, &sh.BookCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	sh.IsExclusive = isExcl != 0
	sh.IsDefault = isDef != 0
	return &sh, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
