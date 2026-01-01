package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// Settings represents user settings.
type Settings struct {
	UserID               string
	Theme                string
	Language             string
	Timezone             string
	ListView             string
	SortBy               string
	SortOrder            string
	NotificationsEnabled bool
	EmailNotifications   bool
	UpdatedAt            time.Time
}

// GetSettings retrieves settings for a user.
func (s *Store) GetSettings(ctx context.Context, userID string) (*Settings, error) {
	settings := &Settings{}
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, theme, language, timezone, list_view, sort_by, sort_order, notifications_enabled, email_notifications, updated_at
		FROM settings WHERE user_id = ?
	`, userID).Scan(&settings.UserID, &settings.Theme, &settings.Language, &settings.Timezone, &settings.ListView, &settings.SortBy, &settings.SortOrder, &settings.NotificationsEnabled, &settings.EmailNotifications, &settings.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return settings, err
}

// CreateSettings inserts new settings for a user.
func (s *Store) CreateSettings(ctx context.Context, settings *Settings) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (user_id, theme, language, timezone, list_view, sort_by, sort_order, notifications_enabled, email_notifications, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, settings.UserID, settings.Theme, settings.Language, settings.Timezone, settings.ListView, settings.SortBy, settings.SortOrder, settings.NotificationsEnabled, settings.EmailNotifications, settings.UpdatedAt)
	return err
}

// UpdateSettings updates settings for a user.
func (s *Store) UpdateSettings(ctx context.Context, settings *Settings) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE settings SET theme = ?, language = ?, timezone = ?, list_view = ?, sort_by = ?, sort_order = ?, notifications_enabled = ?, email_notifications = ?, updated_at = ?
		WHERE user_id = ?
	`, settings.Theme, settings.Language, settings.Timezone, settings.ListView, settings.SortBy, settings.SortOrder, settings.NotificationsEnabled, settings.EmailNotifications, settings.UpdatedAt, settings.UserID)
	return err
}

// UpsertSettings creates or updates settings for a user.
func (s *Store) UpsertSettings(ctx context.Context, settings *Settings) error {
	existing, err := s.GetSettings(ctx, settings.UserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return s.CreateSettings(ctx, settings)
	}
	return s.UpdateSettings(ctx, settings)
}

// DeleteSettings deletes settings for a user.
func (s *Store) DeleteSettings(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM settings WHERE user_id = ?`, userID)
	return err
}

// GetOrCreateDefaultSettings gets settings or creates defaults.
func (s *Store) GetOrCreateDefaultSettings(ctx context.Context, userID string) (*Settings, error) {
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	if settings != nil {
		return settings, nil
	}

	// Create default settings
	settings = &Settings{
		UserID:               userID,
		Theme:                "system",
		Language:             "en",
		Timezone:             "UTC",
		ListView:             "list",
		SortBy:               "name",
		SortOrder:            "asc",
		NotificationsEnabled: true,
		EmailNotifications:   true,
		UpdatedAt:            time.Now(),
	}
	if err := s.CreateSettings(ctx, settings); err != nil {
		return nil, err
	}
	return settings, nil
}
