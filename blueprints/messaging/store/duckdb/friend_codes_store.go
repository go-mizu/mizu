package duckdb

import (
	"context"
	"database/sql"

	"github.com/go-mizu/blueprints/messaging/feature/friendcode"
)

// FriendCodesStore implements friendcode.Store.
type FriendCodesStore struct {
	db *sql.DB
}

// NewFriendCodesStore creates a new FriendCodesStore.
func NewFriendCodesStore(db *sql.DB) *FriendCodesStore {
	return &FriendCodesStore{db: db}
}

// Insert creates a new friend code.
func (s *FriendCodesStore) Insert(ctx context.Context, fc *friendcode.FriendCode) error {
	query := `
		INSERT INTO friend_codes (id, user_id, code, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query, fc.ID, fc.UserID, fc.Code, fc.ExpiresAt, fc.CreatedAt)
	return err
}

// GetByUserID retrieves a friend code by user ID.
func (s *FriendCodesStore) GetByUserID(ctx context.Context, userID string) (*friendcode.FriendCode, error) {
	query := `
		SELECT id, user_id, code, expires_at, created_at
		FROM friend_codes WHERE user_id = ?
	`
	fc := &friendcode.FriendCode{}
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&fc.ID, &fc.UserID, &fc.Code, &fc.ExpiresAt, &fc.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, friendcode.ErrNotFound
	}
	return fc, err
}

// GetByCode retrieves a friend code by code string.
func (s *FriendCodesStore) GetByCode(ctx context.Context, code string) (*friendcode.FriendCode, error) {
	query := `
		SELECT id, user_id, code, expires_at, created_at
		FROM friend_codes WHERE code = ?
	`
	fc := &friendcode.FriendCode{}
	err := s.db.QueryRowContext(ctx, query, code).Scan(
		&fc.ID, &fc.UserID, &fc.Code, &fc.ExpiresAt, &fc.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, friendcode.ErrNotFound
	}
	return fc, err
}

// Delete removes a friend code by ID.
func (s *FriendCodesStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM friend_codes WHERE id = ?", id)
	return err
}

// DeleteByUserID removes all friend codes for a user.
func (s *FriendCodesStore) DeleteByUserID(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM friend_codes WHERE user_id = ?", userID)
	return err
}

// DeleteExpired removes all expired friend codes.
func (s *FriendCodesStore) DeleteExpired(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM friend_codes WHERE expires_at < CURRENT_TIMESTAMP")
	return err
}

// FriendCodeUserStore adapts UsersStore for friendcode.UserStore.
type FriendCodeUserStore struct {
	users *UsersStore
}

// NewFriendCodeUserStore creates a new adapter.
func NewFriendCodeUserStore(users *UsersStore) *FriendCodeUserStore {
	return &FriendCodeUserStore{users: users}
}

// GetByID retrieves user info for friend code resolution.
func (s *FriendCodeUserStore) GetByID(ctx context.Context, id string) (*friendcode.ResolvedUser, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &friendcode.ResolvedUser{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarURL:   u.AvatarURL,
		Status:      u.Status,
	}, nil
}

// FriendCodeContactStore adapts contacts operations for friendcode.
type FriendCodeContactStore struct {
	db *sql.DB
}

// NewFriendCodeContactStore creates a new adapter.
func NewFriendCodeContactStore(db *sql.DB) *FriendCodeContactStore {
	return &FriendCodeContactStore{db: db}
}

// Exists checks if a contact relationship exists.
func (s *FriendCodeContactStore) Exists(ctx context.Context, userID, contactUserID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM contacts WHERE user_id = ? AND contact_user_id = ?)",
		userID, contactUserID,
	).Scan(&exists)
	return exists, err
}

// Insert adds a new contact.
func (s *FriendCodeContactStore) Insert(ctx context.Context, userID, contactUserID, displayName string) error {
	query := `
		INSERT INTO contacts (user_id, contact_user_id, display_name, is_blocked, is_favorite, created_at)
		VALUES (?, ?, ?, FALSE, FALSE, CURRENT_TIMESTAMP)
	`
	_, err := s.db.ExecContext(ctx, query, userID, contactUserID, displayName)
	return err
}
