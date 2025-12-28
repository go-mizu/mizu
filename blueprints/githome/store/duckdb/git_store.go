package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/githome/feature/git"
)

// GitStore handles git object cache data access.
type GitStore struct {
	db *sql.DB
}

// NewGitStore creates a new git store.
func NewGitStore(db *sql.DB) *GitStore {
	return &GitStore{db: db}
}

// CacheBlob stores a blob in the cache.
func (s *GitStore) CacheBlob(ctx context.Context, repoID int64, blob *git.Blob) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO git_blobs (repo_id, sha, node_id, size, content, encoding)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (repo_id, sha) DO UPDATE SET
			node_id = excluded.node_id,
			size = excluded.size,
			content = excluded.content,
			encoding = excluded.encoding
	`, repoID, blob.SHA, blob.NodeID, blob.Size, blob.Content, blob.Encoding)
	return err
}

// GetCachedBlob retrieves a blob from the cache.
func (s *GitStore) GetCachedBlob(ctx context.Context, repoID int64, sha string) (*git.Blob, error) {
	blob := &git.Blob{}
	err := s.db.QueryRowContext(ctx, `
		SELECT sha, node_id, size, content, encoding
		FROM git_blobs
		WHERE repo_id = $1 AND sha = $2
	`, repoID, sha).Scan(&blob.SHA, &blob.NodeID, &blob.Size, &blob.Content, &blob.Encoding)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return blob, nil
}

// DeleteCachedBlob removes a blob from the cache.
func (s *GitStore) DeleteCachedBlob(ctx context.Context, repoID int64, sha string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM git_blobs WHERE repo_id = $1 AND sha = $2
	`, repoID, sha)
	return err
}

// DeleteAllCachedBlobs removes all blobs for a repository.
func (s *GitStore) DeleteAllCachedBlobs(ctx context.Context, repoID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM git_blobs WHERE repo_id = $1
	`, repoID)
	return err
}
