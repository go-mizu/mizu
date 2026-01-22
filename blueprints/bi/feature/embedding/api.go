// Package embedding provides dashboard/question embedding functionality.
package embedding

import (
	"context"
	"errors"
	"time"
)

// Errors
var (
	ErrNotFound       = errors.New("resource not found")
	ErrInvalidToken   = errors.New("invalid or expired token")
	ErrEmbeddingDisabled = errors.New("embedding is disabled")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrExpired        = errors.New("token has expired")
)

// ResourceType represents the type of embeddable resource.
type ResourceType string

const (
	ResourceQuestion  ResourceType = "question"
	ResourceDashboard ResourceType = "dashboard"
)

// PublicLink represents a public sharing link.
type PublicLink struct {
	ID           string       `json:"id"`
	UUID         string       `json:"uuid"`
	ResourceType ResourceType `json:"resource_type"`
	ResourceID   string       `json:"resource_id"`
	CreatorID    string       `json:"creator_id"`
	ExpiresAt    *time.Time   `json:"expires_at,omitempty"`
	ViewCount    int64        `json:"view_count"`
	Active       bool         `json:"active"`
	CreatedAt    time.Time    `json:"created_at"`
}

// EmbedToken represents a signed embedding token.
type EmbedToken struct {
	Token        string            `json:"token"`
	ResourceType ResourceType      `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Parameters   map[string]any    `json:"parameters,omitempty"`
	Permissions  *EmbedPermissions `json:"permissions,omitempty"`
	ExpiresAt    time.Time         `json:"expires_at"`
	CreatedAt    time.Time         `json:"created_at"`
}

// EmbedPermissions defines what the embedded view can do.
type EmbedPermissions struct {
	CanDownload    bool `json:"can_download"`
	CanFullscreen  bool `json:"can_fullscreen"`
	CanRefresh     bool `json:"can_refresh"`
	CanFilter      bool `json:"can_filter"`
	ShowBranding   bool `json:"show_branding"`
}

// CreatePublicLinkIn contains input for creating a public link.
type CreatePublicLinkIn struct {
	ResourceType ResourceType `json:"resource_type"`
	ResourceID   string       `json:"resource_id"`
	CreatorID    string       `json:"-"`
	ExpiresIn    *time.Duration `json:"expires_in,omitempty"` // nil = never expires
}

// CreateEmbedTokenIn contains input for creating an embed token.
type CreateEmbedTokenIn struct {
	ResourceType ResourceType      `json:"resource_type"`
	ResourceID   string            `json:"resource_id"`
	Parameters   map[string]any    `json:"parameters,omitempty"`
	Permissions  *EmbedPermissions `json:"permissions,omitempty"`
	ExpiresIn    time.Duration     `json:"expires_in"` // Required
}

// EmbedSettings represents embedding configuration.
type EmbedSettings struct {
	Enabled         bool   `json:"enabled"`
	SecretKey       string `json:"secret_key,omitempty"`
	AllowedOrigins  []string `json:"allowed_origins,omitempty"`
	DefaultExpiry   time.Duration `json:"default_expiry,omitempty"`
}

// API defines the Embedding service contract.
type API interface {
	// CreatePublicLink creates a public sharing link.
	CreatePublicLink(ctx context.Context, in *CreatePublicLinkIn) (*PublicLink, error)

	// GetPublicLink returns a public link by UUID.
	GetPublicLink(ctx context.Context, uuid string) (*PublicLink, error)

	// ListPublicLinks returns all public links for a resource.
	ListPublicLinks(ctx context.Context, resourceType ResourceType, resourceID string) ([]*PublicLink, error)

	// RevokePublicLink revokes a public link.
	RevokePublicLink(ctx context.Context, uuid string) error

	// RecordView records a view of a public link.
	RecordView(ctx context.Context, uuid string) error

	// CreateEmbedToken creates a signed embedding token.
	CreateEmbedToken(ctx context.Context, in *CreateEmbedTokenIn) (*EmbedToken, error)

	// ValidateEmbedToken validates and decodes an embed token.
	ValidateEmbedToken(ctx context.Context, token string) (*EmbedToken, error)

	// GetSettings returns current embedding settings.
	GetSettings(ctx context.Context) (*EmbedSettings, error)

	// UpdateSettings updates embedding settings.
	UpdateSettings(ctx context.Context, settings *EmbedSettings) error
}

// Store defines data access for public links.
type Store interface {
	CreatePublicLink(ctx context.Context, link *PublicLink) error
	GetPublicLinkByUUID(ctx context.Context, uuid string) (*PublicLink, error)
	ListPublicLinks(ctx context.Context, resourceType ResourceType, resourceID string) ([]*PublicLink, error)
	UpdatePublicLink(ctx context.Context, link *PublicLink) error
	DeletePublicLink(ctx context.Context, uuid string) error
	IncrementViewCount(ctx context.Context, uuid string) error
}

// SettingsStore defines data access for settings.
type SettingsStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

// TokenSigner signs and verifies embed tokens.
type TokenSigner interface {
	Sign(data []byte, expiry time.Time) (string, error)
	Verify(token string) ([]byte, time.Time, error)
}
