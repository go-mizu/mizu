package api

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/workspace/feature/export"
)

// Export handles export endpoints.
type Export struct {
	exports   *export.Service
	getUserID func(c *mizu.Ctx) string
}

// NewExport creates a new Export handler.
func NewExport(exports *export.Service, getUserID func(c *mizu.Ctx) string) *Export {
	return &Export{exports: exports, getUserID: getUserID}
}

// ExportPage exports a page to the specified format.
func (h *Export) ExportPage(c *mizu.Ctx) error {
	pageID := c.Param("id")
	userID := h.getUserID(c)

	var req export.Request
	if err := c.BindJSON(&req, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	req.PageID = pageID

	result, err := h.exports.Export(c.Request().Context(), userID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// Download downloads an exported file.
func (h *Export) Download(c *mizu.Ctx) error {
	exportID := c.Param("id")

	reader, exp, err := h.exports.Download(c.Request().Context(), exportID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "export not found"})
	}
	defer reader.Close()

	// Determine content type from filename
	contentType := export.DetectContentType(exp.Filename)

	// Set headers for download
	c.Header().Set("Content-Type", contentType)
	c.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", exp.Filename))
	c.Header().Set("Content-Length", fmt.Sprintf("%d", exp.Size))

	// Stream the file
	w := c.Writer()
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, reader)
	return err
}

// GetExport returns export status/info.
func (h *Export) GetExport(c *mizu.Ctx) error {
	exportID := c.Param("id")

	exp, err := h.exports.GetExport(c.Request().Context(), exportID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "export not found"})
	}

	return c.JSON(http.StatusOK, exp)
}
