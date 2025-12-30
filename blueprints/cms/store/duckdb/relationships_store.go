package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

// RelationshipsStore handles relationship junction table operations.
type RelationshipsStore struct {
	db *sql.DB
}

// NewRelationshipsStore creates a new RelationshipsStore.
func NewRelationshipsStore(db *sql.DB) *RelationshipsStore {
	return &RelationshipsStore{db: db}
}

// Relationship represents a relationship between documents.
type Relationship struct {
	ID               string
	SourceCollection string
	SourceID         string
	SourceField      string
	TargetCollection string
	TargetID         string
	Position         int
	CreatedAt        time.Time
}

// SetRelationships sets the relationships for a document field.
// This replaces all existing relationships for the source document/field.
func (s *RelationshipsStore) SetRelationships(ctx context.Context, sourceCollection, sourceID, sourceField string, targets []RelationshipTarget) error {
	// Delete existing relationships
	deleteQuery := `DELETE FROM _relationships WHERE source_collection = ? AND source_id = ? AND source_field = ?`
	if _, err := s.db.ExecContext(ctx, deleteQuery, sourceCollection, sourceID, sourceField); err != nil {
		return fmt.Errorf("delete relationships: %w", err)
	}

	if len(targets) == 0 {
		return nil
	}

	// Insert new relationships
	now := time.Now()
	insertQuery := `INSERT INTO _relationships (id, source_collection, source_id, source_field, target_collection, target_id, position, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	for i, target := range targets {
		_, err := s.db.ExecContext(ctx, insertQuery,
			ulid.New(), sourceCollection, sourceID, sourceField,
			target.Collection, target.ID, i, now,
		)
		if err != nil {
			return fmt.Errorf("insert relationship: %w", err)
		}
	}

	return nil
}

// RelationshipTarget represents a target document in a relationship.
type RelationshipTarget struct {
	Collection string
	ID         string
}

// GetRelationships gets the relationships for a document field.
func (s *RelationshipsStore) GetRelationships(ctx context.Context, sourceCollection, sourceID, sourceField string) ([]RelationshipTarget, error) {
	query := `SELECT target_collection, target_id FROM _relationships
		WHERE source_collection = ? AND source_id = ? AND source_field = ?
		ORDER BY position`

	rows, err := s.db.QueryContext(ctx, query, sourceCollection, sourceID, sourceField)
	if err != nil {
		return nil, fmt.Errorf("get relationships: %w", err)
	}
	defer rows.Close()

	var targets []RelationshipTarget
	for rows.Next() {
		var target RelationshipTarget
		if err := rows.Scan(&target.Collection, &target.ID); err != nil {
			return nil, fmt.Errorf("scan relationship: %w", err)
		}
		targets = append(targets, target)
	}

	return targets, nil
}

// AddRelationship adds a single relationship.
func (s *RelationshipsStore) AddRelationship(ctx context.Context, sourceCollection, sourceID, sourceField string, target RelationshipTarget) error {
	// Get current max position
	var maxPos int
	posQuery := `SELECT COALESCE(MAX(position), -1) FROM _relationships WHERE source_collection = ? AND source_id = ? AND source_field = ?`
	s.db.QueryRowContext(ctx, posQuery, sourceCollection, sourceID, sourceField).Scan(&maxPos)

	insertQuery := `INSERT INTO _relationships (id, source_collection, source_id, source_field, target_collection, target_id, position, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, insertQuery,
		ulid.New(), sourceCollection, sourceID, sourceField,
		target.Collection, target.ID, maxPos+1, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("add relationship: %w", err)
	}

	return nil
}

// RemoveRelationship removes a single relationship.
func (s *RelationshipsStore) RemoveRelationship(ctx context.Context, sourceCollection, sourceID, sourceField string, target RelationshipTarget) error {
	query := `DELETE FROM _relationships WHERE source_collection = ? AND source_id = ? AND source_field = ? AND target_collection = ? AND target_id = ?`

	_, err := s.db.ExecContext(ctx, query, sourceCollection, sourceID, sourceField, target.Collection, target.ID)
	if err != nil {
		return fmt.Errorf("remove relationship: %w", err)
	}

	return nil
}

// GetReverseRelationships gets documents that reference a target document.
// This is useful for "join" fields.
func (s *RelationshipsStore) GetReverseRelationships(ctx context.Context, targetCollection, targetID string) ([]ReverseRelationship, error) {
	query := `SELECT source_collection, source_id, source_field FROM _relationships
		WHERE target_collection = ? AND target_id = ?`

	rows, err := s.db.QueryContext(ctx, query, targetCollection, targetID)
	if err != nil {
		return nil, fmt.Errorf("get reverse relationships: %w", err)
	}
	defer rows.Close()

	var results []ReverseRelationship
	for rows.Next() {
		var r ReverseRelationship
		if err := rows.Scan(&r.SourceCollection, &r.SourceID, &r.SourceField); err != nil {
			return nil, fmt.Errorf("scan reverse relationship: %w", err)
		}
		results = append(results, r)
	}

	return results, nil
}

// ReverseRelationship represents a document that references another document.
type ReverseRelationship struct {
	SourceCollection string
	SourceID         string
	SourceField      string
}

// DeleteDocumentRelationships removes all relationships for a document (both as source and target).
func (s *RelationshipsStore) DeleteDocumentRelationships(ctx context.Context, collection, docID string) error {
	// Delete as source
	sourceQuery := `DELETE FROM _relationships WHERE source_collection = ? AND source_id = ?`
	if _, err := s.db.ExecContext(ctx, sourceQuery, collection, docID); err != nil {
		return fmt.Errorf("delete source relationships: %w", err)
	}

	// Delete as target
	targetQuery := `DELETE FROM _relationships WHERE target_collection = ? AND target_id = ?`
	if _, err := s.db.ExecContext(ctx, targetQuery, collection, docID); err != nil {
		return fmt.Errorf("delete target relationships: %w", err)
	}

	return nil
}

// CountReverseRelationships counts how many documents reference a target.
func (s *RelationshipsStore) CountReverseRelationships(ctx context.Context, targetCollection, targetID string) (int, error) {
	query := `SELECT COUNT(*) FROM _relationships WHERE target_collection = ? AND target_id = ?`

	var count int
	if err := s.db.QueryRowContext(ctx, query, targetCollection, targetID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count reverse relationships: %w", err)
	}

	return count, nil
}
