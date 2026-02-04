package api

import (
	"net/http"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/email/pkg/email"
	"github.com/go-mizu/mizu/blueprints/email/store"
	"github.com/go-mizu/mizu/blueprints/email/types"
	"github.com/google/uuid"
)

// DraftHandler handles draft-specific API endpoints.
type DraftHandler struct {
	store    store.Store
	driver   email.Driver
	fromAddr string
}

// NewDraftHandler creates a new draft handler.
func NewDraftHandler(st store.Store, driver email.Driver, fromAddr string) *DraftHandler {
	return &DraftHandler{store: st, driver: driver, fromAddr: fromAddr}
}

// Save creates a new draft email.
func (h *DraftHandler) Save(c *mizu.Ctx) error {
	var req types.ComposeRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	now := time.Now().UTC()
	emailID := uuid.New().String()
	messageID := generateMessageID()
	threadID := req.ThreadID
	if threadID == "" {
		threadID = uuid.New().String()
	}

	settings, _ := h.store.GetSettings(c.Context())
	fromName := "Me"
	fromAddress := "me@email.local"
	if settings != nil {
		if settings.DisplayName != "" {
			fromName = settings.DisplayName
		}
		if settings.EmailAddress != "" {
			fromAddress = settings.EmailAddress
		}
	}

	draft := &types.Email{
		ID:           emailID,
		ThreadID:     threadID,
		MessageID:    messageID,
		InReplyTo:    req.InReplyTo,
		FromAddress:  fromAddress,
		FromName:     fromName,
		ToAddresses:  req.To,
		CCAddresses:  req.CC,
		BCCAddresses: req.BCC,
		Subject:      req.Subject,
		BodyHTML:     req.BodyHTML,
		BodyText:     req.BodyText,
		Snippet:      generateSnippet(req.BodyText),
		IsDraft:      true,
		IsRead:       true,
		Labels:       []string{"all", "drafts"},
		ReceivedAt:   now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.store.CreateEmail(c.Context(), draft); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save draft"})
	}

	return c.JSON(http.StatusCreated, draft)
}

// Update updates an existing draft.
func (h *DraftHandler) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "draft id is required"})
	}

	email, err := h.store.GetEmail(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "draft not found"})
	}
	if !email.IsDraft {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email is not a draft"})
	}

	var req types.ComposeRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	updates := map[string]any{
		"subject":       req.Subject,
		"body_html":     req.BodyHTML,
		"body_text":     req.BodyText,
		"snippet":       generateSnippet(req.BodyText),
		"to_addresses":  req.To,
		"cc_addresses":  req.CC,
		"bcc_addresses": req.BCC,
		"updated_at":    time.Now().UTC(),
	}

	if err := h.store.UpdateEmail(c.Context(), id, updates); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update draft"})
	}

	updated, err := h.store.GetEmail(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch updated draft"})
	}

	return c.JSON(http.StatusOK, updated)
}

// Delete permanently removes a draft.
func (h *DraftHandler) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "draft id is required"})
	}

	email, err := h.store.GetEmail(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "draft not found"})
	}
	if !email.IsDraft {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email is not a draft"})
	}

	// Drafts are always permanently deleted
	if err := h.store.DeleteEmail(c.Context(), id, true); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete draft"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "draft deleted"})
}

// Send converts a draft into a sent email, delivering via the configured driver.
func (h *DraftHandler) Send(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "draft id is required"})
	}

	draft, err := h.store.GetEmail(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "draft not found"})
	}
	if !draft.IsDraft {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email is not a draft"})
	}

	if len(draft.ToAddresses) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "draft has no recipients"})
	}

	// Build the message for the email driver
	from := h.resolveFrom(c)
	msg := &email.Message{
		From:     from,
		To:       recipientAddresses(draft.ToAddresses),
		CC:       recipientAddresses(draft.CCAddresses),
		BCC:      recipientAddresses(draft.BCCAddresses),
		Subject:  draft.Subject,
		HTMLBody: draft.BodyHTML,
		TextBody: draft.BodyText,
		Headers:  map[string]string{"Message-ID": draft.MessageID},
	}
	if draft.InReplyTo != "" {
		msg.Headers["In-Reply-To"] = draft.InReplyTo
	}

	// Send via driver
	result, err := h.driver.Send(c.Context(), msg)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send: " + err.Error()})
	}

	now := time.Now().UTC()

	// Update email fields
	updates := map[string]any{
		"is_draft": false,
		"is_sent":  true,
		"sent_at":  now,
	}

	if err := h.store.UpdateEmail(c.Context(), id, updates); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update draft"})
	}

	// Swap labels: remove drafts, add sent
	h.store.RemoveEmailLabel(c.Context(), id, "drafts")
	h.store.AddEmailLabel(c.Context(), id, "sent")

	sent, err := h.store.GetEmail(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch sent email"})
	}

	// Create contacts for recipients
	for _, rcpt := range draft.ToAddresses {
		if rcpt.Address != "" {
			contacts, listErr := h.store.ListContacts(c.Context(), rcpt.Address)
			if listErr != nil || len(contacts) == 0 {
				_ = h.store.CreateContact(c.Context(), &types.Contact{
					ID:           uuid.New().String(),
					Email:        rcpt.Address,
					Name:         rcpt.Name,
					ContactCount: 1,
					CreatedAt:    now,
				})
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"email":              sent,
		"provider_message_id": result.MessageID,
	})
}

// resolveFrom returns the from address, preferring settings over the CLI flag.
func (h *DraftHandler) resolveFrom(c *mizu.Ctx) string {
	settings, _ := h.store.GetSettings(c.Context())
	if settings != nil && settings.EmailAddress != "" {
		name := settings.DisplayName
		if name != "" {
			return name + " <" + settings.EmailAddress + ">"
		}
		return settings.EmailAddress
	}
	if h.fromAddr != "" {
		return h.fromAddr
	}
	return "me@email.local"
}

// recipientAddresses extracts email addresses from a slice of Recipients.
func recipientAddresses(recipients []types.Recipient) []string {
	addrs := make([]string, 0, len(recipients))
	for _, r := range recipients {
		if r.Address != "" {
			addrs = append(addrs, r.Address)
		}
	}
	return addrs
}
