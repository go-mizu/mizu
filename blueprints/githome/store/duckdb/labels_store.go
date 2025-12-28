package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/githome/feature/labels"
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
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO labels (node_id, repo_id, name, description, color, is_default)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "", l.RepoID, l.Name, l.Description, l.Color, l.Default).Scan(&l.ID)
	if err != nil {
		return err
	}

	l.NodeID = generateNodeID("L", l.ID)
	_, err = s.db.ExecContext(ctx, `UPDATE labels SET node_id = $1 WHERE id = $2`, l.NodeID, l.ID)
	return err
}

func (s *LabelsStore) GetByID(ctx context.Context, id int64) (*labels.Label, error) {
	l := &labels.Label{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, name, description, color, is_default
		FROM labels WHERE id = $1
	`, id).Scan(&l.ID, &l.NodeID, &l.RepoID, &l.Name, &l.Description, &l.Color, &l.Default)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return l, err
}

func (s *LabelsStore) GetByName(ctx context.Context, repoID int64, name string) (*labels.Label, error) {
	l := &labels.Label{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, node_id, repo_id, name, description, color, is_default
		FROM labels WHERE repo_id = $1 AND name = $2
	`, repoID, name).Scan(&l.ID, &l.NodeID, &l.RepoID, &l.Name, &l.Description, &l.Color, &l.Default)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return l, err
}

func (s *LabelsStore) GetByNames(ctx context.Context, repoID int64, names []string) ([]*labels.Label, error) {
	if len(names) == 0 {
		return nil, nil
	}

	var result []*labels.Label
	for _, name := range names {
		l, err := s.GetByName(ctx, repoID, name)
		if err != nil {
			return nil, err
		}
		if l != nil {
			result = append(result, l)
		}
	}
	return result, nil
}

func (s *LabelsStore) Update(ctx context.Context, id int64, in *labels.UpdateIn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE labels SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			color = COALESCE($4, color)
		WHERE id = $1
	`, id, nullStringPtr(in.NewName), nullStringPtr(in.Description), nullStringPtr(in.Color))
	return err
}

func (s *LabelsStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM labels WHERE id = $1`, id)
	return err
}

func (s *LabelsStore) List(ctx context.Context, repoID int64, opts *labels.ListOpts) ([]*labels.Label, error) {
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
		SELECT id, node_id, repo_id, name, description, color, is_default
		FROM labels WHERE repo_id = $1
		ORDER BY name ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanLabels(rows)
}

func (s *LabelsStore) ListForIssue(ctx context.Context, issueID int64, opts *labels.ListOpts) ([]*labels.Label, error) {
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
		SELECT l.id, l.node_id, l.repo_id, l.name, l.description, l.color, l.is_default
		FROM labels l
		JOIN issue_labels il ON il.label_id = l.id
		WHERE il.issue_id = $1
		ORDER BY l.name ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanLabels(rows)
}

func (s *LabelsStore) AddToIssue(ctx context.Context, issueID, labelID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO issue_labels (issue_id, label_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, issueID, labelID)
	return err
}

func (s *LabelsStore) RemoveFromIssue(ctx context.Context, issueID, labelID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM issue_labels WHERE issue_id = $1 AND label_id = $2
	`, issueID, labelID)
	return err
}

func (s *LabelsStore) SetForIssue(ctx context.Context, issueID int64, labelIDs []int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issue_labels WHERE issue_id = $1`, issueID)
	if err != nil {
		return err
	}

	for _, labelID := range labelIDs {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO issue_labels (issue_id, label_id) VALUES ($1, $2)
		`, issueID, labelID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *LabelsStore) RemoveAllFromIssue(ctx context.Context, issueID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM issue_labels WHERE issue_id = $1`, issueID)
	return err
}

func (s *LabelsStore) ListForMilestone(ctx context.Context, milestoneID int64, opts *labels.ListOpts) ([]*labels.Label, error) {
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
		SELECT DISTINCT l.id, l.node_id, l.repo_id, l.name, l.description, l.color, l.is_default
		FROM labels l
		JOIN issue_labels il ON il.label_id = l.id
		JOIN issues i ON i.id = il.issue_id
		WHERE i.milestone_id = $1
		ORDER BY l.name ASC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, milestoneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanLabels(rows)
}

func scanLabels(rows *sql.Rows) ([]*labels.Label, error) {
	var list []*labels.Label
	for rows.Next() {
		l := &labels.Label{}
		if err := rows.Scan(&l.ID, &l.NodeID, &l.RepoID, &l.Name, &l.Description, &l.Color, &l.Default); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}
