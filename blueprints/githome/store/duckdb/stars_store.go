package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/stars"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// StarsStore handles star data access.
type StarsStore struct {
	db *sql.DB
}

// NewStarsStore creates a new stars store.
func NewStarsStore(db *sql.DB) *StarsStore {
	return &StarsStore{db: db}
}

func (s *StarsStore) Create(ctx context.Context, userID, repoID int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO stars (user_id, repo_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, userID, repoID, time.Now())
	return err
}

func (s *StarsStore) Delete(ctx context.Context, userID, repoID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM stars WHERE user_id = $1 AND repo_id = $2
	`, userID, repoID)
	return err
}

func (s *StarsStore) Exists(ctx context.Context, userID, repoID int64) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM stars WHERE user_id = $1 AND repo_id = $2
	`, userID, repoID).Scan(&count)
	return count > 0, err
}

func (s *StarsStore) ListStargazers(ctx context.Context, repoID int64, opts *stars.ListOpts) ([]*users.SimpleUser, error) {
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
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin
		FROM users u
		JOIN stars s ON s.user_id = u.id
		WHERE s.repo_id = $1
		ORDER BY s.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimpleUsers(rows)
}

func (s *StarsStore) ListStargazersWithTimestamps(ctx context.Context, repoID int64, opts *stars.ListOpts) ([]*stars.Stargazer, error) {
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
		SELECT u.id, u.node_id, u.login, u.name, u.email, u.avatar_url, u.type, u.site_admin, s.created_at
		FROM users u
		JOIN stars s ON s.user_id = u.id
		WHERE s.repo_id = $1
		ORDER BY s.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*stars.Stargazer
	for rows.Next() {
		sg := &stars.Stargazer{User: &users.SimpleUser{}}
		var email sql.NullString
		if err := rows.Scan(&sg.User.ID, &sg.User.NodeID, &sg.User.Login, &sg.User.Name, &email,
			&sg.User.AvatarURL, &sg.User.Type, &sg.User.SiteAdmin, &sg.StarredAt); err != nil {
			return nil, err
		}
		if email.Valid {
			sg.User.Email = email.String
		}
		list = append(list, sg)
	}
	return list, rows.Err()
}

func (s *StarsStore) ListStarredRepos(ctx context.Context, userID int64, opts *stars.ListOpts) ([]*stars.Repository, error) {
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
		SELECT r.id, r.node_id, r.name, r.full_name, r.owner_id, r.private, r.description
		FROM repositories r
		JOIN stars s ON s.repo_id = r.id
		WHERE s.user_id = $1
		ORDER BY s.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*stars.Repository
	for rows.Next() {
		repo := &stars.Repository{}
		if err := rows.Scan(&repo.ID, &repo.NodeID, &repo.Name, &repo.FullName, &repo.Owner, &repo.Private, &repo.Description); err != nil {
			return nil, err
		}
		list = append(list, repo)
	}
	return list, rows.Err()
}

func (s *StarsStore) ListStarredReposWithTimestamps(ctx context.Context, userID int64, opts *stars.ListOpts) ([]*stars.StarredRepo, error) {
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
		SELECT r.id, r.node_id, r.name, r.full_name, r.owner_id, r.private, r.description, s.created_at
		FROM repositories r
		JOIN stars s ON s.repo_id = r.id
		WHERE s.user_id = $1
		ORDER BY s.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*stars.StarredRepo
	for rows.Next() {
		sr := &stars.StarredRepo{Repository: &stars.Repository{}}
		if err := rows.Scan(&sr.Repository.ID, &sr.Repository.NodeID, &sr.Repository.Name,
			&sr.Repository.FullName, &sr.Repository.Owner, &sr.Repository.Private,
			&sr.Repository.Description, &sr.StarredAt); err != nil {
			return nil, err
		}
		list = append(list, sr)
	}
	return list, rows.Err()
}
