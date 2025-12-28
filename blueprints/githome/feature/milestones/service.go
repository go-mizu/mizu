package milestones

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
)

// Service implements the milestones API
type Service struct {
	store     Store
	repoStore repos.Store
	userStore users.Store
	baseURL   string
}

// NewService creates a new milestones service
func NewService(store Store, repoStore repos.Store, userStore users.Store, baseURL string) *Service {
	return &Service{
		store:     store,
		repoStore: repoStore,
		userStore: userStore,
		baseURL:   baseURL,
	}
}

// List returns milestones for a repository
func (s *Service) List(ctx context.Context, owner, repo string, opts *ListOpts) ([]*Milestone, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	if opts == nil {
		opts = &ListOpts{PerPage: 30, State: "open"}
	}
	if opts.PerPage == 0 {
		opts.PerPage = 30
	}
	if opts.PerPage > 100 {
		opts.PerPage = 100
	}
	if opts.State == "" {
		opts.State = "open"
	}

	milestones, err := s.store.List(ctx, r.ID, opts)
	if err != nil {
		return nil, err
	}

	for _, m := range milestones {
		s.populateURLs(m, owner, repo)
	}
	return milestones, nil
}

// Get retrieves a milestone by number
func (s *Service) Get(ctx context.Context, owner, repo string, number int) (*Milestone, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	m, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrNotFound
	}

	s.populateURLs(m, owner, repo)
	return m, nil
}

// GetByID retrieves a milestone by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*Milestone, error) {
	m, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrNotFound
	}
	return m, nil
}

// Create creates a new milestone
func (s *Service) Create(ctx context.Context, owner, repo string, creatorID int64, in *CreateIn) (*Milestone, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	// Check if milestone with same title exists
	existing, err := s.store.GetByTitle(ctx, r.ID, in.Title)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrMilestoneExists
	}

	creator, err := s.userStore.GetByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}

	number, err := s.store.NextNumber(ctx, r.ID)
	if err != nil {
		return nil, err
	}

	state := in.State
	if state == "" {
		state = "open"
	}

	now := time.Now()
	m := &Milestone{
		Number:       number,
		State:        state,
		Title:        in.Title,
		Description:  in.Description,
		Creator:      creator.ToSimple(),
		OpenIssues:   0,
		ClosedIssues: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
		DueOn:        in.DueOn,
		RepoID:       r.ID,
		CreatorID:    creatorID,
	}

	if err := s.store.Create(ctx, m); err != nil {
		return nil, err
	}

	s.populateURLs(m, owner, repo)
	return m, nil
}

// Update updates a milestone
func (s *Service) Update(ctx context.Context, owner, repo string, number int, in *UpdateIn) (*Milestone, error) {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, repos.ErrNotFound
	}

	m, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrNotFound
	}

	if err := s.store.Update(ctx, m.ID, in); err != nil {
		return nil, err
	}

	return s.Get(ctx, owner, repo, number)
}

// Delete removes a milestone
func (s *Service) Delete(ctx context.Context, owner, repo string, number int) error {
	r, err := s.repoStore.GetByFullName(ctx, owner, repo)
	if err != nil {
		return err
	}
	if r == nil {
		return repos.ErrNotFound
	}

	m, err := s.store.GetByNumber(ctx, r.ID, number)
	if err != nil {
		return err
	}
	if m == nil {
		return ErrNotFound
	}

	return s.store.Delete(ctx, m.ID)
}

// IncrementOpenIssues adjusts the open issues count
func (s *Service) IncrementOpenIssues(ctx context.Context, id int64, delta int) error {
	return s.store.IncrementOpenIssues(ctx, id, delta)
}

// IncrementClosedIssues adjusts the closed issues count
func (s *Service) IncrementClosedIssues(ctx context.Context, id int64, delta int) error {
	return s.store.IncrementClosedIssues(ctx, id, delta)
}

// populateURLs fills in the URL fields for a milestone
func (s *Service) populateURLs(m *Milestone, owner, repo string) {
	m.NodeID = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("Milestone:%d", m.ID)))
	m.URL = fmt.Sprintf("%s/api/v3/repos/%s/%s/milestones/%d", s.baseURL, owner, repo, m.Number)
	m.HTMLURL = fmt.Sprintf("%s/%s/%s/milestone/%d", s.baseURL, owner, repo, m.Number)
	m.LabelsURL = fmt.Sprintf("%s/api/v3/repos/%s/%s/milestones/%d/labels", s.baseURL, owner, repo, m.Number)
}
