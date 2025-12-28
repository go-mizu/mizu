package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/repos"
)

// RepoHandler handles repository endpoints
type RepoHandler struct {
	repos repos.API
}

// NewRepoHandler creates a new repo handler
func NewRepoHandler(repos repos.API) *RepoHandler {
	return &RepoHandler{repos: repos}
}

// ListAuthenticatedUserRepos handles GET /user/repos
func (h *RepoHandler) ListAuthenticatedUserRepos(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	pagination := GetPaginationParams(r)
	opts := &repos.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Type:      QueryParam(r, "type"),
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	repoList, err := h.repos.ListForAuthenticatedUser(r.Context(), user.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, repoList)
}

// CreateAuthenticatedUserRepo handles POST /user/repos
func (h *RepoHandler) CreateAuthenticatedUserRepo(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	var in repos.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	repo, err := h.repos.Create(r.Context(), user.ID, &in)
	if err != nil {
		if err == repos.ErrRepoExists {
			WriteConflict(w, "Repository already exists")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, repo)
}

// ListUserRepos handles GET /users/{username}/repos
func (h *RepoHandler) ListUserRepos(w http.ResponseWriter, r *http.Request) {
	username := PathParam(r, "username")
	pagination := GetPaginationParams(r)
	opts := &repos.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Type:      QueryParam(r, "type"),
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	repoList, err := h.repos.ListForUser(r.Context(), username, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, repoList)
}

// ListOrgRepos handles GET /orgs/{org}/repos
func (h *RepoHandler) ListOrgRepos(w http.ResponseWriter, r *http.Request) {
	org := PathParam(r, "org")
	pagination := GetPaginationParams(r)
	opts := &repos.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Type:      QueryParam(r, "type"),
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	repoList, err := h.repos.ListForOrg(r.Context(), org, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, repoList)
}

// CreateOrgRepo handles POST /orgs/{org}/repos
func (h *RepoHandler) CreateOrgRepo(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")

	var in repos.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	repo, err := h.repos.CreateForOrg(r.Context(), org, &in)
	if err != nil {
		if err == repos.ErrRepoExists {
			WriteConflict(w, "Repository already exists")
			return
		}
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Organization")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, repo)
}

// GetRepo handles GET /repos/{owner}/{repo}
func (h *RepoHandler) GetRepo(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	repo, err := h.repos.Get(r.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, repo)
}

// UpdateRepo handles PATCH /repos/{owner}/{repo}
func (h *RepoHandler) UpdateRepo(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	var in repos.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.repos.Update(r.Context(), owner, repoName, &in)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// DeleteRepo handles DELETE /repos/{owner}/{repo}
func (h *RepoHandler) DeleteRepo(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	if err := h.repos.Delete(r.Context(), owner, repoName); err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListRepoTopics handles GET /repos/{owner}/{repo}/topics
func (h *RepoHandler) ListRepoTopics(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	topics, err := h.repos.ListTopics(r.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string][]string{"names": topics})
}

// ReplaceRepoTopics handles PUT /repos/{owner}/{repo}/topics
func (h *RepoHandler) ReplaceRepoTopics(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	var in struct {
		Names []string `json:"names"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	topics, err := h.repos.ReplaceTopics(r.Context(), owner, repoName, in.Names)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string][]string{"names": topics})
}

// ListRepoLanguages handles GET /repos/{owner}/{repo}/languages
func (h *RepoHandler) ListRepoLanguages(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	languages, err := h.repos.ListLanguages(r.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, languages)
}

// ListRepoContributors handles GET /repos/{owner}/{repo}/contributors
func (h *RepoHandler) ListRepoContributors(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	pagination := GetPaginationParams(r)
	opts := &repos.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	contributors, err := h.repos.ListContributors(r.Context(), owner, repoName, opts)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, contributors)
}

// TransferRepo handles POST /repos/{owner}/{repo}/transfer
func (h *RepoHandler) TransferRepo(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	var in repos.TransferIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.repos.Transfer(r.Context(), owner, repoName, &in)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteAccepted(w, updated)
}

// GetRepoReadme handles GET /repos/{owner}/{repo}/readme
func (h *RepoHandler) GetRepoReadme(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	ref := QueryParam(r, "ref")

	content, err := h.repos.GetReadme(r.Context(), owner, repoName, ref)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "README")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, content)
}

// GetRepoContent handles GET /repos/{owner}/{repo}/contents/{path}
func (h *RepoHandler) GetRepoContent(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	path := PathParam(r, "path")
	ref := QueryParam(r, "ref")

	content, err := h.repos.GetContents(r.Context(), owner, repoName, path, ref)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Content")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, content)
}

// CreateOrUpdateFileContent handles PUT /repos/{owner}/{repo}/contents/{path}
func (h *RepoHandler) CreateOrUpdateFileContent(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	path := PathParam(r, "path")

	var in struct {
		Message   string             `json:"message"`
		Content   string             `json:"content"`
		SHA       string             `json:"sha,omitempty"`
		Branch    string             `json:"branch,omitempty"`
		Committer *repos.CommitAuthor `json:"committer,omitempty"`
		Author    *repos.CommitAuthor `json:"author,omitempty"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	result, err := h.repos.CreateOrUpdateFile(r.Context(), owner, repoName, path, in.Message, in.Content, in.SHA, in.Branch, in.Author)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if in.SHA != "" {
		WriteJSON(w, http.StatusOK, result)
	} else {
		WriteCreated(w, result)
	}
}

// DeleteFileContent handles DELETE /repos/{owner}/{repo}/contents/{path}
func (h *RepoHandler) DeleteFileContent(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	path := PathParam(r, "path")

	var in struct {
		Message   string             `json:"message"`
		SHA       string             `json:"sha"`
		Branch    string             `json:"branch,omitempty"`
		Committer *repos.CommitAuthor `json:"committer,omitempty"`
		Author    *repos.CommitAuthor `json:"author,omitempty"`
	}
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	result, err := h.repos.DeleteFile(r.Context(), owner, repoName, path, in.Message, in.SHA, in.Branch, in.Author)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "File")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, result)
}

// ForkRepo handles POST /repos/{owner}/{repo}/forks
func (h *RepoHandler) ForkRepo(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	var in struct {
		Organization      string `json:"organization,omitempty"`
		Name              string `json:"name,omitempty"`
		DefaultBranchOnly bool   `json:"default_branch_only,omitempty"`
	}
	// Body is optional
	DecodeJSON(r, &in)

	fork, err := h.repos.CreateFork(r.Context(), owner, repoName, in.Organization, in.Name)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteAccepted(w, fork)
}

// ListForks handles GET /repos/{owner}/{repo}/forks
func (h *RepoHandler) ListForks(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	pagination := GetPaginationParams(r)
	opts := &repos.ListOpts{
		Page:      pagination.Page,
		PerPage:   pagination.PerPage,
		Sort:      QueryParam(r, "sort"),
		Direction: QueryParam(r, "direction"),
	}

	forks, err := h.repos.ListForks(r.Context(), owner, repoName, opts)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, forks)
}
