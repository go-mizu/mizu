// Package stories provides story/status management.
package stories

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("story not found")
	ErrExpired      = errors.New("story expired")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// StoryType represents the type of story.
type StoryType string

const (
	TypeText  StoryType = "text"
	TypeImage StoryType = "image"
	TypeVideo StoryType = "video"
)

// Privacy represents story privacy setting.
type Privacy string

const (
	PrivacyContacts Privacy = "contacts"
	PrivacySelected Privacy = "selected"
	PrivacyEveryone Privacy = "everyone"
)

// Story represents a user story/status.
type Story struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Type            StoryType `json:"type"`
	Content         string    `json:"content,omitempty"`
	MediaURL        string    `json:"media_url,omitempty"`
	ThumbnailURL    string    `json:"thumbnail_url,omitempty"`
	BackgroundColor string    `json:"background_color,omitempty"`
	TextStyle       string    `json:"text_style,omitempty"`
	Duration        int       `json:"duration"`
	ViewCount       int       `json:"view_count"`
	Privacy         Privacy   `json:"privacy"`
	IsHighlight     bool      `json:"is_highlight"`
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`

	// Populated from joins
	User    any          `json:"user,omitempty"`
	Viewers []*StoryView `json:"viewers,omitempty"`
}

// StoryView represents a story view.
type StoryView struct {
	StoryID  string    `json:"story_id"`
	ViewerID string    `json:"viewer_id"`
	ViewedAt time.Time `json:"viewed_at"`

	// Joined user info
	Viewer any `json:"viewer,omitempty"`
}

// CreateIn contains input for creating a story.
type CreateIn struct {
	Type            StoryType `json:"type"`
	Content         string    `json:"content,omitempty"`
	MediaURL        string    `json:"media_url,omitempty"`
	ThumbnailURL    string    `json:"thumbnail_url,omitempty"`
	BackgroundColor string    `json:"background_color,omitempty"`
	TextStyle       string    `json:"text_style,omitempty"`
	Duration        int       `json:"duration,omitempty"` // seconds to display
	Privacy         Privacy   `json:"privacy,omitempty"`
	AllowedUsers    []string  `json:"allowed_users,omitempty"`  // for selected privacy
	ExcludedUsers   []string  `json:"excluded_users,omitempty"` // for contacts privacy
}

// API defines the stories service contract.
type API interface {
	Create(ctx context.Context, userID string, in *CreateIn) (*Story, error)
	GetByID(ctx context.Context, id string) (*Story, error)
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string) ([]*Story, error)           // Stories from contacts
	ListByUser(ctx context.Context, userID string) ([]*Story, error)     // User's own stories
	ListHighlights(ctx context.Context, userID string) ([]*Story, error) // User's highlights
	View(ctx context.Context, storyID, viewerID string) error
	GetViewers(ctx context.Context, storyID string, limit int) ([]*StoryView, error)
	MarkAsHighlight(ctx context.Context, storyID, userID string) error
	UnmarkAsHighlight(ctx context.Context, storyID, userID string) error
	MuteUser(ctx context.Context, userID, mutedUserID string) error
	UnmuteUser(ctx context.Context, userID, mutedUserID string) error
	ListMutedUsers(ctx context.Context, userID string) ([]string, error)
	CleanupExpired(ctx context.Context) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, s *Story) error
	GetByID(ctx context.Context, id string) (*Story, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, userID string) ([]*Story, error)
	ListByUser(ctx context.Context, userID string) ([]*Story, error)
	ListHighlights(ctx context.Context, userID string) ([]*Story, error)
	IncrementViewCount(ctx context.Context, storyID string) error
	UpdateHighlight(ctx context.Context, storyID string, isHighlight bool) error

	// Views
	InsertView(ctx context.Context, v *StoryView) error
	GetViewers(ctx context.Context, storyID string, limit int) ([]*StoryView, error)
	HasViewed(ctx context.Context, storyID, viewerID string) (bool, error)

	// Privacy
	InsertPrivacy(ctx context.Context, storyID, userID string, isAllowed bool) error
	CanView(ctx context.Context, storyID, viewerID string) (bool, error)

	// Muting
	Mute(ctx context.Context, userID, mutedUserID string) error
	Unmute(ctx context.Context, userID, mutedUserID string) error
	ListMutedUsers(ctx context.Context, userID string) ([]string, error)
	IsMuted(ctx context.Context, userID, targetUserID string) (bool, error)

	// Cleanup
	DeleteExpired(ctx context.Context) (int64, error)
}
