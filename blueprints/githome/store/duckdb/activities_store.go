package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/activities"
)

// ActivitiesStore handles activity/event data access.
type ActivitiesStore struct {
	db *sql.DB
}

// NewActivitiesStore creates a new activities store.
func NewActivitiesStore(db *sql.DB) *ActivitiesStore {
	return &ActivitiesStore{db: db}
}

func (s *ActivitiesStore) Create(ctx context.Context, e *activities.Event) error {
	if e.ID == "" {
		e.ID = fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}

	actorID := int64(0)
	repoID := int64(0)
	var orgID *int64
	if e.Actor != nil {
		actorID = e.Actor.ID
	}
	if e.Repo != nil {
		repoID = e.Repo.ID
	}
	if e.Org != nil {
		orgID = &e.Org.ID
	}

	payloadJSON, _ := json.Marshal(e.Payload)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO events (id, type, actor_id, repo_id, org_id, payload, public, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, e.ID, e.Type, actorID, repoID, nullInt64Ptr(orgID), string(payloadJSON), e.Public, e.CreatedAt)
	return err
}

func (s *ActivitiesStore) GetByID(ctx context.Context, id string) (*activities.Event, error) {
	e := &activities.Event{
		Actor: &activities.Actor{},
		Repo:  &activities.EventRepo{},
	}
	var orgID sql.NullInt64
	var payloadJSON string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, type, actor_id, repo_id, org_id, payload, public, created_at
		FROM events WHERE id = $1
	`, id).Scan(&e.ID, &e.Type, &e.Actor.ID, &e.Repo.ID, &orgID, &payloadJSON, &e.Public, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if orgID.Valid {
		e.Org = &activities.Actor{ID: orgID.Int64}
	}
	if payloadJSON != "" {
		_ = json.Unmarshal([]byte(payloadJSON), &e.Payload)
	}
	return e, nil
}

func (s *ActivitiesStore) ListPublic(ctx context.Context, opts *activities.ListOpts) ([]*activities.Event, error) {
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
		SELECT id, type, actor_id, repo_id, org_id, payload, public, created_at
		FROM events
		WHERE public = TRUE
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (s *ActivitiesStore) ListForRepo(ctx context.Context, repoID int64, opts *activities.ListOpts) ([]*activities.Event, error) {
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
		SELECT id, type, actor_id, repo_id, org_id, payload, public, created_at
		FROM events
		WHERE repo_id = $1
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (s *ActivitiesStore) ListForOrg(ctx context.Context, orgID int64, opts *activities.ListOpts) ([]*activities.Event, error) {
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
		SELECT id, type, actor_id, repo_id, org_id, payload, public, created_at
		FROM events
		WHERE org_id = $1
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (s *ActivitiesStore) ListForUser(ctx context.Context, userID int64, opts *activities.ListOpts) ([]*activities.Event, error) {
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
		SELECT id, type, actor_id, repo_id, org_id, payload, public, created_at
		FROM events
		WHERE actor_id = $1
		ORDER BY created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (s *ActivitiesStore) ListReceivedByUser(ctx context.Context, userID int64, opts *activities.ListOpts) ([]*activities.Event, error) {
	page, perPage := 1, 30
	if opts != nil {
		if opts.Page > 0 {
			page = opts.Page
		}
		if opts.PerPage > 0 {
			perPage = opts.PerPage
		}
	}

	// Returns events for repos the user is watching or following
	query := `
		SELECT DISTINCT e.id, e.type, e.actor_id, e.repo_id, e.org_id, e.payload, e.public, e.created_at
		FROM events e
		LEFT JOIN watches w ON w.repo_id = e.repo_id AND w.user_id = $1 AND w.subscribed = TRUE
		LEFT JOIN user_follows f ON f.followed_id = e.actor_id AND f.follower_id = $1
		WHERE (w.user_id IS NOT NULL OR f.follower_id IS NOT NULL)
		ORDER BY e.created_at DESC`
	query = applyPagination(query, page, perPage)

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

func scanEvents(rows *sql.Rows) ([]*activities.Event, error) {
	var list []*activities.Event
	for rows.Next() {
		e := &activities.Event{
			Actor: &activities.Actor{},
			Repo:  &activities.EventRepo{},
		}
		var orgID sql.NullInt64
		var payloadJSON string
		if err := rows.Scan(&e.ID, &e.Type, &e.Actor.ID, &e.Repo.ID, &orgID, &payloadJSON, &e.Public, &e.CreatedAt); err != nil {
			return nil, err
		}
		if orgID.Valid {
			e.Org = &activities.Actor{ID: orgID.Int64}
		}
		if payloadJSON != "" {
			_ = json.Unmarshal([]byte(payloadJSON), &e.Payload)
		}
		list = append(list, e)
	}
	return list, rows.Err()
}
