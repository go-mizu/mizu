package accounts

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/cms/pkg/capability"
	"github.com/go-mizu/mizu/blueprints/cms/pkg/password"
	"github.com/go-mizu/mizu/blueprints/cms/pkg/ulid"
	"github.com/go-mizu/mizu/blueprints/cms/store/duckdb"
)

// Service implements the accounts API.
type Service struct {
	users    *duckdb.UsersStore
	usermeta *duckdb.UsermetaStore
	sessions *duckdb.SessionsStore
}

// NewService creates a new accounts service.
func NewService(users *duckdb.UsersStore, usermeta *duckdb.UsermetaStore, sessions *duckdb.SessionsStore) *Service {
	return &Service{
		users:    users,
		usermeta: usermeta,
		sessions: sessions,
	}
}

// Create creates a new user.
func (s *Service) Create(ctx context.Context, in CreateIn) (*User, error) {
	if err := in.Validate(); err != nil {
		return nil, err
	}

	// Check if username is taken
	existing, _ := s.users.GetByLogin(ctx, in.Username)
	if existing != nil {
		return nil, ErrLoginTaken
	}

	// Check if email is taken
	existing, _ = s.users.GetByEmail(ctx, in.Email)
	if existing != nil {
		return nil, ErrEmailTaken
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	now := time.Now()
	displayName := in.DisplayName
	if displayName == "" {
		displayName = in.Username
	}

	user := &duckdb.User{
		ID:             ulid.New(),
		UserLogin:      in.Username,
		UserPass:       hash,
		UserNicename:   strings.ToLower(in.Username),
		UserEmail:      in.Email,
		UserURL:        in.URL,
		UserRegistered: now,
		UserStatus:     0,
		DisplayName:    displayName,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	// Set default role
	roles := in.Roles
	if len(roles) == 0 {
		roles = []string{"subscriber"}
	}
	if err := s.SetRoles(ctx, user.ID, roles); err != nil {
		return nil, err
	}

	// Set description if provided
	if in.Description != "" {
		_ = s.SetMeta(ctx, user.ID, "description", in.Description)
	}

	return s.toUser(ctx, user)
}

// GetByID retrieves a user by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return s.toUser(ctx, user)
}

// GetByLogin retrieves a user by login name.
func (s *Service) GetByLogin(ctx context.Context, login string) (*User, error) {
	user, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return s.toUser(ctx, user)
}

// GetByEmail retrieves a user by email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return s.toUser(ctx, user)
}

// GetBySlug retrieves a user by nicename (slug).
func (s *Service) GetBySlug(ctx context.Context, slug string) (*User, error) {
	user, err := s.users.GetByNicename(ctx, slug)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	return s.toUser(ctx, user)
}

// Update updates a user.
func (s *Service) Update(ctx context.Context, id string, in UpdateIn) (*User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	if in.Email != nil {
		if !EmailRegex.MatchString(*in.Email) {
			return nil, ErrInvalidEmail
		}
		// Check if email is taken
		existing, _ := s.users.GetByEmail(ctx, *in.Email)
		if existing != nil && existing.ID != id {
			return nil, ErrEmailTaken
		}
		user.UserEmail = *in.Email
	}

	if in.DisplayName != nil {
		user.DisplayName = *in.DisplayName
	}

	if in.URL != nil {
		user.UserURL = *in.URL
	}

	if in.Nicename != nil {
		user.UserNicename = *in.Nicename
	}

	if in.Password != nil {
		if len(*in.Password) < PasswordMinLen {
			return nil, ErrInvalidPassword
		}
		hash, err := password.Hash(*in.Password)
		if err != nil {
			return nil, err
		}
		user.UserPass = hash
	}

	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	if in.Description != nil {
		_ = s.SetMeta(ctx, id, "description", *in.Description)
	}

	if len(in.Roles) > 0 {
		if err := s.SetRoles(ctx, id, in.Roles); err != nil {
			return nil, err
		}
	}

	return s.toUser(ctx, user)
}

// Delete deletes a user.
func (s *Service) Delete(ctx context.Context, id string, reassign string) error {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}

	// TODO: Reassign posts to another user if specified

	// Delete user meta
	_ = s.usermeta.DeleteAllForUser(ctx, id)

	// Delete sessions
	_ = s.sessions.DeleteByUser(ctx, id)

	// Delete user
	return s.users.Delete(ctx, id)
}

// Login authenticates a user.
func (s *Service) Login(ctx context.Context, in LoginIn) (*User, error) {
	user, err := s.users.GetByLogin(ctx, in.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		// Try email
		user, err = s.users.GetByEmail(ctx, in.Username)
		if err != nil {
			return nil, err
		}
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if !password.Verify(in.Password, user.UserPass) {
		return nil, ErrInvalidCredentials
	}

	// Check if password needs rehash
	if password.NeedsRehash(user.UserPass) {
		newHash, err := password.Hash(in.Password)
		if err == nil {
			user.UserPass = newHash
			_ = s.users.Update(ctx, user)
		}
	}

	return s.toUser(ctx, user)
}

// CreateSession creates a new session.
func (s *Service) CreateSession(ctx context.Context, userID, userAgent, ip string) (*Session, error) {
	token, err := password.GenerateToken(32)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	sess := &duckdb.Session{
		SessionID:    ulid.New(),
		UserID:       userID,
		Token:        token,
		IPAddress:    ip,
		UserAgent:    userAgent,
		LastActivity: now,
		ExpiresAt:    now.Add(SessionTTL),
	}

	if err := s.sessions.Create(ctx, sess); err != nil {
		return nil, err
	}

	return &Session{
		ID:        sess.SessionID,
		UserID:    sess.UserID,
		Token:     sess.Token,
		UserAgent: sess.UserAgent,
		IP:        sess.IPAddress,
		ExpiresAt: sess.ExpiresAt,
		CreatedAt: now,
	}, nil
}

// GetSession retrieves a session by token.
func (s *Service) GetSession(ctx context.Context, token string) (*Session, error) {
	sess, err := s.sessions.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil
	}

	if time.Now().After(sess.ExpiresAt) {
		_ = s.sessions.Delete(ctx, token)
		return nil, ErrSessionExpired
	}

	// Update last activity
	_ = s.sessions.UpdateLastActivity(ctx, sess.SessionID)

	return &Session{
		ID:        sess.SessionID,
		UserID:    sess.UserID,
		Token:     sess.Token,
		UserAgent: sess.UserAgent,
		IP:        sess.IPAddress,
		ExpiresAt: sess.ExpiresAt,
	}, nil
}

// DeleteSession deletes a session.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, token)
}

// DeleteAllSessions deletes all sessions for a user.
func (s *Service) DeleteAllSessions(ctx context.Context, userID string) error {
	return s.sessions.DeleteByUser(ctx, userID)
}

// GetMeta retrieves a user meta value.
func (s *Service) GetMeta(ctx context.Context, userID, key string) (string, error) {
	return s.usermeta.Get(ctx, userID, key)
}

// SetMeta sets a user meta value.
func (s *Service) SetMeta(ctx context.Context, userID, key, value string) error {
	existing, _ := s.usermeta.Get(ctx, userID, key)
	if existing != "" {
		return s.usermeta.Update(ctx, userID, key, value)
	}
	return s.usermeta.Create(ctx, &duckdb.Usermeta{
		UmetaID:   ulid.New(),
		UserID:    userID,
		MetaKey:   key,
		MetaValue: value,
	})
}

// DeleteMeta deletes a user meta value.
func (s *Service) DeleteMeta(ctx context.Context, userID, key string) error {
	return s.usermeta.Delete(ctx, userID, key)
}

// GetRoles retrieves the roles for a user.
func (s *Service) GetRoles(ctx context.Context, userID string) ([]string, error) {
	value, err := s.usermeta.Get(ctx, userID, "wp_capabilities")
	if err != nil {
		return nil, err
	}
	if value == "" {
		return []string{"subscriber"}, nil
	}
	// Parse serialized PHP array or JSON
	// For simplicity, assume comma-separated roles
	return strings.Split(value, ","), nil
}

// SetRoles sets the roles for a user.
func (s *Service) SetRoles(ctx context.Context, userID string, roles []string) error {
	// Store as comma-separated for simplicity
	value := strings.Join(roles, ",")
	return s.SetMeta(ctx, userID, "wp_capabilities", value)
}

// HasCapability checks if a user has a specific capability.
func (s *Service) HasCapability(ctx context.Context, userID, cap string) (bool, error) {
	roles, err := s.GetRoles(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if capability.HasCapability(role, cap) {
			return true, nil
		}
	}
	return false, nil
}

// List lists users.
func (s *Service) List(ctx context.Context, opts ListOpts) ([]*User, int, error) {
	storeOpts := duckdb.UserListOpts{
		Limit:   opts.PerPage,
		Offset:  (opts.Page - 1) * opts.PerPage,
		OrderBy: opts.OrderBy,
		Order:   opts.Order,
		Search:  opts.Search,
	}

	if storeOpts.Limit == 0 {
		storeOpts.Limit = 10
	}

	users, total, err := s.users.List(ctx, storeOpts)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*User, 0, len(users))
	for _, u := range users {
		user, err := s.toUser(ctx, u)
		if err != nil {
			continue
		}
		result = append(result, user)
	}

	return result, total, nil
}

// Count returns the total number of users.
func (s *Service) Count(ctx context.Context) (int, error) {
	return s.users.Count(ctx)
}

// GetCurrent retrieves the current user from a session token.
func (s *Service) GetCurrent(ctx context.Context, token string) (*User, error) {
	sess, err := s.GetSession(ctx, token)
	if err != nil {
		return nil, err
	}
	if sess == nil {
		return nil, nil
	}
	return s.GetByID(ctx, sess.UserID)
}

// toUser converts a store user to an API user.
func (s *Service) toUser(ctx context.Context, u *duckdb.User) (*User, error) {
	roles, _ := s.GetRoles(ctx, u.ID)
	description, _ := s.GetMeta(ctx, u.ID, "description")

	return &User{
		ID:          u.ID,
		Username:    u.UserLogin,
		Email:       u.UserEmail,
		Nicename:    u.UserNicename,
		URL:         u.UserURL,
		DisplayName: u.DisplayName,
		Registered:  u.UserRegistered,
		Status:      u.UserStatus,
		Roles:       roles,
		Description: description,
	}, nil
}
