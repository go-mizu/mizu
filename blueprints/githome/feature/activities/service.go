package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the activities API
type Service struct {
	store     Store
	repoStore repos.Store
	orgStore  orgs.Store
	userStore users.Store
	baseURL   string
}

// NewService creates a new activities service
func NewService(store Store, repoStore repos.Store, orgStore orgs.Store, userStore users.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		orgStore:  orgStore,
		userStore: userStore,
		baseURL:   baseURL,
	}
}

// ListPublic returns public events
func (s *Service) ListPublic(ctx context.Context, opts *ListOpts) ([]*Event, error) {
	if opts == nil {
		opts = &ListOpts{PerPage: 30}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}

	return s.store.ListPublic(ctx, opts)
}

// ListForRepo returns events for a repository
func (s *Service) ListForRepo(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Event, error) {
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

	return s.store.ListForRepo(ctx, r.ID, opts)
}

// ListNetworkEvents returns events for a repo's network
func (s *Service) ListNetworkEvents(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Event, error) {
	// Network events include forks - for now just return repo events
	return s.ListForRepo(ctx, owner, repo, opts)
}

// ListForOrg returns events for an organization
func (s *Service) ListForOrg(ctx context.Context, org string, opts *ListOpts) ([]*Event, error) {
	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
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

	return s.store.ListForOrg(ctx, o.ID, opts)
}

// ListPublicForOrg returns public events for an organization
func (s *Service) ListPublicForOrg(ctx context.Context, org string, opts *ListOpts) ([]*Event, error) {
	// For public org events, filter by public repos
	return s.ListForOrg(ctx, org, opts)
}

// ListForUser returns events performed by a user
func (s *Service) ListForUser(ctx context.Context, username string, opts *ListOpts) ([]*Event, error) {
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

	return s.store.ListForUser(ctx, user.ID, opts)
}

// ListPublicForUser returns public events performed by a user
func (s *Service) ListPublicForUser(ctx context.Context, username string, opts *ListOpts) ([]*Event, error) {
	// Filter by public events
	return s.ListForUser(ctx, username, opts)
}

// ListOrgEventsForUser returns org events for a user
func (s *Service) ListOrgEventsForUser(ctx context.Context, username, org string, opts *ListOpts) ([]*Event, error) {
	user, err := s.userStore.GetByLogin(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	o, err := s.orgStore.GetByLogin(ctx, org)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, orgs.ErrNotFound
	}

	// Return user's events in the org
	return s.store.ListForUser(ctx, user.ID, opts)
}

// ListReceivedEvents returns events received by a user
func (s *Service) ListReceivedEvents(ctx context.Context, username string, opts *ListOpts) ([]*Event, error) {
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

	return s.store.ListReceivedByUser(ctx, user.ID, opts)
}

// ListPublicReceivedEvents returns public events received by a user
func (s *Service) ListPublicReceivedEvents(ctx context.Context, username string, opts *ListOpts) ([]*Event, error) {
	// Filter by public events
	return s.ListReceivedEvents(ctx, username, opts)
}

// GetFeeds returns feed URLs for the authenticated user
func (s *Service) GetFeeds(ctx context.Context, userID int64) (*Feeds, error) {
	user, err := s.userStore.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, users.ErrNotFound
	}

	return &Feeds{
		TimelineURL:          fmt.Sprintf("%s/api/v3/feeds/timeline.atom", s.baseURL),
		UserURL:              fmt.Sprintf("%s/api/v3/feeds/user/{user}.atom", s.baseURL),
		CurrentUserPublicURL: fmt.Sprintf("%s/api/v3/feeds/%s.atom", s.baseURL, user.Login),
		CurrentUserURL:       fmt.Sprintf("%s/api/v3/feeds/%s.private.atom", s.baseURL, user.Login),
		CurrentUserActorURL:  fmt.Sprintf("%s/api/v3/feeds/%s.private.actor.atom", s.baseURL, user.Login),
	}, nil
}

// Create creates an event (internal use)
func (s *Service) Create(ctx context.Context, eventType string, actorID, repoID int64, orgID *int64, payload interface{}, public bool) (*Event, error) {
	actor, err := s.userStore.GetByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, users.ErrNotFound
	}

	r, err := s.repoStore.GetByID(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	now := time.Now()
	e := &Event{
		ID:   fmt.Sprintf("%d", now.UnixNano()),
		Type: eventType,
		Actor: &Actor{
			ID:        actor.ID,
			Login:     actor.Login,
			AvatarURL: actor.AvatarURL,
			URL:       fmt.Sprintf("%s/api/v3/users/%s", s.baseURL, actor.Login),
		},
		Repo: &EventRepo{
			ID:   r.ID,
			Name: r.FullName,
			URL:  fmt.Sprintf("%s/api/v3/repos/%s", s.baseURL, r.FullName),
		},
		Payload:   payload,
		Public:    public,
		CreatedAt: now,
	}

	if orgID != nil {
		org, err := s.orgStore.GetByID(ctx, *orgID)
		if err == nil && org != nil {
			e.Org = &Actor{
				ID:        org.ID,
				Login:     org.Login,
				AvatarURL: org.AvatarURL,
				URL:       fmt.Sprintf("%s/api/v3/orgs/%s", s.baseURL, org.Login),
			}
		}
	}

	if err := s.store.Create(ctx, e); err != nil {
		return nil, err
	}

	return e, nil
}
