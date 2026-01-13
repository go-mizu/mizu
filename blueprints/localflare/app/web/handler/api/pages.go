package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"
)

// Pages handles Cloudflare Pages requests.
type Pages struct{}

// NewPages creates a new Pages handler.
func NewPages() *Pages {
	return &Pages{}
}

// PagesProject represents a Pages project.
type PagesProject struct {
	Name              string            `json:"name"`
	Subdomain         string            `json:"subdomain"`
	CreatedAt         string            `json:"created_at"`
	ProductionBranch  string            `json:"production_branch"`
	LatestDeployment  *PagesDeployment  `json:"latest_deployment,omitempty"`
	Domains           []string          `json:"domains"`
}

// PagesDeployment represents a deployment.
type PagesDeployment struct {
	ID                string                 `json:"id"`
	URL               string                 `json:"url"`
	Environment       string                 `json:"environment"`
	DeploymentTrigger map[string]any         `json:"deployment_trigger"`
	CreatedAt         string                 `json:"created_at"`
	Status            string                 `json:"status"`
}

// ListProjects lists all Pages projects.
func (h *Pages) ListProjects(c *mizu.Ctx) error {
	now := time.Now()
	projects := []PagesProject{
		{
			Name:             "my-blog",
			Subdomain:        "my-blog",
			CreatedAt:        now.Add(-48 * time.Hour).Format(time.RFC3339),
			ProductionBranch: "main",
			LatestDeployment: &PagesDeployment{
				ID:          "deploy-" + ulid.Make().String()[:8],
				URL:         "https://my-blog.pages.dev",
				Environment: "production",
				DeploymentTrigger: map[string]any{
					"type":     "push",
					"metadata": map[string]string{"branch": "main", "commit_hash": "abc123"},
				},
				CreatedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
				Status:    "success",
			},
			Domains: []string{"blog.example.com"},
		},
		{
			Name:             "docs-site",
			Subdomain:        "docs-site",
			CreatedAt:        now.Add(-168 * time.Hour).Format(time.RFC3339),
			ProductionBranch: "main",
			LatestDeployment: &PagesDeployment{
				ID:          "deploy-" + ulid.Make().String()[:8],
				URL:         "https://docs-site.pages.dev",
				Environment: "production",
				DeploymentTrigger: map[string]any{
					"type":     "push",
					"metadata": map[string]string{"branch": "main", "commit_hash": "def456"},
				},
				CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
				Status:    "success",
			},
			Domains: []string{"docs.example.com"},
		},
		{
			Name:             "marketing-site",
			Subdomain:        "marketing-site",
			CreatedAt:        now.Add(-72 * time.Hour).Format(time.RFC3339),
			ProductionBranch: "main",
			LatestDeployment: &PagesDeployment{
				ID:          "deploy-" + ulid.Make().String()[:8],
				URL:         "https://marketing-site.pages.dev",
				Environment: "production",
				DeploymentTrigger: map[string]any{
					"type":     "push",
					"metadata": map[string]string{"branch": "feature/hero", "commit_hash": "ghi789"},
				},
				CreatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
				Status:    "building",
			},
			Domains: []string{},
		},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"projects": projects,
		},
	})
}

// GetProject retrieves a project by name.
func (h *Pages) GetProject(c *mizu.Ctx) error {
	name := c.Param("name")
	now := time.Now()

	project := PagesProject{
		Name:             name,
		Subdomain:        name,
		CreatedAt:        now.Add(-48 * time.Hour).Format(time.RFC3339),
		ProductionBranch: "main",
		LatestDeployment: &PagesDeployment{
			ID:          "deploy-" + ulid.Make().String()[:8],
			URL:         "https://" + name + ".pages.dev",
			Environment: "production",
			DeploymentTrigger: map[string]any{
				"type":     "push",
				"metadata": map[string]string{"branch": "main", "commit_hash": "abc123"},
			},
			CreatedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
			Status:    "success",
		},
		Domains: []string{},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  project,
	})
}

// CreateProject creates a new project.
func (h *Pages) CreateProject(c *mizu.Ctx) error {
	var input struct {
		Name             string `json:"name"`
		ProductionBranch string `json:"production_branch"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	project := PagesProject{
		Name:             input.Name,
		Subdomain:        input.Name,
		CreatedAt:        time.Now().Format(time.RFC3339),
		ProductionBranch: input.ProductionBranch,
		Domains:          []string{},
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  project,
	})
}

// DeleteProject deletes a project.
func (h *Pages) DeleteProject(c *mizu.Ctx) error {
	name := c.Param("name")
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"name": name},
	})
}

// GetDeployments lists deployments for a project.
func (h *Pages) GetDeployments(c *mizu.Ctx) error {
	name := c.Param("name")
	now := time.Now()

	deployments := []PagesDeployment{
		{
			ID:          "deploy-" + ulid.Make().String()[:8],
			URL:         "https://" + name + ".pages.dev",
			Environment: "production",
			DeploymentTrigger: map[string]any{
				"type":     "push",
				"metadata": map[string]string{"branch": "main", "commit_hash": "abc123"},
			},
			CreatedAt: now.Add(-1 * time.Hour).Format(time.RFC3339),
			Status:    "success",
		},
		{
			ID:          "deploy-" + ulid.Make().String()[:8],
			URL:         "https://" + ulid.Make().String()[:8] + "." + name + ".pages.dev",
			Environment: "preview",
			DeploymentTrigger: map[string]any{
				"type":     "push",
				"metadata": map[string]string{"branch": "feature/new-ui", "commit_hash": "def456"},
			},
			CreatedAt: now.Add(-3 * time.Hour).Format(time.RFC3339),
			Status:    "success",
		},
		{
			ID:          "deploy-" + ulid.Make().String()[:8],
			URL:         "https://" + ulid.Make().String()[:8] + "." + name + ".pages.dev",
			Environment: "preview",
			DeploymentTrigger: map[string]any{
				"type":     "push",
				"metadata": map[string]string{"branch": "fix/bug", "commit_hash": "ghi789"},
			},
			CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
			Status:    "success",
		},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"deployments": deployments,
		},
	})
}
