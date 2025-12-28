package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Actor represents a unified owner (user or org)
type Actor struct {
	ID        string
	ActorType string // "user" or "org"
	UserID    string
	OrgID     string
	CreatedAt time.Time
}

// ActorsStore manages actor records
type ActorsStore struct {
	db *sql.DB
}

// NewActorsStore creates a new actors store
func NewActorsStore(db *sql.DB) *ActorsStore {
	return &ActorsStore{db: db}
}

// Create creates a new actor
func (s *ActorsStore) Create(ctx context.Context, a *Actor) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO actors (id, actor_type, user_id, org_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, a.ID, a.ActorType, nullString(a.UserID), nullString(a.OrgID), a.CreatedAt)
	return err
}

// GetByID retrieves an actor by ID
func (s *ActorsStore) GetByID(ctx context.Context, id string) (*Actor, error) {
	a := &Actor{}
	var userID, orgID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, actor_type, user_id, org_id, created_at
		FROM actors WHERE id = $1
	`, id).Scan(&a.ID, &a.ActorType, &userID, &orgID, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if userID.Valid {
		a.UserID = userID.String
	}
	if orgID.Valid {
		a.OrgID = orgID.String
	}
	return a, nil
}

// GetByUserID retrieves an actor by user ID
func (s *ActorsStore) GetByUserID(ctx context.Context, userID string) (*Actor, error) {
	a := &Actor{}
	var uid, oid sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, actor_type, user_id, org_id, created_at
		FROM actors WHERE user_id = $1
	`, userID).Scan(&a.ID, &a.ActorType, &uid, &oid, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if uid.Valid {
		a.UserID = uid.String
	}
	if oid.Valid {
		a.OrgID = oid.String
	}
	return a, nil
}

// GetByOrgID retrieves an actor by org ID
func (s *ActorsStore) GetByOrgID(ctx context.Context, orgID string) (*Actor, error) {
	a := &Actor{}
	var uid, oid sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, actor_type, user_id, org_id, created_at
		FROM actors WHERE org_id = $1
	`, orgID).Scan(&a.ID, &a.ActorType, &uid, &oid, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if uid.Valid {
		a.UserID = uid.String
	}
	if oid.Valid {
		a.OrgID = oid.String
	}
	return a, nil
}

// GetOrCreateForUser gets or creates an actor for a user
func (s *ActorsStore) GetOrCreateForUser(ctx context.Context, userID string) (*Actor, error) {
	actor, err := s.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if actor != nil {
		return actor, nil
	}

	// Create new actor
	actor = &Actor{
		ID:        generateID(),
		ActorType: "user",
		UserID:    userID,
		CreatedAt: time.Now(),
	}
	if err := s.Create(ctx, actor); err != nil {
		return nil, err
	}
	return actor, nil
}

// GetOrCreateForOrg gets or creates an actor for an org
func (s *ActorsStore) GetOrCreateForOrg(ctx context.Context, orgID string) (*Actor, error) {
	actor, err := s.GetByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if actor != nil {
		return actor, nil
	}

	// Create new actor
	actor = &Actor{
		ID:        generateID(),
		ActorType: "org",
		OrgID:     orgID,
		CreatedAt: time.Now(),
	}
	if err := s.Create(ctx, actor); err != nil {
		return nil, err
	}
	return actor, nil
}

// Delete deletes an actor
func (s *ActorsStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM actors WHERE id = $1`, id)
	return err
}

// DeleteByUserID deletes an actor by user ID
func (s *ActorsStore) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM actors WHERE user_id = $1`, userID)
	return err
}

// DeleteByOrgID deletes an actor by org ID
func (s *ActorsStore) DeleteByOrgID(ctx context.Context, orgID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM actors WHERE org_id = $1`, orgID)
	return err
}

// generateID generates a new ID using ULID
func generateID() string {
	return ulid.New()
}
