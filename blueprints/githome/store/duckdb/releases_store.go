package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/releases"
)

// ReleasesStore implements releases.Store
type ReleasesStore struct {
	db *sql.DB
}

// NewReleasesStore creates a new releases store
func NewReleasesStore(db *sql.DB) *ReleasesStore {
	return &ReleasesStore{db: db}
}

// Create creates a new release
func (s *ReleasesStore) Create(ctx context.Context, r *releases.Release) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO releases (id, repo_id, tag_name, target_commitish, name, body, is_draft, is_prerelease, author_id, created_at, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, r.ID, r.RepoID, r.TagName, r.TargetCommitish, r.Name, r.Body, r.IsDraft, r.IsPrerelease, r.AuthorID, r.CreatedAt, r.PublishedAt)
	return err
}

// GetByID retrieves a release by ID
func (s *ReleasesStore) GetByID(ctx context.Context, id string) (*releases.Release, error) {
	r := &releases.Release{}
	var publishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, tag_name, target_commitish, name, body, is_draft, is_prerelease, author_id, created_at, published_at
		FROM releases WHERE id = $1
	`, id).Scan(&r.ID, &r.RepoID, &r.TagName, &r.TargetCommitish, &r.Name, &r.Body, &r.IsDraft, &r.IsPrerelease, &r.AuthorID, &r.CreatedAt, &publishedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if publishedAt.Valid {
		r.PublishedAt = &publishedAt.Time
	}
	return r, nil
}

// GetByTag retrieves a release by repository ID and tag name
func (s *ReleasesStore) GetByTag(ctx context.Context, repoID, tagName string) (*releases.Release, error) {
	r := &releases.Release{}
	var publishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, tag_name, target_commitish, name, body, is_draft, is_prerelease, author_id, created_at, published_at
		FROM releases WHERE repo_id = $1 AND tag_name = $2
	`, repoID, tagName).Scan(&r.ID, &r.RepoID, &r.TagName, &r.TargetCommitish, &r.Name, &r.Body, &r.IsDraft, &r.IsPrerelease, &r.AuthorID, &r.CreatedAt, &publishedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if publishedAt.Valid {
		r.PublishedAt = &publishedAt.Time
	}
	return r, nil
}

// GetLatest retrieves the latest published release
func (s *ReleasesStore) GetLatest(ctx context.Context, repoID string) (*releases.Release, error) {
	r := &releases.Release{}
	var publishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, repo_id, tag_name, target_commitish, name, body, is_draft, is_prerelease, author_id, created_at, published_at
		FROM releases WHERE repo_id = $1 AND is_draft = FALSE AND is_prerelease = FALSE AND published_at IS NOT NULL
		ORDER BY published_at DESC
		LIMIT 1
	`, repoID).Scan(&r.ID, &r.RepoID, &r.TagName, &r.TargetCommitish, &r.Name, &r.Body, &r.IsDraft, &r.IsPrerelease, &r.AuthorID, &r.CreatedAt, &publishedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if publishedAt.Valid {
		r.PublishedAt = &publishedAt.Time
	}
	return r, nil
}

// Update updates a release
func (s *ReleasesStore) Update(ctx context.Context, r *releases.Release) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE releases SET tag_name = $2, target_commitish = $3, name = $4, body = $5, is_draft = $6, is_prerelease = $7, published_at = $8
		WHERE id = $1
	`, r.ID, r.TagName, r.TargetCommitish, r.Name, r.Body, r.IsDraft, r.IsPrerelease, r.PublishedAt)
	return err
}

// Delete deletes a release
func (s *ReleasesStore) Delete(ctx context.Context, id string) error {
	// Delete assets first
	s.db.ExecContext(ctx, `DELETE FROM release_assets WHERE release_id = $1`, id)
	_, err := s.db.ExecContext(ctx, `DELETE FROM releases WHERE id = $1`, id)
	return err
}

// List lists releases for a repository
func (s *ReleasesStore) List(ctx context.Context, repoID string, limit, offset int) ([]*releases.Release, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, tag_name, target_commitish, name, body, is_draft, is_prerelease, author_id, created_at, published_at
		FROM releases WHERE repo_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, repoID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*releases.Release
	for rows.Next() {
		r := &releases.Release{}
		var publishedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.RepoID, &r.TagName, &r.TargetCommitish, &r.Name, &r.Body, &r.IsDraft, &r.IsPrerelease, &r.AuthorID, &r.CreatedAt, &publishedAt); err != nil {
			return nil, err
		}
		if publishedAt.Valid {
			r.PublishedAt = &publishedAt.Time
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// CreateAsset creates a new asset
func (s *ReleasesStore) CreateAsset(ctx context.Context, a *releases.Asset) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO release_assets (id, release_id, name, label, content_type, size_bytes, download_count, uploader_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, a.ID, a.ReleaseID, a.Name, a.Label, a.ContentType, a.SizeBytes, a.DownloadCount, a.UploaderID, a.CreatedAt, a.UpdatedAt)
	return err
}

// GetAsset retrieves an asset by ID
func (s *ReleasesStore) GetAsset(ctx context.Context, id string) (*releases.Asset, error) {
	a := &releases.Asset{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, release_id, name, label, content_type, size_bytes, download_count, uploader_id, created_at, updated_at
		FROM release_assets WHERE id = $1
	`, id).Scan(&a.ID, &a.ReleaseID, &a.Name, &a.Label, &a.ContentType, &a.SizeBytes, &a.DownloadCount, &a.UploaderID, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

// UpdateAsset updates an asset
func (s *ReleasesStore) UpdateAsset(ctx context.Context, a *releases.Asset) error {
	a.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE release_assets SET name = $2, label = $3, updated_at = $4
		WHERE id = $1
	`, a.ID, a.Name, a.Label, a.UpdatedAt)
	return err
}

// DeleteAsset deletes an asset
func (s *ReleasesStore) DeleteAsset(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM release_assets WHERE id = $1`, id)
	return err
}

// ListAssets lists assets for a release
func (s *ReleasesStore) ListAssets(ctx context.Context, releaseID string) ([]*releases.Asset, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, release_id, name, label, content_type, size_bytes, download_count, uploader_id, created_at, updated_at
		FROM release_assets WHERE release_id = $1
		ORDER BY name ASC
	`, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*releases.Asset
	for rows.Next() {
		a := &releases.Asset{}
		if err := rows.Scan(&a.ID, &a.ReleaseID, &a.Name, &a.Label, &a.ContentType, &a.SizeBytes, &a.DownloadCount, &a.UploaderID, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// IncrementDownload increments the download count for an asset
func (s *ReleasesStore) IncrementDownload(ctx context.Context, assetID string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE release_assets SET download_count = download_count + 1
		WHERE id = $1
	`, assetID)
	return err
}
