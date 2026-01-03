package members

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/go-mizu/blueprints/workspace/feature/users"
	"github.com/go-mizu/blueprints/workspace/pkg/ulid"
)

var (
	ErrMemberExists    = errors.New("user is already a member")
	ErrMemberNotFound  = errors.New("member not found")
	ErrInviteNotFound  = errors.New("invite not found")
	ErrInviteExpired   = errors.New("invite has expired")
	ErrCannotRemoveOwner = errors.New("cannot remove workspace owner")
)

// Service implements the members API.
type Service struct {
	store Store
	users users.API
}

// NewService creates a new members service.
func NewService(store Store, users users.API) *Service {
	return &Service{store: store, users: users}
}

// Add adds a user to a workspace.
func (s *Service) Add(ctx context.Context, workspaceID, userID string, role Role, inviterID string) (*Member, error) {
	// Check if already a member
	existing, _ := s.store.GetByWorkspaceAndUser(ctx, workspaceID, userID)
	if existing != nil {
		return nil, ErrMemberExists
	}

	member := &Member{
		ID:          ulid.New(),
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
		JoinedAt:    time.Now(),
		InvitedBy:   inviterID,
	}

	if err := s.store.Create(ctx, member); err != nil {
		return nil, err
	}

	return s.enrichMember(ctx, member)
}

// GetByID retrieves a member by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Member, error) {
	member, err := s.store.GetByID(ctx, id)
	if err != nil {
		return nil, ErrMemberNotFound
	}
	return s.enrichMember(ctx, member)
}

// GetByWorkspaceAndUser retrieves a member by workspace and user IDs.
func (s *Service) GetByWorkspaceAndUser(ctx context.Context, workspaceID, userID string) (*Member, error) {
	member, err := s.store.GetByWorkspaceAndUser(ctx, workspaceID, userID)
	if err != nil {
		return nil, ErrMemberNotFound
	}
	return s.enrichMember(ctx, member)
}

// List lists all members of a workspace.
func (s *Service) List(ctx context.Context, workspaceID string) ([]*Member, error) {
	members, err := s.store.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	return s.enrichMembers(ctx, members)
}

// UpdateRole updates a member's role.
func (s *Service) UpdateRole(ctx context.Context, id string, role Role) error {
	return s.store.UpdateRole(ctx, id, role)
}

// Remove removes a member from a workspace.
func (s *Service) Remove(ctx context.Context, id string) error {
	member, err := s.store.GetByID(ctx, id)
	if err != nil {
		return ErrMemberNotFound
	}
	if member.Role == RoleOwner {
		return ErrCannotRemoveOwner
	}
	return s.store.Delete(ctx, id)
}

// Invite creates an invitation to join a workspace.
func (s *Service) Invite(ctx context.Context, workspaceID, email string, role Role, inviterID string) (*Invite, error) {
	token := generateToken()

	invite := &Invite{
		ID:          ulid.New(),
		WorkspaceID: workspaceID,
		Email:       email,
		Role:        role,
		Token:       token,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedBy:   inviterID,
		CreatedAt:   time.Now(),
	}

	if err := s.store.CreateInvite(ctx, invite); err != nil {
		return nil, err
	}

	return invite, nil
}

// GetInvite retrieves an invitation by token.
func (s *Service) GetInvite(ctx context.Context, token string) (*Invite, error) {
	invite, err := s.store.GetInviteByToken(ctx, token)
	if err != nil {
		return nil, ErrInviteNotFound
	}
	if time.Now().After(invite.ExpiresAt) {
		return nil, ErrInviteExpired
	}
	return invite, nil
}

// AcceptInvite accepts an invitation and adds the user as a member.
func (s *Service) AcceptInvite(ctx context.Context, token string, userID string) (*Member, error) {
	invite, err := s.GetInvite(ctx, token)
	if err != nil {
		return nil, err
	}

	// Add as member
	member, err := s.Add(ctx, invite.WorkspaceID, userID, invite.Role, invite.CreatedBy)
	if err != nil {
		return nil, err
	}

	// Delete the invite
	s.store.DeleteInvite(ctx, invite.ID)

	return member, nil
}

// RevokeInvite revokes an invitation.
func (s *Service) RevokeInvite(ctx context.Context, id string) error {
	return s.store.DeleteInvite(ctx, id)
}

// ListPendingInvites lists all pending invitations for a workspace.
func (s *Service) ListPendingInvites(ctx context.Context, workspaceID string) ([]*Invite, error) {
	return s.store.ListPendingInvites(ctx, workspaceID)
}

// enrichMember adds user data to a member.
func (s *Service) enrichMember(ctx context.Context, m *Member) (*Member, error) {
	user, _ := s.users.GetByID(ctx, m.UserID)
	m.User = user
	return m, nil
}

// enrichMembers adds user data to multiple members.
func (s *Service) enrichMembers(ctx context.Context, members []*Member) ([]*Member, error) {
	if len(members) == 0 {
		return members, nil
	}

	// Collect user IDs
	userIDs := make([]string, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID
	}

	// Batch fetch users
	usersMap, _ := s.users.GetByIDs(ctx, userIDs)

	// Attach users to members
	for _, m := range members {
		m.User = usersMap[m.UserID]
	}

	return members, nil
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
