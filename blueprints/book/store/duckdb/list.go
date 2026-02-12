package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// ListStore implements store.ListStore backed by SQLite.
type ListStore struct {
	db *sql.DB
}

func (s *ListStore) Create(ctx context.Context, list *types.BookList) error {
	if list.CreatedAt.IsZero() {
		list.CreatedAt = time.Now()
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO book_lists (title, description, created_at, goodreads_url, voter_count)
		VALUES (?, ?, ?, ?, ?)`,
		list.Title, list.Description, list.CreatedAt, list.GoodreadsURL, list.VoterCount)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	list.ID = id
	return nil
}

func (s *ListStore) Get(ctx context.Context, id int64) (*types.BookList, error) {
	var list types.BookList
	err := s.db.QueryRowContext(ctx, `
		SELECT l.id, l.title, l.description, l.created_at, COALESCE(c.cnt, 0), l.goodreads_url, l.voter_count
		FROM book_lists l
		LEFT JOIN (SELECT list_id, COUNT(*) as cnt FROM book_list_items GROUP BY list_id) c ON c.list_id = l.id
		WHERE l.id = ?`, id).
		Scan(&list.ID, &list.Title, &list.Description, &list.CreatedAt, &list.ItemCount,
			&list.GoodreadsURL, &list.VoterCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Load items with books
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT li.id, li.list_id, li.book_id, li.position, li.votes, %s
		FROM book_list_items li
		JOIN books b ON b.id = li.book_id
		WHERE li.list_id = ?
		ORDER BY li.position ASC, li.votes DESC`, bookColumns("b")), id)
	if err != nil {
		return &list, nil
	}
	defer rows.Close()

	for rows.Next() {
		var item types.BookListItem
		var b types.Book
		fields := append([]any{&item.ID, &item.ListID, &item.BookID, &item.Position, &item.Votes}, scanFields(&b)...)
		err := rows.Scan(fields...)
		if err != nil {
			return nil, fmt.Errorf("scan list item: %w", err)
		}
		hydrateBook(&b)
		item.Book = &b
		list.Items = append(list.Items, item)
	}
	if list.Items == nil {
		list.Items = []types.BookListItem{}
	}
	return &list, nil
}

func (s *ListStore) GetAll(ctx context.Context, page, limit int) ([]types.BookList, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int
	s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM book_lists`).Scan(&total)

	rows, err := s.db.QueryContext(ctx, `
		SELECT l.id, l.title, l.description, l.created_at, COALESCE(c.cnt, 0), l.goodreads_url, l.voter_count
		FROM book_lists l
		LEFT JOIN (SELECT list_id, COUNT(*) as cnt FROM book_list_items GROUP BY list_id) c ON c.list_id = l.id
		ORDER BY l.created_at DESC
		LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var lists []types.BookList
	for rows.Next() {
		var l types.BookList
		err := rows.Scan(&l.ID, &l.Title, &l.Description, &l.CreatedAt, &l.ItemCount,
			&l.GoodreadsURL, &l.VoterCount)
		if err != nil {
			return nil, 0, fmt.Errorf("scan list: %w", err)
		}
		lists = append(lists, l)
	}
	if lists == nil {
		lists = []types.BookList{}
	}
	return lists, total, nil
}

func (s *ListStore) AddBook(ctx context.Context, listID, bookID int64, position int) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO book_list_items (list_id, book_id, position, votes)
		VALUES (?, ?, ?, 0) ON CONFLICT(list_id, book_id) DO NOTHING`,
		listID, bookID, position)
	return err
}

func (s *ListStore) SetVotes(ctx context.Context, listID, bookID int64, votes int) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE book_list_items SET votes = ? WHERE list_id = ? AND book_id = ?`,
		votes, listID, bookID)
	return err
}

func (s *ListStore) RemoveBook(ctx context.Context, listID, bookID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM book_list_items WHERE list_id = ? AND book_id = ?`, listID, bookID)
	return err
}

func (s *ListStore) Vote(ctx context.Context, listID, bookID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE book_list_items SET votes = votes + 1 WHERE list_id = ? AND book_id = ?`,
		listID, bookID)
	return err
}

func (s *ListStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM book_lists WHERE id = ?`, id)
	return err
}
