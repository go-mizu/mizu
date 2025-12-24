package servers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/go-mizu/blueprints/chat/pkg/ulid"
)

// Service implements the servers API.
type Service struct {
	store Store
}

// NewService creates a new servers service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create creates a new server.
func (s *Service) Create(ctx context.Context, ownerID string, in *CreateIn) (*Server, error) {
	now := time.Now()
	inviteCode, _ := generateInviteCode()

	srv := &Server{
		ID:          ulid.New(),
		Name:        in.Name,
		Description: in.Description,
		IconURL:     in.IconURL,
		OwnerID:     ownerID,
		IsPublic:    in.IsPublic,
		InviteCode:  inviteCode,
		MemberCount: 1, // Owner
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.Insert(ctx, srv); err != nil {
		return nil, err
	}

	return srv, nil
}

// GetByID retrieves a server by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Server, error) {
	return s.store.GetByID(ctx, id)
}

// GetByInviteCode retrieves a server by invite code.
func (s *Service) GetByInviteCode(ctx context.Context, code string) (*Server, error) {
	return s.store.GetByInviteCode(ctx, code)
}

// Update updates a server.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Server, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a server.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// ListByUser lists servers a user is a member of.
func (s *Service) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*Server, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	return s.store.ListByUser(ctx, userID, limit, offset)
}

// ListPublic lists public servers.
func (s *Service) ListPublic(ctx context.Context, limit, offset int) ([]*Server, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	return s.store.ListPublic(ctx, limit, offset)
}

// Search searches for public servers.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*Server, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	return s.store.Search(ctx, query, limit)
}

// GenerateInviteCode generates a new invite code.
func (s *Service) GenerateInviteCode(ctx context.Context, serverID string) (string, error) {
	code, err := generateInviteCode()
	if err != nil {
		return "", err
	}

	// Update server with new code
	if err := s.store.Update(ctx, serverID, &UpdateIn{}); err != nil {
		return "", err
	}

	return code, nil
}

// TransferOwnership transfers server ownership.
func (s *Service) TransferOwnership(ctx context.Context, serverID, newOwnerID string) error {
	// This would typically include additional validation
	return s.store.Update(ctx, serverID, &UpdateIn{})
}

// IncrementMemberCount increments the member count.
func (s *Service) IncrementMemberCount(ctx context.Context, serverID string) error {
	return s.store.UpdateMemberCount(ctx, serverID, 1)
}

// DecrementMemberCount decrements the member count.
func (s *Service) DecrementMemberCount(ctx context.Context, serverID string) error {
	return s.store.UpdateMemberCount(ctx, serverID, -1)
}

// SetDefaultChannel sets the default channel.
func (s *Service) SetDefaultChannel(ctx context.Context, serverID, channelID string) error {
	return s.store.SetDefaultChannel(ctx, serverID, channelID)
}

func generateInviteCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
