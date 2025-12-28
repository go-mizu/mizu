package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/git"
	"github.com/go-mizu/blueprints/githome/feature/repos"
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
func (h *GitHandler) GetBlob(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	fileSHA := PathParam(r, "file_sha")

	blob, err := h.git.GetBlob(r.Context(), owner, repoName, fileSHA)
	if err != nil {
		if err == git.ErrNotFound {
			WriteNotFound(w, "Blob")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, blob)
}

// CreateBlob handles POST /repos/{owner}/{repo}/git/blobs
func (h *GitHandler) CreateBlob(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	var in git.CreateBlobIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	blob, err := h.git.CreateBlob(r.Context(), owner, repoName, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, blob)
}

// GetGitCommit handles GET /repos/{owner}/{repo}/git/commits/{commit_sha}
func (h *GitHandler) GetGitCommit(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	commitSHA := PathParam(r, "commit_sha")

	commit, err := h.git.GetGitCommit(r.Context(), owner, repoName, commitSHA)
	if err != nil {
		if err == git.ErrNotFound {
			WriteNotFound(w, "Commit")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, commit)
}

// CreateGitCommit handles POST /repos/{owner}/{repo}/git/commits
func (h *GitHandler) CreateGitCommit(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	var in git.CreateGitCommitIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	commit, err := h.git.CreateGitCommit(r.Context(), owner, repoName, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, commit)
}

// GetRef handles GET /repos/{owner}/{repo}/git/ref/{ref}
func (h *GitHandler) GetRef(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	ref := PathParam(r, "ref")

	reference, err := h.git.GetRef(r.Context(), owner, repoName, ref)
	if err != nil {
		if err == git.ErrNotFound {
			WriteNotFound(w, "Reference")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reference)
}

// ListMatchingRefs handles GET /repos/{owner}/{repo}/git/matching-refs/{ref}
func (h *GitHandler) ListMatchingRefs(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	ref := PathParam(r, "ref")

	refs, err := h.git.ListMatchingRefs(r.Context(), owner, repoName, ref)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, refs)
}

// CreateRef handles POST /repos/{owner}/{repo}/git/refs
func (h *GitHandler) CreateRef(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	var in git.CreateRefIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reference, err := h.git.CreateRef(r.Context(), owner, repoName, &in)
	if err != nil {
		if err == git.ErrRefExists {
			WriteConflict(w, "Reference already exists")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, reference)
}

// UpdateRef handles PATCH /repos/{owner}/{repo}/git/refs/{ref}
func (h *GitHandler) UpdateRef(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	ref := PathParam(r, "ref")

	var in struct {
		SHA   string `json:"sha"`
		Force bool   `json:"force,omitempty"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	reference, err := h.git.UpdateRef(r.Context(), owner, repoName, ref, in.SHA, in.Force)
	if err != nil {
		if err == git.ErrNotFound {
			WriteNotFound(w, "Reference")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, reference)
}

// DeleteRef handles DELETE /repos/{owner}/{repo}/git/refs/{ref}
func (h *GitHandler) DeleteRef(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	ref := PathParam(r, "ref")

	if err := h.git.DeleteRef(r.Context(), owner, repoName, ref); err != nil {
		if err == git.ErrNotFound {
			WriteNotFound(w, "Reference")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// GetTree handles GET /repos/{owner}/{repo}/git/trees/{tree_sha}
func (h *GitHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	treeSHA := PathParam(r, "tree_sha")
	recursive := QueryParamBool(r, "recursive")

	tree, err := h.git.GetTree(r.Context(), owner, repoName, treeSHA, recursive)
	if err != nil {
		if err == git.ErrNotFound {
			WriteNotFound(w, "Tree")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, tree)
}

// CreateTree handles POST /repos/{owner}/{repo}/git/trees
func (h *GitHandler) CreateTree(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	var in git.CreateTreeIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	tree, err := h.git.CreateTree(r.Context(), owner, repoName, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, tree)
}

// GetTag handles GET /repos/{owner}/{repo}/git/tags/{tag_sha}
func (h *GitHandler) GetTag(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	tagSHA := PathParam(r, "tag_sha")

	tag, err := h.git.GetTag(r.Context(), owner, repoName, tagSHA)
	if err != nil {
		if err == git.ErrNotFound {
			WriteNotFound(w, "Tag")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, tag)
}

// CreateTag handles POST /repos/{owner}/{repo}/git/tags
func (h *GitHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	var in git.CreateTagIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	tag, err := h.git.CreateTag(r.Context(), owner, repoName, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, tag)
}

// ListTags handles GET /repos/{owner}/{repo}/git/tags (lightweight tags list)
func (h *GitHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	pagination := GetPaginationParams(r)
	opts := &git.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	tags, err := h.git.ListTags(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, tags)
}
