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
	db           *sql.DB
	users        *UsersStore
	stories      *StoriesStore
	comments     *CommentsStore
	votes        *VotesStore
	seedMappings *SeedMappingsStore
}

// Open opens a DuckDB database at the given path and initializes all stores.
func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "news.db")
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
		db:           db,
		users:        NewUsersStore(db),
		stories:      NewStoriesStore(db),
		comments:     NewCommentsStore(db),
		votes:        NewVotesStore(db),
		seedMappings: NewSeedMappingsStore(db),
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

// Users returns the users store.
func (s *Store) Users() *UsersStore {
	return s.users
}

// Stories returns the stories store.
func (s *Store) Stories() *StoriesStore {
	return s.stories
}

// Comments returns the comments store.
func (s *Store) Comments() *CommentsStore {
	return s.comments
}

// Votes returns the votes store.
func (s *Store) Votes() *VotesStore {
	return s.votes
}

// SeedMappings returns the seed mappings store.
func (s *Store) SeedMappings() *SeedMappingsStore {
	return s.seedMappings
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
