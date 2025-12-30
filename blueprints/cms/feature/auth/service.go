package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/cms/config"
	"github.com/go-mizu/blueprints/cms/pkg/jwt"
	"github.com/go-mizu/blueprints/cms/pkg/password"
	"github.com/go-mizu/blueprints/cms/pkg/ulid"
	"github.com/go-mizu/blueprints/cms/store/duckdb"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account is locked")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)

// Service implements the auth API.
type Service struct {
	db          *sql.DB
	sessions    *duckdb.SessionsStore
	jwt         *jwt.Manager
	collections map[string]*config.CollectionConfig
}

// NewService creates a new auth service.
func NewService(db *sql.DB, sessions *duckdb.SessionsStore, secret string, collections []config.CollectionConfig) *Service {
	collectionMap := make(map[string]*config.CollectionConfig)
	for i := range collections {
		if collections[i].Auth != nil {
			collectionMap[collections[i].Slug] = &collections[i]
		}
	}

	jwtManager := jwt.NewManager(jwt.Config{
		Secret:          secret,
		TokenExpiration: 2 * time.Hour, // Default
	})

	return &Service{
		db:          db,
		sessions:    sessions,
		jwt:         jwtManager,
		collections: collectionMap,
	}
}

// Login authenticates a user and returns tokens.
func (s *Service) Login(ctx context.Context, collection string, input *LoginInput) (*LoginResult, error) {
	cfg := s.collections[collection]
	if cfg == nil {
		return nil, fmt.Errorf("collection %s does not support authentication", collection)
	}

	// Find user by email
	user, err := s.findByEmail(ctx, collection, input.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check if locked
	if user.LockUntil != nil && user.LockUntil.After(time.Now()) {
		return nil, ErrAccountLocked
	}

	// Verify password
	if !password.Verify(input.Password, user.Password) {
		// Increment login attempts
		s.incrementLoginAttempts(ctx, collection, user.ID, cfg.Auth.MaxLoginAttempts, cfg.Auth.LockTime)
		return nil, ErrInvalidCredentials
	}

	// Reset login attempts on successful login
	s.resetLoginAttempts(ctx, collection, user.ID)

	// Generate tokens
	token, exp, err := s.jwt.Generate(user.ID, collection)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	refreshToken := generateRefreshToken()

	// Create session
	session := &duckdb.Session{
		UserID:       user.ID,
		Collection:   collection,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    exp,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &LoginResult{
		User:         s.toUser(user),
		Token:        token,
		RefreshToken: refreshToken,
		Exp:          exp.Unix(),
	}, nil
}

// Logout invalidates the current session.
func (s *Service) Logout(ctx context.Context, collection, token string) error {
	return s.sessions.Delete(ctx, token)
}

// Me returns the current authenticated user.
func (s *Service) Me(ctx context.Context, collection, token string) (*User, error) {
	claims, err := s.jwt.Validate(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims.Collection != collection {
		return nil, ErrInvalidToken
	}

	user, err := s.findByID(ctx, collection, claims.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return s.toUser(user), nil
}

// RefreshToken refreshes an expired token.
func (s *Service) RefreshToken(ctx context.Context, collection, refreshToken string) (*RefreshResult, error) {
	session, err := s.sessions.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	if session == nil {
		return nil, ErrInvalidToken
	}

	if session.Collection != collection {
		return nil, ErrInvalidToken
	}

	// Get user
	user, err := s.findByID(ctx, collection, session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Generate new tokens
	token, exp, err := s.jwt.Generate(user.ID, collection)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	newRefreshToken := generateRefreshToken()

	// Update session
	session.Token = token
	session.RefreshToken = newRefreshToken
	session.ExpiresAt = exp
	if err := s.sessions.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update session: %w", err)
	}

	return &RefreshResult{
		User:         s.toUser(user),
		Token:        token,
		RefreshToken: newRefreshToken,
		Exp:          exp.Unix(),
	}, nil
}

// Register creates a new user account.
func (s *Service) Register(ctx context.Context, collection string, input *RegisterInput) (*LoginResult, error) {
	cfg := s.collections[collection]
	if cfg == nil {
		return nil, fmt.Errorf("collection %s does not support authentication", collection)
	}

	// Check if email exists
	existing, err := s.findByEmail(ctx, collection, input.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailExists
	}

	// Hash password
	hashedPassword, err := password.Hash(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Create user
	id := ulid.New()
	now := time.Now()

	query := fmt.Sprintf(`INSERT INTO %s (id, email, password, first_name, last_name, roles, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, collection)

	roles := `["user"]` // Default role
	_, err = s.db.ExecContext(ctx, query, id, input.Email, hashedPassword, input.FirstName, input.LastName, roles, now, now)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Generate tokens
	token, exp, err := s.jwt.Generate(id, collection)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	refreshToken := generateRefreshToken()

	// Create session
	session := &duckdb.Session{
		UserID:       id,
		Collection:   collection,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    exp,
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &LoginResult{
		User: &User{
			ID:        id,
			Email:     input.Email,
			FirstName: input.FirstName,
			LastName:  input.LastName,
			Roles:     []string{"user"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		Token:        token,
		RefreshToken: refreshToken,
		Exp:          exp.Unix(),
	}, nil
}

// ForgotPassword initiates password reset flow.
func (s *Service) ForgotPassword(ctx context.Context, collection string, input *ForgotPasswordInput) error {
	user, err := s.findByEmail(ctx, collection, input.Email)
	if err != nil {
		return err
	}
	if user == nil {
		// Don't reveal if user exists
		return nil
	}

	// Generate reset token
	resetToken := generateRefreshToken()
	expiration := time.Now().Add(1 * time.Hour)

	query := fmt.Sprintf(`UPDATE %s SET reset_password_token = ?, reset_password_expiration = ? WHERE id = ?`, collection)
	_, err = s.db.ExecContext(ctx, query, resetToken, expiration, user.ID)
	if err != nil {
		return fmt.Errorf("save reset token: %w", err)
	}

	// TODO: Send email with reset link
	// In production, this would send an email

	return nil
}

// ResetPassword resets a user's password.
func (s *Service) ResetPassword(ctx context.Context, collection string, input *ResetPasswordInput) error {
	// Find user by reset token
	query := fmt.Sprintf(`SELECT id, reset_password_expiration FROM %s WHERE reset_password_token = ?`, collection)

	var id string
	var expiration sql.NullTime
	err := s.db.QueryRowContext(ctx, query, input.Token).Scan(&id, &expiration)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrInvalidToken
		}
		return fmt.Errorf("find user: %w", err)
	}

	if !expiration.Valid || expiration.Time.Before(time.Now()) {
		return ErrTokenExpired
	}

	// Hash new password
	hashedPassword, err := password.Hash(input.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// Update password and clear reset token
	updateQuery := fmt.Sprintf(`UPDATE %s SET password = ?, reset_password_token = NULL, reset_password_expiration = NULL, updated_at = ? WHERE id = ?`, collection)
	_, err = s.db.ExecContext(ctx, updateQuery, hashedPassword, time.Now(), id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Invalidate all sessions
	s.sessions.DeleteByUser(ctx, id)

	return nil
}

// VerifyEmail verifies a user's email address.
func (s *Service) VerifyEmail(ctx context.Context, collection, token string) error {
	query := fmt.Sprintf(`UPDATE %s SET verified = TRUE, verification_token = NULL WHERE verification_token = ?`, collection)
	result, err := s.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("verify email: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrInvalidToken
	}

	return nil
}

// Unlock removes account lockout.
func (s *Service) Unlock(ctx context.Context, collection string, email string) error {
	query := fmt.Sprintf(`UPDATE %s SET login_attempts = 0, lock_until = NULL WHERE email = ?`, collection)
	_, err := s.db.ExecContext(ctx, query, email)
	return err
}

// ValidateToken validates a token and returns the user ID.
func (s *Service) ValidateToken(token string) (string, string, error) {
	claims, err := s.jwt.Validate(token)
	if err != nil {
		return "", "", err
	}
	return claims.UserID, claims.Collection, nil
}

// Internal types and helpers

type dbUser struct {
	ID        string
	Email     string
	Password  string
	FirstName sql.NullString
	LastName  sql.NullString
	Roles     sql.NullString
	LoginAttempts int
	LockUntil *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Service) findByEmail(ctx context.Context, collection, email string) (*dbUser, error) {
	query := fmt.Sprintf(`SELECT id, email, password, first_name, last_name, roles, login_attempts, lock_until, created_at, updated_at
		FROM %s WHERE LOWER(email) = LOWER(?)`, collection)

	var user dbUser
	var lockUntil sql.NullTime

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Password, &user.FirstName, &user.LastName, &user.Roles,
		&user.LoginAttempts, &lockUntil, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if lockUntil.Valid {
		user.LockUntil = &lockUntil.Time
	}

	return &user, nil
}

func (s *Service) findByID(ctx context.Context, collection, id string) (*dbUser, error) {
	query := fmt.Sprintf(`SELECT id, email, password, first_name, last_name, roles, login_attempts, lock_until, created_at, updated_at
		FROM %s WHERE id = ?`, collection)

	var user dbUser
	var lockUntil sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.Password, &user.FirstName, &user.LastName, &user.Roles,
		&user.LoginAttempts, &lockUntil, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if lockUntil.Valid {
		user.LockUntil = &lockUntil.Time
	}

	return &user, nil
}

func (s *Service) toUser(u *dbUser) *User {
	user := &User{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
	if u.FirstName.Valid {
		user.FirstName = u.FirstName.String
	}
	if u.LastName.Valid {
		user.LastName = u.LastName.String
	}
	if u.Roles.Valid {
		// Parse JSON array
		roles := strings.Trim(u.Roles.String, "[]\"")
		for _, role := range strings.Split(roles, ",") {
			role = strings.Trim(role, "\" ")
			if role != "" {
				user.Roles = append(user.Roles, role)
			}
		}
	}
	return user
}

func (s *Service) incrementLoginAttempts(ctx context.Context, collection, userID string, maxAttempts, lockTime int) {
	query := fmt.Sprintf(`UPDATE %s SET login_attempts = login_attempts + 1 WHERE id = ?`, collection)
	s.db.ExecContext(ctx, query, userID)

	// Check if should lock
	if maxAttempts > 0 {
		var attempts int
		countQuery := fmt.Sprintf(`SELECT login_attempts FROM %s WHERE id = ?`, collection)
		s.db.QueryRowContext(ctx, countQuery, userID).Scan(&attempts)

		if attempts >= maxAttempts {
			lockUntil := time.Now().Add(time.Duration(lockTime) * time.Second)
			lockQuery := fmt.Sprintf(`UPDATE %s SET lock_until = ? WHERE id = ?`, collection)
			s.db.ExecContext(ctx, lockQuery, lockUntil, userID)
		}
	}
}

func (s *Service) resetLoginAttempts(ctx context.Context, collection, userID string) {
	query := fmt.Sprintf(`UPDATE %s SET login_attempts = 0, lock_until = NULL WHERE id = ?`, collection)
	s.db.ExecContext(ctx, query, userID)
}

func generateRefreshToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
