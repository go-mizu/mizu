package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/feature/watches"
)

// WatchesStore handles watch/subscription data access.
type WatchesStore struct {
	db *sql.DB
}

// NewWatchesStore creates a new watches store.
func NewWatchesStore(db *sql.DB) *WatchesStore {
	return &WatchesStore{db: db}
}

func (s *WatchesStore) Create(ctx context.Context, userID, repoID int64, subscribed, ignored bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO watches (user_id, repo_id, subscribed, ignored, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING
	`, userID, repoID, subscribed, ignored, time.Now())
	return err
}

func (s *WatchesStore) Update(ctx context.Context, userID, repoID int64, subscribed, ignored bool) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE watches SET subscribed = $3, ignored = $4
		WHERE user_id = $1 AND repo_id = $2
	`, userID, repoID, subscribed, ignored)
	return err
}

func (s *WatchesStore) Delete(ctx context.Context, userID, repoID int64) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM watches WHERE user_id = $1 AND repo_id = $2
	`, userID, repoID)
	return err
}

func (s *WatchesStore) Get(ctx context.Context, userID, repoID int64) (*watches.Subscription, error) {
	sub := &watches.Subscription{}
	err := s.db.QueryRowContext(ctx, `
		SELECT subscribed, ignored, created_at
		FROM watches WHERE user_id = $1 AND repo_id = $2
	`, userID, repoID).Scan(&sub.Subscribed, &sub.Ignored, &sub.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sub, err
}

func (s *WatchesStore) ListWatchers(ctx context.Context, repoID int64, opts *watches.ListOpts) ([]*users.SimpleUser, error) {
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
		JOIN watches w ON w.user_id = u.id
		WHERE w.repo_id = $1 AND w.subscribed = TRUE
		ORDER BY w.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSimpleUsers(rows)
}

func (s *WatchesStore) ListWatchedRepos(ctx context.Context, userID int64, opts *watches.ListOpts) ([]*watches.Repository, error) {
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
		JOIN watches w ON w.repo_id = r.id
		WHERE w.user_id = $1 AND w.subscribed = TRUE
		ORDER BY w.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*watches.Repository
	for rows.Next() {
		repo := &watches.Repository{}
		if err := rows.Scan(&repo.ID, &repo.NodeID, &repo.Name, &repo.FullName, &repo.Owner, &repo.Private, &repo.Description); err != nil {
			return nil, err
		}
		list = append(list, repo)
	}
	return list, rows.Err()
}
