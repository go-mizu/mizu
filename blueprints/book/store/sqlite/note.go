package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/mizu/blueprints/book/types"
)

// NoteStore implements store.NoteStore backed by SQLite.
type NoteStore struct {
	db *sql.DB
}

func (s *NoteStore) Upsert(ctx context.Context, note *types.BookNote) error {
	now := time.Now()
	if note.CreatedAt.IsZero() {
		note.CreatedAt = now
	}
	note.UpdatedAt = now

	// Try update first
	res, err := s.db.ExecContext(ctx, `
		UPDATE book_notes SET text=?, updated_at=? WHERE book_id=?`,
		note.Text, note.UpdatedAt, note.BookID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows > 0 {
		// Get the ID
		s.db.QueryRowContext(ctx, `SELECT id FROM book_notes WHERE book_id=?`, note.BookID).Scan(&note.ID)
		return nil
	}

	// Insert
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO book_notes (book_id, text, created_at, updated_at)
		VALUES (?, ?, ?, ?)`,
		note.BookID, note.Text, note.CreatedAt, note.UpdatedAt)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	note.ID = id
	return nil
}

func (s *NoteStore) Get(ctx context.Context, bookID int64) (*types.BookNote, error) {
	var note types.BookNote
	err := s.db.QueryRowContext(ctx, `
		SELECT id, book_id, text, created_at, updated_at
		FROM book_notes WHERE book_id=?`, bookID).Scan(
		&note.ID, &note.BookID, &note.Text, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &note, nil
}

func (s *NoteStore) Delete(ctx context.Context, bookID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM book_notes WHERE book_id=?`, bookID)
	return err
}
