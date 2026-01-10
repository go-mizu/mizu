package users

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/go-mizu/blueprints/table/pkg/ulid"
)

// Service implements the users API.
type Service struct {
	store Store
	db    *sql.DB
}

// NewService creates a new users service.
func NewService(store Store, db *sql.DB) *Service {
	return &Service{store: store, db: db}
}

// Create creates a new user.
func (s *Service) Create(ctx context.Context, in CreateIn) (*User, error) {
	if in.Email == "" {
		return nil, ErrInvalidEmail
	}

	// Check for existing user
	existing, _ := s.store.GetByEmail(ctx, in.Email)
	if existing != nil {
		return nil, ErrEmailTaken
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           ulid.New(),
		Email:        in.Email,
		Name:         in.Name,
		PasswordHash: string(hash),
	}

	if err := s.store.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetByID retrieves a user by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.store.GetByID(ctx, id)
}

// GetByEmail retrieves a user by email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetByEmail(ctx, email)
}

// Update updates a user.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		user.Name = *in.Name
	}
	if in.AvatarURL != nil {
		user.AvatarURL = *in.AvatarURL
	}

	if err := s.store.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Delete deletes a user.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Authenticate validates credentials and returns the user.
func (s *Service) Authenticate(ctx context.Context, email, password string) (*User, error) {
	user, err := s.store.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidAuth
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidAuth
	}

	return user, nil
}

// ChangePassword changes a user's password.
func (s *Service) ChangePassword(ctx context.Context, id, oldPassword, newPassword string) error {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidAuth
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hash)
	return s.store.Update(ctx, user)
}

// Session represents a user session.
type Session struct {
	ID        string
	UserID    string
	Token     string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// CreateSession creates a new session for a user.
func (s *Service) CreateSession(ctx context.Context, userID string) (*Session, error) {
	token := generateToken()
	session := &Session{
		ID:        ulid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
		CreatedAt: time.Now(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, session.ID, session.UserID, session.Token, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetBySession retrieves a user by session token.
func (s *Service) GetBySession(ctx context.Context, token string) (*User, error) {
	var userID string
	var expiresAt time.Time

	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, expires_at FROM sessions WHERE token = $1
	`, token).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if time.Now().After(expiresAt) {
		// Session expired, delete it
		s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, token)
		return nil, ErrNotFound
	}

	return s.store.GetByID(ctx, userID)
}

// DeleteSession deletes a session.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

// Register creates a new user and session.
func (s *Service) Register(ctx context.Context, email, name, password string) (*User, *Session, error) {
	user, err := s.Create(ctx, CreateIn{
		Email:    email,
		Name:     name,
		Password: password,
	})
	if err != nil {
		return nil, nil, err
	}

	session, err := s.CreateSession(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

// Login authenticates and creates a session.
func (s *Service) Login(ctx context.Context, email, password string) (*User, *Session, error) {
	user, err := s.Authenticate(ctx, email, password)
	if err != nil {
		return nil, nil, err
	}

	session, err := s.CreateSession(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, session, nil
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
