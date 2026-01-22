package embedding

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// Service implements the Embedding API.
type Service struct {
	links    Store
	settings SettingsStore
	signer   TokenSigner
}

// NewService creates a new Embedding service.
func NewService(links Store, settings SettingsStore, signer TokenSigner) *Service {
	return &Service{
		links:    links,
		settings: settings,
		signer:   signer,
	}
}

// CreatePublicLink creates a public sharing link.
func (s *Service) CreatePublicLink(ctx context.Context, in *CreatePublicLinkIn) (*PublicLink, error) {
	// Check if embedding is enabled
	enabled, _ := s.settings.Get(ctx, "enable_public_sharing")
	if enabled != "true" {
		return nil, ErrEmbeddingDisabled
	}

	// Validate resource type
	if in.ResourceType != ResourceQuestion && in.ResourceType != ResourceDashboard {
		return nil, fmt.Errorf("invalid resource type: %s", in.ResourceType)
	}

	// Generate UUID
	uuid, err := generateUUID()
	if err != nil {
		return nil, fmt.Errorf("generate uuid: %w", err)
	}

	now := time.Now()
	link := &PublicLink{
		ID:           ulid.Make().String(),
		UUID:         uuid,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		CreatorID:    in.CreatorID,
		Active:       true,
		CreatedAt:    now,
	}

	if in.ExpiresIn != nil {
		expiry := now.Add(*in.ExpiresIn)
		link.ExpiresAt = &expiry
	}

	if err := s.links.CreatePublicLink(ctx, link); err != nil {
		return nil, err
	}

	return link, nil
}

// GetPublicLink returns a public link by UUID.
func (s *Service) GetPublicLink(ctx context.Context, uuid string) (*PublicLink, error) {
	link, err := s.links.GetPublicLinkByUUID(ctx, uuid)
	if err != nil {
		return nil, ErrNotFound
	}

	// Check if active
	if !link.Active {
		return nil, ErrNotFound
	}

	// Check expiry
	if link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpired
	}

	return link, nil
}

// ListPublicLinks returns all public links for a resource.
func (s *Service) ListPublicLinks(ctx context.Context, resourceType ResourceType, resourceID string) ([]*PublicLink, error) {
	return s.links.ListPublicLinks(ctx, resourceType, resourceID)
}

// RevokePublicLink revokes a public link.
func (s *Service) RevokePublicLink(ctx context.Context, uuid string) error {
	link, err := s.links.GetPublicLinkByUUID(ctx, uuid)
	if err != nil {
		return ErrNotFound
	}

	link.Active = false
	return s.links.UpdatePublicLink(ctx, link)
}

// RecordView records a view of a public link.
func (s *Service) RecordView(ctx context.Context, uuid string) error {
	return s.links.IncrementViewCount(ctx, uuid)
}

// CreateEmbedToken creates a signed embedding token.
func (s *Service) CreateEmbedToken(ctx context.Context, in *CreateEmbedTokenIn) (*EmbedToken, error) {
	// Check if embedding is enabled
	enabled, _ := s.settings.Get(ctx, "enable_embedding")
	if enabled != "true" {
		return nil, ErrEmbeddingDisabled
	}

	if s.signer == nil {
		return nil, fmt.Errorf("token signer not configured")
	}

	// Default permissions
	permissions := in.Permissions
	if permissions == nil {
		permissions = &EmbedPermissions{
			CanDownload:   false,
			CanFullscreen: true,
			CanRefresh:    true,
			CanFilter:     true,
			ShowBranding:  true,
		}
	}

	// Set expiry
	expiry := time.Now().Add(in.ExpiresIn)

	// Create token payload
	payload := map[string]any{
		"resource_type": in.ResourceType,
		"resource_id":   in.ResourceID,
		"parameters":    in.Parameters,
		"permissions":   permissions,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	// Sign the token
	token, err := s.signer.Sign(payloadBytes, expiry)
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}

	return &EmbedToken{
		Token:        token,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		Parameters:   in.Parameters,
		Permissions:  permissions,
		ExpiresAt:    expiry,
		CreatedAt:    time.Now(),
	}, nil
}

// ValidateEmbedToken validates and decodes an embed token.
func (s *Service) ValidateEmbedToken(ctx context.Context, token string) (*EmbedToken, error) {
	// Check if embedding is enabled
	enabled, _ := s.settings.Get(ctx, "enable_embedding")
	if enabled != "true" {
		return nil, ErrEmbeddingDisabled
	}

	if s.signer == nil {
		return nil, fmt.Errorf("token signer not configured")
	}

	// Verify and decode
	payloadBytes, expiry, err := s.signer.Verify(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Check expiry
	if expiry.Before(time.Now()) {
		return nil, ErrExpired
	}

	// Decode payload
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, ErrInvalidToken
	}

	// Extract resource info
	resourceType, _ := payload["resource_type"].(string)
	resourceID, _ := payload["resource_id"].(string)

	// Extract parameters
	var parameters map[string]any
	if p, ok := payload["parameters"].(map[string]any); ok {
		parameters = p
	}

	// Extract permissions
	var permissions *EmbedPermissions
	if p, ok := payload["permissions"].(map[string]any); ok {
		permissions = &EmbedPermissions{}
		if v, ok := p["can_download"].(bool); ok {
			permissions.CanDownload = v
		}
		if v, ok := p["can_fullscreen"].(bool); ok {
			permissions.CanFullscreen = v
		}
		if v, ok := p["can_refresh"].(bool); ok {
			permissions.CanRefresh = v
		}
		if v, ok := p["can_filter"].(bool); ok {
			permissions.CanFilter = v
		}
		if v, ok := p["show_branding"].(bool); ok {
			permissions.ShowBranding = v
		}
	}

	return &EmbedToken{
		Token:        token,
		ResourceType: ResourceType(resourceType),
		ResourceID:   resourceID,
		Parameters:   parameters,
		Permissions:  permissions,
		ExpiresAt:    expiry,
	}, nil
}

// GetSettings returns current embedding settings.
func (s *Service) GetSettings(ctx context.Context) (*EmbedSettings, error) {
	settings := &EmbedSettings{}

	if v, _ := s.settings.Get(ctx, "enable_embedding"); v == "true" {
		settings.Enabled = true
	}

	// Don't expose secret key
	if v, _ := s.settings.Get(ctx, "embed_secret_key"); v != "" {
		settings.SecretKey = "***configured***"
	}

	if v, _ := s.settings.Get(ctx, "embed_allowed_origins"); v != "" {
		var origins []string
		json.Unmarshal([]byte(v), &origins)
		settings.AllowedOrigins = origins
	}

	return settings, nil
}

// UpdateSettings updates embedding settings.
func (s *Service) UpdateSettings(ctx context.Context, settings *EmbedSettings) error {
	if settings.Enabled {
		s.settings.Set(ctx, "enable_embedding", "true")
	} else {
		s.settings.Set(ctx, "enable_embedding", "false")
	}

	if settings.SecretKey != "" && settings.SecretKey != "***configured***" {
		s.settings.Set(ctx, "embed_secret_key", settings.SecretKey)
	}

	if settings.AllowedOrigins != nil {
		originsJSON, _ := json.Marshal(settings.AllowedOrigins)
		s.settings.Set(ctx, "embed_allowed_origins", string(originsJSON))
	}

	return nil
}

// Helper functions

func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
