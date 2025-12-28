package users

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound          = errors.New("user not found")
	ErrUserExists        = errors.New("username already exists")
	ErrEmailExists       = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// User represents a GitHub user
type User struct {
	ID                int64     `json:"id"`
	NodeID            string    `json:"node_id"`
	Login             string    `json:"login"`
	Name              string    `json:"name,omitempty"`
	Email             string    `json:"email,omitempty"`
	AvatarURL         string    `json:"avatar_url"`
	GravatarID        string    `json:"gravatar_id"`
	URL               string    `json:"url"`
	HTMLURL           string    `json:"html_url"`
	FollowersURL      string    `json:"followers_url"`
	FollowingURL      string    `json:"following_url"`
	GistsURL          string    `json:"gists_url"`
	StarredURL        string    `json:"starred_url"`
	SubscriptionsURL  string    `json:"subscriptions_url"`
	OrganizationsURL  string    `json:"organizations_url"`
	ReposURL          string    `json:"repos_url"`
	EventsURL         string    `json:"events_url"`
	ReceivedEventsURL string    `json:"received_events_url"`
	Type              string    `json:"type"` // User, Organization, Bot
	SiteAdmin         bool      `json:"site_admin"`
	Bio               string    `json:"bio,omitempty"`
	Blog              string    `json:"blog,omitempty"`
	Location          string    `json:"location,omitempty"`
	Company           string    `json:"company,omitempty"`
	Hireable          bool      `json:"hireable,omitempty"`
	TwitterUsername   string    `json:"twitter_username,omitempty"`
	PublicRepos       int       `json:"public_repos"`
	PublicGists       int       `json:"public_gists"`
	Followers         int       `json:"followers"`
	Following         int       `json:"following"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	// Internal fields (not serialized)
	PasswordHash string `json:"-"`
}

// SimpleUser is a compact user representation used in responses
type SimpleUser struct {
	ID        int64  `json:"id"`
	NodeID    string `json:"node_id"`
	Login     string `json:"login"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url"`
	URL       string `json:"url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
	SiteAdmin bool   `json:"site_admin"`
}

// CreateIn represents the input for user registration
type CreateIn struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// UpdateIn represents the input for updating a user
type UpdateIn struct {
	Name            *string `json:"name,omitempty"`
	Email           *string `json:"email,omitempty"`
	Blog            *string `json:"blog,omitempty"`
	TwitterUsername *string `json:"twitter_username,omitempty"`
	Company         *string `json:"company,omitempty"`
	Location        *string `json:"location,omitempty"`
	Hireable        *bool   `json:"hireable,omitempty"`
	Bio             *string `json:"bio,omitempty"`
}

// ListOpts contains pagination options
type ListOpts struct {
	Page    int `json:"page,omitempty"`
	PerPage int `json:"per_page,omitempty"`
	Since   int64 `json:"since,omitempty"` // User ID to start after
}

// API defines the users service interface
type API interface {
	// Create registers a new user
	Create(ctx context.Context, in *CreateIn) (*User, error)

	// Authenticate validates credentials and returns the user
	Authenticate(ctx context.Context, login, password string) (*User, error)

	// GetByID retrieves a user by ID
	GetByID(ctx context.Context, id int64) (*User, error)

	// GetByLogin retrieves a user by login/username
	GetByLogin(ctx context.Context, login string) (*User, error)

	// GetByEmail retrieves a user by email
	GetByEmail(ctx context.Context, email string) (*User, error)

	// Update updates a user's profile
	Update(ctx context.Context, id int64, in *UpdateIn) (*User, error)

	// Delete removes a user
	Delete(ctx context.Context, id int64) error

	// List returns all users with pagination
	List(ctx context.Context, opts *ListOpts) ([]*User, error)

	// ListFollowers returns users following the given user
	ListFollowers(ctx context.Context, login string, opts *ListOpts) ([]*SimpleUser, error)

	// ListFollowing returns users the given user is following
	ListFollowing(ctx context.Context, login string, opts *ListOpts) ([]*SimpleUser, error)

	// IsFollowing checks if user A follows user B
	IsFollowing(ctx context.Context, follower, followed string) (bool, error)

	// Follow makes the authenticated user follow another user
	Follow(ctx context.Context, followerID int64, targetLogin string) error

	// Unfollow removes follow relationship
	Unfollow(ctx context.Context, followerID int64, targetLogin string) error

	// UpdatePassword changes a user's password
	UpdatePassword(ctx context.Context, id int64, oldPassword, newPassword string) error
}

// Store defines the data access interface for users
type Store interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id int64, in *UpdateIn) error
	UpdatePassword(ctx context.Context, id int64, passwordHash string) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, opts *ListOpts) ([]*User, error)

	// Follow relationships
	CreateFollow(ctx context.Context, followerID, followedID int64) error
	DeleteFollow(ctx context.Context, followerID, followedID int64) error
	IsFollowing(ctx context.Context, followerID, followedID int64) (bool, error)
	ListFollowers(ctx context.Context, userID int64, opts *ListOpts) ([]*SimpleUser, error)
	ListFollowing(ctx context.Context, userID int64, opts *ListOpts) ([]*SimpleUser, error)
	IncrementFollowers(ctx context.Context, userID int64, delta int) error
	IncrementFollowing(ctx context.Context, userID int64, delta int) error
}

// ToSimple converts a User to SimpleUser
func (u *User) ToSimple() *SimpleUser {
	return &SimpleUser{
		ID:        u.ID,
		NodeID:    u.NodeID,
		Login:     u.Login,
		Name:      u.Name,
		Email:     u.Email,
		AvatarURL: u.AvatarURL,
		URL:       u.URL,
		HTMLURL:   u.HTMLURL,
		Type:      u.Type,
		SiteAdmin: u.SiteAdmin,
	}
}
