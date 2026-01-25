package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
)

// PreferenceStore handles user preferences and settings.
type PreferenceStore struct {
	db *sql.DB
}

// SetPreference sets a domain preference.
func (s *PreferenceStore) SetPreference(ctx context.Context, pref *store.UserPreference) error {
	if pref.ID == "" {
		pref.ID = generateID()
	}
	pref.CreatedAt = time.Now()

	// Convert level to action for backwards compatibility
	if pref.Action == "" {
		switch pref.Level {
		case -2:
			pref.Action = "block"
		case -1:
			pref.Action = "downvote"
		case 1, 2:
			pref.Action = "upvote"
		default:
			pref.Action = "upvote" // default for level 0
		}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO preferences (id, domain, action, level, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(domain) DO UPDATE SET
			action = excluded.action,
			level = excluded.level,
			created_at = excluded.created_at
	`, pref.ID, pref.Domain, pref.Action, pref.Level, pref.CreatedAt)

	return err
}

// GetPreferences retrieves all domain preferences.
func (s *PreferenceStore) GetPreferences(ctx context.Context) ([]*store.UserPreference, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, domain, action, COALESCE(level, 0), created_at
		FROM preferences
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []*store.UserPreference
	for rows.Next() {
		var p store.UserPreference
		if err := rows.Scan(&p.ID, &p.Domain, &p.Action, &p.Level, &p.CreatedAt); err != nil {
			return nil, err
		}
		prefs = append(prefs, &p)
	}

	return prefs, nil
}

// GetPreference retrieves a specific domain preference.
func (s *PreferenceStore) GetPreference(ctx context.Context, domain string) (*store.UserPreference, error) {
	var p store.UserPreference

	err := s.db.QueryRowContext(ctx, `
		SELECT id, domain, action, COALESCE(level, 0), created_at
		FROM preferences
		WHERE domain = ?
	`, domain).Scan(&p.ID, &p.Domain, &p.Action, &p.Level, &p.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// DeletePreference removes a domain preference.
func (s *PreferenceStore) DeletePreference(ctx context.Context, domain string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM preferences WHERE domain = ?", domain)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("preference not found")
	}

	return nil
}

// CreateLens creates a new search lens.
func (s *PreferenceStore) CreateLens(ctx context.Context, lens *store.SearchLens) error {
	if lens.ID == "" {
		lens.ID = generateID()
	}
	lens.CreatedAt = time.Now()
	lens.UpdatedAt = time.Now()

	domainsJSON, _ := json.Marshal(lens.Domains)
	excludeJSON, _ := json.Marshal(lens.Exclude)
	keywordsJSON, _ := json.Marshal(lens.Keywords)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO lenses (id, name, description, domains, exclude, keywords, is_public, is_built_in, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, lens.ID, lens.Name, lens.Description, string(domainsJSON), string(excludeJSON),
		string(keywordsJSON), boolToInt(lens.IsPublic), boolToInt(lens.IsBuiltIn), lens.CreatedAt, lens.UpdatedAt)

	return err
}

// GetLens retrieves a lens by ID.
func (s *PreferenceStore) GetLens(ctx context.Context, id string) (*store.SearchLens, error) {
	var lens store.SearchLens
	var desc, domainsStr, excludeStr, keywordsStr sql.NullString
	var isPublic, isBuiltIn int

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, domains, exclude, keywords, is_public, is_built_in, created_at, updated_at
		FROM lenses WHERE id = ?
	`, id).Scan(&lens.ID, &lens.Name, &desc, &domainsStr, &excludeStr, &keywordsStr,
		&isPublic, &isBuiltIn, &lens.CreatedAt, &lens.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lens not found")
	}
	if err != nil {
		return nil, err
	}

	if desc.Valid {
		lens.Description = desc.String
	}
	if domainsStr.Valid {
		json.Unmarshal([]byte(domainsStr.String), &lens.Domains)
	}
	if excludeStr.Valid {
		json.Unmarshal([]byte(excludeStr.String), &lens.Exclude)
	}
	if keywordsStr.Valid {
		json.Unmarshal([]byte(keywordsStr.String), &lens.Keywords)
	}
	lens.IsPublic = isPublic == 1
	lens.IsBuiltIn = isBuiltIn == 1

	return &lens, nil
}

// ListLenses lists all lenses.
func (s *PreferenceStore) ListLenses(ctx context.Context) ([]*store.SearchLens, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, domains, exclude, keywords, is_public, is_built_in, created_at, updated_at
		FROM lenses
		ORDER BY is_built_in DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lenses []*store.SearchLens
	for rows.Next() {
		var lens store.SearchLens
		var desc, domainsStr, excludeStr, keywordsStr sql.NullString
		var isPublic, isBuiltIn int

		if err := rows.Scan(&lens.ID, &lens.Name, &desc, &domainsStr, &excludeStr, &keywordsStr,
			&isPublic, &isBuiltIn, &lens.CreatedAt, &lens.UpdatedAt); err != nil {
			return nil, err
		}

		if desc.Valid {
			lens.Description = desc.String
		}
		if domainsStr.Valid {
			json.Unmarshal([]byte(domainsStr.String), &lens.Domains)
		}
		if excludeStr.Valid {
			json.Unmarshal([]byte(excludeStr.String), &lens.Exclude)
		}
		if keywordsStr.Valid {
			json.Unmarshal([]byte(keywordsStr.String), &lens.Keywords)
		}
		lens.IsPublic = isPublic == 1
		lens.IsBuiltIn = isBuiltIn == 1

		lenses = append(lenses, &lens)
	}

	return lenses, nil
}

// UpdateLens updates an existing lens.
func (s *PreferenceStore) UpdateLens(ctx context.Context, lens *store.SearchLens) error {
	lens.UpdatedAt = time.Now()

	domainsJSON, _ := json.Marshal(lens.Domains)
	excludeJSON, _ := json.Marshal(lens.Exclude)
	keywordsJSON, _ := json.Marshal(lens.Keywords)

	result, err := s.db.ExecContext(ctx, `
		UPDATE lenses SET
			name = ?,
			description = ?,
			domains = ?,
			exclude = ?,
			keywords = ?,
			is_public = ?,
			updated_at = ?
		WHERE id = ?
	`, lens.Name, lens.Description, string(domainsJSON), string(excludeJSON),
		string(keywordsJSON), boolToInt(lens.IsPublic), lens.UpdatedAt, lens.ID)

	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("lens not found")
	}

	return nil
}

// DeleteLens removes a lens.
func (s *PreferenceStore) DeleteLens(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, "DELETE FROM lenses WHERE id = ? AND is_built_in = 0", id)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("lens not found or is built-in")
	}

	return nil
}

// GetSettings retrieves user settings.
func (s *PreferenceStore) GetSettings(ctx context.Context) (*store.SearchSettings, error) {
	var settings store.SearchSettings
	var openInNewTab, showThumbnails int

	err := s.db.QueryRowContext(ctx, `
		SELECT safe_search, results_per_page, region, language, theme, open_in_new_tab, show_thumbnails
		FROM settings WHERE id = 1
	`).Scan(&settings.SafeSearch, &settings.ResultsPerPage, &settings.Region,
		&settings.Language, &settings.Theme, &openInNewTab, &showThumbnails)

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
		return nil, err
	}

	settings.OpenInNewTab = openInNewTab == 1
	settings.ShowThumbnails = showThumbnails == 1

	return &settings, nil
}

// UpdateSettings updates user settings.
func (s *PreferenceStore) UpdateSettings(ctx context.Context, settings *store.SearchSettings) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (id, safe_search, results_per_page, region, language, theme, open_in_new_tab, show_thumbnails)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			safe_search = excluded.safe_search,
			results_per_page = excluded.results_per_page,
			region = excluded.region,
			language = excluded.language,
			theme = excluded.theme,
			open_in_new_tab = excluded.open_in_new_tab,
			show_thumbnails = excluded.show_thumbnails
	`, settings.SafeSearch, settings.ResultsPerPage, settings.Region,
		settings.Language, settings.Theme, boolToInt(settings.OpenInNewTab), boolToInt(settings.ShowThumbnails))

	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
