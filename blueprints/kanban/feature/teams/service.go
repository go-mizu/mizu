package teams

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-mizu/blueprints/kanban/pkg/ulid"
)

var (
	ErrKeyExists     = errors.New("team key already exists")
	ErrNotFound      = errors.New("team not found")
	ErrMemberExists  = errors.New("member already exists")
	ErrMemberNotFound = errors.New("member not found")
)

// Service implements the teams API.
type Service struct {
	store Store
}

// NewService creates a new teams service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, workspaceID string, in *CreateIn) (*Team, error) {
	key := strings.ToUpper(in.Key)

	// Check if key exists in workspace
	existing, err := s.store.GetByKey(ctx, workspaceID, key)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrKeyExists
	}

	team := &Team{
		ID:          ulid.New(),
		WorkspaceID: workspaceID,
		Key:         key,
		Name:        in.Name,
	}

	if err := s.store.Create(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Team, error) {
	t, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *Service) GetByKey(ctx context.Context, workspaceID, key string) (*Team, error) {
	t, err := s.store.GetByKey(ctx, workspaceID, strings.ToUpper(key))
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *Service) ListByWorkspace(ctx context.Context, workspaceID string) ([]*Team, error) {
	return s.store.ListByWorkspace(ctx, workspaceID)
}

func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Team, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

func (s *Service) AddMember(ctx context.Context, teamID, userID, role string) error {
	// Check if already a member
	existing, err := s.store.GetMember(ctx, teamID, userID)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrMemberExists
	}

	member := &Member{
		TeamID:   teamID,
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	}

	return s.store.AddMember(ctx, member)
}

func (s *Service) GetMember(ctx context.Context, teamID, userID string) (*Member, error) {
	m, err := s.store.GetMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrMemberNotFound
	}
	return m, nil
}

func (s *Service) ListMembers(ctx context.Context, teamID string) ([]*Member, error) {
	return s.store.ListMembers(ctx, teamID)
}

func (s *Service) UpdateMemberRole(ctx context.Context, teamID, userID, role string) error {
	return s.store.UpdateMemberRole(ctx, teamID, userID, role)
}

func (s *Service) RemoveMember(ctx context.Context, teamID, userID string) error {
	return s.store.RemoveMember(ctx, teamID, userID)
}
