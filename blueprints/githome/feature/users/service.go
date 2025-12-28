package users

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service implements the users API
type Service struct {
	store   Store
	baseURL string
}

// NewService creates a new users service
func NewService(store Store, baseURL string) *Service {
	return &Service{store: store, baseURL: baseURL}
}

// Create registers a new user
func (s *Service) Create(ctx context.Context, in *CreateIn) (*User, error) {
	// Check if login exists
	existing, err := s.store.GetByLogin(ctx, in.Login)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	// Check if email exists
	existing, err = s.store.GetByEmail(ctx, in.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailExists
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now()
	user := &User{
		Login:        in.Login,
		Email:        in.Email,
		Name:         in.Name,
		PasswordHash: string(hash),
		Type:         "User",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Generate URLs after ID is assigned
	if err := s.store.Create(ctx, user); err != nil {
		return nil, err
	}

	s.populateURLs(user)
	return user, nil
}

// Authenticate validates credentials and returns the user
func (s *Service) Authenticate(ctx context.Context, login, password string) (*User, error) {
	user, err := s.store.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if user == nil {
		// Try email
		user, err = s.store.GetByEmail(ctx, login)
		if err != nil {
			return nil, err
		}
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	s.populateURLs(user)
	return user, nil
}

// GetByID retrieves a user by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	s.populateURLs(user)
	return user, nil
}

// GetByLogin retrieves a user by login/username
func (s *Service) GetByLogin(ctx context.Context, login string) (*User, error) {
	user, err := s.store.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	s.populateURLs(user)
	return user, nil
}

// GetByEmail retrieves a user by email
func (s *Service) GetByEmail(ctx context.Context, email string) (*User, error) {
	user, err := s.store.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	s.populateURLs(user)
	return user, nil
}

// Update updates a user's profile
func (s *Service) Update(ctx context.Context, id int64, in *UpdateIn) (*User, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// Delete removes a user
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.store.Delete(ctx, id)
}

// List returns all users with pagination
func (s *Service) List(ctx context.Context, opts *ListOpts) ([]*User, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	users, err := s.store.List(ctx, opts)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		s.populateURLs(u)
	}
	return users, nil
}

// ListFollowers returns users following the given user
func (s *Service) ListFollowers(ctx context.Context, login string, opts *ListOpts) ([]*SimpleUser, error) {
	user, err := s.store.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	return s.store.ListFollowers(ctx, user.ID, opts)
}

// ListFollowing returns users the given user is following
func (s *Service) ListFollowing(ctx context.Context, login string, opts *ListOpts) ([]*SimpleUser, error) {
	user, err := s.store.GetByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	return s.store.ListFollowing(ctx, user.ID, opts)
}

// IsFollowing checks if user A follows user B
func (s *Service) IsFollowing(ctx context.Context, follower, followed string) (bool, error) {
	followerUser, err := s.store.GetByLogin(ctx, follower)
	if err != nil {
		return false, err
	}
	if followerUser == nil {
		return false, ErrNotFound
	}

	followedUser, err := s.store.GetByLogin(ctx, followed)
	if err != nil {
		return false, err
	}
	if followedUser == nil {
		return false, ErrNotFound
	}

	return s.store.IsFollowing(ctx, followerUser.ID, followedUser.ID)
}

// Follow makes the authenticated user follow another user
func (s *Service) Follow(ctx context.Context, followerID int64, targetLogin string) error {
	target, err := s.store.GetByLogin(ctx, targetLogin)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotFound
	}

	// Check if already following
	isFollowing, err := s.store.IsFollowing(ctx, followerID, target.ID)
	if err != nil {
		return err
	}
	if isFollowing {
		return nil // Already following, no-op
	}

	if err := s.store.CreateFollow(ctx, followerID, target.ID); err != nil {
		return err
	}

	// Update counts
	if err := s.store.IncrementFollowing(ctx, followerID, 1); err != nil {
		return err
	}
	return s.store.IncrementFollowers(ctx, target.ID, 1)
}

// Unfollow removes follow relationship
func (s *Service) Unfollow(ctx context.Context, followerID int64, targetLogin string) error {
	target, err := s.store.GetByLogin(ctx, targetLogin)
	if err != nil {
		return err
	}
	if target == nil {
		return ErrNotFound
	}

	// Check if following
	isFollowing, err := s.store.IsFollowing(ctx, followerID, target.ID)
	if err != nil {
		return err
	}
	if !isFollowing {
		return nil // Not following, no-op
	}

	if err := s.store.DeleteFollow(ctx, followerID, target.ID); err != nil {
		return err
	}

	// Update counts
	if err := s.store.IncrementFollowing(ctx, followerID, -1); err != nil {
		return err
	}
	return s.store.IncrementFollowers(ctx, target.ID, -1)
}

// UpdatePassword changes a user's password
func (s *Service) UpdatePassword(ctx context.Context, id int64, oldPassword, newPassword string) error {
	user, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrNotFound
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(oldPassword)); err != nil {
		return ErrInvalidCredentials
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	return s.store.UpdatePassword(ctx, id, string(hash))
}

// populateURLs fills in the URL fields for a user
func (s *Service) populateURLs(u *User) {
	u.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("User:%d", u.ID)))
	u.URL = fmt.Sprintf("%s/api/v3/users/%s", s.baseURL, u.Login)
	u.HTMLURL = fmt.Sprintf("%s/%s", s.baseURL, u.Login)
	u.FollowersURL = fmt.Sprintf("%s/api/v3/users/%s/followers", s.baseURL, u.Login)
	u.FollowingURL = fmt.Sprintf("%s/api/v3/users/%s/following{/other_user}", s.baseURL, u.Login)
	u.GistsURL = fmt.Sprintf("%s/api/v3/users/%s/gists{/gist_id}", s.baseURL, u.Login)
	u.StarredURL = fmt.Sprintf("%s/api/v3/users/%s/starred{/owner}{/repo}", s.baseURL, u.Login)
	u.SubscriptionsURL = fmt.Sprintf("%s/api/v3/users/%s/subscriptions", s.baseURL, u.Login)
	u.OrganizationsURL = fmt.Sprintf("%s/api/v3/users/%s/orgs", s.baseURL, u.Login)
	u.ReposURL = fmt.Sprintf("%s/api/v3/users/%s/repos", s.baseURL, u.Login)
	u.EventsURL = fmt.Sprintf("%s/api/v3/users/%s/events{/privacy}", s.baseURL, u.Login)
	u.ReceivedEventsURL = fmt.Sprintf("%s/api/v3/users/%s/received_events", s.baseURL, u.Login)
	if u.AvatarURL == "" {
		u.AvatarURL = fmt.Sprintf("%s/avatars/%s", s.baseURL, u.Login)
	}
}
