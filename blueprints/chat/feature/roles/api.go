// Package roles provides role and permission management.
package roles

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("role not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// Permissions bit flags
type Permissions uint64

const (
	PermissionViewChannel     Permissions = 1 << 0
	PermissionSendMessages    Permissions = 1 << 1
	PermissionManageMessages  Permissions = 1 << 2
	PermissionManageChannels  Permissions = 1 << 3
	PermissionManageServer    Permissions = 1 << 4
	PermissionKickMembers     Permissions = 1 << 5
	PermissionBanMembers      Permissions = 1 << 6
	PermissionManageRoles     Permissions = 1 << 7
	PermissionMentionEveryone Permissions = 1 << 8
	PermissionAddReactions    Permissions = 1 << 9
	PermissionAttachFiles     Permissions = 1 << 10
	PermissionCreateInvite    Permissions = 1 << 11
	PermissionEmbedLinks      Permissions = 1 << 12
	PermissionUseExternalEmoji Permissions = 1 << 13
	PermissionChangeNickname  Permissions = 1 << 14
	PermissionManageNicknames Permissions = 1 << 15
	PermissionManageEmoji     Permissions = 1 << 16
	PermissionManageWebhooks  Permissions = 1 << 17
	PermissionReadHistory     Permissions = 1 << 18
	PermissionVoiceConnect    Permissions = 1 << 19
	PermissionVoiceSpeak      Permissions = 1 << 20
	PermissionVoiceMute       Permissions = 1 << 21
	PermissionVoiceDeafen     Permissions = 1 << 22
	PermissionVoiceMove       Permissions = 1 << 23
	PermissionAdministrator   Permissions = 1 << 31
)

// Has checks if permission is set.
func (p Permissions) Has(perm Permissions) bool {
	return p&perm == perm || p&PermissionAdministrator == PermissionAdministrator
}

// Role represents a server role.
type Role struct {
	ID            string      `json:"id"`
	ServerID      string      `json:"server_id"`
	Name          string      `json:"name"`
	Color         int         `json:"color"`
	Position      int         `json:"position"`
	Permissions   Permissions `json:"permissions"`
	IsDefault     bool        `json:"is_default"`
	IsHoisted     bool        `json:"is_hoisted"`
	IsMentionable bool        `json:"is_mentionable"`
	IconURL       string      `json:"icon_url,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
}

// ChannelPermission represents a permission override for a channel.
type ChannelPermission struct {
	ChannelID  string      `json:"channel_id"`
	TargetID   string      `json:"target_id"`
	TargetType string      `json:"target_type"` // "role" or "member"
	Allow      Permissions `json:"allow"`
	Deny       Permissions `json:"deny"`
}

// CreateIn contains input for creating a role.
type CreateIn struct {
	Name          string      `json:"name"`
	Color         int         `json:"color,omitempty"`
	Permissions   Permissions `json:"permissions,omitempty"`
	IsHoisted     bool        `json:"is_hoisted,omitempty"`
	IsMentionable bool        `json:"is_mentionable,omitempty"`
}

// UpdateIn contains input for updating a role.
type UpdateIn struct {
	Name          *string      `json:"name,omitempty"`
	Color         *int         `json:"color,omitempty"`
	Position      *int         `json:"position,omitempty"`
	Permissions   *Permissions `json:"permissions,omitempty"`
	IsHoisted     *bool        `json:"is_hoisted,omitempty"`
	IsMentionable *bool        `json:"is_mentionable,omitempty"`
	IconURL       *string      `json:"icon_url,omitempty"`
}

// API defines the roles service contract.
type API interface {
	Create(ctx context.Context, serverID string, in *CreateIn) (*Role, error)
	GetByID(ctx context.Context, id string) (*Role, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Role, error)
	Update(ctx context.Context, id string, in *UpdateIn) (*Role, error)
	Delete(ctx context.Context, id string) error
	ListByServer(ctx context.Context, serverID string) ([]*Role, error)
	GetDefaultRole(ctx context.Context, serverID string) (*Role, error)
	UpdatePositions(ctx context.Context, serverID string, positions map[string]int) error
	ComputePermissions(ctx context.Context, serverID, userID, channelID string) (Permissions, error)
	SetChannelPermission(ctx context.Context, cp *ChannelPermission) error
	GetChannelPermissions(ctx context.Context, channelID string) ([]*ChannelPermission, error)
	DeleteChannelPermission(ctx context.Context, channelID, targetID string) error
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, r *Role) error
	GetByID(ctx context.Context, id string) (*Role, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Role, error)
	Update(ctx context.Context, id string, in *UpdateIn) error
	Delete(ctx context.Context, id string) error
	ListByServer(ctx context.Context, serverID string) ([]*Role, error)
	GetDefaultRole(ctx context.Context, serverID string) (*Role, error)
	UpdatePositions(ctx context.Context, serverID string, positions map[string]int) error
	CreateDefaultRole(ctx context.Context, serverID string) (*Role, error)
	InsertChannelPermission(ctx context.Context, cp *ChannelPermission) error
	GetChannelPermissions(ctx context.Context, channelID string) ([]*ChannelPermission, error)
	DeleteChannelPermission(ctx context.Context, channelID, targetID string) error
}
