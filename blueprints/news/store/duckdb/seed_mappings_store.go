package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SeedMapping represents a mapping between external and local IDs.
type SeedMapping struct {
	Source     string
	EntityType string
	ExternalID string
	LocalID    string
	CreatedAt  time.Time
}

// SeedMappingsStore handles seed mapping operations.
type SeedMappingsStore struct {
	db *sql.DB
}

// NewSeedMappingsStore creates a new seed mappings store.
func NewSeedMappingsStore(db *sql.DB) *SeedMappingsStore {
	return &SeedMappingsStore{db: db}
}

// Create creates a seed mapping.
func (s *SeedMappingsStore) Create(ctx context.Context, mapping *SeedMapping) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO seed_mappings (source, entity_type, external_id, local_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (source, entity_type, external_id) DO NOTHING
	`, mapping.Source, mapping.EntityType, mapping.ExternalID, mapping.LocalID, mapping.CreatedAt)
	return err
}

// GetLocalID retrieves a local ID from an external ID.
func (s *SeedMappingsStore) GetLocalID(ctx context.Context, source, entityType, externalID string) (string, error) {
	var localID string
	err := s.db.QueryRowContext(ctx, `
		SELECT local_id
		FROM seed_mappings
		WHERE source = $1 AND entity_type = $2 AND external_id = $3
	`, source, entityType, externalID).Scan(&localID)

	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return localID, nil
}

// GetLocalIDs retrieves local IDs for multiple external IDs.
func (s *SeedMappingsStore) GetLocalIDs(ctx context.Context, source, entityType string, externalIDs []string) (map[string]string, error) {
	if len(externalIDs) == 0 {
		return make(map[string]string), nil
	}

	placeholders := make([]string, len(externalIDs))
	args := make([]any, len(externalIDs)+2)
	args[0] = source
	args[1] = entityType
	for i, id := range externalIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+3)
		args[i+2] = id
	}

	query := `
		SELECT external_id, local_id
		FROM seed_mappings
		WHERE source = $1 AND entity_type = $2 AND external_id IN (` + strings.Join(placeholders, ",") + `)`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var externalID, localID string
		if err := rows.Scan(&externalID, &localID); err != nil {
			return nil, err
		}
		result[externalID] = localID
	}
	return result, rows.Err()
}

// Exists checks if a mapping exists.
func (s *SeedMappingsStore) Exists(ctx context.Context, source, entityType, externalID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM seed_mappings
			WHERE source = $1 AND entity_type = $2 AND external_id = $3
		)
	`, source, entityType, externalID).Scan(&exists)
	return exists, err
}
