package api

import (
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/releases"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/mizu"
)

// Release handles release endpoints
type Release struct {
	releases  releases.API
	repos     repos.API
	users     users.API
	getUserID func(*mizu.Ctx) string
}

// NewRelease creates a new release handler
func NewRelease(releases releases.API, repos repos.API, users users.API, getUserID func(*mizu.Ctx) string) *Release {
	return &Release{
		releases:  releases,
		repos:     repos,
		users:     users,
		getUserID: getUserID,
	}
}

func (h *Release) getRepo(c *mizu.Ctx) (*repos.Repository, error) {
	owner := c.Param("owner")
	name := c.Param("repo")

	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return nil, repos.ErrNotFound
	}

	return h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
}

// List lists releases for a repository
func (h *Release) List(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &releases.ListOpts{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	releaseList, err := h.releases.List(c.Context(), repo.ID, opts)
	if err != nil {
		return InternalError(c, "failed to list releases")
	}

	return OK(c, releaseList)
}

// Create creates a new release
func (h *Release) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	var in releases.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	release, err := h.releases.Create(c.Context(), repo.ID, userID, &in)
	if err != nil {
		switch err {
		case releases.ErrExists:
			return Conflict(c, "release with this tag already exists")
		case releases.ErrMissingTag:
			return BadRequest(c, "tag name is required")
		default:
			return InternalError(c, "failed to create release")
		}
	}

	return Created(c, release)
}

// Get retrieves a release by ID
func (h *Release) Get(c *mizu.Ctx) error {
	releaseID := c.Param("id")

	release, err := h.releases.GetByID(c.Context(), releaseID)
	if err != nil {
		return NotFound(c, "release not found")
	}

	return OK(c, release)
}

// GetByTag retrieves a release by tag name
func (h *Release) GetByTag(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	tag := c.Param("tag")
	if tag == "" {
		return BadRequest(c, "tag is required")
	}

	release, err := h.releases.GetByTag(c.Context(), repo.ID, tag)
	if err != nil {
		return NotFound(c, "release not found")
	}

	return OK(c, release)
}

// GetLatest retrieves the latest release
func (h *Release) GetLatest(c *mizu.Ctx) error {
	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	release, err := h.releases.GetLatest(c.Context(), repo.ID)
	if err != nil {
		return NotFound(c, "no releases found")
	}

	return OK(c, release)
}

// Update updates a release
func (h *Release) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	releaseID := c.Param("id")

	var in releases.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	release, err := h.releases.Update(c.Context(), releaseID, &in)
	if err != nil {
		return InternalError(c, "failed to update release")
	}

	return OK(c, release)
}

// Delete deletes a release
func (h *Release) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check admin permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	releaseID := c.Param("id")

	if err := h.releases.Delete(c.Context(), releaseID); err != nil {
		return InternalError(c, "failed to delete release")
	}

	return NoContent(c)
}

// Publish publishes a draft release
func (h *Release) Publish(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	releaseID := c.Param("id")

	release, err := h.releases.Publish(c.Context(), releaseID)
	if err != nil {
		switch err {
		case releases.ErrAlreadyPublished:
			return Conflict(c, "release is already published")
		default:
			return InternalError(c, "failed to publish release")
		}
	}

	return OK(c, release)
}

// ListAssets lists assets for a release
func (h *Release) ListAssets(c *mizu.Ctx) error {
	releaseID := c.Param("id")

	assets, err := h.releases.ListAssets(c.Context(), releaseID)
	if err != nil {
		return InternalError(c, "failed to list assets")
	}

	return OK(c, assets)
}

// UploadAsset uploads an asset to a release
func (h *Release) UploadAsset(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	releaseID := c.Param("id")

	var in releases.UploadAssetIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	asset, err := h.releases.UploadAsset(c.Context(), releaseID, userID, &in)
	if err != nil {
		return InternalError(c, "failed to upload asset")
	}

	return Created(c, asset)
}

// GetAsset retrieves an asset
func (h *Release) GetAsset(c *mizu.Ctx) error {
	assetID := c.Param("assetId")

	asset, err := h.releases.GetAsset(c.Context(), assetID)
	if err != nil {
		return NotFound(c, "asset not found")
	}

	return OK(c, asset)
}

// UpdateAsset updates an asset
func (h *Release) UpdateAsset(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	assetID := c.Param("assetId")

	var in struct {
		Name  string `json:"name"`
		Label string `json:"label"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	asset, err := h.releases.UpdateAsset(c.Context(), assetID, in.Name, in.Label)
	if err != nil {
		return InternalError(c, "failed to update asset")
	}

	return OK(c, asset)
}

// DeleteAsset deletes an asset
func (h *Release) DeleteAsset(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	repo, err := h.getRepo(c)
	if err != nil {
		return NotFound(c, "repository not found")
	}

	// Check write permission
	if !h.repos.CanAccess(c.Context(), repo.ID, userID, repos.PermissionWrite) {
		return Forbidden(c, "insufficient permissions")
	}

	assetID := c.Param("assetId")

	if err := h.releases.DeleteAsset(c.Context(), assetID); err != nil {
		return InternalError(c, "failed to delete asset")
	}

	return NoContent(c)
}
