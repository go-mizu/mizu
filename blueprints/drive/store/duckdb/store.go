// Package duckdb provides DuckDB-based storage.
package duckdb

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

//go:embed schema.sql
var schema string

// Store is the main DuckDB store.
type Store struct {
	db *sql.DB
}

// Open opens a DuckDB database.
func Open(dataDir string) (*Store, error) {
	dbPath := filepath.Join(dataDir, "drive.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	return &Store{db: db}, nil
}

// Ensure creates tables if they don't exist.
func (s *Store) Ensure(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, schema)
	return err
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
	return &AccountsStore{db: s.db}
}

// Files returns the files store.
func (s *Store) Files() *FilesStore {
	return &FilesStore{db: s.db}
}

// Folders returns the folders store.
func (s *Store) Folders() *FoldersStore {
	return &FoldersStore{db: s.db}
}

// Shares returns the shares store.
func (s *Store) Shares() *SharesStore {
	return &SharesStore{db: s.db}
}

// ShareLinks returns the share links store.
func (s *Store) ShareLinks() *ShareLinksStore {
	return &ShareLinksStore{db: s.db}
}

// Tags returns the tags store.
func (s *Store) Tags() *TagsStore {
	return &TagsStore{db: s.db}
}

// Comments returns the comments store.
func (s *Store) Comments() *CommentsStore {
	return &CommentsStore{db: s.db}
}

// Activities returns the activities store.
func (s *Store) Activities() *ActivitiesStore {
	return &ActivitiesStore{db: s.db}
}

// Notifications returns the notifications store.
func (s *Store) Notifications() *NotificationsStore {
	return &NotificationsStore{db: s.db}
}

// ChunkedUploads returns the chunked uploads store.
func (s *Store) ChunkedUploads() *ChunkedUploadsStore {
	return &ChunkedUploadsStore{db: s.db}
}

// FileVersions returns the file versions store.
func (s *Store) FileVersions() *FileVersionsStore {
	return &FileVersionsStore{db: s.db}
}
