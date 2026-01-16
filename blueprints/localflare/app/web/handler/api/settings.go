package api

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"
)

// Settings handles settings requests.
type Settings struct {
	store store.Store
}

// NewSettings creates a new Settings handler.
func NewSettings(st store.Store) *Settings {
	return &Settings{store: st}
}

// APITokenResponse represents an API token response.
type APITokenResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Token       string   `json:"token,omitempty"` // Only shown on creation
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
	ExpiresAt   string   `json:"expires_at,omitempty"`
	LastUsed    string   `json:"last_used,omitempty"`
}

// AccountMemberResponse represents a team member response.
type AccountMemberResponse struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
	Joined string `json:"joined"`
}

// ListTokens lists all API tokens.
func (h *Settings) ListTokens(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	tokens, err := h.store.Settings().ListTokens(ctx)
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var result []APITokenResponse
	for _, token := range tokens {
		resp := APITokenResponse{
			ID:          token.ID,
			Name:        token.Name,
			Permissions: token.Permissions,
			CreatedAt:   token.CreatedAt.Format(time.RFC3339),
		}
		if token.ExpiresAt != nil {
			resp.ExpiresAt = token.ExpiresAt.Format(time.RFC3339)
		}
		if token.LastUsedAt != nil {
			resp.LastUsed = token.LastUsedAt.Format(time.RFC3339)
		}
		result = append(result, resp)
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"tokens": result,
		},
	})
}

// CreateToken creates a new API token.
func (h *Settings) CreateToken(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	var input struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		Expiration  string   `json:"expiration"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "Invalid input"}},
		})
	}

	if input.Name == "" {
		return c.JSON(400, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "Name is required"}},
		})
	}

	// Generate token
	tokenValue := "lf_" + ulid.Make().String()

	// Hash the token for storage
	hash := sha256.Sum256([]byte(tokenValue))
	tokenHash := hex.EncodeToString(hash[:])

	// Get last 4 chars for preview
	tokenPreview := tokenValue[len(tokenValue)-4:]

	now := time.Now()
	token := &store.APIToken{
		ID:           "tok-" + ulid.Make().String()[:8],
		Name:         input.Name,
		TokenHash:    tokenHash,
		TokenPreview: tokenPreview,
		Permissions:  input.Permissions,
		CreatedAt:    now,
	}

	if input.Expiration != "" {
		if d, err := time.ParseDuration(input.Expiration); err == nil {
			expiresAt := now.Add(d)
			token.ExpiresAt = &expiresAt
		}
	}

	if err := h.store.Settings().CreateToken(ctx, token); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result": map[string]any{
			"token": tokenValue,
		},
	})
}

// RevokeToken revokes an API token.
func (h *Settings) RevokeToken(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	if err := h.store.Settings().DeleteToken(ctx, id); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListMembers lists team members.
func (h *Settings) ListMembers(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	members, err := h.store.Settings().ListMembers(ctx)
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var result []AccountMemberResponse
	for _, member := range members {
		result = append(result, AccountMemberResponse{
			ID:     member.ID,
			Email:  member.Email,
			Name:   member.Name,
			Role:   member.Role,
			Status: member.Status,
			Joined: member.InvitedAt.Format(time.RFC3339),
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"members": result,
		},
	})
}

// InviteMember invites a new team member.
func (h *Settings) InviteMember(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	var input struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "Invalid input"}},
		})
	}

	if input.Email == "" {
		return c.JSON(400, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "Email is required"}},
		})
	}

	if input.Role == "" {
		input.Role = "member"
	}

	member := &store.TeamMember{
		ID:        "mem-" + ulid.Make().String()[:8],
		Email:     input.Email,
		Role:      input.Role,
		Status:    "pending",
		InvitedAt: time.Now(),
	}

	if err := h.store.Settings().CreateMember(ctx, member); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result": AccountMemberResponse{
			ID:     member.ID,
			Email:  member.Email,
			Role:   member.Role,
			Status: member.Status,
			Joined: member.InvitedAt.Format(time.RFC3339),
		},
	})
}

// RemoveMember removes a team member.
func (h *Settings) RemoveMember(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	id := c.Param("id")

	if err := h.store.Settings().DeleteMember(ctx, id); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}
