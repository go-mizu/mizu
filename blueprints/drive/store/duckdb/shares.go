package duckdb

import (
	"context"
	"database/sql"
	"time"
)

// Share represents a share record.
type Share struct {
	ID               string
	ResourceType     string
	ResourceID       string
	OwnerID          string
	SharedWithID     sql.NullString
	Permission       string
	LinkToken        sql.NullString
	LinkPasswordHash sql.NullString
	ExpiresAt        sql.NullTime
	DownloadLimit    sql.NullInt64
	DownloadCount    int
	PreventDownload  bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CreateShare inserts a new share.
func (s *Store) CreateShare(ctx context.Context, sh *Share) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO shares (id, resource_type, resource_id, owner_id, shared_with_id, permission, link_token, link_password_hash, expires_at, download_limit, download_count, prevent_download, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sh.ID, sh.ResourceType, sh.ResourceID, sh.OwnerID, sh.SharedWithID, sh.Permission, sh.LinkToken, sh.LinkPasswordHash, sh.ExpiresAt, sh.DownloadLimit, sh.DownloadCount, sh.PreventDownload, sh.CreatedAt, sh.UpdatedAt)
	return err
}

// GetShareByID retrieves a share by ID.
func (s *Store) GetShareByID(ctx context.Context, id string) (*Share, error) {
	sh := &Share{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, resource_type, resource_id, owner_id, shared_with_id, permission, link_token, link_password_hash, expires_at, download_limit, download_count, prevent_download, created_at, updated_at
		FROM shares WHERE id = ?
	`, id).Scan(&sh.ID, &sh.ResourceType, &sh.ResourceID, &sh.OwnerID, &sh.SharedWithID, &sh.Permission, &sh.LinkToken, &sh.LinkPasswordHash, &sh.ExpiresAt, &sh.DownloadLimit, &sh.DownloadCount, &sh.PreventDownload, &sh.CreatedAt, &sh.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sh, err
}

// GetShareByToken retrieves a share by link token.
func (s *Store) GetShareByToken(ctx context.Context, token string) (*Share, error) {
	sh := &Share{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, resource_type, resource_id, owner_id, shared_with_id, permission, link_token, link_password_hash, expires_at, download_limit, download_count, prevent_download, created_at, updated_at
		FROM shares WHERE link_token = ?
	`, token).Scan(&sh.ID, &sh.ResourceType, &sh.ResourceID, &sh.OwnerID, &sh.SharedWithID, &sh.Permission, &sh.LinkToken, &sh.LinkPasswordHash, &sh.ExpiresAt, &sh.DownloadLimit, &sh.DownloadCount, &sh.PreventDownload, &sh.CreatedAt, &sh.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sh, err
}

// UpdateShare updates a share.
func (s *Store) UpdateShare(ctx context.Context, sh *Share) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE shares SET permission = ?, link_password_hash = ?, expires_at = ?, download_limit = ?, download_count = ?, prevent_download = ?, updated_at = ?
		WHERE id = ?
	`, sh.Permission, sh.LinkPasswordHash, sh.ExpiresAt, sh.DownloadLimit, sh.DownloadCount, sh.PreventDownload, sh.UpdatedAt, sh.ID)
	return err
}

// DeleteShare deletes a share by ID.
func (s *Store) DeleteShare(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM shares WHERE id = ?`, id)
	return err
}

// ListSharesByOwner lists all shares created by a user.
func (s *Store) ListSharesByOwner(ctx context.Context, ownerID string) ([]*Share, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, resource_type, resource_id, owner_id, shared_with_id, permission, link_token, link_password_hash, expires_at, download_limit, download_count, prevent_download, created_at, updated_at
		FROM shares WHERE owner_id = ? ORDER BY created_at DESC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanShares(rows)
}

// ListSharesWithUser lists all shares shared with a user.
func (s *Store) ListSharesWithUser(ctx context.Context, userID string) ([]*Share, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, resource_type, resource_id, owner_id, shared_with_id, permission, link_token, link_password_hash, expires_at, download_limit, download_count, prevent_download, created_at, updated_at
		FROM shares WHERE shared_with_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanShares(rows)
}

// ListSharesForResource lists all shares for a resource.
func (s *Store) ListSharesForResource(ctx context.Context, resourceType, resourceID string) ([]*Share, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, resource_type, resource_id, owner_id, shared_with_id, permission, link_token, link_password_hash, expires_at, download_limit, download_count, prevent_download, created_at, updated_at
		FROM shares WHERE resource_type = ? AND resource_id = ? ORDER BY created_at DESC
	`, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanShares(rows)
}

// GetShareForUserAndResource checks if a resource is shared with a user.
func (s *Store) GetShareForUserAndResource(ctx context.Context, userID, resourceType, resourceID string) (*Share, error) {
	sh := &Share{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, resource_type, resource_id, owner_id, shared_with_id, permission, link_token, link_password_hash, expires_at, download_limit, download_count, prevent_download, created_at, updated_at
		FROM shares WHERE shared_with_id = ? AND resource_type = ? AND resource_id = ?
	`, userID, resourceType, resourceID).Scan(&sh.ID, &sh.ResourceType, &sh.ResourceID, &sh.OwnerID, &sh.SharedWithID, &sh.Permission, &sh.LinkToken, &sh.LinkPasswordHash, &sh.ExpiresAt, &sh.DownloadLimit, &sh.DownloadCount, &sh.PreventDownload, &sh.CreatedAt, &sh.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sh, err
}

// IncrementDownloadCount increments the download count for a share.
func (s *Store) IncrementDownloadCount(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE shares SET download_count = download_count + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// DeleteSharesForResource deletes all shares for a resource.
func (s *Store) DeleteSharesForResource(ctx context.Context, resourceType, resourceID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM shares WHERE resource_type = ? AND resource_id = ?`, resourceType, resourceID)
	return err
}

// CleanupExpiredShares removes expired shares.
func (s *Store) CleanupExpiredShares(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM shares WHERE expires_at IS NOT NULL AND expires_at <= CURRENT_TIMESTAMP`)
	return err
}

func scanShares(rows *sql.Rows) ([]*Share, error) {
	var shares []*Share
	for rows.Next() {
		sh := &Share{}
		if err := rows.Scan(&sh.ID, &sh.ResourceType, &sh.ResourceID, &sh.OwnerID, &sh.SharedWithID, &sh.Permission, &sh.LinkToken, &sh.LinkPasswordHash, &sh.ExpiresAt, &sh.DownloadLimit, &sh.DownloadCount, &sh.PreventDownload, &sh.CreatedAt, &sh.UpdatedAt); err != nil {
			return nil, err
		}
		shares = append(shares, sh)
	}
	return shares, rows.Err()
}
