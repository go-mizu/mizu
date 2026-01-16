package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
)

// PagesStoreImpl implements store.PagesStore.
type PagesStoreImpl struct {
	db *sql.DB
}

// CreateProject creates a new Pages project.
func (s *PagesStoreImpl) CreateProject(ctx context.Context, project *store.PagesProject) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO pages_projects (id, name, subdomain, production_branch, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		project.ID, project.Name, project.Subdomain, project.ProductionBranch, project.CreatedAt, project.UpdatedAt)
	return err
}

// GetProject retrieves a project by ID.
func (s *PagesStoreImpl) GetProject(ctx context.Context, id string) (*store.PagesProject, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, subdomain, production_branch, created_at, updated_at FROM pages_projects WHERE id = ?`, id)
	return s.scanProject(row)
}

// GetProjectByName retrieves a project by name.
func (s *PagesStoreImpl) GetProjectByName(ctx context.Context, name string) (*store.PagesProject, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, subdomain, production_branch, created_at, updated_at FROM pages_projects WHERE name = ?`, name)
	return s.scanProject(row)
}

// ListProjects returns all projects.
func (s *PagesStoreImpl) ListProjects(ctx context.Context) ([]*store.PagesProject, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, subdomain, production_branch, created_at, updated_at FROM pages_projects ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*store.PagesProject
	for rows.Next() {
		project, err := s.scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	return projects, rows.Err()
}

// UpdateProject updates a project.
func (s *PagesStoreImpl) UpdateProject(ctx context.Context, project *store.PagesProject) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE pages_projects SET name = ?, subdomain = ?, production_branch = ?, updated_at = ? WHERE id = ?`,
		project.Name, project.Subdomain, project.ProductionBranch, time.Now(), project.ID)
	return err
}

// DeleteProject deletes a project.
func (s *PagesStoreImpl) DeleteProject(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pages_projects WHERE id = ?`, id)
	return err
}

// CreateDeployment creates a new deployment.
func (s *PagesStoreImpl) CreateDeployment(ctx context.Context, deployment *store.PagesDeployment) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO pages_deployments (id, project_id, environment, branch, commit_hash, commit_message, url, status, created_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		deployment.ID, deployment.ProjectID, deployment.Environment, deployment.Branch,
		deployment.CommitHash, deployment.CommitMessage, deployment.URL, deployment.Status,
		deployment.CreatedAt, deployment.FinishedAt)
	return err
}

// GetDeployment retrieves a deployment by ID.
func (s *PagesStoreImpl) GetDeployment(ctx context.Context, id string) (*store.PagesDeployment, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, environment, branch, commit_hash, commit_message, url, status, created_at, finished_at
		FROM pages_deployments WHERE id = ?`, id)
	return s.scanDeployment(row)
}

// ListDeployments lists deployments for a project.
func (s *PagesStoreImpl) ListDeployments(ctx context.Context, projectID string, limit int) ([]*store.PagesDeployment, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, environment, branch, commit_hash, commit_message, url, status, created_at, finished_at
		FROM pages_deployments WHERE project_id = ? ORDER BY created_at DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deployments []*store.PagesDeployment
	for rows.Next() {
		deployment, err := s.scanDeployment(rows)
		if err != nil {
			return nil, err
		}
		deployments = append(deployments, deployment)
	}
	return deployments, rows.Err()
}

// UpdateDeployment updates a deployment.
func (s *PagesStoreImpl) UpdateDeployment(ctx context.Context, deployment *store.PagesDeployment) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE pages_deployments SET status = ?, url = ?, finished_at = ? WHERE id = ?`,
		deployment.Status, deployment.URL, deployment.FinishedAt, deployment.ID)
	return err
}

// GetLatestDeployment returns the latest production deployment for a project.
func (s *PagesStoreImpl) GetLatestDeployment(ctx context.Context, projectID string) (*store.PagesDeployment, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, project_id, environment, branch, commit_hash, commit_message, url, status, created_at, finished_at
		FROM pages_deployments WHERE project_id = ? AND environment = 'production' ORDER BY created_at DESC LIMIT 1`, projectID)
	return s.scanDeployment(row)
}

// AddDomain adds a custom domain to a project.
func (s *PagesStoreImpl) AddDomain(ctx context.Context, domain *store.PagesDomain) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO pages_domains (id, project_id, domain, status, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		domain.ID, domain.ProjectID, domain.Domain, domain.Status, domain.CreatedAt)
	return err
}

// ListDomains lists custom domains for a project.
func (s *PagesStoreImpl) ListDomains(ctx context.Context, projectID string) ([]*store.PagesDomain, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, project_id, domain, status, created_at FROM pages_domains WHERE project_id = ?`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []*store.PagesDomain
	for rows.Next() {
		var domain store.PagesDomain
		if err := rows.Scan(&domain.ID, &domain.ProjectID, &domain.Domain, &domain.Status, &domain.CreatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, &domain)
	}
	return domains, rows.Err()
}

// DeleteDomain deletes a custom domain.
func (s *PagesStoreImpl) DeleteDomain(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pages_domains WHERE id = ?`, id)
	return err
}

func (s *PagesStoreImpl) scanProject(row scanner) (*store.PagesProject, error) {
	var project store.PagesProject
	if err := row.Scan(&project.ID, &project.Name, &project.Subdomain, &project.ProductionBranch,
		&project.CreatedAt, &project.UpdatedAt); err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *PagesStoreImpl) scanDeployment(row scanner) (*store.PagesDeployment, error) {
	var deployment store.PagesDeployment
	var commitHash, commitMessage, url sql.NullString
	var finishedAt sql.NullTime
	if err := row.Scan(&deployment.ID, &deployment.ProjectID, &deployment.Environment, &deployment.Branch,
		&commitHash, &commitMessage, &url, &deployment.Status, &deployment.CreatedAt, &finishedAt); err != nil {
		return nil, err
	}
	deployment.CommitHash = commitHash.String
	deployment.CommitMessage = commitMessage.String
	deployment.URL = url.String
	if finishedAt.Valid {
		deployment.FinishedAt = &finishedAt.Time
	}
	return &deployment, nil
}
