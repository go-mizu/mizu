package watches

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the watches API
type Service struct {
	store     Store
	repoStore repos.Store
	userStore users.Store
	baseURL   string
}

// NewService creates a new watches service
func NewService(store Store, repoStore repos.Store, userStore users.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		userStore: userStore,
		baseURL:   baseURL,
	}
}

// ListWatchers returns users watching a repo
func (s *Service) ListWatchers(ctx context.Context, owner, repo string, opts *ListOpts) ([]*users.SimpleUser, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListWatchers(ctx, r.ID, opts)
}

// ListForAuthenticatedUser returns watched repos for authenticated user
func (s *Service) ListForAuthenticatedUser(ctx context.Context, userID int64, opts *ListOpts) ([]*Repository, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListWatchedRepos(ctx, userID, opts)
}

// ListForUser returns watched repos for a user
func (s *Service) ListForUser(ctx context.Context, username string, opts *ListOpts) ([]*Repository, error) {
	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListWatchedRepos(ctx, user.ID, opts)
}

// GetSubscription returns the user's subscription for a repo
func (s *Service) GetSubscription(ctx context.Context, userID int64, owner, repo string) (*Subscription, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	sub, err := s.store.Get(ctx, userID, r.ID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrNotFound
	}

	s.populateURLs(sub, owner, repo)
	return sub, nil
}

// SetSubscription sets the user's subscription for a repo
func (s *Service) SetSubscription(ctx context.Context, userID int64, owner, repo string, subscribed, ignored bool) (*Subscription, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Check if subscription exists
	existing, err := s.store.Get(ctx, userID, r.ID)
	if err != nil {
		return nil, err
	}

	wasSubscribed := existing != nil && existing.Subscribed

	if existing != nil {
		if err := s.store.Update(ctx, userID, r.ID, subscribed, ignored); err != nil {
			return nil, err
		}
	} else {
		if err := s.store.Create(ctx, userID, r.ID, subscribed, ignored); err != nil {
			return nil, err
		}
	}

	// Update watchers count
	if subscribed && !wasSubscribed {
		_ = s.repoStore.IncrementWatchers(ctx, r.ID, 1)
	} else if !subscribed && wasSubscribed {
		_ = s.repoStore.IncrementWatchers(ctx, r.ID, -1)
	}

	sub := &Subscription{
		Subscribed: subscribed,
		Ignored:    ignored,
		CreatedAt:  time.Now(),
	}
	s.populateURLs(sub, owner, repo)
	return sub, nil
}

// DeleteSubscription removes the user's subscription for a repo
func (s *Service) DeleteSubscription(ctx context.Context, userID int64, owner, repo string) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	// Check if subscription exists
	existing, err := s.store.Get(ctx, userID, r.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil // Not subscribed, no-op
	}

	if err := s.store.Delete(ctx, userID, r.ID); err != nil {
		return err
	}

	// Decrement watchers count if was subscribed
	if existing.Subscribed {
		return s.repoStore.IncrementWatchers(ctx, r.ID, -1)
	}
	return nil
}

// populateURLs fills in the URL fields for a subscription
func (s *Service) populateURLs(sub *Subscription, owner, repo string) {
	sub.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/subscription", s.baseURL, owner, repo)
	sub.RepositoryURL = fmt.Sprintf("%s/api/v3/repos/%s/%s", s.baseURL, owner, repo)
}
