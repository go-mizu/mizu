package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/releases"
)

// ReleasesStore handles release data access.
type ReleasesStore struct {
	db *sql.DB
}

// NewReleasesStore creates a new releases store.
func NewReleasesStore(db *sql.DB) *ReleasesStore {
	return &ReleasesStore{db: db}
}

func (s *ReleasesStore) Create(ctx context.Context, r *releases.Release) error {
	now := time.Now()
	r.CreatedAt = now

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO releases (node_id, repo_id, tag_name, target_commitish, name, body,
			draft, prerelease, author_id, published_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`, "", r.RepoID, r.TagName, r.TargetCommitish, r.Name, r.Body, r.Draft, r.Prerelease,
		r.AuthorID, nullTime(r.PublishedAt), r.CreatedAt).Scan(&r.ID)
	if err != nil {
		return err
	}

	r.NodeID = generateNodeID("RE", r.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE releases SET node_id = $1 WHERE id = $2`, r.NodeID, r.ID)
	return err
}

func (s *ReleasesStore) GetByID(ctx context.Context, id int64) (*releases.Release, error) {
	r := &releases.Release{}
	var publishedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, tag_name, target_commitish, name, body,
			draft, prerelease, author_id, published_at, created_at
		FROM releases WHERE id = $1
	`, id).Scan(&r.ID, &r.NodeID, &r.RepoID, &r.TagName, &r.TargetCommitish, &r.Name, &r.Body,
		&r.Draft, &r.Prerelease, &r.AuthorID, &publishedAt, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if publishedAt.Valid {
		r.PublishedAt = &publishedAt.Time
	}
	return r, err
}

func (s *ReleasesStore) GetByTag(ctx context.Context, repoID int64, tag string) (*releases.Release, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM releases WHERE repo_id = $1 AND tag_name = $2`, repoID, tag).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *ReleasesStore) GetLatest(ctx context.Context, repoID int64) (*releases.Release, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
		SELECT id FROM releases
		WHERE repo_id = $1 AND draft = FALSE AND prerelease = FALSE
		ORDER BY created_at DESC LIMIT 1
	`, repoID).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *ReleasesStore) Update(ctx context.Context, id int64, in *releases.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE releases SET
			tag_name = COALESCE($2, tag_name),
			target_commitish = COALESCE($3, target_commitish),
			name = COALESCE($4, name),
			body = COALESCE($5, body),
			draft = COALESCE($6, draft),
			prerelease = COALESCE($7, prerelease)
		WHERE id = $1
	`, id, nullStringPtr(in.TagName), nullStringPtr(in.TargetCommitish), nullStringPtr(in.Name),
		nullStringPtr(in.Body), nullBoolPtr(in.Draft), nullBoolPtr(in.Prerelease))
	return err
}

func (s *ReleasesStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM releases WHERE id = $1`, id)
	return err
}

func (s *ReleasesStore) List(ctx context.Context, repoID int64, opts *releases.ListOpts) ([]*releases.Release, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT id, node_id, repo_id, tag_name, target_commitish, name, body,
			draft, prerelease, author_id, published_at, created_at
		FROM releases WHERE repo_id = $1
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*releases.Release
	for rows.Next() {
		r := &releases.Release{}
		var publishedAt sql.NullTime
		if err := rows.Scan(&r.ID, &r.NodeID, &r.RepoID, &r.TagName, &r.TargetCommitish, &r.Name, &r.Body,
			&r.Draft, &r.Prerelease, &r.AuthorID, &publishedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		if publishedAt.Valid {
			r.PublishedAt = &publishedAt.Time
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

// Asset methods

func (s *ReleasesStore) CreateAsset(ctx context.Context, a *releases.Asset) error {
	now := time.Now()
	a.CreatedAt = now
	a.UpdatedAt = now

	err := s.db.QueryRowContext(ctx, `
		INSERT INTO release_assets (node_id, release_id, uploader_id, name, label, state,
			content_type, size, download_count, storage_path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, "", a.ReleaseID, a.UploaderID, a.Name, a.Label, a.State, a.ContentType,
		a.Size, a.DownloadCount, a.StoragePath, a.CreatedAt, a.UpdatedAt).Scan(&a.ID)
	if err != nil {
		return err
	}

	a.NodeID = generateNodeID("RA", a.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE release_assets SET node_id = $1 WHERE id = $2`, a.NodeID, a.ID)
	return err
}

func (s *ReleasesStore) GetAssetByID(ctx context.Context, id int64) (*releases.Asset, error) {
	a := &releases.Asset{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, release_id, uploader_id, name, label, state, content_type,
			size, download_count, storage_path, created_at, updated_at
		FROM release_assets WHERE id = $1
	`, id).Scan(&a.ID, &a.NodeID, &a.ReleaseID, &a.UploaderID, &a.Name, &a.Label, &a.State,
		&a.ContentType, &a.Size, &a.DownloadCount, &a.StoragePath, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

func (s *ReleasesStore) UpdateAsset(ctx context.Context, id int64, in *releases.UpdateAssetIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE release_assets SET
			name = COALESCE($2, name),
			label = COALESCE($3, label),
			updated_at = $4
		WHERE id = $1
	`, id, nullStringPtr(in.Name), nullStringPtr(in.Label), time.Now())
	return err
}

func (s *ReleasesStore) DeleteAsset(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM release_assets WHERE id = $1`, id)
	return err
}

func (s *ReleasesStore) ListAssets(ctx context.Context, releaseID int64, opts *releases.ListOpts) ([]*releases.Asset, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	query := `
		SELECT id, node_id, release_id, uploader_id, name, label, state, content_type,
			size, download_count, storage_path, created_at, updated_at
		FROM release_assets WHERE release_id = $1
		ORDER BY created_at ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, releaseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*releases.Asset
	for rows.Next() {
		a := &releases.Asset{}
		if err := rows.Scan(&a.ID, &a.NodeID, &a.ReleaseID, &a.UploaderID, &a.Name, &a.Label, &a.State,
			&a.ContentType, &a.Size, &a.DownloadCount, &a.StoragePath, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

func (s *ReleasesStore) IncrementDownloadCount(ctx context.Context, assetID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE release_assets SET download_count = download_count + 1 WHERE id = $1
	`, assetID)
	return err
}
