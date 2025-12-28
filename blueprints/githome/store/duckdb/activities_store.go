package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/githome/feature/activities"
)

// ActivitiesStore implements activities.Store
type ActivitiesStore struct {
	db *sql.DB
}

// NewActivitiesStore creates a new activities store
func NewActivitiesStore(db *sql.DB) *ActivitiesStore {
	return &ActivitiesStore{db: db}
}

// Create creates a new activity
func (s *ActivitiesStore) Create(ctx context.Context, a *activities.Activity) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO activities (id, actor_id, event_type, repo_id, target_type, target_id, ref, ref_type, payload, is_public, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, a.ID, a.ActorID, a.EventType, nullString(a.RepoID), nullString(a.TargetType), nullString(a.TargetID), a.Ref, a.RefType, a.Payload, a.IsPublic, a.CreatedAt)
	return err
}

// GetByID retrieves an activity by ID
func (s *ActivitiesStore) GetByID(ctx context.Context, id string) (*activities.Activity, error) {
	a := &activities.Activity{}
	var repoID, targetType, targetID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, actor_id, event_type, repo_id, target_type, target_id, ref, ref_type, payload, is_public, created_at
		FROM activities WHERE id = $1
	`, id).Scan(&a.ID, &a.ActorID, &a.EventType, &repoID, &targetType, &targetID, &a.Ref, &a.RefType, &a.Payload, &a.IsPublic, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if repoID.Valid {
		a.RepoID = repoID.String
	}
	if targetType.Valid {
		a.TargetType = targetType.String
	}
	if targetID.Valid {
		a.TargetID = targetID.String
	}
	return a, nil
}

// ListByUser lists activities for a user
func (s *ActivitiesStore) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*activities.Activity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, actor_id, event_type, repo_id, target_type, target_id, ref, ref_type, payload, is_public, created_at
		FROM activities WHERE actor_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanActivities(rows)
}

// ListByRepo lists activities for a repository
func (s *ActivitiesStore) ListByRepo(ctx context.Context, repoID string, limit, offset int) ([]*activities.Activity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, actor_id, event_type, repo_id, target_type, target_id, ref, ref_type, payload, is_public, created_at
		FROM activities WHERE repo_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, repoID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanActivities(rows)
}

// ListPublic lists public activities
func (s *ActivitiesStore) ListPublic(ctx context.Context, limit, offset int) ([]*activities.Activity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, actor_id, event_type, repo_id, target_type, target_id, ref, ref_type, payload, is_public, created_at
		FROM activities WHERE is_public = TRUE
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanActivities(rows)
}

// Delete deletes an activity
func (s *ActivitiesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM activities WHERE id = $1`, id)
	return err
}

func (s *ActivitiesStore) scanActivities(rows *sql.Rows) ([]*activities.Activity, error) {
	var list []*activities.Activity
	for rows.Next() {
		a := &activities.Activity{}
		var repoID, targetType, targetID sql.NullString
		if err := rows.Scan(&a.ID, &a.ActorID, &a.EventType, &repoID, &targetType, &targetID, &a.Ref, &a.RefType, &a.Payload, &a.IsPublic, &a.CreatedAt); err != nil {
			return nil, err
		}
		if repoID.Valid {
			a.RepoID = repoID.String
		}
		if targetType.Valid {
			a.TargetType = targetType.String
		}
		if targetID.Valid {
			a.TargetID = targetID.String
		}
		list = append(list, a)
	}
	return list, rows.Err()
}
