package rest

import (
	"net/http"

	"github.com/go-mizu/blueprints/cms/config"
	"github.com/go-mizu/mizu"
)

// Access handles access control endpoints.
type Access struct {
	collections []config.CollectionConfig
	globals     []config.GlobalConfig
}

// NewAccess creates a new Access handler.
func NewAccess(collections []config.CollectionConfig, globals []config.GlobalConfig) *Access {
	return &Access{
		collections: collections,
		globals:     globals,
	}
}

// GetAccess handles GET /api/access
func (h *Access) GetAccess(c *mizu.Ctx) error {
	// Get user from context (set by auth middleware)
	user := getUserFromContext(c)

	collectionsAccess := make(map[string]map[string]any)
	for _, col := range h.collections {
		collectionsAccess[col.Slug] = map[string]any{
			"create": h.checkAccess(col.Access, "create", user),
			"read":   h.checkAccess(col.Access, "read", user),
			"update": h.checkAccess(col.Access, "update", user),
			"delete": h.checkAccess(col.Access, "delete", user),
		}
	}

	globalsAccess := make(map[string]map[string]any)
	for _, g := range h.globals {
		globalsAccess[g.Slug] = map[string]any{
			"read":   h.checkGlobalAccess(g.Access, "read", user),
			"update": h.checkGlobalAccess(g.Access, "update", user),
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"canAccessAdmin": user != nil,
		"collections":    collectionsAccess,
		"globals":        globalsAccess,
	})
}

func (h *Access) checkAccess(access *config.AccessConfig, operation string, user map[string]any) bool {
	if access == nil {
		return user != nil // Default: authenticated users only
	}

	var fn config.AccessFn
	switch operation {
	case "create":
		fn = access.Create
	case "read":
		fn = access.Read
	case "update":
		fn = access.Update
	case "delete":
		fn = access.Delete
	}

	if fn == nil {
		return user != nil
	}

	result, err := fn(&config.AccessContext{User: user})
	if err != nil {
		return false
	}
	return result.Allowed
}

func (h *Access) checkGlobalAccess(access *config.AccessConfig, operation string, user map[string]any) bool {
	if access == nil {
		return user != nil
	}

	var fn config.AccessFn
	switch operation {
	case "read":
		fn = access.Read
	case "update":
		fn = access.Update
	}

	if fn == nil {
		return user != nil
	}

	result, err := fn(&config.AccessContext{User: user})
	if err != nil {
		return false
	}
	return result.Allowed
}

func getUserFromContext(c *mizu.Ctx) map[string]any {
	user, _ := c.Request().Context().Value(userContextKey{}).(map[string]any)
	return user
}

// userContextKey matches the middleware key.
type userContextKey struct{}
