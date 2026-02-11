package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/go-mizu/mizu/blueprints/book/store"
)

// Store is the SQLite-backed implementation of store.Store.
type Store struct {
	db        *sql.DB
	book      *BookStore
	author    *AuthorStore
	shelf     *ShelfStore
	review    *ReviewStore
	progress  *ProgressStore
	challenge *ChallengeStore
	list      *ListStore
	quote     *QuoteStore
	stats     *StatsStore
	feed      *FeedStore
}

// New opens (or creates) a SQLite database at dbPath and returns a Store.
func New(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dsn := fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Store{db: db}
	s.book = &BookStore{db: db}
	s.author = &AuthorStore{db: db}
	s.shelf = &ShelfStore{db: db}
	s.review = &ReviewStore{db: db}
	s.progress = &ProgressStore{db: db}
	s.challenge = &ChallengeStore{db: db}
	s.list = &ListStore{db: db}
	s.quote = &QuoteStore{db: db}
	s.stats = &StatsStore{db: db}
	s.feed = &FeedStore{db: db}

	return s, nil
}

// Ensure creates all tables and indexes if they do not exist.
func (s *Store) Ensure(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// Close closes the underlying database connection.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Book() store.BookStore         { return s.book }
func (s *Store) Author() store.AuthorStore     { return s.author }
func (s *Store) Shelf() store.ShelfStore       { return s.shelf }
func (s *Store) Review() store.ReviewStore     { return s.review }
func (s *Store) Progress() store.ProgressStore { return s.progress }
func (s *Store) Challenge() store.ChallengeStore { return s.challenge }
func (s *Store) List() store.ListStore         { return s.list }
func (s *Store) Quote() store.QuoteStore       { return s.quote }
func (s *Store) Stats() store.StatsStore       { return s.stats }
func (s *Store) Feed() store.FeedStore         { return s.feed }

// Ensure interface compliance.
var _ store.Store = (*Store)(nil)
