package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// RepoStorage represents the storage location for a repository
type RepoStorage struct {
	RepoID         string
	StorageBackend string // fs, s3, r2, other
	StoragePath    string
	CreatedAt      time.Time
}

// RepoStorageStore manages repo_storage records
type RepoStorageStore struct {
	db *sql.DB
}

// NewRepoStorageStore creates a new repo storage store
func NewRepoStorageStore(db *sql.DB) *RepoStorageStore {
	return &RepoStorageStore{db: db}
}

// Create creates a new repo storage record
func (s *RepoStorageStore) Create(ctx context.Context, rs *RepoStorage) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO repo_storage (repo_id, storage_backend, storage_path, created_at)
		VALUES ($1, $2, $3, $4)
	`, rs.RepoID, rs.StorageBackend, rs.StoragePath, rs.CreatedAt)
	return err
}

// GetByRepoID retrieves storage info for a repository
func (s *RepoStorageStore) GetByRepoID(ctx context.Context, repoID string) (*RepoStorage, error) {
	rs := &RepoStorage{}
	err := s.db.QueryRowContext(ctx, `
		SELECT repo_id, storage_backend, storage_path, created_at
		FROM repo_storage WHERE repo_id = $1
	`, repoID).Scan(&rs.RepoID, &rs.StorageBackend, &rs.StoragePath, &rs.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return rs, err
}

// Update updates a repo storage record
func (s *RepoStorageStore) Update(ctx context.Context, rs *RepoStorage) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE repo_storage SET storage_backend = $2, storage_path = $3
		WHERE repo_id = $1
	`, rs.RepoID, rs.StorageBackend, rs.StoragePath)
	return err
}

// Delete deletes a repo storage record
func (s *RepoStorageStore) Delete(ctx context.Context, repoID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM repo_storage WHERE repo_id = $1`, repoID)
	return err
}

// ListByBackend lists all storage records for a given backend type
func (s *RepoStorageStore) ListByBackend(ctx context.Context, backend string) ([]*RepoStorage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT repo_id, storage_backend, storage_path, created_at
		FROM repo_storage WHERE storage_backend = $1
		ORDER BY created_at DESC
	`, backend)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*RepoStorage
	for rows.Next() {
		rs := &RepoStorage{}
		if err := rows.Scan(&rs.RepoID, &rs.StorageBackend, &rs.StoragePath, &rs.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, rs)
	}
	return list, rows.Err()
}
