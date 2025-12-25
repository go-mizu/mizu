package stories

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/messaging/pkg/ulid"
)

const (
	defaultExpiryHours = 24
	defaultDuration    = 5 // seconds
)

// Service implements the stories API.
type Service struct {
	store Store
}

// NewService creates a new stories service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new story.
func (s *Service) Create(ctx context.Context, userID string, in *CreateIn) (*Story, error) {
	now := time.Now()

	duration := in.Duration
	if duration <= 0 {
		duration = defaultDuration
	}

	privacy := in.Privacy
	if privacy == "" {
		privacy = PrivacyContacts
	}

	story := &Story{
		ID:              ulid.New(),
		UserID:          userID,
		Type:            in.Type,
		Content:         in.Content,
		MediaURL:        in.MediaURL,
		ThumbnailURL:    in.ThumbnailURL,
		BackgroundColor: in.BackgroundColor,
		TextStyle:       in.TextStyle,
		Duration:        duration,
		ViewCount:       0,
		Privacy:         privacy,
		IsHighlight:     false,
		ExpiresAt:       now.Add(defaultExpiryHours * time.Hour),
		CreatedAt:       now,
	}

	if err := s.store.Insert(ctx, story); err != nil {
		return nil, err
	}

	// Set privacy exceptions
	for _, uid := range in.AllowedUsers {
		s.store.InsertPrivacy(ctx, story.ID, uid, true)
	}
	for _, uid := range in.ExcludedUsers {
		s.store.InsertPrivacy(ctx, story.ID, uid, false)
	}

	return story, nil
}

// GetByID retrieves a story by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Story, error) {
	story, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if story.ExpiresAt.Before(time.Now()) && !story.IsHighlight {
		return nil, ErrExpired
	}

	return story, nil
}

// Delete deletes a story.
func (s *Service) Delete(ctx context.Context, id, userID string) error {
	story, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if story.UserID != userID {
		return ErrForbidden
	}

	return s.store.Delete(ctx, id)
}

// List lists stories from contacts.
func (s *Service) List(ctx context.Context, userID string) ([]*Story, error) {
	return s.store.List(ctx, userID)
}

// ListByUser lists a user's own stories.
func (s *Service) ListByUser(ctx context.Context, userID string) ([]*Story, error) {
	return s.store.ListByUser(ctx, userID)
}

// ListHighlights lists a user's story highlights.
func (s *Service) ListHighlights(ctx context.Context, userID string) ([]*Story, error) {
	return s.store.ListHighlights(ctx, userID)
}

// View records a story view.
func (s *Service) View(ctx context.Context, storyID, viewerID string) error {
	// Check if already viewed
	viewed, err := s.store.HasViewed(ctx, storyID, viewerID)
	if err != nil {
		return err
	}
	if viewed {
		return nil
	}

	// Record view
	view := &StoryView{
		StoryID:  storyID,
		ViewerID: viewerID,
		ViewedAt: time.Now(),
	}

	if err := s.store.InsertView(ctx, view); err != nil {
		return err
	}

	return s.store.IncrementViewCount(ctx, storyID)
}

// GetViewers gets story viewers.
func (s *Service) GetViewers(ctx context.Context, storyID string, limit int) ([]*StoryView, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.GetViewers(ctx, storyID, limit)
}

// MarkAsHighlight marks a story as a highlight.
func (s *Service) MarkAsHighlight(ctx context.Context, storyID, userID string) error {
	story, err := s.store.GetByID(ctx, storyID)
	if err != nil {
		return err
	}

	if story.UserID != userID {
		return ErrForbidden
	}

	return s.store.UpdateHighlight(ctx, storyID, true)
}

// UnmarkAsHighlight removes highlight status from a story.
func (s *Service) UnmarkAsHighlight(ctx context.Context, storyID, userID string) error {
	story, err := s.store.GetByID(ctx, storyID)
	if err != nil {
		return err
	}

	if story.UserID != userID {
		return ErrForbidden
	}

	return s.store.UpdateHighlight(ctx, storyID, false)
}

// MuteUser mutes a user's stories.
func (s *Service) MuteUser(ctx context.Context, userID, mutedUserID string) error {
	return s.store.Mute(ctx, userID, mutedUserID)
}

// UnmuteUser unmutes a user's stories.
func (s *Service) UnmuteUser(ctx context.Context, userID, mutedUserID string) error {
	return s.store.Unmute(ctx, userID, mutedUserID)
}

// ListMutedUsers lists muted users.
func (s *Service) ListMutedUsers(ctx context.Context, userID string) ([]string, error) {
	return s.store.ListMutedUsers(ctx, userID)
}

// CleanupExpired removes expired stories.
func (s *Service) CleanupExpired(ctx context.Context) error {
	_, err := s.store.DeleteExpired(ctx)
	return err
}
