package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

// VersionsStore handles document version operations.
type VersionsStore struct {
	db *sql.DB
}

// NewVersionsStore creates a new VersionsStore.
func NewVersionsStore(db *sql.DB) *VersionsStore {
	return &VersionsStore{db: db}
}

// Version represents a document version.
type Version struct {
	ID        string
	Parent    string
	Version   int
	Snapshot  map[string]any
	Published bool
	Autosave  bool
	CreatedAt time.Time
	UpdatedBy string
}

// Create creates a new version for a document.
func (s *VersionsStore) Create(ctx context.Context, collection string, version *Version) error {
	version.ID = ulid.New()
	version.CreatedAt = time.Now()

	snapshotJSON, err := json.Marshal(version.Snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	tableName := collection + "_versions"
	query := fmt.Sprintf(`INSERT INTO %s (id, parent, version, snapshot, published, autosave, created_at, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, tableName)

	_, err = s.db.ExecContext(ctx, query,
		version.ID, version.Parent, version.Version, string(snapshotJSON),
		version.Published, version.Autosave, version.CreatedAt, version.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("create version: %w", err)
	}

	return nil
}

// GetByID retrieves a version by ID.
func (s *VersionsStore) GetByID(ctx context.Context, collection, id string) (*Version, error) {
	tableName := collection + "_versions"
	query := fmt.Sprintf(`SELECT id, parent, version, snapshot, published, autosave, created_at, updated_by
		FROM %s WHERE id = ?`, tableName)

	var v Version
	var snapshotJSON string
	var updatedBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&v.ID, &v.Parent, &v.Version, &snapshotJSON, &v.Published, &v.Autosave, &v.CreatedAt, &updatedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get version: %w", err)
	}

	if err := json.Unmarshal([]byte(snapshotJSON), &v.Snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	if updatedBy.Valid {
		v.UpdatedBy = updatedBy.String
	}

	return &v, nil
}

// ListByParent lists all versions for a document.
func (s *VersionsStore) ListByParent(ctx context.Context, collection, parentID string, limit, page int) ([]*Version, int, error) {
	tableName := collection + "_versions"

	// Count total
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE parent = ?`, tableName)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, parentID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count versions: %w", err)
	}

	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`SELECT id, parent, version, snapshot, published, autosave, created_at, updated_by
		FROM %s WHERE parent = ? ORDER BY version DESC LIMIT ? OFFSET ?`, tableName)

	rows, err := s.db.QueryContext(ctx, query, parentID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	var versions []*Version
	for rows.Next() {
		var v Version
		var snapshotJSON string
		var updatedBy sql.NullString

		if err := rows.Scan(&v.ID, &v.Parent, &v.Version, &snapshotJSON, &v.Published, &v.Autosave, &v.CreatedAt, &updatedBy); err != nil {
			return nil, 0, fmt.Errorf("scan version: %w", err)
		}

		if err := json.Unmarshal([]byte(snapshotJSON), &v.Snapshot); err != nil {
			return nil, 0, fmt.Errorf("unmarshal snapshot: %w", err)
		}
		if updatedBy.Valid {
			v.UpdatedBy = updatedBy.String
		}

		versions = append(versions, &v)
	}

	return versions, total, nil
}

// GetLatestVersion gets the latest version number for a document.
func (s *VersionsStore) GetLatestVersion(ctx context.Context, collection, parentID string) (int, error) {
	tableName := collection + "_versions"
	query := fmt.Sprintf(`SELECT COALESCE(MAX(version), 0) FROM %s WHERE parent = ?`, tableName)

	var version int
	if err := s.db.QueryRowContext(ctx, query, parentID).Scan(&version); err != nil {
		return 0, fmt.Errorf("get latest version: %w", err)
	}

	return version, nil
}

// DeleteOldVersions deletes versions beyond the max count.
func (s *VersionsStore) DeleteOldVersions(ctx context.Context, collection, parentID string, maxVersions int) error {
	tableName := collection + "_versions"

	// Get IDs of versions to keep
	query := fmt.Sprintf(`DELETE FROM %s WHERE parent = ? AND id NOT IN (
		SELECT id FROM %s WHERE parent = ? ORDER BY version DESC LIMIT ?
	)`, tableName, tableName)

	_, err := s.db.ExecContext(ctx, query, parentID, parentID, maxVersions)
	if err != nil {
		return fmt.Errorf("delete old versions: %w", err)
	}

	return nil
}

// CreateGlobalVersion creates a new version for a global.
func (s *VersionsStore) CreateGlobalVersion(ctx context.Context, version *GlobalVersion) error {
	version.ID = ulid.New()
	version.CreatedAt = time.Now()

	snapshotJSON, err := json.Marshal(version.Snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	query := `INSERT INTO _globals_versions (id, global_slug, version, snapshot, created_at, updated_by)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err = s.db.ExecContext(ctx, query,
		version.ID, version.GlobalSlug, version.Version, string(snapshotJSON),
		version.CreatedAt, version.UpdatedBy,
	)
	if err != nil {
		return fmt.Errorf("create global version: %w", err)
	}

	return nil
}

// GlobalVersion represents a global document version.
type GlobalVersion struct {
	ID         string
	GlobalSlug string
	Version    int
	Snapshot   map[string]any
	CreatedAt  time.Time
	UpdatedBy  string
}

// ListGlobalVersions lists all versions for a global.
func (s *VersionsStore) ListGlobalVersions(ctx context.Context, globalSlug string, limit, page int) ([]*GlobalVersion, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM _globals_versions WHERE global_slug = ?`
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, globalSlug).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count global versions: %w", err)
	}

	if limit <= 0 {
		limit = 10
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := `SELECT id, global_slug, version, snapshot, created_at, updated_by
		FROM _globals_versions WHERE global_slug = ? ORDER BY version DESC LIMIT ? OFFSET ?`

	rows, err := s.db.QueryContext(ctx, query, globalSlug, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list global versions: %w", err)
	}
	defer rows.Close()

	var versions []*GlobalVersion
	for rows.Next() {
		var v GlobalVersion
		var snapshotJSON string
		var updatedBy sql.NullString

		if err := rows.Scan(&v.ID, &v.GlobalSlug, &v.Version, &snapshotJSON, &v.CreatedAt, &updatedBy); err != nil {
			return nil, 0, fmt.Errorf("scan global version: %w", err)
		}

		if err := json.Unmarshal([]byte(snapshotJSON), &v.Snapshot); err != nil {
			return nil, 0, fmt.Errorf("unmarshal snapshot: %w", err)
		}
		if updatedBy.Valid {
			v.UpdatedBy = updatedBy.String
		}

		versions = append(versions, &v)
	}

	return versions, total, nil
}
