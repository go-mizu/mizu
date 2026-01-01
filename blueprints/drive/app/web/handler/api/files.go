package api

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/activity"
	"github.com/go-mizu/blueprints/drive/feature/files"
)

// Files handles file endpoints.
type Files struct {
	files     files.API
	activity  activity.API
	getUserID func(*mizu.Ctx) string
}

// NewFiles creates a new Files handler.
func NewFiles(files files.API, activity activity.API, getUserID func(*mizu.Ctx) string) *Files {
	return &Files{
		files:     files,
		activity:  activity,
		getUserID: getUserID,
	}
}

// List lists files in a folder.
func (h *Files) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	parentID := c.Query("parent_id")

	filesList, err := h.files.ListByUser(c.Context(), userID, parentID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, filesList)
}

// Create creates a new file (metadata only).
func (h *Files) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in files.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	file, err := h.files.Create(c.Context(), userID, &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Log activity
	h.activity.Log(c.Context(), userID, &activity.LogIn{
		Action:       activity.ActionFileUpload,
		ResourceType: "file",
		ResourceID:   file.ID,
		ResourceName: file.Name,
	})

	return c.JSON(http.StatusCreated, file)
}

// Get gets a file by ID.
func (h *Files) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	file, err := h.files.GetByID(c.Context(), id)
	if err != nil {
		if err == files.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, file)
}

// Update updates a file.
func (h *Files) Update(c *mizu.Ctx) error {
	id := c.Param("id")

	var in files.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	file, err := h.files.Update(c.Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, file)
}

// Delete deletes a file permanently.
func (h *Files) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	file, _ := h.files.GetByID(c.Context(), id)

	if err := h.files.Delete(c.Context(), id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if file != nil {
		h.activity.Log(c.Context(), userID, &activity.LogIn{
			Action:       activity.ActionFileDelete,
			ResourceType: "file",
			ResourceID:   id,
			ResourceName: file.Name,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Download handles file download.
func (h *Files) Download(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	file, err := h.files.GetByID(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
	}

	h.activity.Log(c.Context(), userID, &activity.LogIn{
		Action:       activity.ActionFileDownload,
		ResourceType: "file",
		ResourceID:   id,
		ResourceName: file.Name,
	})

	// Return file info for now (actual download would stream from storage)
	return c.JSON(http.StatusOK, map[string]any{
		"file":        file,
		"storage_key": file.StorageKey,
	})
}

// Copy copies a file.
func (h *Files) Copy(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	var in files.CopyIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	file, err := h.files.Copy(c.Context(), id, userID, &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	h.activity.Log(c.Context(), userID, &activity.LogIn{
		Action:       activity.ActionFileCopy,
		ResourceType: "file",
		ResourceID:   file.ID,
		ResourceName: file.Name,
	})

	return c.JSON(http.StatusCreated, file)
}

// Move moves a file.
func (h *Files) Move(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	var in files.MoveIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	file, err := h.files.Move(c.Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	h.activity.Log(c.Context(), userID, &activity.LogIn{
		Action:       activity.ActionFileMove,
		ResourceType: "file",
		ResourceID:   file.ID,
		ResourceName: file.Name,
	})

	return c.JSON(http.StatusOK, file)
}

// Star stars a file.
func (h *Files) Star(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	if err := h.files.Star(c.Context(), id, userID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Unstar unstars a file.
func (h *Files) Unstar(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	if err := h.files.Unstar(c.Context(), id, userID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Trash moves a file to trash.
func (h *Files) Trash(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	file, _ := h.files.GetByID(c.Context(), id)

	if err := h.files.Trash(c.Context(), id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if file != nil {
		h.activity.Log(c.Context(), userID, &activity.LogIn{
			Action:       activity.ActionFileTrash,
			ResourceType: "file",
			ResourceID:   id,
			ResourceName: file.Name,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Restore restores a file from trash.
func (h *Files) Restore(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	file, _ := h.files.GetByID(c.Context(), id)

	if err := h.files.Restore(c.Context(), id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if file != nil {
		h.activity.Log(c.Context(), userID, &activity.LogIn{
			Action:       activity.ActionFileRestore,
			ResourceType: "file",
			ResourceID:   id,
			ResourceName: file.Name,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// ListVersions lists file versions.
func (h *Files) ListVersions(c *mizu.Ctx) error {
	id := c.Param("id")

	versions, err := h.files.ListVersions(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, versions)
}

// RestoreVersion restores a file version.
func (h *Files) RestoreVersion(c *mizu.Ctx) error {
	id := c.Param("id")
	versionStr := c.Param("version")
	userID := h.getUserID(c)

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid version"})
	}

	file, err := h.files.RestoreVersion(c.Context(), id, version, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, file)
}

// ListStarred lists starred files.
func (h *Files) ListStarred(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	filesList, err := h.files.ListStarred(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, filesList)
}

// ListRecent lists recent files.
func (h *Files) ListRecent(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	filesList, err := h.files.ListRecent(c.Context(), userID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, filesList)
}

// ListTrashed lists trashed files.
func (h *Files) ListTrashed(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	filesList, err := h.files.ListTrashed(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, filesList)
}

// EmptyTrash empties the trash.
func (h *Files) EmptyTrash(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	trashed, err := h.files.ListTrashed(c.Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	for _, file := range trashed {
		h.files.Delete(c.Context(), file.ID)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Search searches files.
func (h *Files) Search(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	query := c.Query("q")

	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query is required"})
	}

	filesList, err := h.files.Search(c.Context(), userID, query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, filesList)
}

// Preview returns preview metadata for a file.
func (h *Files) Preview(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	file, err := h.files.GetByID(c.Context(), id)
	if err != nil {
		if err == files.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Log preview activity
	h.activity.Log(c.Context(), userID, &activity.LogIn{
		Action:       "view",
		ResourceType: "file",
		ResourceID:   id,
		ResourceName: file.Name,
	})

	// Get base URL from request
	scheme := "http"
	if c.Request().TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + c.Request().Host

	// Create preview info
	previewInfo := files.GetPreviewInfo(file, baseURL)

	// Get sibling files for navigation
	parentID := file.ParentID
	siblings, _ := h.files.ListByUser(c.Context(), userID, parentID)

	response := &files.PreviewResponse{
		PreviewInfo: previewInfo,
	}

	// Find prev/next files
	for i, sibling := range siblings {
		if sibling.ID == file.ID {
			if i > 0 {
				response.Siblings.Prev = &files.SiblingFile{
					ID:   siblings[i-1].ID,
					Name: siblings[i-1].Name,
				}
			}
			if i < len(siblings)-1 {
				response.Siblings.Next = &files.SiblingFile{
					ID:   siblings[i+1].ID,
					Name: siblings[i+1].Name,
				}
			}
			break
		}
	}

	return c.JSON(http.StatusOK, response)
}
