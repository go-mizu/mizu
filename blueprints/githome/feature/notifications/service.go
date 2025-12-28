package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// Service implements the notifications API
type Service struct {
	store     Store
	repoStore repos.Store
	baseURL   string
}

// NewService creates a new notifications service
func NewService(store Store, repoStore repos.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		baseURL:   baseURL,
	}
}

// List returns notifications for the authenticated user
func (s *Service) List(ctx context.Context, userID int64, opts *ListOpts) ([]*Notification, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 50}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 50
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	notifications, err := s.store.List(ctx, userID, opts)
	if err != nil {
		return nil, err
	}

	for _, n := range notifications {
		s.populateURLs(n)
	}
	return notifications, nil
}

// ListForRepo returns notifications for a repository
func (s *Service) ListForRepo(ctx context.Context, userID int64, owner, repo string, opts *ListOpts) ([]*Notification, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 50}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 50
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	notifications, err := s.store.ListForRepo(ctx, userID, r.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, n := range notifications {
		s.populateURLs(n)
	}
	return notifications, nil
}

// MarkAsRead marks all notifications as read
func (s *Service) MarkAsRead(ctx context.Context, userID int64, lastReadAt time.Time) error {
	if lastReadAt.IsZero() {
		lastReadAt = time.Now()
	}
	return s.store.MarkAsRead(ctx, userID, lastReadAt)
}

// MarkRepoAsRead marks all notifications for a repo as read
func (s *Service) MarkRepoAsRead(ctx context.Context, userID int64, owner, repo string, lastReadAt time.Time) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	if lastReadAt.IsZero() {
		lastReadAt = time.Now()
	}
	return s.store.MarkRepoAsRead(ctx, userID, r.ID, lastReadAt)
}

// GetThread retrieves a notification thread
func (s *Service) GetThread(ctx context.Context, userID int64, threadID string) (*Notification, error) {
	n, err := s.store.GetByID(ctx, threadID, userID)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, ErrNotFound
	}

	s.populateURLs(n)
	return n, nil
}

// MarkThreadAsRead marks a thread as read
func (s *Service) MarkThreadAsRead(ctx context.Context, userID int64, threadID string) error {
	n, err := s.store.GetByID(ctx, threadID, userID)
	if err != nil {
		return err
	}
	if n == nil {
		return ErrNotFound
	}

	return s.store.MarkThreadAsRead(ctx, threadID, userID)
}

// MarkThreadAsDone marks a thread as done (removes it)
func (s *Service) MarkThreadAsDone(ctx context.Context, userID int64, threadID string) error {
	n, err := s.store.GetByID(ctx, threadID, userID)
	if err != nil {
		return err
	}
	if n == nil {
		return ErrNotFound
	}

	return s.store.Delete(ctx, threadID, userID)
}

// GetThreadSubscription returns the thread subscription
func (s *Service) GetThreadSubscription(ctx context.Context, userID int64, threadID string) (*ThreadSubscription, error) {
	n, err := s.store.GetByID(ctx, threadID, userID)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, ErrNotFound
	}

	sub, err := s.store.GetSubscription(ctx, threadID, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		// Return default subscription
		sub = &ThreadSubscription{
			Subscribed: true,
			Ignored:    false,
			CreatedAt:  n.UpdatedAt,
		}
	}

	sub.URL = fmt.Sprintf("%s/api/v3/notifications/threads/%s/subscription", s.baseURL, threadID)
	sub.ThreadURL = fmt.Sprintf("%s/api/v3/notifications/threads/%s", s.baseURL, threadID)
	return sub, nil
}

// SetThreadSubscription sets the thread subscription
func (s *Service) SetThreadSubscription(ctx context.Context, userID int64, threadID string, ignored bool) (*ThreadSubscription, error) {
	n, err := s.store.GetByID(ctx, threadID, userID)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, ErrNotFound
	}

	if err := s.store.SetSubscription(ctx, threadID, userID, ignored); err != nil {
		return nil, err
	}

	return s.GetThreadSubscription(ctx, userID, threadID)
}

// DeleteThreadSubscription removes the thread subscription
func (s *Service) DeleteThreadSubscription(ctx context.Context, userID int64, threadID string) error {
	n, err := s.store.GetByID(ctx, threadID, userID)
	if err != nil {
		return err
	}
	if n == nil {
		return ErrNotFound
	}

	return s.store.DeleteSubscription(ctx, threadID, userID)
}

// Create creates a notification (internal use)
func (s *Service) Create(ctx context.Context, userID, repoID int64, reason, subjectType, subjectTitle, subjectURL string) (*Notification, error) {
	// Get repo info
	r, err := s.repoStore.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	now := time.Now()
	n := &Notification{
		ID:        fmt.Sprintf("%d", now.UnixNano()),
		Unread:    true,
		Reason:    reason,
		UpdatedAt: now,
		Subject: &Subject{
			Title: subjectTitle,
			URL:   subjectURL,
			Type:  subjectType,
		},
		Repository: &Repository{
			ID:       r.ID,
			Name:     r.Name,
			FullName: r.FullName,
			Private:  r.Private,
		},
	}

	if err := s.store.Create(ctx, n, userID); err != nil {
		return nil, err
	}

	s.populateURLs(n)
	return n, nil
}

// populateURLs fills in the URL fields for a notification
func (s *Service) populateURLs(n *Notification) {
	n.URL = fmt.Sprintf("%s/api/v3/notifications/threads/%s", s.baseURL, n.ID)
	n.SubscriptionURL = fmt.Sprintf("%s/api/v3/notifications/threads/%s/subscription", s.baseURL, n.ID)
	if n.Repository != nil {
		n.Repository.URL = fmt.Sprintf("%s/api/v3/repos/%s", s.baseURL, n.Repository.FullName)
		n.Repository.HTMLURL = fmt.Sprintf("%s/%s", s.baseURL, n.Repository.FullName)
	}
}
