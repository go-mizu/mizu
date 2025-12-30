package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/cms/pkg/ulid"
)

// PreferencesStore handles user preference operations.
type PreferencesStore struct {
	db *sql.DB
}

// NewPreferencesStore creates a new PreferencesStore.
func NewPreferencesStore(db *sql.DB) *PreferencesStore {
	return &PreferencesStore{db: db}
}

// Preference represents a user preference.
type Preference struct {
	ID        string
	UserID    string
	Key       string
	Value     any
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Get retrieves a preference by user and key.
func (s *PreferencesStore) Get(ctx context.Context, userID, key string) (*Preference, error) {
	query := `SELECT id, user_id, key, value, created_at, updated_at FROM _preferences WHERE user_id = ? AND key = ?`

	var pref Preference
	var valueJSON string

	err := s.db.QueryRowContext(ctx, query, userID, key).Scan(
		&pref.ID, &pref.UserID, &pref.Key, &valueJSON, &pref.CreatedAt, &pref.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get preference: %w", err)
	}

	if err := json.Unmarshal([]byte(valueJSON), &pref.Value); err != nil {
		pref.Value = valueJSON // Use as string if not JSON
	}

	return &pref, nil
}

// Set creates or updates a preference.
func (s *PreferencesStore) Set(ctx context.Context, userID, key string, value any) (*Preference, error) {
	now := time.Now()

	valueJSON, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal preference value: %w", err)
	}

	// Check if exists
	existing, err := s.Get(ctx, userID, key)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// Update
		query := `UPDATE _preferences SET value = ?, updated_at = ? WHERE user_id = ? AND key = ?`
		_, err = s.db.ExecContext(ctx, query, string(valueJSON), now, userID, key)
		if err != nil {
			return nil, fmt.Errorf("update preference: %w", err)
		}
		return &Preference{
			ID:        existing.ID,
			UserID:    userID,
			Key:       key,
			Value:     value,
			CreatedAt: existing.CreatedAt,
			UpdatedAt: now,
		}, nil
	}

	// Create
	id := ulid.New()
	query := `INSERT INTO _preferences (id, user_id, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = s.db.ExecContext(ctx, query, id, userID, key, string(valueJSON), now, now)
	if err != nil {
		return nil, fmt.Errorf("create preference: %w", err)
	}

	return &Preference{
		ID:        id,
		UserID:    userID,
		Key:       key,
		Value:     value,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Delete removes a preference.
func (s *PreferencesStore) Delete(ctx context.Context, userID, key string) error {
	query := `DELETE FROM _preferences WHERE user_id = ? AND key = ?`
	_, err := s.db.ExecContext(ctx, query, userID, key)
	if err != nil {
		return fmt.Errorf("delete preference: %w", err)
	}
	return nil
}

// ListByUser lists all preferences for a user.
func (s *PreferencesStore) ListByUser(ctx context.Context, userID string) ([]*Preference, error) {
	query := `SELECT id, user_id, key, value, created_at, updated_at FROM _preferences WHERE user_id = ? ORDER BY key`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list preferences: %w", err)
	}
	defer rows.Close()

	var prefs []*Preference
	for rows.Next() {
		var pref Preference
		var valueJSON string

		if err := rows.Scan(&pref.ID, &pref.UserID, &pref.Key, &valueJSON, &pref.CreatedAt, &pref.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan preference: %w", err)
		}

		if err := json.Unmarshal([]byte(valueJSON), &pref.Value); err != nil {
			pref.Value = valueJSON
		}

		prefs = append(prefs, &pref)
	}

	return prefs, nil
}
