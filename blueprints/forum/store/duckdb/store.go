package duckdb

import (
	"context"
	"database/sql"
	_ "embed"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

//go:embed schema.sql
var schema string

// Store wraps a DuckDB connection and provides access to all stores.
type Store struct {
	db            *sql.DB
	accounts      *AccountsStore
	boards        *BoardsStore
	threads       *ThreadsStore
	comments      *CommentsStore
	votes         *VotesStore
	bookmarks     *BookmarksStore
	notifications *NotificationsStore
}

// Open opens a DuckDB database at the given path and initializes all stores.
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "forum.db")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}

	// Initialize schema
	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{
		db:            db,
		accounts:      NewAccountsStore(db),
		boards:        NewBoardsStore(db),
		threads:       NewThreadsStore(db),
		comments:      NewCommentsStore(db),
		votes:         NewVotesStore(db),
		bookmarks:     NewBookmarksStore(db),
		notifications: NewNotificationsStore(db),
	}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Accounts returns the accounts store.
func (s *Store) Accounts() *AccountsStore {
	return s.accounts
}

// Boards returns the boards store.
func (s *Store) Boards() *BoardsStore {
	return s.boards
}

// Threads returns the threads store.
func (s *Store) Threads() *ThreadsStore {
	return s.threads
}

// Comments returns the comments store.
func (s *Store) Comments() *CommentsStore {
	return s.comments
}

// Votes returns the votes store.
func (s *Store) Votes() *VotesStore {
	return s.votes
}

// Bookmarks returns the bookmarks store.
func (s *Store) Bookmarks() *BookmarksStore {
	return s.bookmarks
}

// Notifications returns the notifications store.
func (s *Store) Notifications() *NotificationsStore {
	return s.notifications
}

// Tx executes a function within a transaction.
func (s *Store) Tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
