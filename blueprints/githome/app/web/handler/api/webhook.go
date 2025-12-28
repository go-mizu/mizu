package api

import (
	"net/http"

	"github.com/go-mizu/blueprints/githome/feature/repos"
	"github.com/go-mizu/blueprints/githome/feature/webhooks"
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
func (h *WebhookHandler) ListRepoWebhooks(w http.ResponseWriter, r *http.Request) {
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

	pagination := GetPaginationParams(r)
	opts := &webhooks.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	hookList, err := h.webhooks.ListForRepo(r.Context(), owner, repoName, opts)
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	hook, err := h.webhooks.GetForRepo(r.Context(), owner, repoName, hookID)
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

	var in webhooks.CreateIn
	if err := DecodeJSON(r, &in); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	hook, err := h.webhooks.CreateForRepo(r.Context(), owner, repoName, &in)
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

	hook, err := h.webhooks.UpdateForRepo(r.Context(), owner, repoName, hookID, &in)
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	if err := h.webhooks.DeleteForRepo(r.Context(), owner, repoName, hookID); err != nil {
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	if err := h.webhooks.PingRepo(r.Context(), owner, repoName, hookID); err != nil {
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

	hookID, err := PathParamInt64(r, "hook_id")
	if err != nil {
		WriteBadRequest(w, "Invalid hook ID")
		return
	}

	if err := h.webhooks.TestRepo(r.Context(), owner, repoName, hookID); err != nil {
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

	deliveries, err := h.webhooks.ListDeliveriesForRepo(r.Context(), owner, repoName, hookID, opts)
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

	delivery, err := h.webhooks.GetDeliveryForRepo(r.Context(), owner, repoName, hookID, deliveryID)
	if err != nil {
		if err == webhooks.ErrDeliveryNotFound {
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

	delivery, err := h.webhooks.RedeliverForRepo(r.Context(), owner, repoName, hookID, deliveryID)
	if err != nil {
		if err == webhooks.ErrDeliveryNotFound {
			WriteNotFound(w, "Delivery")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteAccepted(w, delivery)
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

	hook, err := h.webhooks.GetForOrg(r.Context(), org, hookID)
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

	hook, err := h.webhooks.UpdateForOrg(r.Context(), org, hookID, &in)
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

	if err := h.webhooks.DeleteForOrg(r.Context(), org, hookID); err != nil {
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

	if err := h.webhooks.PingOrg(r.Context(), org, hookID); err != nil {
		if err == webhooks.ErrNotFound {
			WriteNotFound(w, "Hook")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteNoContent(w)
}

// ListOrgWebhookDeliveries handles GET /orgs/{org}/hooks/{hook_id}/deliveries
func (h *WebhookHandler) ListOrgWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
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

	pagination := GetPaginationParams(r)
	opts := &webhooks.ListOpts{
		Page:    pagination.Page,
		PerPage: pagination.PerPage,
	}

	deliveries, err := h.webhooks.ListDeliveriesForOrg(r.Context(), org, hookID, opts)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, deliveries)
}

// GetOrgWebhookDelivery handles GET /orgs/{org}/hooks/{hook_id}/deliveries/{delivery_id}
func (h *WebhookHandler) GetOrgWebhookDelivery(w http.ResponseWriter, r *http.Request) {
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

	deliveryID, err := PathParamInt64(r, "delivery_id")
	if err != nil {
		WriteBadRequest(w, "Invalid delivery ID")
		return
	}

	delivery, err := h.webhooks.GetDeliveryForOrg(r.Context(), org, hookID, deliveryID)
	if err != nil {
		if err == webhooks.ErrDeliveryNotFound {
			WriteNotFound(w, "Delivery")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, delivery)
}

// RedeliverOrgWebhook handles POST /orgs/{org}/hooks/{hook_id}/deliveries/{delivery_id}/attempts
func (h *WebhookHandler) RedeliverOrgWebhook(w http.ResponseWriter, r *http.Request) {
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

	deliveryID, err := PathParamInt64(r, "delivery_id")
	if err != nil {
		WriteBadRequest(w, "Invalid delivery ID")
		return
	}

	delivery, err := h.webhooks.RedeliverForOrg(r.Context(), org, hookID, deliveryID)
	if err != nil {
		if err == webhooks.ErrDeliveryNotFound {
			WriteNotFound(w, "Delivery")
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteAccepted(w, delivery)
}
