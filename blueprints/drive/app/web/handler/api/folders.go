package api

import (
	"net/http"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/activity"
	"github.com/go-mizu/blueprints/drive/feature/files"
	"github.com/go-mizu/blueprints/drive/feature/folders"
)

// Folders handles folder endpoints.
type Folders struct {
	folders   folders.API
	files     files.API
	activity  activity.API
	getUserID func(*mizu.Ctx) string
}

// NewFolders creates a new Folders handler.
func NewFolders(folders folders.API, activity activity.API, getUserID func(*mizu.Ctx) string) *Folders {
	return &Folders{
		folders:   folders,
		activity:  activity,
		getUserID: getUserID,
	}
}

// List lists root folders.
func (h *Folders) List(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	parentID := c.Query("parent_id")

	foldersList, err := h.folders.ListByParent(c.Context(), userID, parentID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, foldersList)
}

// Create creates a new folder.
func (h *Folders) Create(c *mizu.Ctx) error {
	userID := h.getUserID(c)

	var in folders.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	folder, err := h.folders.Create(c.Context(), userID, &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	h.activity.Log(c.Context(), userID, &activity.LogIn{
		Action:       activity.ActionFolderCreate,
		ResourceType: "folder",
		ResourceID:   folder.ID,
		ResourceName: folder.Name,
	})

	return c.JSON(http.StatusCreated, folder)
}

// Get gets a folder by ID.
func (h *Folders) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	folder, err := h.folders.GetByID(c.Context(), id)
	if err != nil {
		if err == folders.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "folder not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, folder)
}

// Contents lists folder contents (folders and files).
func (h *Folders) Contents(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	subfolders, err := h.folders.ListByParent(c.Context(), userID, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Files would need to be fetched here too
	// For now, return just folders
	return c.JSON(http.StatusOK, map[string]any{
		"folders": subfolders,
		"files":   []any{}, // TODO: fetch files
	})
}

// Update updates a folder.
func (h *Folders) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	var in folders.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	folder, err := h.folders.Update(c.Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if in.Name != nil {
		h.activity.Log(c.Context(), userID, &activity.LogIn{
			Action:       activity.ActionFolderRename,
			ResourceType: "folder",
			ResourceID:   id,
			ResourceName: folder.Name,
		})
	}

	return c.JSON(http.StatusOK, folder)
}

// Delete deletes a folder.
func (h *Folders) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	folder, _ := h.folders.GetByID(c.Context(), id)

	if err := h.folders.Delete(c.Context(), id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if folder != nil {
		h.activity.Log(c.Context(), userID, &activity.LogIn{
			Action:       activity.ActionFolderDelete,
			ResourceType: "folder",
			ResourceID:   id,
			ResourceName: folder.Name,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Move moves a folder.
func (h *Folders) Move(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	var in folders.MoveIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return c.JSON(http.StatusBadRequest, errResponse("invalid request body"))
	}

	folder, err := h.folders.Move(c.Context(), id, &in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	h.activity.Log(c.Context(), userID, &activity.LogIn{
		Action:       activity.ActionFolderMove,
		ResourceType: "folder",
		ResourceID:   folder.ID,
		ResourceName: folder.Name,
	})

	return c.JSON(http.StatusOK, folder)
}

// Star stars a folder.
func (h *Folders) Star(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	if err := h.folders.Star(c.Context(), id, userID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Unstar unstars a folder.
func (h *Folders) Unstar(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	if err := h.folders.Unstar(c.Context(), id, userID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Trash moves a folder to trash.
func (h *Folders) Trash(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	folder, _ := h.folders.GetByID(c.Context(), id)

	if err := h.folders.Trash(c.Context(), id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if folder != nil {
		h.activity.Log(c.Context(), userID, &activity.LogIn{
			Action:       activity.ActionFolderTrash,
			ResourceType: "folder",
			ResourceID:   id,
			ResourceName: folder.Name,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Restore restores a folder from trash.
func (h *Folders) Restore(c *mizu.Ctx) error {
	id := c.Param("id")
	userID := h.getUserID(c)

	folder, _ := h.folders.GetByID(c.Context(), id)

	if err := h.folders.Restore(c.Context(), id); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if folder != nil {
		h.activity.Log(c.Context(), userID, &activity.LogIn{
			Action:       activity.ActionFolderRestore,
			ResourceType: "folder",
			ResourceID:   id,
			ResourceName: folder.Name,
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// Path returns the path to a folder.
func (h *Folders) Path(c *mizu.Ctx) error {
	id := c.Param("id")

	path, err := h.folders.GetPath(c.Context(), id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, path)
}
