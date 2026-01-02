package duckdb

import (
	"context"
	"database/sql"
	_ "embed"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
)

//go:embed schema.sql
var schema string

// Store wraps a DuckDB connection and provides access to all stores.
type Store struct {
	db            *sql.DB
	accounts      *AccountsStore
	questions     *QuestionsStore
	answers       *AnswersStore
	comments      *CommentsStore
	votes         *VotesStore
	tags          *TagsStore
	bookmarks     *BookmarksStore
	badges        *BadgesStore
	notifications *NotificationsStore
}

// Open opens a DuckDB database at the given path and initializes all stores.
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "qa.db")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(context.Background(), schema); err != nil {
		db.Close()
		return nil, err
	}

	return &Store{
		db:            db,
		accounts:      NewAccountsStore(db),
		questions:     NewQuestionsStore(db),
		answers:       NewAnswersStore(db),
		comments:      NewCommentsStore(db),
		votes:         NewVotesStore(db),
		tags:          NewTagsStore(db),
		bookmarks:     NewBookmarksStore(db),
		badges:        NewBadgesStore(db),
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

func (s *Store) Accounts() *AccountsStore { return s.accounts }
func (s *Store) Questions() *QuestionsStore { return s.questions }
func (s *Store) Answers() *AnswersStore { return s.answers }
func (s *Store) Comments() *CommentsStore { return s.comments }
func (s *Store) Votes() *VotesStore { return s.votes }
func (s *Store) Tags() *TagsStore { return s.tags }
func (s *Store) Bookmarks() *BookmarksStore { return s.bookmarks }
func (s *Store) Badges() *BadgesStore { return s.badges }
func (s *Store) Notifications() *NotificationsStore { return s.notifications }

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
