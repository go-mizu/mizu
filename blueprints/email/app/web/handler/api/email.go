package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/email/pkg/email"
	"github.com/go-mizu/mizu/blueprints/email/store"
	"github.com/go-mizu/mizu/blueprints/email/types"
	"github.com/google/uuid"
)

// EmailHandler handles email API endpoints.
type EmailHandler struct {
	store    store.Store
	driver   email.Driver
	fromAddr string
}

// NewEmailHandler creates a new email handler.
func NewEmailHandler(st store.Store, driver email.Driver, fromAddr string) *EmailHandler {
	return &EmailHandler{store: st, driver: driver, fromAddr: fromAddr}
}

// List returns a paginated list of emails filtered by label, query, read/starred status.
func (h *EmailHandler) List(c *mizu.Ctx) error {
	filter := store.EmailFilter{
		LabelID: c.Query("label"),
		Query:   c.Query("q"),
		Page:    1,
		PerPage: 50,
	}

	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			filter.Page = v
		}
	}
	if pp := c.Query("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 && v <= 100 {
			filter.PerPage = v
		}
	}
	if ir := c.Query("is_read"); ir != "" {
		val := ir == "true"
		filter.IsRead = &val
	}
	if is := c.Query("is_starred"); is != "" {
		val := is == "true"
		filter.IsStarred = &val
	}

	resp, err := h.store.ListEmails(c.Context(), filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list emails"})
	}

	return c.JSON(http.StatusOK, resp)
}

// Get returns a single email by ID along with thread context.
func (h *EmailHandler) Get(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email id is required"})
	}

	email, err := h.store.GetEmail(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "email not found"})
	}

	var thread *types.Thread
	if email.ThreadID != "" {
		thread, _ = h.store.GetThread(c.Context(), email.ThreadID)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"email":  email,
		"thread": thread,
	})
}

// Create handles composing and sending a new email.
func (h *EmailHandler) Create(c *mizu.Ctx) error {
	var req types.ComposeRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if len(req.To) == 0 && !req.IsDraft {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "at least one recipient is required"})
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

	newEmail := &types.Email{
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
		IsDraft:      req.IsDraft,
		IsSent:       !req.IsDraft,
		IsRead:       true,
		Labels:       []string{"all"},
		SentAt:       &now,
		ReceivedAt:   now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	var providerMsgID string
	if !req.IsDraft {
		newEmail.Labels = append(newEmail.Labels, "sent")

		// Send via driver
		from := h.resolveFrom(c)
		msg := &email.Message{
			From:     from,
			To:       recipientAddresses(newEmail.ToAddresses),
			CC:       recipientAddresses(newEmail.CCAddresses),
			BCC:      recipientAddresses(newEmail.BCCAddresses),
			Subject:  newEmail.Subject,
			HTMLBody: newEmail.BodyHTML,
			TextBody: newEmail.BodyText,
			Headers:  map[string]string{"Message-ID": newEmail.MessageID},
		}
		result, sendErr := h.driver.Send(c.Context(), msg)
		if sendErr != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send: " + sendErr.Error()})
		}
		providerMsgID = result.MessageID
	} else {
		newEmail.Labels = append(newEmail.Labels, "drafts")
		newEmail.SentAt = nil
	}

	if err := h.store.CreateEmail(c.Context(), newEmail); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create email"})
	}

	// Create contacts for new recipients
	for _, rcpt := range req.To {
		h.ensureContact(c, rcpt)
	}
	for _, rcpt := range req.CC {
		h.ensureContact(c, rcpt)
	}

	resp := map[string]any{"email": newEmail}
	if providerMsgID != "" {
		resp["provider_message_id"] = providerMsgID
	}
	return c.JSON(http.StatusCreated, resp)
}

// Update partially updates an email's metadata (read, starred, important, labels).
func (h *EmailHandler) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email id is required"})
	}

	// Verify email exists
	if _, err := h.store.GetEmail(c.Context(), id); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "email not found"})
	}

	var updates map[string]any
	if err := c.BindJSON(&updates, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Only allow specific fields to be updated
	allowed := map[string]bool{
		"is_read":      true,
		"is_starred":   true,
		"is_important": true,
		"labels":       true,
	}
	filtered := make(map[string]any)
	for k, v := range updates {
		if allowed[k] {
			filtered[k] = v
		}
	}
	filtered["updated_at"] = time.Now().UTC()

	if err := h.store.UpdateEmail(c.Context(), id, filtered); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update email"})
	}

	email, err := h.store.GetEmail(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch updated email"})
	}

	return c.JSON(http.StatusOK, email)
}

// Delete removes an email. If permanent=true query param is set, hard deletes; otherwise moves to trash.
func (h *EmailHandler) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email id is required"})
	}

	if _, err := h.store.GetEmail(c.Context(), id); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "email not found"})
	}

	permanent := c.Query("permanent") == "true"
	if err := h.store.DeleteEmail(c.Context(), id, permanent); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete email"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "email deleted"})
}

// Reply creates a reply email in the same thread as the original.
func (h *EmailHandler) Reply(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email id is required"})
	}

	original, err := h.store.GetEmail(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "original email not found"})
	}

	var req types.ComposeRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if len(req.To) == 0 {
		// Default reply to the original sender
		req.To = []types.Recipient{{Name: original.FromName, Address: original.FromAddress}}
	}

	now := time.Now().UTC()
	emailID := uuid.New().String()
	messageID := generateMessageID()

	subject := req.Subject
	if subject == "" {
		subject = original.Subject
		if !strings.HasPrefix(strings.ToLower(subject), "re:") {
			subject = "Re: " + subject
		}
	}

	// Build references chain
	refs := make([]string, 0, len(original.References)+1)
	refs = append(refs, original.References...)
	refs = append(refs, original.MessageID)

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

	reply := &types.Email{
		ID:           emailID,
		ThreadID:     original.ThreadID,
		MessageID:    messageID,
		InReplyTo:    original.MessageID,
		References:   refs,
		FromAddress:  fromAddress,
		FromName:     fromName,
		ToAddresses:  req.To,
		CCAddresses:  req.CC,
		BCCAddresses: req.BCC,
		Subject:      subject,
		BodyHTML:     req.BodyHTML,
		BodyText:     req.BodyText,
		Snippet:      generateSnippet(req.BodyText),
		IsSent:       true,
		IsRead:       true,
		Labels:       []string{"all", "sent"},
		SentAt:       &now,
		ReceivedAt:   now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Send via driver
	from := h.resolveFrom(c)
	msg := &email.Message{
		From:     from,
		To:       recipientAddresses(req.To),
		CC:       recipientAddresses(req.CC),
		BCC:      recipientAddresses(req.BCC),
		Subject:  subject,
		HTMLBody: req.BodyHTML,
		TextBody: req.BodyText,
		Headers: map[string]string{
			"Message-ID": messageID,
			"In-Reply-To": original.MessageID,
		},
	}
	if _, sendErr := h.driver.Send(c.Context(), msg); sendErr != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send reply: " + sendErr.Error()})
	}

	if err := h.store.CreateEmail(c.Context(), reply); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create reply"})
	}

	for _, rcpt := range req.To {
		h.ensureContact(c, rcpt)
	}

	return c.JSON(http.StatusCreated, reply)
}

// Forward creates a forwarded copy of an email.
func (h *EmailHandler) Forward(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email id is required"})
	}

	original, err := h.store.GetEmail(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "original email not found"})
	}

	var req types.ComposeRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if len(req.To) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "at least one recipient is required"})
	}

	now := time.Now().UTC()
	emailID := uuid.New().String()
	messageID := generateMessageID()

	subject := req.Subject
	if subject == "" {
		subject = original.Subject
		if !strings.HasPrefix(strings.ToLower(subject), "fwd:") {
			subject = "Fwd: " + subject
		}
	}

	bodyText := req.BodyText
	if bodyText == "" {
		bodyText = original.BodyText
	}
	bodyHTML := req.BodyHTML
	if bodyHTML == "" {
		bodyHTML = original.BodyHTML
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

	fwd := &types.Email{
		ID:           emailID,
		ThreadID:     uuid.New().String(),
		MessageID:    messageID,
		FromAddress:  fromAddress,
		FromName:     fromName,
		ToAddresses:  req.To,
		CCAddresses:  req.CC,
		BCCAddresses: req.BCC,
		Subject:      subject,
		BodyHTML:     bodyHTML,
		BodyText:     bodyText,
		Snippet:      generateSnippet(bodyText),
		IsSent:       true,
		IsRead:       true,
		Labels:       []string{"all", "sent"},
		SentAt:       &now,
		ReceivedAt:   now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Send via driver
	from := h.resolveFrom(c)
	msg := &email.Message{
		From:     from,
		To:       recipientAddresses(req.To),
		CC:       recipientAddresses(req.CC),
		BCC:      recipientAddresses(req.BCC),
		Subject:  subject,
		HTMLBody: bodyHTML,
		TextBody: bodyText,
		Headers:  map[string]string{"Message-ID": messageID},
	}
	if _, sendErr := h.driver.Send(c.Context(), msg); sendErr != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send forward: " + sendErr.Error()})
	}

	if err := h.store.CreateEmail(c.Context(), fwd); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to forward email"})
	}

	for _, rcpt := range req.To {
		h.ensureContact(c, rcpt)
	}

	return c.JSON(http.StatusCreated, fwd)
}

// Batch handles batch operations on multiple emails.
func (h *EmailHandler) Batch(c *mizu.Ctx) error {
	var action types.BatchAction
	if err := c.BindJSON(&action, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if len(action.IDs) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no email ids provided"})
	}
	if action.Action == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "action is required"})
	}

	validActions := map[string]bool{
		"archive": true, "trash": true, "delete": true,
		"read": true, "unread": true,
		"star": true, "unstar": true,
		"important": true, "unimportant": true,
		"add_label": true, "remove_label": true,
	}
	if !validActions[action.Action] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid action: " + action.Action})
	}

	if err := h.store.BatchUpdateEmails(c.Context(), &action); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to perform batch action"})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message":  "batch action completed",
		"affected": len(action.IDs),
	})
}

// Search searches emails by query string.
func (h *EmailHandler) Search(c *mizu.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "search query is required"})
	}

	page := 1
	perPage := 50
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if pp := c.Query("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 && v <= 100 {
			perPage = v
		}
	}

	resp, err := h.store.SearchEmails(c.Context(), q, page, perPage)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "search failed"})
	}

	return c.JSON(http.StatusOK, resp)
}

// Snooze snoozes an email until a specified time.
func (h *EmailHandler) Snooze(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email id is required"})
	}

	var req struct {
		Until time.Time `json:"until"`
	}
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Until.IsZero() {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "snooze time is required"})
	}

	updates := map[string]any{
		"snoozed_until": req.Until,
	}
	if err := h.store.UpdateEmail(c.Context(), id, updates); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to snooze email"})
	}

	// Move from inbox to snoozed
	h.store.RemoveEmailLabel(c.Context(), id, "inbox")
	h.store.AddEmailLabel(c.Context(), id, "snoozed")

	email, _ := h.store.GetEmail(c.Context(), id)
	return c.JSON(http.StatusOK, email)
}

// Unsnooze removes the snooze from an email.
func (h *EmailHandler) Unsnooze(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email id is required"})
	}

	updates := map[string]any{
		"snoozed_until": (*time.Time)(nil),
	}
	if err := h.store.UpdateEmail(c.Context(), id, updates); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to unsnooze email"})
	}

	// Move back to inbox
	h.store.RemoveEmailLabel(c.Context(), id, "snoozed")
	h.store.AddEmailLabel(c.Context(), id, "inbox")

	email, _ := h.store.GetEmail(c.Context(), id)
	return c.JSON(http.StatusOK, email)
}

// ReplyAll creates a reply to all recipients.
func (h *EmailHandler) ReplyAll(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email id is required"})
	}

	original, err := h.store.GetEmail(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "original email not found"})
	}

	var req types.ComposeRequest
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
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

	// Build To: original sender + all original To (minus current user)
	if len(req.To) == 0 {
		toRecipients := []types.Recipient{{Name: original.FromName, Address: original.FromAddress}}
		for _, r := range original.ToAddresses {
			if !strings.EqualFold(r.Address, fromAddress) {
				toRecipients = append(toRecipients, r)
			}
		}
		req.To = toRecipients
	}
	// Build CC: original CC (minus current user)
	if len(req.CC) == 0 {
		for _, r := range original.CCAddresses {
			if !strings.EqualFold(r.Address, fromAddress) {
				req.CC = append(req.CC, r)
			}
		}
	}

	now := time.Now().UTC()
	emailID := uuid.New().String()
	messageID := generateMessageID()

	subject := req.Subject
	if subject == "" {
		subject = original.Subject
		if !strings.HasPrefix(strings.ToLower(subject), "re:") {
			subject = "Re: " + subject
		}
	}

	refs := make([]string, 0, len(original.References)+1)
	refs = append(refs, original.References...)
	refs = append(refs, original.MessageID)

	reply := &types.Email{
		ID:           emailID,
		ThreadID:     original.ThreadID,
		MessageID:    messageID,
		InReplyTo:    original.MessageID,
		References:   refs,
		FromAddress:  fromAddress,
		FromName:     fromName,
		ToAddresses:  req.To,
		CCAddresses:  req.CC,
		BCCAddresses: req.BCC,
		Subject:      subject,
		BodyHTML:     req.BodyHTML,
		BodyText:     req.BodyText,
		Snippet:      generateSnippet(req.BodyText),
		IsSent:       true,
		IsRead:       true,
		Labels:       []string{"all", "sent"},
		SentAt:       &now,
		ReceivedAt:   now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Send via driver
	from := h.resolveFrom(c)
	replyMsg := &email.Message{
		From:     from,
		To:       recipientAddresses(req.To),
		CC:       recipientAddresses(req.CC),
		BCC:      recipientAddresses(req.BCC),
		Subject:  subject,
		HTMLBody: req.BodyHTML,
		TextBody: req.BodyText,
		Headers: map[string]string{
			"Message-ID":  messageID,
			"In-Reply-To": original.MessageID,
		},
	}
	if _, sendErr := h.driver.Send(c.Context(), replyMsg); sendErr != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send reply-all: " + sendErr.Error()})
	}

	if err := h.store.CreateEmail(c.Context(), reply); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create reply"})
	}

	for _, rcpt := range req.To {
		h.ensureContact(c, rcpt)
	}

	return c.JSON(http.StatusCreated, reply)
}

// resolveFrom returns the from address, preferring settings over the CLI flag.
func (h *EmailHandler) resolveFrom(c *mizu.Ctx) string {
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

// ensureContact creates a contact entry if one does not already exist for the given recipient.
func (h *EmailHandler) ensureContact(c *mizu.Ctx, rcpt types.Recipient) {
	if rcpt.Address == "" {
		return
	}
	contacts, err := h.store.ListContacts(c.Context(), rcpt.Address)
	if err != nil || len(contacts) > 0 {
		return
	}
	now := time.Now().UTC()
	_ = h.store.CreateContact(c.Context(), &types.Contact{
		ID:           uuid.New().String(),
		Email:        rcpt.Address,
		Name:         rcpt.Name,
		ContactCount: 1,
		CreatedAt:    now,
	})
}

// generateSnippet returns the first 100 characters of the body text.
func generateSnippet(bodyText string) string {
	s := strings.TrimSpace(bodyText)
	if len(s) > 100 {
		return s[:100]
	}
	return s
}

// generateMessageID returns a message ID in RFC 2822 format.
func generateMessageID() string {
	return fmt.Sprintf("<%s@email.local>", uuid.New().String())
}
