package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/bi/store"
)

// QuestionStore implements store.QuestionStore.
type QuestionStore struct {
	db *sql.DB
}

func (s *QuestionStore) Create(ctx context.Context, q *store.Question) error {
	if q.ID == "" {
		q.ID = generateID()
	}
	now := time.Now()
	q.CreatedAt = now
	q.UpdatedAt = now

	var collID interface{}
	if q.CollectionID != "" {
		collID = q.CollectionID
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO questions (id, name, description, collection_id, datasource_id, query_type, query, visualization, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, q.ID, q.Name, q.Description, collID, q.DataSourceID, q.QueryType, toJSON(q.Query), toJSON(q.Visualization), q.CreatedBy, q.CreatedAt, q.UpdatedAt)
	return err
}

func (s *QuestionStore) GetByID(ctx context.Context, id string) (*store.Question, error) {
	var q store.Question
	var query, viz string
	var collID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, collection_id, datasource_id, query_type, query, visualization, created_by, created_at, updated_at
		FROM questions WHERE id = ?
	`, id).Scan(&q.ID, &q.Name, &q.Description, &collID, &q.DataSourceID, &q.QueryType, &query, &viz, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	q.CollectionID = collID.String
	fromJSON(query, &q.Query)
	fromJSON(viz, &q.Visualization)
	return &q, nil
}

func (s *QuestionStore) List(ctx context.Context) ([]*store.Question, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, collection_id, datasource_id, query_type, query, visualization, created_by, created_at, updated_at
		FROM questions ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Question
	for rows.Next() {
		var q store.Question
		var query, viz string
		var collID sql.NullString
		if err := rows.Scan(&q.ID, &q.Name, &q.Description, &collID, &q.DataSourceID, &q.QueryType, &query, &viz, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		q.CollectionID = collID.String
		fromJSON(query, &q.Query)
		fromJSON(viz, &q.Visualization)
		result = append(result, &q)
	}
	return result, rows.Err()
}

func (s *QuestionStore) ListByCollection(ctx context.Context, collectionID string) ([]*store.Question, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, collection_id, datasource_id, query_type, query, visualization, created_by, created_at, updated_at
		FROM questions WHERE collection_id = ? ORDER BY name
	`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*store.Question
	for rows.Next() {
		var q store.Question
		var query, viz string
		var collID sql.NullString
		if err := rows.Scan(&q.ID, &q.Name, &q.Description, &collID, &q.DataSourceID, &q.QueryType, &query, &viz, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		q.CollectionID = collID.String
		fromJSON(query, &q.Query)
		fromJSON(viz, &q.Visualization)
		result = append(result, &q)
	}
	return result, rows.Err()
}

func (s *QuestionStore) Update(ctx context.Context, q *store.Question) error {
	q.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE questions SET name=?, description=?, collection_id=?, query_type=?, query=?, visualization=?, updated_at=?
		WHERE id=?
	`, q.Name, q.Description, q.CollectionID, q.QueryType, toJSON(q.Query), toJSON(q.Visualization), q.UpdatedAt, q.ID)
	return err
}

func (s *QuestionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM questions WHERE id=?`, id)
	return err
}
