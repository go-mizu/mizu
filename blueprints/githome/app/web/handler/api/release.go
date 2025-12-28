package api

import (
	"io"
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/releases"
	"github.com/go-mizu/blueprints/githome/feature/repos"
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

// ListReleases handles GET /repos/{owner}/{repo}/releases
func (h *ReleaseHandler) ListReleases(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	releaseList, err := h.releases.List(r.Context(), owner, repoName, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, releaseList)
}

// GetRelease handles GET /repos/{owner}/{repo}/releases/{release_id}
func (h *ReleaseHandler) GetRelease(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	release, err := h.releases.Get(r.Context(), owner, repoName, releaseID)
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
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	release, err := h.releases.GetLatest(r.Context(), owner, repoName)
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
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			WriteNotFound(w, "Repository")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tag := PathParam(r, "tag")

	release, err := h.releases.GetByTag(r.Context(), owner, repoName, tag)
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

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	release, err := h.releases.Create(r.Context(), owner, repoName, user.ID, &in)
	if err != nil {
		if err == releases.ErrReleaseExists {
			WriteBadRequest(w, "Release already exists")
			return
		}
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

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	var in releases.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	updated, err := h.releases.Update(r.Context(), owner, repoName, releaseID, &in)
	if err != nil {
		if err == releases.ErrNotFound {
			WriteNotFound(w, "Release")
			return
		}
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

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	if err := h.releases.Delete(r.Context(), owner, repoName, releaseID); err != nil {
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

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	notes, err := h.releases.GenerateNotes(r.Context(), owner, repoName, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, notes)
}

// ListReleaseAssets handles GET /repos/{owner}/{repo}/releases/{release_id}/assets
func (h *ReleaseHandler) ListReleaseAssets(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	assets, err := h.releases.ListAssets(r.Context(), owner, repoName, releaseID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, assets)
}

// GetReleaseAsset handles GET /repos/{owner}/{repo}/releases/assets/{asset_id}
func (h *ReleaseHandler) GetReleaseAsset(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	asset, err := h.releases.GetAsset(r.Context(), owner, repoName, assetID)
	if err != nil {
		if err == releases.ErrAssetNotFound {
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

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	asset, err := h.releases.UpdateAsset(r.Context(), owner, repoName, assetID, &in)
	if err != nil {
		if err == releases.ErrAssetNotFound {
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

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	if err := h.releases.DeleteAsset(r.Context(), owner, repoName, assetID); err != nil {
		if err == releases.ErrAssetNotFound {
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

	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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
	if name == "" {
		WriteBadRequest(w, "Asset name is required")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	asset, err := h.releases.UploadAsset(r.Context(), owner, repoName, releaseID, user.ID, name, contentType, r.Body)
	if err != nil {
		if err == releases.ErrNotFound {
			WriteNotFound(w, "Release")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, asset)
}

// DownloadReleaseAsset handles GET /repos/{owner}/{repo}/releases/assets/{asset_id}/download
func (h *ReleaseHandler) DownloadReleaseAsset(w http.ResponseWriter, r *http.Request) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")

	_, err := h.repos.Get(r.Context(), owner, repoName)
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

	reader, contentType, err := h.releases.DownloadAsset(r.Context(), owner, repoName, assetID)
	if err != nil {
		if err == releases.ErrAssetNotFound {
			WriteNotFound(w, "Asset")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", contentType)
	io.Copy(w, reader)
}
