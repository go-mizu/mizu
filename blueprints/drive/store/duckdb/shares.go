package duckdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/drive/feature/shares"
)

// SharesStore handles share persistence.
type SharesStore struct {
	db *sql.DB
}

// Create inserts a new share.
func (s *SharesStore) Create(ctx context.Context, sh *shares.Share) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO shares (id, item_id, item_type, owner_id, shared_with, permission, notify, message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sh.ID, sh.ItemID, sh.ItemType, sh.OwnerID, sh.SharedWith, sh.Permission, sh.Notify, sh.Message, sh.CreatedAt, sh.UpdatedAt)
	return err
}

// GetByID retrieves a share by ID.
func (s *SharesStore) GetByID(ctx context.Context, id string) (*shares.Share, error) {
	sh := &shares.Share{}
	var message sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, item_id, item_type, owner_id, shared_with, permission, notify, message, created_at, updated_at
		FROM shares WHERE id = ?`, id).Scan(
		&sh.ID, &sh.ItemID, &sh.ItemType, &sh.OwnerID, &sh.SharedWith, &sh.Permission, &sh.Notify, &message, &sh.CreatedAt, &sh.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, shares.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	sh.Message = message.String
	return sh, nil
}

// GetByItemAndUser retrieves a share by item and user.
func (s *SharesStore) GetByItemAndUser(ctx context.Context, itemID, itemType, sharedWith string) (*shares.Share, error) {
	sh := &shares.Share{}
	var message sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, item_id, item_type, owner_id, shared_with, permission, notify, message, created_at, updated_at
		FROM shares WHERE item_id = ? AND item_type = ? AND shared_with = ?`, itemID, itemType, sharedWith).Scan(
		&sh.ID, &sh.ItemID, &sh.ItemType, &sh.OwnerID, &sh.SharedWith, &sh.Permission, &sh.Notify, &message, &sh.CreatedAt, &sh.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, shares.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	sh.Message = message.String
	return sh, nil
}

// ListByOwner lists shares by owner.
func (s *SharesStore) ListByOwner(ctx context.Context, ownerID string) ([]*shares.Share, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, item_id, item_type, owner_id, shared_with, permission, notify, message, created_at, updated_at
		FROM shares WHERE owner_id = ? ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanShares(rows)
}

// ListBySharedWith lists shares shared with a user.
func (s *SharesStore) ListBySharedWith(ctx context.Context, accountID string) ([]*shares.Share, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, item_id, item_type, owner_id, shared_with, permission, notify, message, created_at, updated_at
		FROM shares WHERE shared_with = ? ORDER BY created_at DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanShares(rows)
}

// ListByItem lists shares for an item.
func (s *SharesStore) ListByItem(ctx context.Context, itemID, itemType string) ([]*shares.Share, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, item_id, item_type, owner_id, shared_with, permission, notify, message, created_at, updated_at
		FROM shares WHERE item_id = ? AND item_type = ? ORDER BY created_at DESC`, itemID, itemType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanShares(rows)
}

// Update updates a share.
func (s *SharesStore) Update(ctx context.Context, id string, permission string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE shares SET permission = ?, updated_at = ? WHERE id = ?`, permission, time.Now(), id)
	return err
}

// Delete deletes a share.
func (s *SharesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM shares WHERE id = ?`, id)
	return err
}

func scanShares(rows *sql.Rows) ([]*shares.Share, error) {
	var result []*shares.Share
	for rows.Next() {
		sh := &shares.Share{}
		var message sql.NullString

		if err := rows.Scan(&sh.ID, &sh.ItemID, &sh.ItemType, &sh.OwnerID, &sh.SharedWith, &sh.Permission, &sh.Notify, &message, &sh.CreatedAt, &sh.UpdatedAt); err != nil {
			return nil, err
		}

		sh.Message = message.String
		result = append(result, sh)
	}
	return result, rows.Err()
}

// ShareLinksStore handles share link persistence.
type ShareLinksStore struct {
	db *sql.DB
}

// Create inserts a new share link.
func (s *ShareLinksStore) Create(ctx context.Context, l *shares.ShareLink) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO share_links (id, item_id, item_type, owner_id, token, permission, password_hash,
			expires_at, download_limit, download_count, allow_download, disabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.ItemID, l.ItemType, l.OwnerID, l.Token, l.Permission, nullString(l.PasswordHash),
		l.ExpiresAt, l.DownloadLimit, l.DownloadCount, l.AllowDownload, l.Disabled, l.CreatedAt)
	return err
}

// GetByID retrieves a link by ID.
func (s *ShareLinksStore) GetByID(ctx context.Context, id string) (*shares.ShareLink, error) {
	return s.scanLink(s.db.QueryRowContext(ctx, `
		SELECT id, item_id, item_type, owner_id, token, permission, password_hash,
			expires_at, download_limit, download_count, allow_download, disabled, created_at, accessed_at
		FROM share_links WHERE id = ?`, id))
}

// GetByToken retrieves a link by token.
func (s *ShareLinksStore) GetByToken(ctx context.Context, token string) (*shares.ShareLink, error) {
	return s.scanLink(s.db.QueryRowContext(ctx, `
		SELECT id, item_id, item_type, owner_id, token, permission, password_hash,
			expires_at, download_limit, download_count, allow_download, disabled, created_at, accessed_at
		FROM share_links WHERE token = ?`, token))
}

// ListByItem lists links for an item.
func (s *ShareLinksStore) ListByItem(ctx context.Context, itemID, itemType string) ([]*shares.ShareLink, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, item_id, item_type, owner_id, token, permission, password_hash,
			expires_at, download_limit, download_count, allow_download, disabled, created_at, accessed_at
		FROM share_links WHERE item_id = ? AND item_type = ? ORDER BY created_at DESC`, itemID, itemType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*shares.ShareLink
	for rows.Next() {
		l := &shares.ShareLink{}
		var passwordHash sql.NullString
		var expiresAt, accessedAt sql.NullTime
		var downloadLimit sql.NullInt64

		if err := rows.Scan(&l.ID, &l.ItemID, &l.ItemType, &l.OwnerID, &l.Token, &l.Permission, &passwordHash,
			&expiresAt, &downloadLimit, &l.DownloadCount, &l.AllowDownload, &l.Disabled, &l.CreatedAt, &accessedAt); err != nil {
			return nil, err
		}

		l.PasswordHash = passwordHash.String
		l.HasPassword = passwordHash.Valid && passwordHash.String != ""
		if expiresAt.Valid {
			l.ExpiresAt = &expiresAt.Time
		}
		if downloadLimit.Valid {
			limit := int(downloadLimit.Int64)
			l.DownloadLimit = &limit
		}
		if accessedAt.Valid {
			l.AccessedAt = &accessedAt.Time
		}

		result = append(result, l)
	}
	return result, rows.Err()
}

// Update updates a link.
func (s *ShareLinksStore) Update(ctx context.Context, id string, in *shares.UpdateLinkIn, passwordHash string) error {
	if in.Permission != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE share_links SET permission = ? WHERE id = ?`, *in.Permission, id); err != nil {
			return err
		}
	}
	if passwordHash != "" {
		if _, err := s.db.ExecContext(ctx, `UPDATE share_links SET password_hash = ? WHERE id = ?`, passwordHash, id); err != nil {
			return err
		}
	}
	if in.ExpiresAt != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE share_links SET expires_at = ? WHERE id = ?`, in.ExpiresAt, id); err != nil {
			return err
		}
	}
	if in.DownloadLimit != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE share_links SET download_limit = ? WHERE id = ?`, *in.DownloadLimit, id); err != nil {
			return err
		}
	}
	if in.AllowDownload != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE share_links SET allow_download = ? WHERE id = ?`, *in.AllowDownload, id); err != nil {
			return err
		}
	}
	if in.Disabled != nil {
		if _, err := s.db.ExecContext(ctx, `UPDATE share_links SET disabled = ? WHERE id = ?`, *in.Disabled, id); err != nil {
			return err
		}
	}
	return nil
}

// UpdateAccess updates access time.
func (s *ShareLinksStore) UpdateAccess(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE share_links SET accessed_at = ? WHERE id = ?`, time.Now(), id)
	return err
}

// IncrementDownloads increments download count.
func (s *ShareLinksStore) IncrementDownloads(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE share_links SET download_count = download_count + 1 WHERE id = ?`, id)
	return err
}

// Delete deletes a link.
func (s *ShareLinksStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM share_links WHERE id = ?`, id)
	return err
}

func (s *ShareLinksStore) scanLink(row *sql.Row) (*shares.ShareLink, error) {
	l := &shares.ShareLink{}
	var passwordHash sql.NullString
	var expiresAt, accessedAt sql.NullTime
	var downloadLimit sql.NullInt64

	err := row.Scan(&l.ID, &l.ItemID, &l.ItemType, &l.OwnerID, &l.Token, &l.Permission, &passwordHash,
		&expiresAt, &downloadLimit, &l.DownloadCount, &l.AllowDownload, &l.Disabled, &l.CreatedAt, &accessedAt)

	if err == sql.ErrNoRows {
		return nil, shares.ErrLinkNotFound
	}
	if err != nil {
		return nil, err
	}

	l.PasswordHash = passwordHash.String
	l.HasPassword = passwordHash.Valid && passwordHash.String != ""
	if expiresAt.Valid {
		l.ExpiresAt = &expiresAt.Time
	}
	if downloadLimit.Valid {
		limit := int(downloadLimit.Int64)
		l.DownloadLimit = &limit
	}
	if accessedAt.Valid {
		l.AccessedAt = &accessedAt.Time
	}

	return l, nil
}
