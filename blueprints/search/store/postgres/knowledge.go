package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// KnowledgeStore implements store.KnowledgeStore using PostgreSQL.
type KnowledgeStore struct {
	db *sql.DB
}

// GetEntity retrieves a knowledge panel for a query.
func (s *KnowledgeStore) GetEntity(ctx context.Context, query string) (*store.KnowledgePanel, error) {
	var entity struct {
		Name        string
		Type        string
		Description sql.NullString
		Image       sql.NullString
		Facts       []byte
		Links       []byte
	}

	// Use trigram similarity for fuzzy matching
	err := s.db.QueryRowContext(ctx, `
		SELECT name, type, description, image, facts, links
		FROM search.entities
		WHERE name ILIKE $1 OR similarity(name, $1) > 0.3
		ORDER BY similarity(name, $1) DESC
		LIMIT 1
	`, query).Scan(&entity.Name, &entity.Type, &entity.Description, &entity.Image, &entity.Facts, &entity.Links)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	panel := &store.KnowledgePanel{
		Title:    entity.Name,
		Subtitle: entity.Type,
	}

	if entity.Description.Valid {
		panel.Description = entity.Description.String
	}
	if entity.Image.Valid {
		panel.Image = entity.Image.String
	}

	// Parse facts
	if len(entity.Facts) > 0 {
		var facts map[string]any
		if err := json.Unmarshal(entity.Facts, &facts); err == nil {
			for k, v := range facts {
				panel.Facts = append(panel.Facts, store.Fact{
					Label: k,
					Value: fmt.Sprintf("%v", v),
				})
			}
		}
	}

	// Parse links
	if len(entity.Links) > 0 {
		json.Unmarshal(entity.Links, &panel.Links)
	}

	return panel, nil
}

// CreateEntity creates a new knowledge entity.
func (s *KnowledgeStore) CreateEntity(ctx context.Context, entity *store.Entity) error {
	now := time.Now()
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = now
	}
	entity.UpdatedAt = now

	factsJSON, err := json.Marshal(entity.Facts)
	if err != nil {
		factsJSON = []byte("{}")
	}

	linksJSON, err := json.Marshal(entity.Links)
	if err != nil {
		linksJSON = []byte("[]")
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO search.entities (name, type, description, image, facts, links, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, entity.Name, entity.Type, entity.Description, entity.Image, factsJSON, linksJSON, entity.CreatedAt, entity.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create entity: %w", err)
	}

	return nil
}

// UpdateEntity updates an existing knowledge entity.
func (s *KnowledgeStore) UpdateEntity(ctx context.Context, entity *store.Entity) error {
	entity.UpdatedAt = time.Now()

	factsJSON, err := json.Marshal(entity.Facts)
	if err != nil {
		factsJSON = []byte("{}")
	}

	linksJSON, err := json.Marshal(entity.Links)
	if err != nil {
		linksJSON = []byte("[]")
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE search.entities SET
			name = $2,
			type = $3,
			description = $4,
			image = $5,
			facts = $6,
			links = $7,
			updated_at = $8
		WHERE id = $1
	`, entity.ID, entity.Name, entity.Type, entity.Description, entity.Image, factsJSON, linksJSON, entity.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update entity: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entity not found")
	}

	return nil
}

// DeleteEntity deletes a knowledge entity.
func (s *KnowledgeStore) DeleteEntity(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM search.entities WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entity not found")
	}

	return nil
}

// ListEntities lists entities by type with pagination.
func (s *KnowledgeStore) ListEntities(ctx context.Context, entityType string, limit, offset int) ([]*store.Entity, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, name, type, description, image, facts, links, created_at, updated_at
		FROM search.entities
	`
	var args []interface{}
	argIdx := 1

	if entityType != "" {
		query += fmt.Sprintf(" WHERE type = $%d", argIdx)
		args = append(args, entityType)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY name LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list entities: %w", err)
	}
	defer rows.Close()

	var entities []*store.Entity
	for rows.Next() {
		var e store.Entity
		var description, image sql.NullString
		var factsJSON, linksJSON []byte

		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &description, &image, &factsJSON, &linksJSON, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan entity: %w", err)
		}

		if description.Valid {
			e.Description = description.String
		}
		if image.Valid {
			e.Image = image.String
		}
		if len(factsJSON) > 0 {
			json.Unmarshal(factsJSON, &e.Facts)
		}
		if len(linksJSON) > 0 {
			json.Unmarshal(linksJSON, &e.Links)
		}

		entities = append(entities, &e)
	}

	return entities, nil
}
