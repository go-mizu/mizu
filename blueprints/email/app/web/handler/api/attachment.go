package api

import (
	"io"
	"net/http"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/email/store"
	"github.com/go-mizu/mizu/blueprints/email/types"
	"github.com/google/uuid"
)

// AttachmentHandler handles attachment API endpoints.
type AttachmentHandler struct {
	store store.Store
}

// NewAttachmentHandler creates a new attachment handler.
func NewAttachmentHandler(st store.Store) *AttachmentHandler {
	return &AttachmentHandler{store: st}
}

// Upload handles file upload for an email.
func (h *AttachmentHandler) Upload(c *mizu.Ctx) error {
	emailID := c.Param("id")
	if emailID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email id is required"})
	}

	// Parse multipart form (max 25MB)
	if err := c.Request().ParseMultipartForm(25 << 20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to parse form"})
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "file is required"})
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read file"})
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	attachment := &types.Attachment{
		ID:          uuid.New().String(),
		EmailID:     emailID,
		Filename:    header.Filename,
		ContentType: contentType,
		SizeBytes:   int64(len(data)),
	}

	if err := h.store.CreateAttachment(c.Context(), attachment, data); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save attachment"})
	}

	return c.JSON(http.StatusCreated, attachment)
}

// List returns attachments for an email.
func (h *AttachmentHandler) List(c *mizu.Ctx) error {
	emailID := c.Param("id")
	if emailID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "email id is required"})
	}

	attachments, err := h.store.ListAttachments(c.Context(), emailID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list attachments"})
	}

	return c.JSON(http.StatusOK, attachments)
}

// Download returns attachment data.
func (h *AttachmentHandler) Download(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "attachment id is required"})
	}

	attachment, data, err := h.store.GetAttachment(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "attachment not found"})
	}

	c.Header().Set("Content-Type", attachment.ContentType)
	c.Header().Set("Content-Disposition", "attachment; filename=\""+attachment.Filename+"\"")
	_, writeErr := c.Writer().Write(data)
	return writeErr
}

// Delete removes an attachment.
func (h *AttachmentHandler) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "attachment id is required"})
	}

	if err := h.store.DeleteAttachment(c.Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete attachment"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "attachment deleted"})
}
