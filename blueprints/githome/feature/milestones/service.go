package milestones

import (
	"context"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/githome/pkg/ulid"
)

// Service implements the milestones API
type Service struct {
	store Store
}

// NewService creates a new milestones service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new milestone
func (s *Service) Create(ctx context.Context, repoID string, in *CreateIn) (*Milestone, error) {
	if in.Title == "" {
		return nil, ErrMissingTitle
	}

	// Get next milestone number
	number, err := s.store.GetNextNumber(ctx, repoID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	milestone := &Milestone{
		ID:          ulid.New(),
		RepoID:      repoID,
		Number:      number,
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		State:       "open",
		DueDate:     in.DueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Create(ctx, milestone); err != nil {
		return nil, err
	}

	return milestone, nil
}

// GetByID retrieves a milestone by ID
func (s *Service) GetByID(ctx context.Context, id string) (*Milestone, error) {
	milestone, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if milestone == nil {
		return nil, ErrNotFound
	}
	return milestone, nil
}

// GetByNumber retrieves a milestone by repository ID and number
func (s *Service) GetByNumber(ctx context.Context, repoID string, number int) (*Milestone, error) {
	milestone, err := s.store.GetByNumber(ctx, repoID, number)
	if err != nil {
		return nil, err
	}
	if milestone == nil {
		return nil, ErrNotFound
	}
	return milestone, nil
}

// Update updates a milestone
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Milestone, error) {
	milestone, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if milestone == nil {
		return nil, ErrNotFound
	}

	if in.Title != nil {
		milestone.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		milestone.Description = *in.Description
	}
	if in.State != nil {
		newState := *in.State
		if newState == "closed" && milestone.State == "open" {
			now := time.Now()
			milestone.ClosedAt = &now
		} else if newState == "open" && milestone.State == "closed" {
			milestone.ClosedAt = nil
		}
		milestone.State = newState
	}
	if in.DueDate != nil {
		milestone.DueDate = in.DueDate
	}

	milestone.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, milestone); err != nil {
		return nil, err
	}

	return milestone, nil
}

// Delete deletes a milestone
func (s *Service) Delete(ctx context.Context, id string) error {
	milestone, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if milestone == nil {
		return ErrNotFound
	}
	return s.store.Delete(ctx, id)
}

// List lists milestones for a repository
func (s *Service) List(ctx context.Context, repoID string, opts *ListOpts) ([]*Milestone, error) {
	state := "all"
	if opts != nil && opts.State != "" {
		state = opts.State
	}
	return s.store.List(ctx, repoID, state)
}

// Close closes a milestone
func (s *Service) Close(ctx context.Context, id string) error {
	milestone, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if milestone == nil {
		return ErrNotFound
	}

	if milestone.State == "closed" {
		return ErrAlreadyClosed
	}

	now := time.Now()
	milestone.State = "closed"
	milestone.ClosedAt = &now
	milestone.UpdatedAt = now

	return s.store.Update(ctx, milestone)
}

// Reopen reopens a milestone
func (s *Service) Reopen(ctx context.Context, id string) error {
	milestone, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if milestone == nil {
		return ErrNotFound
	}

	if milestone.State == "open" {
		return ErrAlreadyOpen
	}

	milestone.State = "open"
	milestone.ClosedAt = nil
	milestone.UpdatedAt = time.Now()

	return s.store.Update(ctx, milestone)
}

// IncrementOpenIssues increments the open issues count
func (s *Service) IncrementOpenIssues(ctx context.Context, id string) error {
	milestone, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if milestone == nil {
		return ErrNotFound
	}

	milestone.OpenIssues++
	milestone.UpdatedAt = time.Now()
	return s.store.Update(ctx, milestone)
}

// DecrementOpenIssues decrements the open issues count
func (s *Service) DecrementOpenIssues(ctx context.Context, id string) error {
	milestone, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if milestone == nil {
		return ErrNotFound
	}

	if milestone.OpenIssues > 0 {
		milestone.OpenIssues--
	}
	milestone.UpdatedAt = time.Now()
	return s.store.Update(ctx, milestone)
}

// IncrementClosedIssues increments the closed issues count
func (s *Service) IncrementClosedIssues(ctx context.Context, id string) error {
	milestone, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if milestone == nil {
		return ErrNotFound
	}

	milestone.ClosedIssues++
	milestone.UpdatedAt = time.Now()
	return s.store.Update(ctx, milestone)
}

// DecrementClosedIssues decrements the closed issues count
func (s *Service) DecrementClosedIssues(ctx context.Context, id string) error {
	milestone, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if milestone == nil {
		return ErrNotFound
	}

	if milestone.ClosedIssues > 0 {
		milestone.ClosedIssues--
	}
	milestone.UpdatedAt = time.Now()
	return s.store.Update(ctx, milestone)
}
