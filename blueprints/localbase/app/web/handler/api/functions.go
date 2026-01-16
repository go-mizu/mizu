package api

import (
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/localbase/store"
	"github.com/go-mizu/mizu/blueprints/localbase/store/postgres"
	"github.com/oklog/ulid/v2"
)

// FunctionsHandler handles edge functions endpoints.
type FunctionsHandler struct {
	store *postgres.Store
}

// NewFunctionsHandler creates a new functions handler.
func NewFunctionsHandler(store *postgres.Store) *FunctionsHandler {
	return &FunctionsHandler{store: store}
}

// ListFunctions lists all functions.
func (h *FunctionsHandler) ListFunctions(c *mizu.Ctx) error {
	functions, err := h.store.Functions().ListFunctions(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list functions"})
	}

	return c.JSON(200, functions)
}

// CreateFunction creates a new function.
func (h *FunctionsHandler) CreateFunction(c *mizu.Ctx) error {
	var req struct {
		Name       string `json:"name"`
		Entrypoint string `json:"entrypoint"`
		ImportMap  string `json:"import_map"`
		VerifyJWT  bool   `json:"verify_jwt"`
		SourceCode string `json:"source_code"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" {
		return c.JSON(400, map[string]string{"error": "name required"})
	}

	entrypoint := req.Entrypoint
	if entrypoint == "" {
		entrypoint = "index.ts"
	}

	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))

	fn := &store.Function{
		ID:         ulid.Make().String(),
		Name:       req.Name,
		Slug:       slug,
		Version:    1,
		Status:     "active",
		Entrypoint: entrypoint,
		ImportMap:  req.ImportMap,
		VerifyJWT:  req.VerifyJWT,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := h.store.Functions().CreateFunction(c.Context(), fn); err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return c.JSON(400, map[string]string{"error": "function already exists"})
		}
		return c.JSON(500, map[string]string{"error": "failed to create function"})
	}

	// Create initial deployment if source code provided
	if req.SourceCode != "" {
		deployment := &store.Deployment{
			ID:         ulid.Make().String(),
			FunctionID: fn.ID,
			Version:    1,
			SourceCode: req.SourceCode,
			Status:     "deployed",
			DeployedAt: time.Now(),
		}
		_ = h.store.Functions().CreateDeployment(c.Context(), deployment)
	}

	return c.JSON(201, fn)
}

// GetFunction gets a function by ID.
func (h *FunctionsHandler) GetFunction(c *mizu.Ctx) error {
	id := c.Param("id")

	fn, err := h.store.Functions().GetFunction(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	return c.JSON(200, fn)
}

// UpdateFunction updates a function.
func (h *FunctionsHandler) UpdateFunction(c *mizu.Ctx) error {
	id := c.Param("id")

	fn, err := h.store.Functions().GetFunction(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	var req struct {
		Name       string `json:"name"`
		Entrypoint string `json:"entrypoint"`
		ImportMap  string `json:"import_map"`
		VerifyJWT  *bool  `json:"verify_jwt"`
		Status     string `json:"status"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name != "" {
		fn.Name = req.Name
		fn.Slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	}
	if req.Entrypoint != "" {
		fn.Entrypoint = req.Entrypoint
	}
	if req.ImportMap != "" {
		fn.ImportMap = req.ImportMap
	}
	if req.VerifyJWT != nil {
		fn.VerifyJWT = *req.VerifyJWT
	}
	if req.Status != "" {
		fn.Status = req.Status
	}
	fn.UpdatedAt = time.Now()

	if err := h.store.Functions().UpdateFunction(c.Context(), fn); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to update function"})
	}

	return c.JSON(200, fn)
}

// DeleteFunction deletes a function.
func (h *FunctionsHandler) DeleteFunction(c *mizu.Ctx) error {
	id := c.Param("id")

	if err := h.store.Functions().DeleteFunction(c.Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete function"})
	}

	return c.NoContent()
}

// DeployFunction deploys a new version of a function.
func (h *FunctionsHandler) DeployFunction(c *mizu.Ctx) error {
	id := c.Param("id")

	fn, err := h.store.Functions().GetFunction(c.Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	var req struct {
		SourceCode string `json:"source_code"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.SourceCode == "" {
		return c.JSON(400, map[string]string{"error": "source_code required"})
	}

	// Increment version
	fn.Version++
	fn.UpdatedAt = time.Now()
	_ = h.store.Functions().UpdateFunction(c.Context(), fn)

	// Create deployment
	deployment := &store.Deployment{
		ID:         ulid.Make().String(),
		FunctionID: fn.ID,
		Version:    fn.Version,
		SourceCode: req.SourceCode,
		Status:     "deployed",
		DeployedAt: time.Now(),
	}

	if err := h.store.Functions().CreateDeployment(c.Context(), deployment); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create deployment"})
	}

	return c.JSON(201, deployment)
}

// ListDeployments lists deployments for a function.
func (h *FunctionsHandler) ListDeployments(c *mizu.Ctx) error {
	id := c.Param("id")
	limit := queryInt(c, "limit", 10)

	deployments, err := h.store.Functions().ListDeployments(c.Context(), id, limit)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list deployments"})
	}

	return c.JSON(200, deployments)
}

// InvokeFunction invokes a function.
func (h *FunctionsHandler) InvokeFunction(c *mizu.Ctx) error {
	name := c.Param("name")

	fn, err := h.store.Functions().GetFunctionByName(c.Context(), name)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "function not found"})
	}

	if fn.Status != "active" {
		return c.JSON(503, map[string]string{"error": "function is not active"})
	}

	// Get latest deployment
	deployment, err := h.store.Functions().GetLatestDeployment(c.Context(), fn.ID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "no deployment found"})
	}

	// In production, we'd execute the function using Deno runtime
	// For now, return a placeholder response
	return c.JSON(200, map[string]any{
		"message":     "Function executed",
		"function":    fn.Name,
		"version":     deployment.Version,
		"executed_at": time.Now().Format(time.RFC3339),
	})
}

// ListSecrets lists all secrets (names only).
func (h *FunctionsHandler) ListSecrets(c *mizu.Ctx) error {
	secrets, err := h.store.Functions().ListSecrets(c.Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": "failed to list secrets"})
	}

	return c.JSON(200, secrets)
}

// CreateSecret creates or updates a secret.
func (h *FunctionsHandler) CreateSecret(c *mizu.Ctx) error {
	var req struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := c.BindJSON(&req, 0); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request body"})
	}

	if req.Name == "" || req.Value == "" {
		return c.JSON(400, map[string]string{"error": "name and value required"})
	}

	secret := &store.Secret{
		ID:        ulid.Make().String(),
		Name:      req.Name,
		Value:     req.Value, // In production, encrypt this
		CreatedAt: time.Now(),
	}

	if err := h.store.Functions().CreateSecret(c.Context(), secret); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to create secret"})
	}

	return c.JSON(201, map[string]string{
		"name":       secret.Name,
		"created_at": secret.CreatedAt.Format(time.RFC3339),
	})
}

// DeleteSecret deletes a secret.
func (h *FunctionsHandler) DeleteSecret(c *mizu.Ctx) error {
	name := c.Param("name")

	if err := h.store.Functions().DeleteSecret(c.Context(), name); err != nil {
		return c.JSON(500, map[string]string{"error": "failed to delete secret"})
	}

	return c.NoContent()
}
