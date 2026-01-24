package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// PreferenceStore implements store.PreferenceStore using PostgreSQL.
type PreferenceStore struct {
	db *sql.DB
}

// SetPreference sets a domain preference (upvote, downvote, block).
func (s *PreferenceStore) SetPreference(ctx context.Context, pref *store.UserPreference) error {
	if pref.CreatedAt.IsZero() {
		pref.CreatedAt = time.Now()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO search.preferences (domain, action, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (domain) DO UPDATE SET
			action = EXCLUDED.action
	`, pref.Domain, pref.Action, pref.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to set preference: %w", err)
	}

	return nil
}

// GetPreferences retrieves all user preferences.
func (s *PreferenceStore) GetPreferences(ctx context.Context) ([]*store.UserPreference, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, domain, action, created_at
		FROM search.preferences
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get preferences: %w", err)
	}
	defer rows.Close()

	var prefs []*store.UserPreference
	for rows.Next() {
		var p store.UserPreference
		if err := rows.Scan(&p.ID, &p.Domain, &p.Action, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan preference: %w", err)
		}
		prefs = append(prefs, &p)
	}

	return prefs, nil
}

// DeletePreference removes a domain preference.
func (s *PreferenceStore) DeletePreference(ctx context.Context, domain string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM search.preferences WHERE domain = $1", domain)
	if err != nil {
		return fmt.Errorf("failed to delete preference: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("preference not found")
	}

	return nil
}

// CreateLens creates a new search lens.
func (s *PreferenceStore) CreateLens(ctx context.Context, lens *store.SearchLens) error {
	now := time.Now()
	if lens.CreatedAt.IsZero() {
		lens.CreatedAt = now
	}
	lens.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO search.lenses (name, description, domains, exclude, keywords, is_public, is_built_in, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, lens.Name, lens.Description, pq.Array(lens.Domains), pq.Array(lens.Exclude),
		pq.Array(lens.Keywords), lens.IsPublic, lens.IsBuiltIn, lens.CreatedAt, lens.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create lens: %w", err)
	}

	return nil
}

// GetLens retrieves a lens by ID.
func (s *PreferenceStore) GetLens(ctx context.Context, id string) (*store.SearchLens, error) {
	var lens store.SearchLens
	var description sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, domains, exclude, keywords, is_public, is_built_in, created_at, updated_at
		FROM search.lenses WHERE id = $1
	`, id).Scan(&lens.ID, &lens.Name, &description, pq.Array(&lens.Domains), pq.Array(&lens.Exclude),
		pq.Array(&lens.Keywords), &lens.IsPublic, &lens.IsBuiltIn, &lens.CreatedAt, &lens.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lens not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get lens: %w", err)
	}

	if description.Valid {
		lens.Description = description.String
	}

	return &lens, nil
}

// ListLenses retrieves all lenses.
func (s *PreferenceStore) ListLenses(ctx context.Context) ([]*store.SearchLens, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, domains, exclude, keywords, is_public, is_built_in, created_at, updated_at
		FROM search.lenses
		ORDER BY is_built_in DESC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list lenses: %w", err)
	}
	defer rows.Close()

	var lenses []*store.SearchLens
	for rows.Next() {
		var lens store.SearchLens
		var description sql.NullString
		if err := rows.Scan(&lens.ID, &lens.Name, &description, pq.Array(&lens.Domains), pq.Array(&lens.Exclude),
			pq.Array(&lens.Keywords), &lens.IsPublic, &lens.IsBuiltIn, &lens.CreatedAt, &lens.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan lens: %w", err)
		}
		if description.Valid {
			lens.Description = description.String
		}
		lenses = append(lenses, &lens)
	}

	return lenses, nil
}

// UpdateLens updates a lens.
func (s *PreferenceStore) UpdateLens(ctx context.Context, lens *store.SearchLens) error {
	lens.UpdatedAt = time.Now()

	result, err := s.db.ExecContext(ctx, `
		UPDATE search.lenses SET
			name = $2,
			description = $3,
			domains = $4,
			exclude = $5,
			keywords = $6,
			is_public = $7,
			updated_at = $8
		WHERE id = $1
	`, lens.ID, lens.Name, lens.Description, pq.Array(lens.Domains), pq.Array(lens.Exclude),
		pq.Array(lens.Keywords), lens.IsPublic, lens.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update lens: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("lens not found")
	}

	return nil
}

// DeleteLens deletes a lens.
func (s *PreferenceStore) DeleteLens(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM search.lenses WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete lens: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("lens not found")
	}

	return nil
}

// GetSettings retrieves user settings.
func (s *PreferenceStore) GetSettings(ctx context.Context) (*store.SearchSettings, error) {
	var settings store.SearchSettings

	err := s.db.QueryRowContext(ctx, `
		SELECT safe_search, results_per_page, region, language, theme, open_in_new_tab, show_thumbnails
		FROM search.settings WHERE id = 1
	`).Scan(&settings.SafeSearch, &settings.ResultsPerPage, &settings.Region, &settings.Language,
		&settings.Theme, &settings.OpenInNewTab, &settings.ShowThumbnails)
	if err == sql.ErrNoRows {
		// Return defaults
		return &store.SearchSettings{
			SafeSearch:     "moderate",
			ResultsPerPage: 10,
			Region:         "us",
			Language:       "en",
			Theme:          "system",
			OpenInNewTab:   false,
			ShowThumbnails: true,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	return &settings, nil
}

// UpdateSettings updates user settings.
func (s *PreferenceStore) UpdateSettings(ctx context.Context, settings *store.SearchSettings) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO search.settings (id, safe_search, results_per_page, region, language, theme, open_in_new_tab, show_thumbnails, updated_at)
		VALUES (1, $1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (id) DO UPDATE SET
			safe_search = EXCLUDED.safe_search,
			results_per_page = EXCLUDED.results_per_page,
			region = EXCLUDED.region,
			language = EXCLUDED.language,
			theme = EXCLUDED.theme,
			open_in_new_tab = EXCLUDED.open_in_new_tab,
			show_thumbnails = EXCLUDED.show_thumbnails,
			updated_at = EXCLUDED.updated_at
	`, settings.SafeSearch, settings.ResultsPerPage, settings.Region, settings.Language,
		settings.Theme, settings.OpenInNewTab, settings.ShowThumbnails)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	return nil
}
