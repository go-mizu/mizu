package api

import (
	"net/http"

	"github.com/mizu-framework/mizu/blueprints/githome/feature/repos"
	"github.com/mizu-framework/mizu/blueprints/githome/feature/webhooks"
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

// getRepoFromPath gets repository from path parameters
func (h *WebhookHandler) getRepoFromPath(r *http.Request) (*repos.Repository, error) {
	owner := PathParam(r, "owner")
	repoName := PathParam(r, "repo")
	return h.repos.GetByFullName(r.Context(), owner, repoName)
}

// ListRepoWebhooks handles GET /repos/{owner}/{repo}/hooks
func (h *WebhookHandler) ListRepoWebhooks(w http.ResponseWriter, r *http.Request) {
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

	pagination := GetPaginationParams(r)
	opts := &webhooks.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	hookList, err := h.webhooks.ListForRepo(r.Context(), repo.ID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, hookList)
}

// GetRepoWebhook handles GET /repos/{owner}/{repo}/hooks/{hook_id}
func (h *WebhookHandler) GetRepoWebhook(w http.ResponseWriter, r *http.Request) {
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	hook, err := h.webhooks.GetByID(r.Context(), repo.ID, hookID)
	if err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Hook")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, hook)
}

// CreateRepoWebhook handles POST /repos/{owner}/{repo}/hooks
func (h *WebhookHandler) CreateRepoWebhook(w http.ResponseWriter, r *http.Request) {
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

	var in webhooks.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	hook, err := h.webhooks.CreateForRepo(r.Context(), repo.ID, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, hook)
}

// UpdateRepoWebhook handles PATCH /repos/{owner}/{repo}/hooks/{hook_id}
func (h *WebhookHandler) UpdateRepoWebhook(w http.ResponseWriter, r *http.Request) {
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	var in webhooks.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	hook, err := h.webhooks.Update(r.Context(), repo.ID, hookID, &in)
	if err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Hook")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, hook)
}

// DeleteRepoWebhook handles DELETE /repos/{owner}/{repo}/hooks/{hook_id}
func (h *WebhookHandler) DeleteRepoWebhook(w http.ResponseWriter, r *http.Request) {
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	if err := h.webhooks.Delete(r.Context(), repo.ID, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Hook")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// PingRepoWebhook handles POST /repos/{owner}/{repo}/hooks/{hook_id}/pings
func (h *WebhookHandler) PingRepoWebhook(w http.ResponseWriter, r *http.Request) {
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	if err := h.webhooks.Ping(r.Context(), repo.ID, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Hook")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// TestRepoWebhook handles POST /repos/{owner}/{repo}/hooks/{hook_id}/tests
func (h *WebhookHandler) TestRepoWebhook(w http.ResponseWriter, r *http.Request) {
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	if err := h.webhooks.Test(r.Context(), repo.ID, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Hook")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListWebhookDeliveries handles GET /repos/{owner}/{repo}/hooks/{hook_id}/deliveries
func (h *WebhookHandler) ListWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	pagination := GetPaginationParams(r)
	opts := &webhooks.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	deliveries, err := h.webhooks.ListDeliveries(r.Context(), repo.ID, hookID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, deliveries)
}

// GetWebhookDelivery handles GET /repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}
func (h *WebhookHandler) GetWebhookDelivery(w http.ResponseWriter, r *http.Request) {
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	deliveryID, err := PathParamInt64(r, "delivery_id")
	if err != nil {
		WriteBadRequest(w, "Invalid delivery ID")
		return
	}

	delivery, err := h.webhooks.GetDelivery(r.Context(), repo.ID, hookID, deliveryID)
	if err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Delivery")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, delivery)
}

// RedeliverWebhook handles POST /repos/{owner}/{repo}/hooks/{hook_id}/deliveries/{delivery_id}/attempts
func (h *WebhookHandler) RedeliverWebhook(w http.ResponseWriter, r *http.Request) {
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	deliveryID, err := PathParamInt64(r, "delivery_id")
	if err != nil {
		WriteBadRequest(w, "Invalid delivery ID")
		return
	}

	if err := h.webhooks.Redeliver(r.Context(), repo.ID, hookID, deliveryID); err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Delivery")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteAccepted(w, map[string]string{"message": "Redelivery triggered"})
}

// ListOrgWebhooks handles GET /orgs/{org}/hooks
func (h *WebhookHandler) ListOrgWebhooks(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	pagination := GetPaginationParams(r)
	opts := &webhooks.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	hookList, err := h.webhooks.ListForOrg(r.Context(), org, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, hookList)
}

// GetOrgWebhook handles GET /orgs/{org}/hooks/{hook_id}
func (h *WebhookHandler) GetOrgWebhook(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	hook, err := h.webhooks.GetOrgHook(r.Context(), org, hookID)
	if err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Hook")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, hook)
}

// CreateOrgWebhook handles POST /orgs/{org}/hooks
func (h *WebhookHandler) CreateOrgWebhook(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")

	var in webhooks.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	hook, err := h.webhooks.CreateForOrg(r.Context(), org, &in)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteCreated(w, hook)
}

// UpdateOrgWebhook handles PATCH /orgs/{org}/hooks/{hook_id}
func (h *WebhookHandler) UpdateOrgWebhook(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	var in webhooks.UpdateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	hook, err := h.webhooks.UpdateOrgHook(r.Context(), org, hookID, &in)
	if err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Hook")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, hook)
}

// DeleteOrgWebhook handles DELETE /orgs/{org}/hooks/{hook_id}
func (h *WebhookHandler) DeleteOrgWebhook(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	if err := h.webhooks.DeleteOrgHook(r.Context(), org, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Hook")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// PingOrgWebhook handles POST /orgs/{org}/hooks/{hook_id}/pings
func (h *WebhookHandler) PingOrgWebhook(w http.ResponseWriter, r *http.Request) {
	user := GetUser(r.Context())
	if user == nil {
		WriteUnauthorized(w)
		return
	}

	org := PathParam(r, "org")
	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	if err := h.webhooks.PingOrgHook(r.Context(), org, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Hook")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}
