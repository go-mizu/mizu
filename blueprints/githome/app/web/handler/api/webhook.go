package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/webhooks"
	"github.com/go-mizu/mizu"
)

// WebhookHandler handles webhook endpoints
type WebhookHandler struct {
	webhooks webhooks.API
	repos    repos.API
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(webhooks webhooks.API, repos repos.API) *WebhookHandler {
	return &WebhookHandler{webhooks: webhooks, repos: repos}
}

// ListRepoWebhooks handles GET /repos/{owner}/{repo}/hooks
func (h *WebhookHandler) ListRepoWebhooks(c *mizu.Ctx) error {
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

	pagination := GetPagination(c)
	opts := &webhooks.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	hookList, err := h.webhooks.ListForRepo(c.Context(), owner, repoName, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, hookList)
}

// GetRepoWebhook handles GET /repos/{owner}/{repo}/hooks/{hook_id}
func (h *WebhookHandler) GetRepoWebhook(c *mizu.Ctx) error {
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

	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	hook, err := h.webhooks.GetForRepo(c.Context(), owner, repoName, hookID)
	if err != nil {
		if err == webhooks.ErrNotFound {
			return NotFound(c, "Hook")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, hook)
}

// CreateRepoWebhook handles POST /repos/{owner}/{repo}/hooks
func (h *WebhookHandler) CreateRepoWebhook(c *mizu.Ctx) error {
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

	var in webhooks.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	hook, err := h.webhooks.CreateForRepo(c.Context(), owner, repoName, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, hook)
}

// UpdateRepoWebhook handles PATCH /repos/{owner}/{repo}/hooks/{hook_id}
func (h *WebhookHandler) UpdateRepoWebhook(c *mizu.Ctx) error {
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

	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	var in webhooks.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	hook, err := h.webhooks.UpdateForRepo(c.Context(), owner, repoName, hookID, &in)
	if err != nil {
		if err == webhooks.ErrNotFound {
			return NotFound(c, "Hook")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, hook)
}

// DeleteRepoWebhook handles DELETE /repos/{owner}/{repo}/hooks/{hook_id}
func (h *WebhookHandler) DeleteRepoWebhook(c *mizu.Ctx) error {
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

	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	if err := h.webhooks.DeleteForRepo(c.Context(), owner, repoName, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			return NotFound(c, "Hook")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// PingRepoWebhook handles POST /repos/{owner}/{repo}/hooks/{hook_id}/pings
func (h *WebhookHandler) PingRepoWebhook(c *mizu.Ctx) error {
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

	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	if err := h.webhooks.PingRepo(c.Context(), owner, repoName, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			return NotFound(c, "Hook")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// TestRepoWebhook handles POST /repos/{owner}/{repo}/hooks/{hook_id}/tests
func (h *WebhookHandler) TestRepoWebhook(c *mizu.Ctx) error {
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

	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	if err := h.webhooks.TestRepo(c.Context(), owner, repoName, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			return NotFound(c, "Hook")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListWebhookDeliveries handles GET /repos/{owner}/{repo}/hooks/{hook_id}/deliveries
func (h *WebhookHandler) ListWebhookDeliveries(c *mizu.Ctx) error {
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

	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	pagination := GetPagination(c)
	opts := &webhooks.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	deliveries, err := h.webhooks.ListDeliveriesForRepo(c.Context(), owner, repoName, hookID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, deliveries)
}

// GetWebhookDelivery handles GET /repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}
func (h *WebhookHandler) GetWebhookDelivery(c *mizu.Ctx) error {
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

	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	deliveryID, err := ParamInt64(c, "delivery_id")
	if err != nil {
		return BadRequest(c, "Invalid delivery ID")
	}

	delivery, err := h.webhooks.GetDeliveryForRepo(c.Context(), owner, repoName, hookID, deliveryID)
	if err != nil {
		if err == webhooks.ErrDeliveryNotFound {
			return NotFound(c, "Delivery")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, delivery)
}

// RedeliverWebhook handles POST /repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}/attempts
func (h *WebhookHandler) RedeliverWebhook(c *mizu.Ctx) error {
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

	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	deliveryID, err := ParamInt64(c, "delivery_id")
	if err != nil {
		return BadRequest(c, "Invalid delivery ID")
	}

	delivery, err := h.webhooks.RedeliverForRepo(c.Context(), owner, repoName, hookID, deliveryID)
	if err != nil {
		if err == webhooks.ErrDeliveryNotFound {
			return NotFound(c, "Delivery")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Accepted(c, delivery)
}

// ListOrgWebhooks handles GET /orgs/{org}/hooks
func (h *WebhookHandler) ListOrgWebhooks(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	pagination := GetPagination(c)
	opts := &webhooks.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	hookList, err := h.webhooks.ListForOrg(c.Context(), org, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, hookList)
}

// GetOrgWebhook handles GET /orgs/{org}/hooks/{hook_id}
func (h *WebhookHandler) GetOrgWebhook(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	hook, err := h.webhooks.GetForOrg(c.Context(), org, hookID)
	if err != nil {
		if err == webhooks.ErrNotFound {
			return NotFound(c, "Hook")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, hook)
}

// CreateOrgWebhook handles POST /orgs/{org}/hooks
func (h *WebhookHandler) CreateOrgWebhook(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")

	var in webhooks.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	hook, err := h.webhooks.CreateForOrg(c.Context(), org, &in)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Created(c, hook)
}

// UpdateOrgWebhook handles PATCH /orgs/{org}/hooks/{hook_id}
func (h *WebhookHandler) UpdateOrgWebhook(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	var in webhooks.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	hook, err := h.webhooks.UpdateForOrg(c.Context(), org, hookID, &in)
	if err != nil {
		if err == webhooks.ErrNotFound {
			return NotFound(c, "Hook")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, hook)
}

// DeleteOrgWebhook handles DELETE /orgs/{org}/hooks/{hook_id}
func (h *WebhookHandler) DeleteOrgWebhook(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	if err := h.webhooks.DeleteForOrg(c.Context(), org, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			return NotFound(c, "Hook")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// PingOrgWebhook handles POST /orgs/{org}/hooks/{hook_id}/pings
func (h *WebhookHandler) PingOrgWebhook(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	if err := h.webhooks.PingOrg(c.Context(), org, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			return NotFound(c, "Hook")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return NoContent(c)
}

// ListOrgWebhookDeliveries handles GET /orgs/{org}/hooks/{hook_id}/deliveries
func (h *WebhookHandler) ListOrgWebhookDeliveries(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	pagination := GetPagination(c)
	opts := &webhooks.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	deliveries, err := h.webhooks.ListDeliveriesForOrg(c.Context(), org, hookID, opts)
	if err != nil {
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, deliveries)
}

// GetOrgWebhookDelivery handles GET /orgs/{org}/hooks/{hook_id}/deliveries/{delivery_id}
func (h *WebhookHandler) GetOrgWebhookDelivery(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	deliveryID, err := ParamInt64(c, "delivery_id")
	if err != nil {
		return BadRequest(c, "Invalid delivery ID")
	}

	delivery, err := h.webhooks.GetDeliveryForOrg(c.Context(), org, hookID, deliveryID)
	if err != nil {
		if err == webhooks.ErrDeliveryNotFound {
			return NotFound(c, "Delivery")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, delivery)
}

// RedeliverOrgWebhook handles POST /orgs/{org}/hooks/{hook_id}/deliveries/{delivery_id}/attempts
func (h *WebhookHandler) RedeliverOrgWebhook(c *mizu.Ctx) error {
	user := GetUserFromCtx(c)
	if user == nil {
		return Unauthorized(c)
	}

	org := c.Param("org")
	hookID, err := ParamInt64(c, "hook_id")
	if err != nil {
		return BadRequest(c, "Invalid hook ID")
	}

	deliveryID, err := ParamInt64(c, "delivery_id")
	if err != nil {
		return BadRequest(c, "Invalid delivery ID")
	}

	delivery, err := h.webhooks.RedeliverForOrg(c.Context(), org, hookID, deliveryID)
	if err != nil {
		if err == webhooks.ErrDeliveryNotFound {
			return NotFound(c, "Delivery")
		}
		return WriteError(c, http.StatusInternalServerError, err.Error())
	}

	return Accepted(c, delivery)
}
