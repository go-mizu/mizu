package roles

import (
	"context"
	"time"

	"github.com/go-mizu/blueprints/chat/pkg/ulid"
)

// MemberRoleGetter is an interface for getting member roles.
type MemberRoleGetter interface {
	GetMemberRoleIDs(ctx context.Context, serverID, userID string) ([]string, error)
}

// Service implements the roles API.
type Service struct {
	store        Store
	memberGetter MemberRoleGetter
}

// NewService creates a new roles service.
func NewService(store Store, memberGetter MemberRoleGetter) *Service {
	return &Service{
		store:        store,
		memberGetter: memberGetter,
	}
}

// Create creates a new role.
func (s *Service) Create(ctx context.Context, serverID string, in *CreateIn) (*Role, error) {
	// Get current roles to determine position
	existing, err := s.store.ListByServer(ctx, serverID)
	if err != nil {
		return nil, err
	}

	role := &Role{
		ID:            ulid.New(),
		ServerID:      serverID,
		Name:          in.Name,
		Color:         in.Color,
		Position:      len(existing),
		Permissions:   in.Permissions,
		IsHoisted:     in.IsHoisted,
		IsMentionable: in.IsMentionable,
		CreatedAt:     time.Now(),
	}

	if err := s.store.Insert(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

// GetByID retrieves a role by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Role, error) {
	return s.store.GetByID(ctx, id)
}

// GetByIDs retrieves multiple roles by IDs.
func (s *Service) GetByIDs(ctx context.Context, ids []string) ([]*Role, error) {
	return s.store.GetByIDs(ctx, ids)
}

// Update updates a role.
func (s *Service) Update(ctx context.Context, id string, in *UpdateIn) (*Role, error) {
	if err := s.store.Update(ctx, id, in); err != nil {
		return nil, err
	}
	return s.store.GetByID(ctx, id)
}

// Delete deletes a role.
func (s *Service) Delete(ctx context.Context, id string) error {
	role, err := s.store.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role.IsDefault {
		return ErrForbidden // Cannot delete @everyone role
	}
	return s.store.Delete(ctx, id)
}

// ListByServer lists roles in a server.
func (s *Service) ListByServer(ctx context.Context, serverID string) ([]*Role, error) {
	return s.store.ListByServer(ctx, serverID)
}

// GetDefaultRole gets the @everyone role.
func (s *Service) GetDefaultRole(ctx context.Context, serverID string) (*Role, error) {
	return s.store.GetDefaultRole(ctx, serverID)
}

// UpdatePositions updates role positions.
func (s *Service) UpdatePositions(ctx context.Context, serverID string, positions map[string]int) error {
	return s.store.UpdatePositions(ctx, serverID, positions)
}

// ComputePermissions computes effective permissions for a user in a channel.
func (s *Service) ComputePermissions(ctx context.Context, serverID, userID, channelID string) (Permissions, error) {
	// Get member's role IDs
	roleIDs, err := s.memberGetter.GetMemberRoleIDs(ctx, serverID, userID)
	if err != nil {
		return 0, err
	}

	// Always include default role
	defaultRole, _ := s.store.GetDefaultRole(ctx, serverID)
	if defaultRole != nil {
		roleIDs = append(roleIDs, defaultRole.ID)
	}

	// Get all roles
	roles, err := s.store.GetByIDs(ctx, roleIDs)
	if err != nil {
		return 0, err
	}

	// Compute base permissions from roles
	var perms Permissions
	for _, role := range roles {
		perms |= role.Permissions
	}

	// Admin has all permissions
	if perms.Has(PermissionAdministrator) {
		return ^Permissions(0), nil
	}

	// Apply channel permission overrides
	if channelID != "" {
		overrides, _ := s.store.GetChannelPermissions(ctx, channelID)
		for _, override := range overrides {
			// Check if this override applies to user or their roles
			applies := override.TargetID == userID
			if !applies {
				for _, roleID := range roleIDs {
					if override.TargetID == roleID {
						applies = true
						break
					}
				}
			}

			if applies {
				perms &= ^override.Deny  // Remove denied permissions
				perms |= override.Allow   // Add allowed permissions
			}
		}
	}

	return perms, nil
}

// SetChannelPermission sets a channel permission override.
func (s *Service) SetChannelPermission(ctx context.Context, cp *ChannelPermission) error {
	return s.store.InsertChannelPermission(ctx, cp)
}

// GetChannelPermissions gets permission overrides for a channel.
func (s *Service) GetChannelPermissions(ctx context.Context, channelID string) ([]*ChannelPermission, error) {
	return s.store.GetChannelPermissions(ctx, channelID)
}

// DeleteChannelPermission deletes a permission override.
func (s *Service) DeleteChannelPermission(ctx context.Context, channelID, targetID string) error {
	return s.store.DeleteChannelPermission(ctx, channelID, targetID)
}

// CreateDefaultRole creates the @everyone role for a new server.
func (s *Service) CreateDefaultRole(ctx context.Context, serverID string) (*Role, error) {
	return s.store.CreateDefaultRole(ctx, serverID)
}
