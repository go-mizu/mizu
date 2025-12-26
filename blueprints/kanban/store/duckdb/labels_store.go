package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/kanban/feature/labels"
)

// LabelsStore handles label data access.
type LabelsStore struct {
	db *sql.DB
}

// NewLabelsStore creates a new labels store.
func NewLabelsStore(db *sql.DB) *LabelsStore {
	return &LabelsStore{db: db}
}

func (s *LabelsStore) Create(ctx context.Context, l *labels.Label) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO labels (id, project_id, name, color, description)
		VALUES ($1, $2, $3, $4, $5)
	`, l.ID, l.ProjectID, l.Name, l.Color, l.Description)
	return err
}

func (s *LabelsStore) GetByID(ctx context.Context, id string) (*labels.Label, error) {
	l := &labels.Label{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, color, description
		FROM labels WHERE id = $1
	`, id).Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color, &l.Description)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return l, err
}

func (s *LabelsStore) ListByProject(ctx context.Context, projectID string) ([]*labels.Label, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, name, color, description
		FROM labels WHERE project_id = $1
		ORDER BY name
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*labels.Label
	for rows.Next() {
		l := &labels.Label{}
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color, &l.Description); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

func (s *LabelsStore) Update(ctx context.Context, id string, in *labels.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE labels SET
			name = COALESCE($2, name),
			color = COALESCE($3, color),
			description = COALESCE($4, description)
		WHERE id = $1
	`, id, in.Name, in.Color, in.Description)
	return err
}

func (s *LabelsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM labels WHERE id = $1`, id)
	return err
}

func (s *LabelsStore) GetByIssue(ctx context.Context, issueID string) ([]*labels.Label, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT l.id, l.project_id, l.name, l.color, l.description
		FROM labels l
		INNER JOIN issue_labels il ON l.id = il.label_id
		WHERE il.issue_id = $1
		ORDER BY l.name
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*labels.Label
	for rows.Next() {
		l := &labels.Label{}
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.Name, &l.Color, &l.Description); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}
