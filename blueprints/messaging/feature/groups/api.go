// Package groups provides group-specific management.
package groups

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound        = errors.New("group not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrNotAdmin        = errors.New("admin privileges required")
	ErrInviteExpired   = errors.New("invite link expired")
	ErrInviteMaxUses   = errors.New("invite link max uses reached")
	ErrAlreadyMember   = errors.New("already a member")
	ErrMemberLimitReached = errors.New("member limit reached")
)

// Group represents group-specific settings.
type Group struct {
	ChatID                string     `json:"chat_id"`
	InviteLink            string     `json:"invite_link,omitempty"`
	InviteLinkExpiresAt   *time.Time `json:"invite_link_expires_at,omitempty"`
	InviteLinkCreatedBy   string     `json:"invite_link_created_by,omitempty"`
	MemberCount           int        `json:"member_count"`
	MaxMembers            int        `json:"max_members"`
	OnlyAdminsCanSend     bool       `json:"only_admins_can_send"`
	OnlyAdminsCanEdit     bool       `json:"only_admins_can_edit"`
	DisappearingMessagesTTL int      `json:"disappearing_messages_ttl,omitempty"` // seconds
	CreatedAt             time.Time  `json:"created_at"`
}

// Invite represents a group invite.
type Invite struct {
	Code      string     `json:"code"`
	ChatID    string     `json:"chat_id"`
	CreatedBy string     `json:"created_by"`
	MaxUses   int        `json:"max_uses"`
	Uses      int        `json:"uses"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// UpdateIn contains input for updating group settings.
type UpdateIn struct {
	OnlyAdminsCanSend     *bool `json:"only_admins_can_send,omitempty"`
	OnlyAdminsCanEdit     *bool `json:"only_admins_can_edit,omitempty"`
	DisappearingMessagesTTL *int `json:"disappearing_messages_ttl,omitempty"`
	MaxMembers            *int  `json:"max_members,omitempty"`
}

// CreateInviteIn contains input for creating an invite.
type CreateInviteIn struct {
	MaxUses   int `json:"max_uses,omitempty"`   // 0 = unlimited
	ExpiresIn int `json:"expires_in,omitempty"` // seconds, 0 = never
}

// API defines the groups service contract.
type API interface {
	Create(ctx context.Context, chatID string) (*Group, error)
	GetByChatID(ctx context.Context, chatID string) (*Group, error)
	Update(ctx context.Context, chatID, userID string, in *UpdateIn) (*Group, error)
	Delete(ctx context.Context, chatID string) error

	// Invites
	CreateInvite(ctx context.Context, chatID, userID string, in *CreateInviteIn) (*Invite, error)
	GetInvite(ctx context.Context, code string) (*Invite, error)
	RevokeInvite(ctx context.Context, chatID, userID string) error
	JoinByInvite(ctx context.Context, code, userID string) (*Group, error)

	// Member management
	PromoteToAdmin(ctx context.Context, chatID, userID, targetUserID string) error
	DemoteFromAdmin(ctx context.Context, chatID, userID, targetUserID string) error
	IsAdmin(ctx context.Context, chatID, userID string) (bool, error)
	IsOwner(ctx context.Context, chatID, userID string) (bool, error)
	TransferOwnership(ctx context.Context, chatID, userID, newOwnerID string) error

	// Member count
	IncrementMemberCount(ctx context.Context, chatID string) error
	DecrementMemberCount(ctx context.Context, chatID string) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, g *Group) error
	GetByChatID(ctx context.Context, chatID string) (*Group, error)
	Update(ctx context.Context, chatID string, in *UpdateIn) error
	Delete(ctx context.Context, chatID string) error

	// Invites
	InsertInvite(ctx context.Context, inv *Invite) error
	GetInvite(ctx context.Context, code string) (*Invite, error)
	UpdateInviteLink(ctx context.Context, chatID, link string, expiresAt *time.Time, createdBy string) error
	IncrementInviteUses(ctx context.Context, code string) error
	DeleteInvite(ctx context.Context, code string) error
	DeleteInvitesByChatID(ctx context.Context, chatID string) error

	// Member count
	IncrementMemberCount(ctx context.Context, chatID string) error
	DecrementMemberCount(ctx context.Context, chatID string) error
}
