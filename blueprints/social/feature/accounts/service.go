package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/go-mizu/blueprints/social/feature/relationships"
	"github.com/go-mizu/blueprints/social/pkg/password"
	"github.com/go-mizu/blueprints/social/pkg/ulid"
)

const (
	sessionDuration = 30 * 24 * time.Hour // 30 days
	tokenLength     = 32
)

// Service implements the accounts API.
type Service struct {
	store         Store
	relationships relationships.Store
}

// NewService creates a new accounts service.
func NewService(store Store, rels relationships.Store) *Service {
	return &Service{
		store:         store,
		relationships: rels,
	}
}

// Create creates a new account.
func (s *Service) Create(ctx context.Context, in *CreateIn) (*Account, error) {
	// Check username availability
	exists, err := s.store.ExistsUsername(ctx, in.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	// Check email availability
	exists, err = s.store.ExistsEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailTaken
	}

	// Hash password
	hash, err := password.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	// Create account
	now := time.Now()
	account := &Account{
		ID:           ulid.New(),
		Username:     in.Username,
		Email:        in.Email,
		DisplayName:  in.DisplayName,
		Discoverable: true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.Insert(ctx, account, hash); err != nil {
		return nil, err
	}

	return account, nil
}

// GetByID retrieves an account by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Account, error) {
	return s.store.GetByID(ctx, id)
}

// GetByIDs retrieves multiple accounts by IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) ([]*Account, error) {
	return s.store.GetByIDs(ctx, ids)
}

// GetByUsername retrieves an account by username.
func (s *Service) GetByUsername(ctx context.Context, username string) (*Account, error) {
	return s.store.GetByUsername(ctx, username)
}

// GetByEmail retrieves an account by email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*Account, error) {
	return s.store.GetByEmail(ctx, email)
}

// Update updates an account.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Account, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Login authenticates a user and creates a session.
func (s *Service) Login(ctx context.Context, in *LoginIn) (*Session, error) {
	id, hash, suspended, err := s.store.GetPasswordHash(ctx, in.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if suspended {
		return nil, ErrAccountSuspended
	}

	if !password.Verify(in.Password, hash) {
		return nil, ErrInvalidCredentials
	}

	return s.CreateSession(ctx, id, "", "")
}

// CreateSession creates a new session for an account.
func (s *Service) CreateSession(ctx context.Context, accountID, userAgent, ipAddress string) (*Session, error) {
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session := &Session{
		ID:        ulid.New(),
		AccountID: accountID,
		Token:     token,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		ExpiresAt: now.Add(sessionDuration),
		CreatedAt: now,
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	return session, nil
}

// GetSession retrieves a session by token.
func (s *Service) GetSession(ctx context.Context, token string) (*Session, error) {
	session, err := s.store.GetSession(ctx, token)
	if err != nil {
		return nil, ErrInvalidSession
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.store.DeleteSession(ctx, token)
		return nil, ErrInvalidSession
	}

	return session, nil
}

// DeleteSession deletes a session.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	return s.store.DeleteSession(ctx, token)
}

// Verify sets the verified status of an account.
func (s *Service) Verify(ctx context.Context, id string, verified bool) error {
	return s.store.SetVerified(ctx, id, verified)
}

// Suspend sets the suspended status of an account.
func (s *Service) Suspend(ctx context.Context, id string, suspended bool) error {
	return s.store.SetSuspended(ctx, id, suspended)
}

// SetAdmin sets the admin status of an account.
func (s *Service) SetAdmin(ctx context.Context, id string, admin bool) error {
	return s.store.SetAdmin(ctx, id, admin)
}

// List returns a paginated list of accounts.
func (s *Service) List(ctx context.Context, opts ListOpts) (*AccountList, error) {
	accounts, total, err := s.store.List(ctx, opts.Limit, opts.Offset)
	if err != nil {
		return nil, err
	}

	return &AccountList{
		Accounts: accounts,
		Total:    total,
	}, nil
}

// Search searches for accounts by username or display name.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*Account, error) {
	return s.store.Search(ctx, query, limit)
}

// PopulateStats populates follower/following/post counts for an account.
func (s *Service) PopulateStats(ctx context.Context, a *Account) error {
	var err error

	a.FollowersCount, err = s.store.GetFollowersCount(ctx, a.ID)
	if err != nil {
		return err
	}

	a.FollowingCount, err = s.store.GetFollowingCount(ctx, a.ID)
	if err != nil {
		return err
	}

	a.PostsCount, err = s.store.GetPostsCount(ctx, a.ID)
	if err != nil {
		return err
	}

	return nil
}

// PopulateRelationship populates relationship status from viewer's perspective.
func (s *Service) PopulateRelationship(ctx context.Context, a *Account, viewerID string) error {
	if viewerID == "" || viewerID == a.ID {
		return nil
	}

	if s.relationships == nil {
		return nil
	}

	rel, err := s.relationships.GetRelationship(ctx, viewerID, a.ID)
	if err != nil {
		return err
	}

	a.Following = rel.Following
	a.FollowedBy = rel.FollowedBy
	a.Requested = rel.Requested
	a.Blocking = rel.Blocking
	a.Muting = rel.Muting

	return nil
}

func generateToken() (string, error) {
	b := make([]byte, tokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
