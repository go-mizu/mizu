package groups

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/go-mizu/blueprints/messaging/feature/chats"
)

const defaultMaxMembers = 1024

// Service implements the groups API.
type Service struct {
	store     Store
	chatStore chats.Store
}

// NewService creates a new groups service.
func NewService(store Store, chatStore chats.Store) *Service {
	return &Service{
		store:     store,
		chatStore: chatStore,
	}
}

// Create creates group settings for a chat.
func (s *Service) Create(ctx context.Context, chatID string) (*Group, error) {
	g := &Group{
		ChatID:            chatID,
		MemberCount:       1,
		MaxMembers:        defaultMaxMembers,
		OnlyAdminsCanSend: false,
		OnlyAdminsCanEdit: false,
		CreatedAt:         time.Now(),
	}

	if err := s.store.Insert(ctx, g); err != nil {
		return nil, err
	}

	return g, nil
}

// GetByChatID retrieves group settings by chat ID.
func (s *Service) GetByChatID(ctx context.Context, chatID string) (*Group, error) {
	return s.store.GetByChatID(ctx, chatID)
}

// Update updates group settings.
func (s *Service) Update(ctx context.Context, chatID, userID string, in *UpdateIn) (*Group, error) {
	// Check if user is admin
	isAdmin, err := s.IsAdmin(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, ErrNotAdmin
	}

	if err := s.store.Update(ctx, chatID, in); err != nil {
		return nil, err
	}

	return s.store.GetByChatID(ctx, chatID)
}

// Delete deletes group settings.
func (s *Service) Delete(ctx context.Context, chatID string) error {
	// Delete all invites first
	if err := s.store.DeleteInvitesByChatID(ctx, chatID); err != nil {
		return err
	}
	return s.store.Delete(ctx, chatID)
}

// CreateInvite creates an invite link.
func (s *Service) CreateInvite(ctx context.Context, chatID, userID string, in *CreateInviteIn) (*Invite, error) {
	isAdmin, err := s.IsAdmin(ctx, chatID, userID)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, ErrNotAdmin
	}

	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	inv := &Invite{
		Code:      code,
		ChatID:    chatID,
		CreatedBy: userID,
		MaxUses:   in.MaxUses,
		Uses:      0,
		CreatedAt: now,
	}

	if in.ExpiresIn > 0 {
		expiresAt := now.Add(time.Duration(in.ExpiresIn) * time.Second)
		inv.ExpiresAt = &expiresAt
	}

	if err := s.store.InsertInvite(ctx, inv); err != nil {
		return nil, err
	}

	// Update group with invite link
	if err := s.store.UpdateInviteLink(ctx, chatID, code, inv.ExpiresAt, userID); err != nil {
		return nil, err
	}

	return inv, nil
}

// GetInvite retrieves an invite by code.
func (s *Service) GetInvite(ctx context.Context, code string) (*Invite, error) {
	return s.store.GetInvite(ctx, code)
}

// RevokeInvite revokes the current invite link.
func (s *Service) RevokeInvite(ctx context.Context, chatID, userID string) error {
	isAdmin, err := s.IsAdmin(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrNotAdmin
	}

	return s.store.DeleteInvitesByChatID(ctx, chatID)
}

// JoinByInvite joins a group using an invite code.
func (s *Service) JoinByInvite(ctx context.Context, code, userID string) (*Group, error) {
	inv, err := s.store.GetInvite(ctx, code)
	if err != nil {
		return nil, err
	}

	// Check expiry
	if inv.ExpiresAt != nil && inv.ExpiresAt.Before(time.Now()) {
		return nil, ErrInviteExpired
	}

	// Check max uses
	if inv.MaxUses > 0 && inv.Uses >= inv.MaxUses {
		return nil, ErrInviteMaxUses
	}

	// Check if already member
	isMember, err := s.chatStore.IsParticipant(ctx, inv.ChatID, userID)
	if err != nil {
		return nil, err
	}
	if isMember {
		return nil, ErrAlreadyMember
	}

	// Check member limit
	g, err := s.store.GetByChatID(ctx, inv.ChatID)
	if err != nil {
		return nil, err
	}
	if g.MemberCount >= g.MaxMembers {
		return nil, ErrMemberLimitReached
	}

	// Add participant
	p := &chats.Participant{
		ChatID:   inv.ChatID,
		UserID:   userID,
		Role:     "member",
		JoinedAt: time.Now(),
	}
	if err := s.chatStore.InsertParticipant(ctx, p); err != nil {
		return nil, err
	}

	// Increment counters
	if err := s.store.IncrementInviteUses(ctx, code); err != nil {
		return nil, err
	}
	if err := s.store.IncrementMemberCount(ctx, inv.ChatID); err != nil {
		return nil, err
	}

	return s.store.GetByChatID(ctx, inv.ChatID)
}

// PromoteToAdmin promotes a member to admin.
func (s *Service) PromoteToAdmin(ctx context.Context, chatID, userID, targetUserID string) error {
	isAdmin, err := s.IsAdmin(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !isAdmin {
		return ErrNotAdmin
	}

	return s.chatStore.UpdateParticipantRole(ctx, chatID, targetUserID, "admin")
}

// DemoteFromAdmin demotes an admin to member.
func (s *Service) DemoteFromAdmin(ctx context.Context, chatID, userID, targetUserID string) error {
	isOwner, err := s.IsOwner(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return ErrForbidden
	}

	return s.chatStore.UpdateParticipantRole(ctx, chatID, targetUserID, "member")
}

// IsAdmin checks if a user is an admin.
func (s *Service) IsAdmin(ctx context.Context, chatID, userID string) (bool, error) {
	p, err := s.chatStore.GetParticipant(ctx, chatID, userID)
	if err != nil {
		return false, err
	}
	return p.Role == "admin" || p.Role == "owner", nil
}

// IsOwner checks if a user is the owner.
func (s *Service) IsOwner(ctx context.Context, chatID, userID string) (bool, error) {
	p, err := s.chatStore.GetParticipant(ctx, chatID, userID)
	if err != nil {
		return false, err
	}
	return p.Role == "owner", nil
}

// TransferOwnership transfers group ownership.
func (s *Service) TransferOwnership(ctx context.Context, chatID, userID, newOwnerID string) error {
	isOwner, err := s.IsOwner(ctx, chatID, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return ErrForbidden
	}

	// Demote current owner to admin
	if err := s.chatStore.UpdateParticipantRole(ctx, chatID, userID, "admin"); err != nil {
		return err
	}

	// Promote new owner
	return s.chatStore.UpdateParticipantRole(ctx, chatID, newOwnerID, "owner")
}

// IncrementMemberCount increments the member count.
func (s *Service) IncrementMemberCount(ctx context.Context, chatID string) error {
	return s.store.IncrementMemberCount(ctx, chatID)
}

// DecrementMemberCount decrements the member count.
func (s *Service) DecrementMemberCount(ctx context.Context, chatID string) error {
	return s.store.DecrementMemberCount(ctx, chatID)
}

func generateInviteCode() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:22], nil
}
