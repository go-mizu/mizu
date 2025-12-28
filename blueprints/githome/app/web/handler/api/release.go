package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/releases"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/repos"
)

// ReleaseHandler handles release endpoints
type ReleaseHandler struct {
	releases releases.API
	repos    repos.API
}

// NewReleaseHandler creates a new release handler
func NewReleaseHandler(releases releases.API, repos repos.API) *ReleaseHandler {
	return &ReleaseHandler{releases: releases, repos: repos}
}

// getRepoFromPath gets repository from path parameters
func (h *ReleaseHandler) getRepoFromPath(r *http.Request) (*repos.Repository, error) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	return h.repos.GetByFullName(r.Context(), owner, repoName)
}

// ListReleases handles GET /repos/{owner}/{repo}/releases
func (h *ReleaseHandler) ListReleases(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pagination := GetPaginationParams(r)
	opts := &releases.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	releaseList, err := h.releases.ListForRepo(r.Context(), repo.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, releaseList)
}

// GetRelease handles GET /repos/{owner}/{repo}/releases/{release_id}
func (h *ReleaseHandler) GetRelease(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	releaseID, err := PathParamInt64(r, "release_id")
	if err != nil {
		WriteBadRequest(w, "Invalid release ID")
		return
	}

	release, err := h.releases.GetByID(r.Context(), repo.ID, releaseID)
	if err != nil {
		if err == releases.ErrNotFound {
			WriteNotFound(w, "Release")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, release)
}

// GetLatestRelease handles GET /repos/{owner}/{repo}/releases/latest
func (h *ReleaseHandler) GetLatestRelease(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	release, err := h.releases.GetLatest(r.Context(), repo.ID)
	if err != nil {
		if err == releases.ErrNotFound {
			WriteNotFound(w, "Release")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, release)
}

// GetReleaseByTag handles GET /repos/{owner}/{repo}/releases/tags/{tag}
func (h *ReleaseHandler) GetReleaseByTag(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tag := PathParam(r, "tag")

	release, err := h.releases.GetByTag(r.Context(), repo.ID, tag)
	if err != nil {
		if err == releases.ErrNotFound {
			WriteNotFound(w, "Release")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, release)
}

// CreateRelease handles POST /repos/{owner}/{repo}/releases
func (h *ReleaseHandler) CreateRelease(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in releases.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	release, err := h.releases.Create(r.Context(), repo.ID, user.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, release)
}

// UpdateRelease handles PATCH /repos/{owner}/{repo}/releases/{release_id}
func (h *ReleaseHandler) UpdateRelease(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	releaseID, err := PathParamInt64(r, "release_id")
	if err != nil {
		WriteBadRequest(w, "Invalid release ID")
		return
	}

	release, err := h.releases.GetByID(r.Context(), repo.ID, releaseID)
	if err != nil {
		if err == releases.ErrNotFound {
			WriteNotFound(w, "Release")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in releases.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.releases.Update(r.Context(), release.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, updated)
}

// DeleteRelease handles DELETE /repos/{owner}/{repo}/releases/{release_id}
func (h *ReleaseHandler) DeleteRelease(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	releaseID, err := PathParamInt64(r, "release_id")
	if err != nil {
		WriteBadRequest(w, "Invalid release ID")
		return
	}

	if err := h.releases.Delete(r.Context(), repo.ID, releaseID); err != nil {
		if err == releases.ErrNotFound {
			WriteNotFound(w, "Release")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// GenerateReleaseNotes handles POST /repos/{owner}/{repo}/releases/generate-notes
func (h *ReleaseHandler) GenerateReleaseNotes(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var in releases.GenerateNotesIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	notes, err := h.releases.GenerateNotes(r.Context(), repo.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, notes)
}

// ListReleaseAssets handles GET /repos/{owner}/{repo}/releases/{release_id}/assets
func (h *ReleaseHandler) ListReleaseAssets(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	releaseID, err := PathParamInt64(r, "release_id")
	if err != nil {
		WriteBadRequest(w, "Invalid release ID")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &releases.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	assets, err := h.releases.ListAssets(r.Context(), repo.ID, releaseID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, assets)
}

// GetReleaseAsset handles GET /repos/{owner}/{repo}/releases/assets/{asset_id}
func (h *ReleaseHandler) GetReleaseAsset(w http.ResponseWriter, r *http.Request) {
	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	assetID, err := PathParamInt64(r, "asset_id")
	if err != nil {
		WriteBadRequest(w, "Invalid asset ID")
		return
	}

	asset, err := h.releases.GetAsset(r.Context(), repo.ID, assetID)
	if err != nil {
		if err == releases.ErrNotFound {
			WriteNotFound(w, "Asset")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, asset)
}

// UpdateReleaseAsset handles PATCH /repos/{owner}/{repo}/releases/assets/{asset_id}
func (h *ReleaseHandler) UpdateReleaseAsset(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	assetID, err := PathParamInt64(r, "asset_id")
	if err != nil {
		WriteBadRequest(w, "Invalid asset ID")
		return
	}

	var in releases.UpdateAssetIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	asset, err := h.releases.UpdateAsset(r.Context(), repo.ID, assetID, &in)
	if err != nil {
		if err == releases.ErrNotFound {
			WriteNotFound(w, "Asset")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, asset)
}

// DeleteReleaseAsset handles DELETE /repos/{owner}/{repo}/releases/assets/{asset_id}
func (h *ReleaseHandler) DeleteReleaseAsset(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	assetID, err := PathParamInt64(r, "asset_id")
	if err != nil {
		WriteBadRequest(w, "Invalid asset ID")
		return
	}

	if err := h.releases.DeleteAsset(r.Context(), repo.ID, assetID); err != nil {
		if err == releases.ErrNotFound {
			WriteNotFound(w, "Asset")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// UploadReleaseAsset handles POST /repos/{owner}/{repo}/releases/{release_id}/assets
func (h *ReleaseHandler) UploadReleaseAsset(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	repo, err := h.getRepoFromPath(r)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	releaseID, err := PathParamInt64(r, "release_id")
	if err != nil {
		WriteBadRequest(w, "Invalid release ID")
		return
	}

	name := QueryParam(r, "name")
	label := QueryParam(r, "label")

	asset, err := h.releases.UploadAsset(r.Context(), repo.ID, releaseID, name, label, r.Body, r.ContentLength)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, asset)
}
