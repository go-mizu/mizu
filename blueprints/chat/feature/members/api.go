// Package members provides server membership management.
package members

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound     = errors.New("member not found")
	ErrAlreadyMember = errors.New("already a member")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrBanned       = errors.New("user is banned")
)

// Member represents a server member.
type Member struct {
	ServerID   string    `json:"server_id"`
	UserID     string    `json:"user_id"`
	Nickname   string    `json:"nickname,omitempty"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	RoleIDs    []string  `json:"role_ids,omitempty"`
	IsMuted    bool      `json:"is_muted"`
	IsDeafened bool      `json:"is_deafened"`
	JoinedAt   time.Time `json:"joined_at"`

	// Populated from joins
	User any `json:"user,omitempty"`
}

// UpdateIn contains input for updating a member.
type UpdateIn struct {
	Nickname   *string `json:"nickname,omitempty"`
	AvatarURL  *string `json:"avatar_url,omitempty"`
	IsMuted    *bool   `json:"is_muted,omitempty"`
	IsDeafened *bool   `json:"is_deafened,omitempty"`
}

// Ban represents a server ban.
type Ban struct {
	ServerID  string    `json:"server_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason,omitempty"`
	BannedBy  string    `json:"banned_by"`
	CreatedAt time.Time `json:"created_at"`

	// Populated from joins
	User any `json:"user,omitempty"`
}

// API defines the members service contract.
type API interface {
	Join(ctx context.Context, serverID, userID string) (*Member, error)
	Leave(ctx context.Context, serverID, userID string) error
	Get(ctx context.Context, serverID, userID string) (*Member, error)
	Update(ctx context.Context, serverID, userID string, in *UpdateIn) (*Member, error)
	Kick(ctx context.Context, serverID, userID string) error
	List(ctx context.Context, serverID string, limit, offset int) ([]*Member, error)
	Count(ctx context.Context, serverID string) (int, error)
	IsMember(ctx context.Context, serverID, userID string) (bool, error)
	Search(ctx context.Context, serverID, query string, limit int) ([]*Member, error)
	AddRole(ctx context.Context, serverID, userID, roleID string) error
	RemoveRole(ctx context.Context, serverID, userID, roleID string) error
	Ban(ctx context.Context, serverID, userID, bannedBy, reason string) error
	Unban(ctx context.Context, serverID, userID string) error
	IsBanned(ctx context.Context, serverID, userID string) (bool, error)
	ListBans(ctx context.Context, serverID string, limit, offset int) ([]*Ban, error)
}

// Store defines the data access contract.
type Store interface {
	Insert(ctx context.Context, m *Member) error
	Get(ctx context.Context, serverID, userID string) (*Member, error)
	Update(ctx context.Context, serverID, userID string, in *UpdateIn) error
	Delete(ctx context.Context, serverID, userID string) error
	List(ctx context.Context, serverID string, limit, offset int) ([]*Member, error)
	Count(ctx context.Context, serverID string) (int, error)
	IsMember(ctx context.Context, serverID, userID string) (bool, error)
	Search(ctx context.Context, serverID, query string, limit int) ([]*Member, error)
	AddRole(ctx context.Context, serverID, userID, roleID string) error
	RemoveRole(ctx context.Context, serverID, userID, roleID string) error
	ListByRole(ctx context.Context, serverID, roleID string) ([]*Member, error)
	Ban(ctx context.Context, serverID, userID, bannedBy, reason string) error
	Unban(ctx context.Context, serverID, userID string) error
	IsBanned(ctx context.Context, serverID, userID string) (bool, error)
	ListBans(ctx context.Context, serverID string, limit, offset int) ([]*Ban, error)
}
