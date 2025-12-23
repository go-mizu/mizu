package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/mizu/blueprints/forum/store"
)

// SeedMappingsStore handles seed mapping persistence.
type SeedMappingsStore struct {
	db *sql.DB
}

// NewSeedMappingsStore creates a new seed mappings store.
func NewSeedMappingsStore(db *sql.DB) *SeedMappingsStore {
	return &SeedMappingsStore{db: db}
}

// Create creates a new seed mapping.
func (s *SeedMappingsStore) Create(ctx context.Context, mapping *store.SeedMapping) error {
	query := `
		INSERT INTO seed_mappings (source, entity_type, external_id, local_id, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	mapping.CreatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, query,
		mapping.Source,
		mapping.EntityType,
		mapping.ExternalID,
		mapping.LocalID,
		mapping.CreatedAt,
	)
	return err
}

// GetByExternalID returns a mapping by external ID.
func (s *SeedMappingsStore) GetByExternalID(ctx context.Context, source, entityType, externalID string) (*store.SeedMapping, error) {
	query := `
		SELECT source, entity_type, external_id, local_id, created_at
		FROM seed_mappings
		WHERE source = ? AND entity_type = ? AND external_id = ?
	`
	var m store.SeedMapping
	err := s.db.QueryRowContext(ctx, query, source, entityType, externalID).Scan(
		&m.Source,
		&m.EntityType,
		&m.ExternalID,
		&m.LocalID,
		&m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByLocalID returns a mapping by local ID.
func (s *SeedMappingsStore) GetByLocalID(ctx context.Context, localID string) (*store.SeedMapping, error) {
	query := `
		SELECT source, entity_type, external_id, local_id, created_at
		FROM seed_mappings
		WHERE local_id = ?
	`
	var m store.SeedMapping
	err := s.db.QueryRowContext(ctx, query, localID).Scan(
		&m.Source,
		&m.EntityType,
		&m.ExternalID,
		&m.LocalID,
		&m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Exists checks if a mapping exists.
func (s *SeedMappingsStore) Exists(ctx context.Context, source, entityType, externalID string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM seed_mappings
		WHERE source = ? AND entity_type = ? AND external_id = ?
	`
	var count int
	err := s.db.QueryRowContext(ctx, query, source, entityType, externalID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Delete deletes a mapping.
func (s *SeedMappingsStore) Delete(ctx context.Context, source, entityType, externalID string) error {
	query := `DELETE FROM seed_mappings WHERE source = ? AND entity_type = ? AND external_id = ?`
	_, err := s.db.ExecContext(ctx, query, source, entityType, externalID)
	return err
}

// DeleteBySource deletes all mappings for a source.
func (s *SeedMappingsStore) DeleteBySource(ctx context.Context, source string) error {
	query := `DELETE FROM seed_mappings WHERE source = ?`
	_, err := s.db.ExecContext(ctx, query, source)
	return err
}

// List returns all mappings for a source and entity type.
func (s *SeedMappingsStore) List(ctx context.Context, source, entityType string) ([]*store.SeedMapping, error) {
	query := `
		SELECT source, entity_type, external_id, local_id, created_at
		FROM seed_mappings
		WHERE source = ? AND entity_type = ?
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, source, entityType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []*store.SeedMapping
	for rows.Next() {
		var m store.SeedMapping
		if err := rows.Scan(&m.Source, &m.EntityType, &m.ExternalID, &m.LocalID, &m.CreatedAt); err != nil {
			return nil, err
		}
		mappings = append(mappings, &m)
	}
	return mappings, rows.Err()
}
