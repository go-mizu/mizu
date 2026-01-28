package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// KnowledgeStore handles knowledge graph operations.
type KnowledgeStore struct {
	db *sql.DB
}

// GetEntity retrieves a knowledge panel for a query.
func (s *KnowledgeStore) GetEntity(ctx context.Context, query string) (*store.KnowledgePanel, error) {
	var entity struct {
		Name        string
		Type        string
		Description string
		Image       string
		Facts       string
		Links       string
	}

	// Try exact match first, then FTS
	err := s.db.QueryRowContext(ctx, `
		SELECT name, type, description, image, facts, links
		FROM entities
		WHERE name = ? COLLATE NOCASE
		LIMIT 1
	`, query).Scan(&entity.Name, &entity.Type, &entity.Description, &entity.Image, &entity.Facts, &entity.Links)

	if err == sql.ErrNoRows {
		// Try FTS search
		err = s.db.QueryRowContext(ctx, `
			SELECT e.name, e.type, e.description, e.image, e.facts, e.links
			FROM entities e
			JOIN entities_fts fts ON e.rowid = fts.rowid
			WHERE entities_fts MATCH ?
			ORDER BY bm25(entities_fts)
			LIMIT 1
		`, query+"*").Scan(&entity.Name, &entity.Type, &entity.Description, &entity.Image, &entity.Facts, &entity.Links)
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse facts
	var facts map[string]any
	if entity.Facts != "" && entity.Facts != "{}" {
		json.Unmarshal([]byte(entity.Facts), &facts)
	}

	// Parse links
	var links []store.Link
	if entity.Links != "" && entity.Links != "[]" {
		json.Unmarshal([]byte(entity.Links), &links)
	}

	// Build knowledge panel
	panel := &store.KnowledgePanel{
		Title:       entity.Name,
		Subtitle:    entity.Type,
		Description: entity.Description,
		Image:       entity.Image,
		Links:       links,
		Source:      "Knowledge Graph",
	}

	// Convert facts to panel facts
	for k, v := range facts {
		panel.Facts = append(panel.Facts, store.Fact{
			Label: k,
			Value: fmt.Sprintf("%v", v),
		})
	}

	return panel, nil
}

// CreateEntity creates a new knowledge entity.
func (s *KnowledgeStore) CreateEntity(ctx context.Context, entity *store.Entity) error {
	if entity.ID == "" {
		entity.ID = generateID()
	}
	entity.CreatedAt = time.Now()
	entity.UpdatedAt = time.Now()

	factsJSON, _ := json.Marshal(entity.Facts)
	linksJSON, _ := json.Marshal(entity.Links)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO entities (id, name, type, description, image, facts, links, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entity.ID, entity.Name, entity.Type, entity.Description, entity.Image,
		string(factsJSON), string(linksJSON), entity.CreatedAt, entity.UpdatedAt)

	return err
}

// UpdateEntity updates an existing entity.
func (s *KnowledgeStore) UpdateEntity(ctx context.Context, entity *store.Entity) error {
	entity.UpdatedAt = time.Now()

	factsJSON, _ := json.Marshal(entity.Facts)
	linksJSON, _ := json.Marshal(entity.Links)

	result, err := s.db.ExecContext(ctx, `
		UPDATE entities SET
			name = ?,
			type = ?,
			description = ?,
			image = ?,
			facts = ?,
			links = ?,
			updated_at = ?
		WHERE id = ?
	`, entity.Name, entity.Type, entity.Description, entity.Image,
		string(factsJSON), string(linksJSON), entity.UpdatedAt, entity.ID)

	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entity not found")
	}

	return nil
}

// DeleteEntity removes an entity.
func (s *KnowledgeStore) DeleteEntity(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM entities WHERE id = ?", id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("entity not found")
	}

	return nil
}

// ListEntities lists entities by type.
func (s *KnowledgeStore) ListEntities(ctx context.Context, entityType string, limit, offset int) ([]*store.Entity, error) {
	var rows *sql.Rows
	var err error

	if entityType != "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, type, description, image, facts, links, created_at, updated_at
			FROM entities
			WHERE type = ?
			ORDER BY name
			LIMIT ? OFFSET ?
		`, entityType, limit, offset)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, name, type, description, image, facts, links, created_at, updated_at
			FROM entities
			ORDER BY name
			LIMIT ? OFFSET ?
		`, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []*store.Entity
	for rows.Next() {
		var e store.Entity
		var desc, image, factsStr, linksStr sql.NullString

		if err := rows.Scan(&e.ID, &e.Name, &e.Type, &desc, &image, &factsStr, &linksStr, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}

		if desc.Valid {
			e.Description = desc.String
		}
		if image.Valid {
			e.Image = image.String
		}
		if factsStr.Valid && factsStr.String != "" {
			json.Unmarshal([]byte(factsStr.String), &e.Facts)
		}
		if linksStr.Valid && linksStr.String != "" {
			json.Unmarshal([]byte(linksStr.String), &e.Links)
		}

		entities = append(entities, &e)
	}

	return entities, nil
}
