package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/table/feature/shares"
)

// SharesStore provides SQLite-based share storage.
type SharesStore struct {
	db *sql.DB
}

// NewSharesStore creates a new shares store.
func NewSharesStore(db *sql.DB) *SharesStore {
	return &SharesStore{db: db}
}

// Create creates a new share.
func (s *SharesStore) Create(ctx context.Context, share *shares.Share) error {
	share.CreatedAt = time.Now()

	var token *string
	if share.Token != "" {
		token = &share.Token
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO shares (id, base_id, table_id, view_id, type, permission, user_id, email, token, expires_at, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, share.ID, share.BaseID, share.TableID, share.ViewID, share.Type, share.Permission, share.UserID, share.Email, token, share.ExpiresAt, share.CreatedBy, share.CreatedAt)
	return err
}

// GetByID retrieves a share by ID.
func (s *SharesStore) GetByID(ctx context.Context, id string) (*shares.Share, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, base_id, table_id, view_id, type, permission, user_id, email, token, expires_at, created_by, created_at
		FROM shares WHERE id = ?
	`, id)
	return s.scanShare(row)
}

// GetByToken retrieves a share by token.
func (s *SharesStore) GetByToken(ctx context.Context, token string) (*shares.Share, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, base_id, table_id, view_id, type, permission, user_id, email, token, expires_at, created_by, created_at
		FROM shares WHERE token = ?
	`, token)
	return s.scanShare(row)
}

// Update updates a share.
func (s *SharesStore) Update(ctx context.Context, share *shares.Share) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE shares SET
			permission = ?, expires_at = ?
		WHERE id = ?
	`, share.Permission, share.ExpiresAt, share.ID)
	return err
}

// Delete deletes a share.
func (s *SharesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM shares WHERE id = ?`, id)
	return err
}

// ListByBase lists all shares for a base.
func (s *SharesStore) ListByBase(ctx context.Context, baseID string) ([]*shares.Share, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, base_id, table_id, view_id, type, permission, user_id, email, token, expires_at, created_by, created_at
		FROM shares WHERE base_id = ?
		ORDER BY created_at DESC
	`, baseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanShares(rows)
}

// ListByUser lists all shares for a user.
func (s *SharesStore) ListByUser(ctx context.Context, userID string) ([]*shares.Share, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, base_id, table_id, view_id, type, permission, user_id, email, token, expires_at, created_by, created_at
		FROM shares WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanShares(rows)
}

func (s *SharesStore) scanShare(row *sql.Row) (*shares.Share, error) {
	share := &shares.Share{}
	var tableID, viewID, userID, email, token sql.NullString
	var expiresAt sql.NullTime

	err := row.Scan(&share.ID, &share.BaseID, &tableID, &viewID, &share.Type, &share.Permission, &userID, &email, &token, &expiresAt, &share.CreatedBy, &share.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, shares.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if tableID.Valid {
		share.TableID = tableID.String
	}
	if viewID.Valid {
		share.ViewID = viewID.String
	}
	if userID.Valid {
		share.UserID = userID.String
	}
	if email.Valid {
		share.Email = email.String
	}
	if token.Valid {
		share.Token = token.String
	}
	if expiresAt.Valid {
		share.ExpiresAt = &expiresAt.Time
	}

	return share, nil
}

func (s *SharesStore) scanShares(rows *sql.Rows) ([]*shares.Share, error) {
	var shareList []*shares.Share
	for rows.Next() {
		share := &shares.Share{}
		var tableID, viewID, userID, email, token sql.NullString
		var expiresAt sql.NullTime

		err := rows.Scan(&share.ID, &share.BaseID, &tableID, &viewID, &share.Type, &share.Permission, &userID, &email, &token, &expiresAt, &share.CreatedBy, &share.CreatedAt)
		if err != nil {
			return nil, err
		}

		if tableID.Valid {
			share.TableID = tableID.String
		}
		if viewID.Valid {
			share.ViewID = viewID.String
		}
		if userID.Valid {
			share.UserID = userID.String
		}
		if email.Valid {
			share.Email = email.String
		}
		if token.Valid {
			share.Token = token.String
		}
		if expiresAt.Valid {
			share.ExpiresAt = &expiresAt.Time
		}

		shareList = append(shareList, share)
	}
	return shareList, rows.Err()
}
