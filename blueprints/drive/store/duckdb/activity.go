package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// Activity represents an activity log record.
type Activity struct {
	ID           string
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	ResourceName sql.NullString
	Details      sql.NullString
	IPAddress    sql.NullString
	UserAgent    sql.NullString
	CreatedAt    time.Time
}

// CreateActivity inserts a new activity.
func (s *Store) CreateActivity(ctx context.Context, a *Activity) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO activities (id, user_id, action, resource_type, resource_id, resource_name, details, ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, a.ID, a.UserID, a.Action, a.ResourceType, a.ResourceID, a.ResourceName, a.Details, a.IPAddress, a.UserAgent, a.CreatedAt)
	return err
}

// ListActivitiesByUser lists all activities for a user.
func (s *Store) ListActivitiesByUser(ctx context.Context, userID string, limit int) ([]*Activity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, action, resource_type, resource_id, resource_name, details, ip_address, user_agent, created_at
		FROM activities WHERE user_id = ? ORDER BY created_at DESC LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanActivities(rows)
}

// ListActivitiesForResource lists activities for a specific resource.
func (s *Store) ListActivitiesForResource(ctx context.Context, resourceType, resourceID string, limit int) ([]*Activity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, action, resource_type, resource_id, resource_name, details, ip_address, user_agent, created_at
		FROM activities WHERE resource_type = ? AND resource_id = ? ORDER BY created_at DESC LIMIT ?
	`, resourceType, resourceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanActivities(rows)
}

// ListRecentActivities lists the most recent activities across all users.
func (s *Store) ListRecentActivities(ctx context.Context, limit int) ([]*Activity, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, action, resource_type, resource_id, resource_name, details, ip_address, user_agent, created_at
		FROM activities ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanActivities(rows)
}

// DeleteActivitiesForResource deletes all activities for a resource.
func (s *Store) DeleteActivitiesForResource(ctx context.Context, resourceType, resourceID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM activities WHERE resource_type = ? AND resource_id = ?`, resourceType, resourceID)
	return err
}

// DeleteOldActivities deletes activities older than the specified duration.
func (s *Store) DeleteOldActivities(ctx context.Context, before time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM activities WHERE created_at < ?`, before)
	return err
}

// CountActivitiesByAction counts activities by action type for a user.
func (s *Store) CountActivitiesByAction(ctx context.Context, userID string) (map[string]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT action, COUNT(*) as count FROM activities WHERE user_id = ? GROUP BY action
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var action string
		var count int64
		if err := rows.Scan(&action, &count); err != nil {
			return nil, err
		}
		counts[action] = count
	}
	return counts, rows.Err()
}

func scanActivities(rows *sql.Rows) ([]*Activity, error) {
	var activities []*Activity
	for rows.Next() {
		a := &Activity{}
		if err := rows.Scan(&a.ID, &a.UserID, &a.Action, &a.ResourceType, &a.ResourceID, &a.ResourceName, &a.Details, &a.IPAddress, &a.UserAgent, &a.CreatedAt); err != nil {
			return nil, err
		}
		activities = append(activities, a)
	}
	return activities, rows.Err()
}
