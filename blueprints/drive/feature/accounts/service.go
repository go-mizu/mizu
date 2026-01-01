package accounts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/go-mizu/blueprints/drive/pkg/password"
	"github.com/go-mizu/blueprints/drive/pkg/ulid"
	"github.com/go-mizu/blueprints/drive/store/duckdb"
)

// Service implements the accounts API.
type Service struct {
	store *duckdb.Store
}

// NewService creates a new accounts service.
func NewService(store *duckdb.Store) *Service {
	return &Service{store: store}
}

func (s *Service) Register(ctx context.Context, in *RegisterIn) (*User, *Session, error) {
	if in.Email == "" {
		return nil, nil, ErrMissingEmail
	}
	if in.Name == "" {
		return nil, nil, ErrMissingName
	}
	if in.Password == "" {
		return nil, nil, ErrMissingPassword
	}

	// Check if user already exists
	existing, err := s.store.GetUserByEmail(ctx, in.Email)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		return nil, nil, ErrUserExists
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now()
	dbUser := &duckdb.User{
		ID:            ulid.New(),
		Email:         in.Email,
		Name:          in.Name,
		PasswordHash:  hash,
		StorageQuota:  10 * 1024 * 1024 * 1024, // 10GB
		StorageUsed:   0,
		IsAdmin:       false,
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.store.CreateUser(ctx, dbUser); err != nil {
		return nil, nil, err
	}

	user := dbUserToUser(dbUser)

	// Create session
	session, err := s.createSession(ctx, user.ID, "", "")
	if err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

func (s *Service) Login(ctx context.Context, in *LoginIn) (*User, *Session, error) {
	if in.Email == "" {
		return nil, nil, ErrMissingEmail
	}
	if in.Password == "" {
		return nil, nil, ErrMissingPassword
	}

	dbUser, err := s.store.GetUserByEmail(ctx, in.Email)
	if err != nil {
		return nil, nil, err
	}
	if dbUser == nil {
		return nil, nil, ErrInvalidEmail
	}

	valid, err := password.Verify(in.Password, dbUser.PasswordHash)
	if err != nil {
		return nil, nil, err
	}
	if !valid {
		return nil, nil, ErrInvalidPassword
	}

	user := dbUserToUser(dbUser)

	session, err := s.createSession(ctx, user.ID, "", "")
	if err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.store.DeleteSession(ctx, sessionID)
}

func (s *Service) LogoutAll(ctx context.Context, userID string) error {
	return s.store.DeleteUserSessions(ctx, userID)
}

func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	dbUser, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbUser == nil {
		return nil, ErrNotFound
	}
	return dbUserToUser(dbUser), nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	dbUser, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if dbUser == nil {
		return nil, nil
	}
	return dbUserToUser(dbUser), nil
}

func (s *Service) GetBySession(ctx context.Context, token string) (*User, error) {
	tokenHash := hashToken(token)
	sess, err := s.store.GetSessionByToken(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, ErrSessionNotFound
	}

	if time.Now().After(sess.ExpiresAt) {
		_ = s.store.DeleteSession(ctx, sess.ID)
		return nil, ErrSessionExpired
	}

	// Update last active
	_ = s.store.UpdateSessionActivity(ctx, sess.ID)

	dbUser, err := s.store.GetUserByID(ctx, sess.UserID)
	if err != nil {
		return nil, err
	}
	if dbUser == nil {
		return nil, ErrNotFound
	}

	return dbUserToUser(dbUser), nil
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*User, error) {
	dbUser, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if dbUser == nil {
		return nil, ErrNotFound
	}

	if in.Name != nil {
		dbUser.Name = *in.Name
	}
	if in.AvatarURL != nil {
		dbUser.AvatarURL = sql.NullString{String: *in.AvatarURL, Valid: true}
	}
	dbUser.UpdatedAt = time.Now()

	if err := s.store.UpdateUser(ctx, dbUser); err != nil {
		return nil, err
	}

	return dbUserToUser(dbUser), nil
}

func (s *Service) ChangePassword(ctx context.Context, id string, in *ChangePasswordIn) error {
	dbUser, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	if dbUser == nil {
		return ErrNotFound
	}

	valid, err := password.Verify(in.CurrentPassword, dbUser.PasswordHash)
	if err != nil {
		return err
	}
	if !valid {
		return ErrInvalidPassword
	}

	hash, err := password.Hash(in.NewPassword)
	if err != nil {
		return err
	}

	dbUser.PasswordHash = hash
	dbUser.UpdatedAt = time.Now()

	return s.store.UpdateUser(ctx, dbUser)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	// Delete user's sessions first
	_ = s.store.DeleteUserSessions(ctx, id)
	return s.store.DeleteUser(ctx, id)
}

func (s *Service) ListSessions(ctx context.Context, userID string) ([]*Session, error) {
	dbSessions, err := s.store.ListUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	sessions := make([]*Session, len(dbSessions))
	for i, sess := range dbSessions {
		sessions[i] = dbSessionToSession(sess)
	}
	return sessions, nil
}

func (s *Service) DeleteSession(ctx context.Context, userID, sessionID string) error {
	sess, err := s.store.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess == nil {
		return ErrSessionNotFound
	}
	if sess.UserID != userID {
		return ErrUnauthorized
	}
	return s.store.DeleteSession(ctx, sessionID)
}

func (s *Service) createSession(ctx context.Context, userID, ipAddress, userAgent string) (*Session, error) {
	token := ulid.New()
	tokenHash := hashToken(token)
	now := time.Now()

	dbSession := &duckdb.Session{
		ID:           ulid.New(),
		UserID:       userID,
		TokenHash:    tokenHash,
		IPAddress:    sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:    sql.NullString{String: userAgent, Valid: userAgent != ""},
		LastActiveAt: sql.NullTime{Time: now, Valid: true},
		ExpiresAt:    now.Add(7 * 24 * time.Hour),
		CreatedAt:    now,
	}

	if err := s.store.CreateSession(ctx, dbSession); err != nil {
		return nil, err
	}

	session := dbSessionToSession(dbSession)
	session.TokenHash = token // Return the raw token, not the hash
	return session, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func dbUserToUser(u *duckdb.User) *User {
	return &User{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		PasswordHash:  u.PasswordHash,
		AvatarURL:     u.AvatarURL.String,
		StorageQuota:  u.StorageQuota,
		StorageUsed:   u.StorageUsed,
		IsAdmin:       u.IsAdmin,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

func dbSessionToSession(s *duckdb.Session) *Session {
	return &Session{
		ID:           s.ID,
		UserID:       s.UserID,
		TokenHash:    s.TokenHash,
		IPAddress:    s.IPAddress.String,
		UserAgent:    s.UserAgent.String,
		LastActiveAt: s.LastActiveAt.Time,
		ExpiresAt:    s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
	}
}
