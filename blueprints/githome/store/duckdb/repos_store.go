package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// ReposStore implements repos.Store
type ReposStore struct {
	db *sql.DB
}

// NewReposStore creates a new repos store
func NewReposStore(db *sql.DB) *ReposStore {
	return &ReposStore{db: db}
}

// Create creates a new repository
func (s *ReposStore) Create(ctx context.Context, r *repos.Repository) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO repositories (id, owner_actor_id, name, slug, description, website, default_branch, is_private, is_archived, is_template, is_fork, forked_from_repo_id, star_count, fork_count, watcher_count, open_issue_count, open_pr_count, size_kb, license, has_issues, has_wiki, has_projects, created_at, updated_at, pushed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
	`, r.ID, r.OwnerActorID, r.Name, r.Slug, r.Description, r.Website, r.DefaultBranch, r.IsPrivate, r.IsArchived, r.IsTemplate, r.IsFork, nullString(r.ForkedFromID), r.StarCount, r.ForkCount, r.WatcherCount, r.OpenIssueCount, r.OpenPRCount, r.SizeKB, r.License, r.HasIssues, r.HasWiki, r.HasProjects, r.CreatedAt, r.UpdatedAt, nullTime(r.PushedAt))
	if err != nil {
		return err
	}

	// Insert topics to repo_topics table
	for _, topic := range r.Topics {
		if topic != "" {
			s.db.ExecContext(ctx, `
				INSERT INTO repo_topics (repo_id, topic, created_at)
				VALUES ($1, $2, $3)
			`, r.ID, topic, time.Now())
		}
	}

	return nil
}

// GetByID retrieves a repository by ID
func (s *ReposStore) GetByID(ctx context.Context, id string) (*repos.Repository, error) {
	r := &repos.Repository{}
	var pushedAt sql.NullTime
	var forkedFromID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, owner_actor_id, name, slug, description, website, default_branch, is_private, is_archived, is_template, is_fork, forked_from_repo_id, star_count, fork_count, watcher_count, open_issue_count, open_pr_count, size_kb, license, has_issues, has_wiki, has_projects, created_at, updated_at, pushed_at
		FROM repositories WHERE id = $1
	`, id).Scan(&r.ID, &r.OwnerActorID, &r.Name, &r.Slug, &r.Description, &r.Website, &r.DefaultBranch, &r.IsPrivate, &r.IsArchived, &r.IsTemplate, &r.IsFork, &forkedFromID, &r.StarCount, &r.ForkCount, &r.WatcherCount, &r.OpenIssueCount, &r.OpenPRCount, &r.SizeKB, &r.License, &r.HasIssues, &r.HasWiki, &r.HasProjects, &r.CreatedAt, &r.UpdatedAt, &pushedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if pushedAt.Valid {
		r.PushedAt = pushedAt.Time
	}
	if forkedFromID.Valid {
		r.ForkedFromID = forkedFromID.String
	}

	// Load topics
	r.Topics, _ = s.getTopics(ctx, r.ID)

	return r, nil
}

// GetByOwnerAndName retrieves a repository by owner actor ID and slug
func (s *ReposStore) GetByOwnerAndName(ctx context.Context, ownerActorID, _, name string) (*repos.Repository, error) {
	r := &repos.Repository{}
	var pushedAt sql.NullTime
	var forkedFromID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, owner_actor_id, name, slug, description, website, default_branch, is_private, is_archived, is_template, is_fork, forked_from_repo_id, star_count, fork_count, watcher_count, open_issue_count, open_pr_count, size_kb, license, has_issues, has_wiki, has_projects, created_at, updated_at, pushed_at
		FROM repositories WHERE owner_actor_id = $1 AND slug = $2
	`, ownerActorID, name).Scan(&r.ID, &r.OwnerActorID, &r.Name, &r.Slug, &r.Description, &r.Website, &r.DefaultBranch, &r.IsPrivate, &r.IsArchived, &r.IsTemplate, &r.IsFork, &forkedFromID, &r.StarCount, &r.ForkCount, &r.WatcherCount, &r.OpenIssueCount, &r.OpenPRCount, &r.SizeKB, &r.License, &r.HasIssues, &r.HasWiki, &r.HasProjects, &r.CreatedAt, &r.UpdatedAt, &pushedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if pushedAt.Valid {
		r.PushedAt = pushedAt.Time
	}
	if forkedFromID.Valid {
		r.ForkedFromID = forkedFromID.String
	}

	// Load topics
	r.Topics, _ = s.getTopics(ctx, r.ID)

	return r, nil
}

// Update updates a repository
func (s *ReposStore) Update(ctx context.Context, r *repos.Repository) error {
	r.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE repositories SET name = $2, slug = $3, description = $4, website = $5, default_branch = $6, is_private = $7, is_archived = $8, is_template = $9, star_count = $10, fork_count = $11, watcher_count = $12, open_issue_count = $13, open_pr_count = $14, size_kb = $15, license = $16, has_issues = $17, has_wiki = $18, has_projects = $19, updated_at = $20, pushed_at = $21
		WHERE id = $1
	`, r.ID, r.Name, r.Slug, r.Description, r.Website, r.DefaultBranch, r.IsPrivate, r.IsArchived, r.IsTemplate, r.StarCount, r.ForkCount, r.WatcherCount, r.OpenIssueCount, r.OpenPRCount, r.SizeKB, r.License, r.HasIssues, r.HasWiki, r.HasProjects, r.UpdatedAt, nullTime(r.PushedAt))
	if err != nil {
		return err
	}

	// Update topics: delete all and re-insert
	s.db.ExecContext(ctx, `DELETE FROM repo_topics WHERE repo_id = $1`, r.ID)
	for _, topic := range r.Topics {
		if topic != "" {
			s.db.ExecContext(ctx, `
				INSERT INTO repo_topics (repo_id, topic, created_at)
				VALUES ($1, $2, $3)
			`, r.ID, topic, time.Now())
		}
	}

	return nil
}

// Delete deletes a repository
func (s *ReposStore) Delete(ctx context.Context, id string) error {
	// Topics are deleted by CASCADE
	_, err := s.db.ExecContext(ctx, `DELETE FROM repositories WHERE id = $1`, id)
	return err
}

// ListByOwner lists repositories by owner actor ID
func (s *ReposStore) ListByOwner(ctx context.Context, ownerActorID, _ string, limit, offset int) ([]*repos.Repository, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_actor_id, name, slug, description, website, default_branch, is_private, is_archived, is_template, is_fork, forked_from_repo_id, star_count, fork_count, watcher_count, open_issue_count, open_pr_count, size_kb, license, has_issues, has_wiki, has_projects, created_at, updated_at, pushed_at
		FROM repositories WHERE owner_actor_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3
	`, ownerActorID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRepos(ctx, rows)
}

// ListPublic lists public repositories
func (s *ReposStore) ListPublic(ctx context.Context, limit, offset int) ([]*repos.Repository, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_actor_id, name, slug, description, website, default_branch, is_private, is_archived, is_template, is_fork, forked_from_repo_id, star_count, fork_count, watcher_count, open_issue_count, open_pr_count, size_kb, license, has_issues, has_wiki, has_projects, created_at, updated_at, pushed_at
		FROM repositories WHERE is_private = FALSE ORDER BY star_count DESC, updated_at DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRepos(ctx, rows)
}

// ListByIDs lists repositories by their IDs
func (s *ReposStore) ListByIDs(ctx context.Context, ids []string) ([]*repos.Repository, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Build query with placeholders
	query := `
		SELECT id, owner_actor_id, name, slug, description, website, default_branch, is_private, is_archived, is_template, is_fork, forked_from_repo_id, star_count, fork_count, watcher_count, open_issue_count, open_pr_count, size_kb, license, has_issues, has_wiki, has_projects, created_at, updated_at, pushed_at
		FROM repositories WHERE id IN (`
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query += `) ORDER BY updated_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRepos(ctx, rows)
}

func (s *ReposStore) scanRepos(ctx context.Context, rows *sql.Rows) ([]*repos.Repository, error) {
	var list []*repos.Repository
	for rows.Next() {
		r := &repos.Repository{}
		var pushedAt sql.NullTime
		var forkedFromID sql.NullString
		if err := rows.Scan(&r.ID, &r.OwnerActorID, &r.Name, &r.Slug, &r.Description, &r.Website, &r.DefaultBranch, &r.IsPrivate, &r.IsArchived, &r.IsTemplate, &r.IsFork, &forkedFromID, &r.StarCount, &r.ForkCount, &r.WatcherCount, &r.OpenIssueCount, &r.OpenPRCount, &r.SizeKB, &r.License, &r.HasIssues, &r.HasWiki, &r.HasProjects, &r.CreatedAt, &r.UpdatedAt, &pushedAt); err != nil {
			return nil, err
		}
		if pushedAt.Valid {
			r.PushedAt = pushedAt.Time
		}
		if forkedFromID.Valid {
			r.ForkedFromID = forkedFromID.String
		}
		// Load topics for each repo
		r.Topics, _ = s.getTopics(ctx, r.ID)
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *ReposStore) getTopics(ctx context.Context, repoID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT topic FROM repo_topics WHERE repo_id = $1`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []string
	for rows.Next() {
		var topic string
		if err := rows.Scan(&topic); err != nil {
			return nil, err
		}
		topics = append(topics, topic)
	}
	return topics, rows.Err()
}

// Star stars a repository - uses composite PK (user_id, repo_id)
func (s *ReposStore) Star(ctx context.Context, star *repos.Star) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO stars (user_id, repo_id, created_at)
		VALUES ($1, $2, $3)
	`, star.UserID, star.RepoID, star.CreatedAt)
	return err
}

// Unstar removes a star from a repository
func (s *ReposStore) Unstar(ctx context.Context, userID, repoID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM stars WHERE user_id = $1 AND repo_id = $2`, userID, repoID)
	return err
}

// IsStarred checks if a user has starred a repository
func (s *ReposStore) IsStarred(ctx context.Context, userID, repoID string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM stars WHERE user_id = $1 AND repo_id = $2`, userID, repoID).Scan(&count)
	return count > 0, err
}

// ListStarredByUser lists repositories starred by a user
func (s *ReposStore) ListStarredByUser(ctx context.Context, userID string, limit, offset int) ([]*repos.Repository, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.owner_actor_id, r.name, r.slug, r.description, r.website, r.default_branch, r.is_private, r.is_archived, r.is_template, r.is_fork, r.forked_from_repo_id, r.star_count, r.fork_count, r.watcher_count, r.open_issue_count, r.open_pr_count, r.size_kb, r.license, r.has_issues, r.has_wiki, r.has_projects, r.created_at, r.updated_at, r.pushed_at
		FROM repositories r
		JOIN stars s ON r.id = s.repo_id
		WHERE s.user_id = $1
		ORDER BY s.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanRepos(ctx, rows)
}

// AddCollaborator adds a collaborator to a repository - uses composite PK (repo_id, user_id)
func (s *ReposStore) AddCollaborator(ctx context.Context, collab *repos.Collaborator) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO collaborators (repo_id, user_id, permission, created_at)
		VALUES ($1, $2, $3, $4)
	`, collab.RepoID, collab.UserID, collab.Permission, collab.CreatedAt)
	return err
}

// RemoveCollaborator removes a collaborator from a repository
func (s *ReposStore) RemoveCollaborator(ctx context.Context, repoID, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM collaborators WHERE repo_id = $1 AND user_id = $2`, repoID, userID)
	return err
}

// GetCollaborator gets a collaborator
func (s *ReposStore) GetCollaborator(ctx context.Context, repoID, userID string) (*repos.Collaborator, error) {
	c := &repos.Collaborator{}
	err := s.db.QueryRowContext(ctx, `
		SELECT repo_id, user_id, permission, created_at
		FROM collaborators WHERE repo_id = $1 AND user_id = $2
	`, repoID, userID).Scan(&c.RepoID, &c.UserID, &c.Permission, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// ListCollaborators lists collaborators for a repository
func (s *ReposStore) ListCollaborators(ctx context.Context, repoID string) ([]*repos.Collaborator, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT repo_id, user_id, permission, created_at
		FROM collaborators WHERE repo_id = $1
	`, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*repos.Collaborator
	for rows.Next() {
		c := &repos.Collaborator{}
		if err := rows.Scan(&c.RepoID, &c.UserID, &c.Permission, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// nullTime converts time.Time to sql.NullTime for zero values
func nullTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}
