package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/go-mizu/blueprints/chat/feature/presence"
)

// PresenceStore implements presence.Store.
type PresenceStore struct {
	db *sql.DB
}

// NewPresenceStore creates a new PresenceStore.
func NewPresenceStore(db *sql.DB) *PresenceStore {
	return &PresenceStore{db: db}
}

// Upsert creates or updates presence.
func (s *PresenceStore) Upsert(ctx context.Context, p *presence.Presence) error {
	activitiesJSON, _ := json.Marshal(p.Activities)
	clientStatusJSON, _ := json.Marshal(p.ClientStatus)

	query := `
		INSERT INTO presence (user_id, status, custom_status, activities, client_status, last_seen_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			status = EXCLUDED.status,
			custom_status = EXCLUDED.custom_status,
			activities = EXCLUDED.activities,
			client_status = EXCLUDED.client_status,
			last_seen_at = EXCLUDED.last_seen_at
	`
	_, err := s.db.ExecContext(ctx, query,
		p.UserID, p.Status, p.CustomStatus, activitiesJSON, clientStatusJSON, p.LastSeenAt,
	)
	return err
}

// Get retrieves presence for a user.
func (s *PresenceStore) Get(ctx context.Context, userID string) (*presence.Presence, error) {
	query := `
		SELECT user_id, status, custom_status, activities, client_status, last_seen_at
		FROM presence WHERE user_id = ?
	`
	p := &presence.Presence{}
	var customStatus sql.NullString
	var activitiesRaw, clientStatusRaw any
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&p.UserID, &p.Status, &customStatus, &activitiesRaw, &clientStatusRaw, &p.LastSeenAt,
	)
	if err == sql.ErrNoRows {
		return &presence.Presence{
			UserID:     userID,
			Status:     presence.StatusOffline,
			LastSeenAt: time.Now(),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	p.CustomStatus = customStatus.String

	// DuckDB returns JSON columns as native Go types, re-marshal to unmarshal into target structs
	if activitiesRaw != nil {
		if b, err := json.Marshal(activitiesRaw); err == nil {
			json.Unmarshal(b, &p.Activities)
		}
	}
	if clientStatusRaw != nil {
		if b, err := json.Marshal(clientStatusRaw); err == nil {
			json.Unmarshal(b, &p.ClientStatus)
		}
	}

	return p, nil
}

// GetBulk retrieves presence for multiple users.
func (s *PresenceStore) GetBulk(ctx context.Context, userIDs []string) ([]*presence.Presence, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	// Build placeholders
	placeholders := make([]string, len(userIDs))
	args := make([]any, len(userIDs))
	for i, id := range userIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT user_id, status, custom_status, activities, client_status, last_seen_at
		FROM presence
		WHERE user_id IN (` + stringJoin(placeholders, ",") + `)
	`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	presenceMap := make(map[string]*presence.Presence)
	for rows.Next() {
		p := &presence.Presence{}
		var customStatus sql.NullString
		var activitiesRaw, clientStatusRaw any
		if err := rows.Scan(
			&p.UserID, &p.Status, &customStatus, &activitiesRaw, &clientStatusRaw, &p.LastSeenAt,
		); err != nil {
			return nil, err
		}
		p.CustomStatus = customStatus.String
		// DuckDB returns JSON columns as native Go types
		if activitiesRaw != nil {
			if b, err := json.Marshal(activitiesRaw); err == nil {
				json.Unmarshal(b, &p.Activities)
			}
		}
		if clientStatusRaw != nil {
			if b, err := json.Marshal(clientStatusRaw); err == nil {
				json.Unmarshal(b, &p.ClientStatus)
			}
		}
		presenceMap[p.UserID] = p
	}

	// Build result with default offline for missing users
	result := make([]*presence.Presence, len(userIDs))
	for i, id := range userIDs {
		if p, ok := presenceMap[id]; ok {
			result[i] = p
		} else {
			result[i] = &presence.Presence{
				UserID:     id,
				Status:     presence.StatusOffline,
				LastSeenAt: time.Now(),
			}
		}
	}
	return result, rows.Err()
}

// UpdateStatus updates just the status.
func (s *PresenceStore) UpdateStatus(ctx context.Context, userID string, status presence.Status) error {
	query := `
		INSERT INTO presence (user_id, status, last_seen_at)
		VALUES (?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			status = EXCLUDED.status,
			last_seen_at = EXCLUDED.last_seen_at
	`
	_, err := s.db.ExecContext(ctx, query, userID, status, time.Now())
	return err
}

// SetOffline sets a user offline.
func (s *PresenceStore) SetOffline(ctx context.Context, userID string) error {
	return s.UpdateStatus(ctx, userID, presence.StatusOffline)
}

// Delete removes presence.
func (s *PresenceStore) Delete(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM presence WHERE user_id = ?", userID)
	return err
}

// CleanupStale removes stale presence entries.
func (s *PresenceStore) CleanupStale(ctx context.Context, before time.Time) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE presence SET status = 'offline' WHERE last_seen_at < ? AND status != 'offline'",
		before,
	)
	return err
}

func stringJoin(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for i := 1; i < len(s); i++ {
		result += sep + s[i]
	}
	return result
}
