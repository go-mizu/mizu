package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/git"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
)

// GitHandler handles low-level Git data endpoints
type GitHandler struct {
	git   git.API
	repos repos.API
}

// NewGitHandler creates a new git handler
func NewGitHandler(gitAPI git.API, repos repos.API) *GitHandler {
	return &GitHandler{git: gitAPI, repos: repos}
}

// GetBlob handles GET /repos/{owner}/{repo}/git/blobs/{file_sha}
func (h *GitHandler) GetBlob(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	fileSHA := c.Param("file_sha")

	blob, err := h.git.GetBlob(c.Context(), owner, repoName, fileSHA)
	if err != nil {
		if err == git.ErrNotFound {
			return NotFound(c, "Blob")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, blob)
}

// CreateBlob handles POST /repos/{owner}/{repo}/git/blobs
func (h *GitHandler) CreateBlob(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	var in git.CreateBlobIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	blob, err := h.git.CreateBlob(c.Context(), owner, repoName, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, blob)
}

// GetGitCommit handles GET /repos/{owner}/{repo}/git/commits/{commit_sha}
func (h *GitHandler) GetGitCommit(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	commitSHA := c.Param("commit_sha")

	commit, err := h.git.GetGitCommit(c.Context(), owner, repoName, commitSHA)
	if err != nil {
		if err == git.ErrNotFound {
			return NotFound(c, "Commit")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, commit)
}

// CreateGitCommit handles POST /repos/{owner}/{repo}/git/commits
func (h *GitHandler) CreateGitCommit(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	var in git.CreateGitCommitIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	commit, err := h.git.CreateGitCommit(c.Context(), owner, repoName, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, commit)
}

// GetRef handles GET /repos/{owner}/{repo}/git/ref/{ref}
func (h *GitHandler) GetRef(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")

	reference, err := h.git.GetRef(c.Context(), owner, repoName, ref)
	if err != nil {
		if err == git.ErrNotFound {
			return NotFound(c, "Reference")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, reference)
}

// ListMatchingRefs handles GET /repos/{owner}/{repo}/git/matching-refs/{ref}
func (h *GitHandler) ListMatchingRefs(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")

	refs, err := h.git.ListMatchingRefs(c.Context(), owner, repoName, ref)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, refs)
}

// CreateRef handles POST /repos/{owner}/{repo}/git/refs
func (h *GitHandler) CreateRef(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	var in git.CreateRefIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	reference, err := h.git.CreateRef(c.Context(), owner, repoName, &in)
	if err != nil {
		if err == git.ErrRefExists {
			return Conflict(c, "Reference already exists")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, reference)
}

// UpdateRef handles PATCH /repos/{owner}/{repo}/git/refs/{ref}
func (h *GitHandler) UpdateRef(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")

	var in struct {
		SHA   string `json:"sha"`
		Force bool   `json:"force,omitempty"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	reference, err := h.git.UpdateRef(c.Context(), owner, repoName, ref, in.SHA, in.Force)
	if err != nil {
		if err == git.ErrNotFound {
			return NotFound(c, "Reference")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, reference)
}

// DeleteRef handles DELETE /repos/{owner}/{repo}/git/refs/{ref}
func (h *GitHandler) DeleteRef(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")
	ref := c.Param("ref")

	if err := h.git.DeleteRef(c.Context(), owner, repoName, ref); err != nil {
		if err == git.ErrNotFound {
			return NotFound(c, "Reference")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// GetTree handles GET /repos/{owner}/{repo}/git/trees/{tree_sha}
func (h *GitHandler) GetTree(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	treeSHA := c.Param("tree_sha")
	recursive := QueryBool(c, "recursive")

	tree, err := h.git.GetTree(c.Context(), owner, repoName, treeSHA, recursive)
	if err != nil {
		if err == git.ErrNotFound {
			return NotFound(c, "Tree")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, tree)
}

// CreateTree handles POST /repos/{owner}/{repo}/git/trees
func (h *GitHandler) CreateTree(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	var in git.CreateTreeIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	tree, err := h.git.CreateTree(c.Context(), owner, repoName, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, tree)
}

// GetTag handles GET /repos/{owner}/{repo}/git/tags/{tag_sha}
func (h *GitHandler) GetTag(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")
	tagSHA := c.Param("tag_sha")

	tag, err := h.git.GetTag(c.Context(), owner, repoName, tagSHA)
	if err != nil {
		if err == git.ErrNotFound {
			return NotFound(c, "Tag")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, tag)
}

// CreateTag handles POST /repos/{owner}/{repo}/git/tags
func (h *GitHandler) CreateTag(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	var in git.CreateTagIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	tag, err := h.git.CreateTag(c.Context(), owner, repoName, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, tag)
}

// ListTags handles GET /repos/{owner}/{repo}/git/tags (lightweight tags list)
func (h *GitHandler) ListTags(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	pagination := GetPagination(c)
	opts := &git.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	tags, err := h.git.ListTags(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, tags)
}
