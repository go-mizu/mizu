package cycles

import (
	"context"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrNotFound      = errors.New("cycle not found")
	ErrInvalidStatus = errors.New("invalid cycle status")
)

// Service implements the cycles API.
type Service struct {
	store Store
}

// NewService creates a new cycles service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, teamID string, in *CreateIn) (*Cycle, error) {
	// Get next cycle number for team
	number, err := s.store.GetNextNumber(ctx, teamID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	cycle := &Cycle{
		ID:        ulid.New(),
		TeamID:    teamID,
		Number:    number,
		Name:      in.Name,
		Status:    StatusPlanning,
		StartDate: in.StartDate,
		EndDate:   in.EndDate,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Create(ctx, cycle); err != nil {
		return nil, err
	}

	return cycle, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Cycle, error) {
	c, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *Service) GetByNumber(ctx context.Context, teamID string, number int) (*Cycle, error) {
	c, err := s.store.GetByNumber(ctx, teamID, number)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *Service) ListByTeam(ctx context.Context, teamID string) ([]*Cycle, error) {
	return s.store.ListByTeam(ctx, teamID)
}

func (s *Service) GetActive(ctx context.Context, teamID string) (*Cycle, error) {
	c, err := s.store.GetActive(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrNotFound
	}
	return c, nil
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Cycle, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) UpdateStatus(ctx context.Context, id, status string) error {
	// Validate status
	switch status {
	case StatusPlanning, StatusActive, StatusCompleted:
		// Valid
	default:
		return ErrInvalidStatus
	}

	return s.store.UpdateStatus(ctx, id, status)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}
