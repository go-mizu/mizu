package api

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"
)

// Pages handles Cloudflare Pages requests.
type Pages struct {
	store store.Store
}

// NewPages creates a new Pages handler.
func NewPages(st store.Store) *Pages {
	return &Pages{store: st}
}

// PagesProjectResponse represents a Pages project response.
type PagesProjectResponse struct {
	Name             string                    `json:"name"`
	Subdomain        string                    `json:"subdomain"`
	CreatedAt        string                    `json:"created_at"`
	ProductionBranch string                    `json:"production_branch"`
	LatestDeployment *PagesDeploymentResponse  `json:"latest_deployment,omitempty"`
	Domains          []string                  `json:"domains"`
}

// PagesDeploymentResponse represents a deployment response.
type PagesDeploymentResponse struct {
	ID                string         `json:"id"`
	URL               string         `json:"url"`
	Environment       string         `json:"environment"`
	DeploymentTrigger map[string]any `json:"deployment_trigger"`
	CreatedAt         string         `json:"created_at"`
	Status            string         `json:"status"`
}

// ListProjects lists all Pages projects.
func (h *Pages) ListProjects(c *mizu.Ctx) error {
	projects, err := h.store.Pages().ListProjects(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var result []PagesProjectResponse
	for _, p := range projects {
		resp := h.projectToResponse(c.Request().Context(), p)
		result = append(result, resp)
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"projects": result,
		},
	})
}

// GetProject retrieves a project by name.
func (h *Pages) GetProject(c *mizu.Ctx) error {
	name := c.Param("name")

	project, err := h.store.Pages().GetProjectByName(c.Request().Context(), name)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(404, map[string]any{
				"success": false,
				"errors":  []map[string]any{{"code": 1001, "message": "Project not found"}},
			})
		}
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	resp := h.projectToResponse(c.Request().Context(), project)
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  resp,
	})
}

// CreateProject creates a new project.
func (h *Pages) CreateProject(c *mizu.Ctx) error {
	var input struct {
		Name             string `json:"name"`
		ProductionBranch string `json:"production_branch"`
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

	if input.ProductionBranch == "" {
		input.ProductionBranch = "main"
	}

	now := time.Now()
	project := &store.PagesProject{
		ID:               "pages_" + ulid.Make().String(),
		Name:             input.Name,
		Subdomain:        input.Name,
		ProductionBranch: input.ProductionBranch,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := h.store.Pages().CreateProject(c.Request().Context(), project); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	resp := h.projectToResponse(c.Request().Context(), project)
	return c.JSON(201, map[string]any{
		"success": true,
		"result":  resp,
	})
}

// DeleteProject deletes a project.
func (h *Pages) DeleteProject(c *mizu.Ctx) error {
	name := c.Param("name")

	project, err := h.store.Pages().GetProjectByName(c.Request().Context(), name)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(404, map[string]any{
				"success": false,
				"errors":  []map[string]any{{"code": 1001, "message": "Project not found"}},
			})
		}
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	if err := h.store.Pages().DeleteProject(c.Request().Context(), project.ID); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"name": name},
	})
}

// GetDeployments lists deployments for a project.
func (h *Pages) GetDeployments(c *mizu.Ctx) error {
	name := c.Param("name")

	project, err := h.store.Pages().GetProjectByName(c.Request().Context(), name)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(404, map[string]any{
				"success": false,
				"errors":  []map[string]any{{"code": 1001, "message": "Project not found"}},
			})
		}
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	deployments, err := h.store.Pages().ListDeployments(c.Request().Context(), project.ID, 20)
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var result []PagesDeploymentResponse
	for _, d := range deployments {
		result = append(result, h.deploymentToResponse(d))
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"deployments": result,
		},
	})
}

// CreateDeployment creates a new deployment.
func (h *Pages) CreateDeployment(c *mizu.Ctx) error {
	name := c.Param("name")

	project, err := h.store.Pages().GetProjectByName(c.Request().Context(), name)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(404, map[string]any{
				"success": false,
				"errors":  []map[string]any{{"code": 1001, "message": "Project not found"}},
			})
		}
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var input struct {
		Branch        string `json:"branch"`
		CommitHash    string `json:"commit_hash"`
		CommitMessage string `json:"commit_message"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "Invalid input"}},
		})
	}

	if input.Branch == "" {
		input.Branch = project.ProductionBranch
	}

	environment := "preview"
	if input.Branch == project.ProductionBranch {
		environment = "production"
	}

	deployID := ulid.Make().String()[:8]
	now := time.Now()

	deployment := &store.PagesDeployment{
		ID:            "deploy-" + deployID,
		ProjectID:     project.ID,
		Environment:   environment,
		Branch:        input.Branch,
		CommitHash:    input.CommitHash,
		CommitMessage: input.CommitMessage,
		URL:           "https://" + deployID + "." + project.Subdomain + ".pages.dev",
		Status:        "success", // Simplified - would be "building" in real impl
		CreatedAt:     now,
		FinishedAt:    &now,
	}

	if environment == "production" {
		deployment.URL = "https://" + project.Subdomain + ".pages.dev"
	}

	if err := h.store.Pages().CreateDeployment(c.Request().Context(), deployment); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  h.deploymentToResponse(deployment),
	})
}

func (h *Pages) projectToResponse(ctx context.Context, p *store.PagesProject) PagesProjectResponse {
	resp := PagesProjectResponse{
		Name:             p.Name,
		Subdomain:        p.Subdomain,
		CreatedAt:        p.CreatedAt.Format(time.RFC3339),
		ProductionBranch: p.ProductionBranch,
		Domains:          []string{},
	}

	// Get latest deployment
	if latest, err := h.store.Pages().GetLatestDeployment(ctx, p.ID); err == nil && latest != nil {
		resp.LatestDeployment = ptrTo(h.deploymentToResponse(latest))
	}

	// Get custom domains
	if domains, err := h.store.Pages().ListDomains(ctx, p.ID); err == nil {
		for _, d := range domains {
			resp.Domains = append(resp.Domains, d.Domain)
		}
	}

	return resp
}

func (h *Pages) deploymentToResponse(d *store.PagesDeployment) PagesDeploymentResponse {
	return PagesDeploymentResponse{
		ID:          d.ID,
		URL:         d.URL,
		Environment: d.Environment,
		DeploymentTrigger: map[string]any{
			"type": "push",
			"metadata": map[string]string{
				"branch":      d.Branch,
				"commit_hash": d.CommitHash,
			},
		},
		CreatedAt: d.CreatedAt.Format(time.RFC3339),
		Status:    d.Status,
	}
}

func ptrTo[T any](v T) *T {
	return &v
}
