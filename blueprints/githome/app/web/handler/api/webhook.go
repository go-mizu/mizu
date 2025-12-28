package api

import (
	"strconv"

	"github.com/go-mizu/blueprints/githome/feature/orgs"
	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/users"
	"github.com/go-mizu/blueprints/githome/feature/webhooks"
	"github.com/go-mizu/mizu"
)

// Webhook handles webhook endpoints
type Webhook struct {
	webhooks  webhooks.API
	repos     repos.API
	orgs      orgs.API
	users     users.API
	getUserID func(*mizu.Ctx) string
}

// NewWebhook creates a new webhook handler
func NewWebhook(webhooks webhooks.API, repos repos.API, orgs orgs.API, users users.API, getUserID func(*mizu.Ctx) string) *Webhook {
	return &Webhook{
		webhooks:  webhooks,
		repos:     repos,
		orgs:      orgs,
		users:     users,
		getUserID: getUserID,
	}
}

func (h *Webhook) getRepo(c *mizu.Ctx) (*repos.Repository, error) {
	owner := c.Param("owner")
	name := c.Param("repo")

	user, err := h.users.GetByUsername(c.Context(), owner)
	if err != nil {
		return nil, repos.ErrNotFound
	}

	return h.repos.GetByOwnerAndName(c.Context(), user.ID, "user", name)
}

// ListByRepo lists webhooks for a repository
func (h *Webhook) ListByRepo(c *mizu.Ctx) error {
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

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &webhooks.ListOpts{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	webhookList, err := h.webhooks.ListByRepo(c.Context(), repo.ID, opts)
	if err != nil {
		return InternalError(c, "failed to list webhooks")
	}

	return OK(c, webhookList)
}

// ListByOrg lists webhooks for an organization
func (h *Webhook) ListByOrg(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	slug := c.Param("org")
	org, err := h.orgs.GetBySlug(c.Context(), slug)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &webhooks.ListOpts{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	webhookList, err := h.webhooks.ListByOrg(c.Context(), org.ID, opts)
	if err != nil {
		return InternalError(c, "failed to list webhooks")
	}

	return OK(c, webhookList)
}

// CreateForRepo creates a webhook for a repository
func (h *Webhook) CreateForRepo(c *mizu.Ctx) error {
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

	var in webhooks.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	in.RepoID = repo.ID

	webhook, err := h.webhooks.Create(c.Context(), &in)
	if err != nil {
		switch err {
		case webhooks.ErrMissingURL:
			return BadRequest(c, "webhook URL is required")
		default:
			return InternalError(c, "failed to create webhook")
		}
	}

	return Created(c, webhook)
}

// CreateForOrg creates a webhook for an organization
func (h *Webhook) CreateForOrg(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	slug := c.Param("org")
	org, err := h.orgs.GetBySlug(c.Context(), slug)
	if err != nil {
		return NotFound(c, "organization not found")
	}

	// Check if user is owner or admin
	member, _ := h.orgs.GetMember(c.Context(), org.ID, userID)
	if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
		return Forbidden(c, "insufficient permissions")
	}

	var in webhooks.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	in.OrgID = org.ID

	webhook, err := h.webhooks.Create(c.Context(), &in)
	if err != nil {
		switch err {
		case webhooks.ErrMissingURL:
			return BadRequest(c, "webhook URL is required")
		default:
			return InternalError(c, "failed to create webhook")
		}
	}

	return Created(c, webhook)
}

// Get retrieves a webhook
func (h *Webhook) Get(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	webhookID := c.Param("id")

	webhook, err := h.webhooks.GetByID(c.Context(), webhookID)
	if err != nil {
		return NotFound(c, "webhook not found")
	}

	// Check permission based on webhook type
	if webhook.RepoID != "" {
		if !h.repos.CanAccess(c.Context(), webhook.RepoID, userID, repos.PermissionAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	} else if webhook.OrgID != "" {
		member, _ := h.orgs.GetMember(c.Context(), webhook.OrgID, userID)
		if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	}

	return OK(c, webhook)
}

// Update updates a webhook
func (h *Webhook) Update(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	webhookID := c.Param("id")

	webhook, err := h.webhooks.GetByID(c.Context(), webhookID)
	if err != nil {
		return NotFound(c, "webhook not found")
	}

	// Check permission based on webhook type
	if webhook.RepoID != "" {
		if !h.repos.CanAccess(c.Context(), webhook.RepoID, userID, repos.PermissionAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	} else if webhook.OrgID != "" {
		member, _ := h.orgs.GetMember(c.Context(), webhook.OrgID, userID)
		if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	}

	var in webhooks.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "invalid request body")
	}

	webhook, err = h.webhooks.Update(c.Context(), webhookID, &in)
	if err != nil {
		return InternalError(c, "failed to update webhook")
	}

	return OK(c, webhook)
}

// Delete deletes a webhook
func (h *Webhook) Delete(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	webhookID := c.Param("id")

	webhook, err := h.webhooks.GetByID(c.Context(), webhookID)
	if err != nil {
		return NotFound(c, "webhook not found")
	}

	// Check permission based on webhook type
	if webhook.RepoID != "" {
		if !h.repos.CanAccess(c.Context(), webhook.RepoID, userID, repos.PermissionAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	} else if webhook.OrgID != "" {
		member, _ := h.orgs.GetMember(c.Context(), webhook.OrgID, userID)
		if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	}

	if err := h.webhooks.Delete(c.Context(), webhookID); err != nil {
		return InternalError(c, "failed to delete webhook")
	}

	return NoContent(c)
}

// Ping sends a ping event to a webhook
func (h *Webhook) Ping(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	webhookID := c.Param("id")

	webhook, err := h.webhooks.GetByID(c.Context(), webhookID)
	if err != nil {
		return NotFound(c, "webhook not found")
	}

	// Check permission based on webhook type
	if webhook.RepoID != "" {
		if !h.repos.CanAccess(c.Context(), webhook.RepoID, userID, repos.PermissionAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	} else if webhook.OrgID != "" {
		member, _ := h.orgs.GetMember(c.Context(), webhook.OrgID, userID)
		if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	}

	delivery, err := h.webhooks.Ping(c.Context(), webhookID)
	if err != nil {
		return InternalError(c, "failed to ping webhook")
	}

	return OK(c, delivery)
}

// Test sends a test event to a webhook
func (h *Webhook) Test(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	webhookID := c.Param("id")

	webhook, err := h.webhooks.GetByID(c.Context(), webhookID)
	if err != nil {
		return NotFound(c, "webhook not found")
	}

	// Check permission based on webhook type
	if webhook.RepoID != "" {
		if !h.repos.CanAccess(c.Context(), webhook.RepoID, userID, repos.PermissionAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	} else if webhook.OrgID != "" {
		member, _ := h.orgs.GetMember(c.Context(), webhook.OrgID, userID)
		if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	}

	event := c.Query("event")
	if event == "" {
		event = "push"
	}

	delivery, err := h.webhooks.Test(c.Context(), webhookID, event)
	if err != nil {
		return InternalError(c, "failed to test webhook")
	}

	return OK(c, delivery)
}

// ListDeliveries lists deliveries for a webhook
func (h *Webhook) ListDeliveries(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	webhookID := c.Param("id")

	webhook, err := h.webhooks.GetByID(c.Context(), webhookID)
	if err != nil {
		return NotFound(c, "webhook not found")
	}

	// Check permission based on webhook type
	if webhook.RepoID != "" {
		if !h.repos.CanAccess(c.Context(), webhook.RepoID, userID, repos.PermissionAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	} else if webhook.OrgID != "" {
		member, _ := h.orgs.GetMember(c.Context(), webhook.OrgID, userID)
		if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	}

	page, _ := strconv.Atoi(c.Query("page"))
	perPage, _ := strconv.Atoi(c.Query("per_page"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}

	opts := &webhooks.ListOpts{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	deliveries, err := h.webhooks.ListDeliveries(c.Context(), webhookID, opts)
	if err != nil {
		return InternalError(c, "failed to list deliveries")
	}

	return OK(c, deliveries)
}

// GetDelivery retrieves a delivery
func (h *Webhook) GetDelivery(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	webhookID := c.Param("id")
	deliveryID := c.Param("did")

	webhook, err := h.webhooks.GetByID(c.Context(), webhookID)
	if err != nil {
		return NotFound(c, "webhook not found")
	}

	// Check permission based on webhook type
	if webhook.RepoID != "" {
		if !h.repos.CanAccess(c.Context(), webhook.RepoID, userID, repos.PermissionAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	} else if webhook.OrgID != "" {
		member, _ := h.orgs.GetMember(c.Context(), webhook.OrgID, userID)
		if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	}

	delivery, err := h.webhooks.GetDelivery(c.Context(), deliveryID)
	if err != nil {
		return NotFound(c, "delivery not found")
	}

	return OK(c, delivery)
}

// Redeliver redelivers a webhook delivery
func (h *Webhook) Redeliver(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID == "" {
		return Unauthorized(c, "not authenticated")
	}

	webhookID := c.Param("id")
	deliveryID := c.Param("did")

	webhook, err := h.webhooks.GetByID(c.Context(), webhookID)
	if err != nil {
		return NotFound(c, "webhook not found")
	}

	// Check permission based on webhook type
	if webhook.RepoID != "" {
		if !h.repos.CanAccess(c.Context(), webhook.RepoID, userID, repos.PermissionAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	} else if webhook.OrgID != "" {
		member, _ := h.orgs.GetMember(c.Context(), webhook.OrgID, userID)
		if member == nil || (member.Role != orgs.RoleOwner && member.Role != orgs.RoleAdmin) {
			return Forbidden(c, "insufficient permissions")
		}
	}

	delivery, err := h.webhooks.Redeliver(c.Context(), deliveryID)
	if err != nil {
		return InternalError(c, "failed to redeliver webhook")
	}

	return OK(c, delivery)
}
