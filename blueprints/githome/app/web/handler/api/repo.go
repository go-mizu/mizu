package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
)

// RepoHandler handles repository endpoints
type RepoHandler struct {
	repos repos.API
}

// NewRepoHandler creates a new repo handler
func NewRepoHandler(repos repos.API) *RepoHandler {
	return &RepoHandler{repos: repos}
}

// ListPublicRepos handles GET /repositories
func (h *RepoHandler) ListPublicRepos(c *mizu.Ctx) error {
	// TODO: Implement when repos.API.ListPublic is available
	// For now, return empty list
	return c.JSON(http.StatusOK, []*repos.Repository{})
}

// ListAuthenticatedUserRepos handles GET /user/repos
func (h *RepoHandler) ListAuthenticatedUserRepos(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	pagination := GetPagination(c)
	opts := &repos.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Type:      c.Query("type"),
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	repoList, err := h.repos.ListForAuthenticatedUser(c.Context(), user.ID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, repoList)
}

// CreateAuthenticatedUserRepo handles POST /user/repos
func (h *RepoHandler) CreateAuthenticatedUserRepo(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	var in repos.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	repo, err := h.repos.Create(c.Context(), user.ID, &in)
	if err != nil {
		if err == repos.ErrRepoExists {
			return Conflict(c, "Repository already exists")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, repo)
}

// ListUserRepos handles GET /users/{username}/repos
func (h *RepoHandler) ListUserRepos(c *mizu.Ctx) error {
	username := c.Param("username")
	pagination := GetPagination(c)
	opts := &repos.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Type:      c.Query("type"),
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	repoList, err := h.repos.ListForUser(c.Context(), username, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, repoList)
}

// ListOrgRepos handles GET /orgs/{org}/repos
func (h *RepoHandler) ListOrgRepos(c *mizu.Ctx) error {
	org := c.Param("org")
	pagination := GetPagination(c)
	opts := &repos.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Type:      c.Query("type"),
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	repoList, err := h.repos.ListForOrg(c.Context(), org, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, repoList)
}

// CreateOrgRepo handles POST /orgs/{org}/repos
func (h *RepoHandler) CreateOrgRepo(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")

	var in repos.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	repo, err := h.repos.CreateForOrg(c.Context(), org, &in)
	if err != nil {
		if err == repos.ErrRepoExists {
			return Conflict(c, "Repository already exists")
		}
		if err == repos.ErrNotFound {
			return NotFound(c, "Organization")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, repo)
}

// GetRepo handles GET /repos/{owner}/{repo}
func (h *RepoHandler) GetRepo(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	repo, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, repo)
}

// UpdateRepo handles PATCH /repos/{owner}/{repo}
func (h *RepoHandler) UpdateRepo(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	var in repos.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.repos.Update(c.Context(), owner, repoName, &in)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// DeleteRepo handles DELETE /repos/{owner}/{repo}
func (h *RepoHandler) DeleteRepo(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	if err := h.repos.Delete(c.Context(), owner, repoName); err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListRepoTopics handles GET /repos/{owner}/{repo}/topics
func (h *RepoHandler) ListRepoTopics(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	topics, err := h.repos.ListTopics(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string][]string{"names": topics})
}

// ReplaceRepoTopics handles PUT /repos/{owner}/{repo}/topics
func (h *RepoHandler) ReplaceRepoTopics(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	var in struct {
		Names []string `json:"names"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	topics, err := h.repos.ReplaceTopics(c.Context(), owner, repoName, in.Names)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, map[string][]string{"names": topics})
}

// ListRepoLanguages handles GET /repos/{owner}/{repo}/languages
func (h *RepoHandler) ListRepoLanguages(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	languages, err := h.repos.ListLanguages(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, languages)
}

// ListRepoContributors handles GET /repos/{owner}/{repo}/contributors
func (h *RepoHandler) ListRepoContributors(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	pagination := GetPagination(c)
	opts := &repos.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	contributors, err := h.repos.ListContributors(c.Context(), owner, repoName, opts)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, contributors)
}

// ListRepoTags handles GET /repos/{owner}/{repo}/tags
func (h *RepoHandler) ListRepoTags(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	// Verify repo exists
	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	// TODO: Implement when repos.API.ListTags is available
	// For now, return empty list
	return c.JSON(http.StatusOK, []any{})
}

// TransferRepo handles POST /repos/{owner}/{repo}/transfer
func (h *RepoHandler) TransferRepo(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	var in repos.TransferIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.repos.Transfer(c.Context(), owner, repoName, &in)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Accepted(c, updated)
}

// GetRepoReadme handles GET /repos/{owner}/{repo}/readme
func (h *RepoHandler) GetRepoReadme(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Query("ref")

	content, err := h.repos.GetReadme(c.Context(), owner, repoName, ref)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "README")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, content)
}

// GetRepoContent handles GET /repos/{owner}/{repo}/contents/{path}
func (h *RepoHandler) GetRepoContent(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	path := c.Param("path")
	ref := c.Query("ref")

	content, err := h.repos.GetContents(c.Context(), owner, repoName, path, ref)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Content")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, content)
}

// CreateOrUpdateFileContent handles PUT /repos/{owner}/{repo}/contents/{path}
func (h *RepoHandler) CreateOrUpdateFileContent(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")
	path := c.Param("path")

	var in struct {
		Message   string              `json:"message"`
		Content   string              `json:"content"`
		SHA       string              `json:"sha,omitempty"`
		Branch    string              `json:"branch,omitempty"`
		Committer *repos.CommitAuthor `json:"committer,omitempty"`
		Author    *repos.CommitAuthor `json:"author,omitempty"`
	}
	if err := c.BindJSON(&in, 10<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	result, err := h.repos.CreateOrUpdateFile(c.Context(), owner, repoName, path, in.Message, in.Content, in.SHA, in.Branch, in.Author)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	if in.SHA != "" {
		return c.JSON(http.StatusOK, result)
	}
	return Created(c, result)
}

// DeleteFileContent handles DELETE /repos/{owner}/{repo}/contents/{path}
func (h *RepoHandler) DeleteFileContent(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")
	path := c.Param("path")

	var in struct {
		Message   string              `json:"message"`
		SHA       string              `json:"sha"`
		Branch    string              `json:"branch,omitempty"`
		Committer *repos.CommitAuthor `json:"committer,omitempty"`
		Author    *repos.CommitAuthor `json:"author,omitempty"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	result, err := h.repos.DeleteFile(c.Context(), owner, repoName, path, in.Message, in.SHA, in.Branch, in.Author)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "File")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// ForkRepo handles POST /repos/{owner}/{repo}/forks
func (h *RepoHandler) ForkRepo(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	var in struct {
		Organization      string `json:"organization,omitempty"`
		Name              string `json:"name,omitempty"`
		DefaultBranchOnly bool   `json:"default_branch_only,omitempty"`
	}
	// Body is optional
	c.BindJSON(&in, 1<<20)

	fork, err := h.repos.CreateFork(c.Context(), owner, repoName, in.Organization, in.Name)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Accepted(c, fork)
}

// ListForks handles GET /repos/{owner}/{repo}/forks
func (h *RepoHandler) ListForks(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	pagination := GetPagination(c)
	opts := &repos.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      c.Query("sort"),
		Direction: c.Query("direction"),
	}

	forks, err := h.repos.ListForks(c.Context(), owner, repoName, opts)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, forks)
}
