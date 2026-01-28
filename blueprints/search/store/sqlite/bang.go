package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/mizu/blueprints/search/store"
	"github.com/go-mizu/mizu/blueprints/search/types"
)

// BangStore handles bang shortcuts storage.
type BangStore struct {
	db *sql.DB
}

// CreateBang creates a new bang shortcut.
func (s *BangStore) CreateBang(ctx context.Context, bang *store.Bang) error {
	bang.CreatedAt = time.Now()

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO bangs (trigger, name, url_template, category, is_builtin, user_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(trigger) DO UPDATE SET
			name = excluded.name,
			url_template = excluded.url_template,
			category = excluded.category
	`, bang.Trigger, bang.Name, bang.URLTemplate, bang.Category, boolToInt(bang.IsBuiltin), bang.UserID, bang.CreatedAt)

	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	bang.ID = id
	return nil
}

// GetBang retrieves a bang by trigger.
func (s *BangStore) GetBang(ctx context.Context, trigger string) (*store.Bang, error) {
	var bang store.Bang
	var isBuiltin int
	var userID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, trigger, name, url_template, category, is_builtin, user_id, created_at
		FROM bangs WHERE trigger = ?
	`, trigger).Scan(&bang.ID, &bang.Trigger, &bang.Name, &bang.URLTemplate,
		&bang.Category, &isBuiltin, &userID, &bang.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	bang.IsBuiltin = isBuiltin == 1
	if userID.Valid {
		bang.UserID = userID.String
	}

	return &bang, nil
}

// ListBangs returns all bangs.
func (s *BangStore) ListBangs(ctx context.Context) ([]*store.Bang, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, trigger, name, url_template, category, is_builtin, user_id, created_at
		FROM bangs
		ORDER BY is_builtin DESC, trigger ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bangs []*store.Bang
	for rows.Next() {
		var bang store.Bang
		var isBuiltin int
		var userID sql.NullString

		if err := rows.Scan(&bang.ID, &bang.Trigger, &bang.Name, &bang.URLTemplate,
			&bang.Category, &isBuiltin, &userID, &bang.CreatedAt); err != nil {
			return nil, err
		}

		bang.IsBuiltin = isBuiltin == 1
		if userID.Valid {
			bang.UserID = userID.String
		}
		bangs = append(bangs, &bang)
	}

	return bangs, nil
}

// ListUserBangs returns bangs for a specific user.
func (s *BangStore) ListUserBangs(ctx context.Context, userID string) ([]*store.Bang, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, trigger, name, url_template, category, is_builtin, user_id, created_at
		FROM bangs
		WHERE user_id = ? OR is_builtin = 1
		ORDER BY is_builtin DESC, trigger ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bangs []*store.Bang
	for rows.Next() {
		var bang store.Bang
		var isBuiltin int
		var uid sql.NullString

		if err := rows.Scan(&bang.ID, &bang.Trigger, &bang.Name, &bang.URLTemplate,
			&bang.Category, &isBuiltin, &uid, &bang.CreatedAt); err != nil {
			return nil, err
		}

		bang.IsBuiltin = isBuiltin == 1
		if uid.Valid {
			bang.UserID = uid.String
		}
		bangs = append(bangs, &bang)
	}

	return bangs, nil
}

// DeleteBang removes a bang by ID.
func (s *BangStore) DeleteBang(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM bangs WHERE id = ? AND is_builtin = 0", id)
	return err
}

// SeedBuiltinBangs inserts all built-in bangs.
func (s *BangStore) SeedBuiltinBangs(ctx context.Context) error {
	for _, bang := range types.ExternalBangs {
		b := &store.Bang{
			Trigger:     bang.Trigger,
			Name:        bang.Name,
			URLTemplate: bang.URLTemplate,
			Category:    bang.Category,
			IsBuiltin:   true,
		}
		if err := s.CreateBang(ctx, b); err != nil {
			return err
		}
	}
	return nil
}
