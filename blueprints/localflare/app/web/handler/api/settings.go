package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"
)

// Settings handles settings requests.
type Settings struct{}

// NewSettings creates a new Settings handler.
func NewSettings() *Settings {
	return &Settings{}
}

// APIToken represents an API token.
type APIToken struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Token       string   `json:"token,omitempty"` // Only shown on creation
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
	ExpiresAt   string   `json:"expires_at,omitempty"`
	LastUsed    string   `json:"last_used,omitempty"`
}

// AccountMember represents a team member.
type AccountMember struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
	Joined string `json:"joined"`
}

// ListTokens lists all API tokens.
func (h *Settings) ListTokens(c *mizu.Ctx) error {
	now := time.Now()
	tokens := []APIToken{
		{
			ID:          "tok-" + ulid.Make().String()[:8],
			Name:        "CI/CD Pipeline Token",
			Permissions: []string{"workers:read", "workers:write", "kv:read", "kv:write"},
			CreatedAt:   now.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
			ExpiresAt:   now.Add(335 * 24 * time.Hour).Format(time.RFC3339),
			LastUsed:    now.Add(-2 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:          "tok-" + ulid.Make().String()[:8],
			Name:        "Monitoring Service",
			Permissions: []string{"analytics:read", "logs:read"},
			CreatedAt:   now.Add(-60 * 24 * time.Hour).Format(time.RFC3339),
			LastUsed:    now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
		{
			ID:          "tok-" + ulid.Make().String()[:8],
			Name:        "Development Token",
			Permissions: []string{"*:read", "*:write"},
			CreatedAt:   now.Add(-7 * 24 * time.Hour).Format(time.RFC3339),
			ExpiresAt:   now.Add(23 * 24 * time.Hour).Format(time.RFC3339),
			LastUsed:    now.Add(-1 * time.Hour).Format(time.RFC3339),
		},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"tokens": tokens,
		},
	})
}

// CreateToken creates a new API token.
func (h *Settings) CreateToken(c *mizu.Ctx) error {
	var input struct {
		Name        string `json:"name"`
		Permissions string `json:"permissions"`
		Expiration  string `json:"expiration"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	now := time.Now()
	tokenValue := "lf_" + ulid.Make().String()

	token := APIToken{
		ID:          "tok-" + ulid.Make().String()[:8],
		Name:        input.Name,
		Token:       tokenValue,
		Permissions: []string{input.Permissions},
		CreatedAt:   now.Format(time.RFC3339),
	}

	if input.Expiration != "" {
		if d, err := time.ParseDuration(input.Expiration); err == nil {
			token.ExpiresAt = now.Add(d).Format(time.RFC3339)
		}
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
	id := c.Param("id")
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}

// ListMembers lists team members.
func (h *Settings) ListMembers(c *mizu.Ctx) error {
	now := time.Now()
	members := []AccountMember{
		{
			ID:     "mem-" + ulid.Make().String()[:8],
			Email:  "admin@example.com",
			Name:   "Admin User",
			Role:   "admin",
			Status: "active",
			Joined: now.Add(-365 * 24 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:     "mem-" + ulid.Make().String()[:8],
			Email:  "developer@example.com",
			Name:   "Developer One",
			Role:   "developer",
			Status: "active",
			Joined: now.Add(-90 * 24 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:     "mem-" + ulid.Make().String()[:8],
			Email:  "viewer@example.com",
			Name:   "Viewer User",
			Role:   "viewer",
			Status: "active",
			Joined: now.Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		},
		{
			ID:     "mem-" + ulid.Make().String()[:8],
			Email:  "pending@example.com",
			Name:   "",
			Role:   "developer",
			Status: "pending",
			Joined: now.Add(-1 * 24 * time.Hour).Format(time.RFC3339),
		},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"members": members,
		},
	})
}

// InviteMember invites a new team member.
func (h *Settings) InviteMember(c *mizu.Ctx) error {
	var input struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	member := AccountMember{
		ID:     "mem-" + ulid.Make().String()[:8],
		Email:  input.Email,
		Role:   input.Role,
		Status: "pending",
		Joined: time.Now().Format(time.RFC3339),
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  member,
	})
}

// RemoveMember removes a team member.
func (h *Settings) RemoveMember(c *mizu.Ctx) error {
	id := c.Param("id")
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"id": id},
	})
}
