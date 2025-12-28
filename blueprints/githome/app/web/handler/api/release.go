package api

import (
	"io"
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/releases"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/mizu"
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
func (h *ReleaseHandler) ListReleases(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	pagination := GetPagination(c)
	opts := &releases.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	releaseList, err := h.releases.List(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, releaseList)
}

// GetRelease handles GET /repos/{owner}/{repo}/releases/{release_id}
func (h *ReleaseHandler) GetRelease(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	releaseID, err := ParamInt64(c, "release_id")
	if err != nil {
		return BadRequest(c, "Invalid release ID")
	}

	release, err := h.releases.Get(c.Context(), owner, repoName, releaseID)
	if err != nil {
		if err == releases.ErrNotFound {
			return NotFound(c, "Release")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, release)
}

// GetLatestRelease handles GET /repos/{owner}/{repo}/releases/latest
func (h *ReleaseHandler) GetLatestRelease(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	release, err := h.releases.GetLatest(c.Context(), owner, repoName)
	if err != nil {
		if err == releases.ErrNotFound {
			return NotFound(c, "Release")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, release)
}

// GetReleaseByTag handles GET /repos/{owner}/{repo}/releases/tags/{tag}
func (h *ReleaseHandler) GetReleaseByTag(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	tag := c.Param("tag")

	release, err := h.releases.GetByTag(c.Context(), owner, repoName, tag)
	if err != nil {
		if err == releases.ErrNotFound {
			return NotFound(c, "Release")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, release)
}

// CreateRelease handles POST /repos/{owner}/{repo}/releases
func (h *ReleaseHandler) CreateRelease(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	var in releases.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	release, err := h.releases.Create(c.Context(), owner, repoName, user.ID, &in)
	if err != nil {
		if err == releases.ErrReleaseExists {
			return BadRequest(c, "Release already exists")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, release)
}

// UpdateRelease handles PATCH /repos/{owner}/{repo}/releases/{release_id}
func (h *ReleaseHandler) UpdateRelease(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	releaseID, err := ParamInt64(c, "release_id")
	if err != nil {
		return BadRequest(c, "Invalid release ID")
	}

	var in releases.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	updated, err := h.releases.Update(c.Context(), owner, repoName, releaseID, &in)
	if err != nil {
		if err == releases.ErrNotFound {
			return NotFound(c, "Release")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, updated)
}

// DeleteRelease handles DELETE /repos/{owner}/{repo}/releases/{release_id}
func (h *ReleaseHandler) DeleteRelease(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	releaseID, err := ParamInt64(c, "release_id")
	if err != nil {
		return BadRequest(c, "Invalid release ID")
	}

	if err := h.releases.Delete(c.Context(), owner, repoName, releaseID); err != nil {
		if err == releases.ErrNotFound {
			return NotFound(c, "Release")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// GenerateReleaseNotes handles POST /repos/{owner}/{repo}/releases/generate-notes
func (h *ReleaseHandler) GenerateReleaseNotes(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	var in releases.GenerateNotesIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	notes, err := h.releases.GenerateNotes(c.Context(), owner, repoName, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, notes)
}

// ListReleaseAssets handles GET /repos/{owner}/{repo}/releases/{release_id}/assets
func (h *ReleaseHandler) ListReleaseAssets(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	releaseID, err := ParamInt64(c, "release_id")
	if err != nil {
		return BadRequest(c, "Invalid release ID")
	}

	pagination := GetPagination(c)
	opts := &releases.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	assets, err := h.releases.ListAssets(c.Context(), owner, repoName, releaseID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, assets)
}

// GetReleaseAsset handles GET /repos/{owner}/{repo}/releases/assets/{asset_id}
func (h *ReleaseHandler) GetReleaseAsset(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	assetID, err := ParamInt64(c, "asset_id")
	if err != nil {
		return BadRequest(c, "Invalid asset ID")
	}

	asset, err := h.releases.GetAsset(c.Context(), owner, repoName, assetID)
	if err != nil {
		if err == releases.ErrAssetNotFound {
			return NotFound(c, "Asset")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, asset)
}

// UpdateReleaseAsset handles PATCH /repos/{owner}/{repo}/releases/assets/{asset_id}
func (h *ReleaseHandler) UpdateReleaseAsset(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	assetID, err := ParamInt64(c, "asset_id")
	if err != nil {
		return BadRequest(c, "Invalid asset ID")
	}

	var in releases.UpdateAssetIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	asset, err := h.releases.UpdateAsset(c.Context(), owner, repoName, assetID, &in)
	if err != nil {
		if err == releases.ErrAssetNotFound {
			return NotFound(c, "Asset")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, asset)
}

// DeleteReleaseAsset handles DELETE /repos/{owner}/{repo}/releases/assets/{asset_id}
func (h *ReleaseHandler) DeleteReleaseAsset(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	assetID, err := ParamInt64(c, "asset_id")
	if err != nil {
		return BadRequest(c, "Invalid asset ID")
	}

	if err := h.releases.DeleteAsset(c.Context(), owner, repoName, assetID); err != nil {
		if err == releases.ErrAssetNotFound {
			return NotFound(c, "Asset")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// UploadReleaseAsset handles POST /repos/{owner}/{repo}/releases/{release_id}/assets
func (h *ReleaseHandler) UploadReleaseAsset(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	releaseID, err := ParamInt64(c, "release_id")
	if err != nil {
		return BadRequest(c, "Invalid release ID")
	}

	name := c.Query("name")
	if name == "" {
		return BadRequest(c, "Asset name is required")
	}

	req := c.Request()
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	asset, err := h.releases.UploadAsset(c.Context(), owner, repoName, releaseID, user.ID, name, contentType, req.Body)
	if err != nil {
		if err == releases.ErrNotFound {
			return NotFound(c, "Release")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, asset)
}

// DownloadReleaseAsset handles GET /repos/{owner}/{repo}/releases/assets/{asset_id}/download
func (h *ReleaseHandler) DownloadReleaseAsset(c *mizu.Ctx) error {
	owner := c.Param("owner")
	repoName := c.Param("repo")

	_, err := h.repos.Get(c.Context(), owner, repoName)
	if err != nil {
		if err == repos.ErrNotFound {
			return NotFound(c, "Repository")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	assetID, err := ParamInt64(c, "asset_id")
	if err != nil {
		return BadRequest(c, "Invalid asset ID")
	}

	reader, contentType, err := h.releases.DownloadAsset(c.Context(), owner, repoName, assetID)
	if err != nil {
		if err == releases.ErrAssetNotFound {
			return NotFound(c, "Asset")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}
	defer reader.Close()

	c.Header().Set("Content-Type", contentType)
	io.Copy(c.Writer(), reader)
	return nil
}
