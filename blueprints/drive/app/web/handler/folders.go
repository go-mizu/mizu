package handler

import (
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/accounts"
	"github.com/go-mizu/blueprints/drive/feature/files"
	"github.com/go-mizu/blueprints/drive/feature/folders"
)

// Folders handles folder endpoints.
type Folders struct {
	folders  folders.API
	files    files.API
	accounts accounts.API
}

// NewFolders creates a new folders handler.
func NewFolders(folders folders.API, files files.API) *Folders {
	return &Folders{folders: folders, files: files}
}

// Create creates a folder.
func (h *Folders) Create(c *mizu.Ctx) error {
	accountID := GetAccountID(c)
	if accountID == "" {
		cookie, _ := c.Request().Cookie("session")
		if cookie == nil {
			return Unauthorized(c, "Not authenticated")
		}
		// For now, use empty - the actual auth will be handled by middleware
	}

	var in folders.CreateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	// Get account ID from session cookie
	cookie, err := c.Request().Cookie("session")
	if err != nil {
		return Unauthorized(c, "Not authenticated")
	}

	// We need to get accountID - for now we'll use a simplified approach
	// In real implementation, this would be extracted from middleware context
	_ = cookie

	// For now, create folder with empty owner (will fail - needs proper auth)
	folder, err := h.folders.Create(c.Request().Context(), accountID, &in)
	if err != nil {
		switch err {
		case folders.ErrNameTaken:
			return Conflict(c, "A folder with this name already exists")
		case folders.ErrInvalidParent:
			return BadRequest(c, "Invalid parent folder")
		case folders.ErrNotOwner:
			return Forbidden(c, "Not folder owner")
		default:
			return InternalError(c, "Failed to create folder")
		}
	}

	return Created(c, folder)
}

// Get retrieves folder metadata.
func (h *Folders) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	folder, err := h.folders.GetByID(c.Request().Context(), id)
	if err != nil {
		return NotFound(c, "Folder")
	}

	return OK(c, folder)
}

// Contents lists folder contents.
func (h *Folders) Contents(c *mizu.Ctx) error {
	id := c.Param("id")

	folder, err := h.folders.GetByID(c.Request().Context(), id)
	if err != nil {
		return NotFound(c, "Folder")
	}

	// Parse query params
	limit := 50
	offset := 0
	orderBy := "name"
	order := "asc"

	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	if s := c.Query("sort"); s != "" {
		orderBy = s
	}
	if o := c.Query("order"); o == "desc" {
		order = "desc"
	}

	// List subfolders
	folderList, err := h.folders.List(c.Request().Context(), folder.OwnerID, &folders.ListIn{
		ParentID: id,
		Trashed:  false,
		Limit:    limit,
		Offset:   offset,
		OrderBy:  orderBy,
		Order:    order,
	})
	if err != nil {
		return InternalError(c, "Failed to list folders")
	}

	// List files
	fileList, err := h.files.List(c.Request().Context(), folder.OwnerID, &files.ListIn{
		FolderID: id,
		Trashed:  false,
		Limit:    limit,
		Offset:   offset,
		OrderBy:  orderBy,
		Order:    order,
	})
	if err != nil {
		return InternalError(c, "Failed to list files")
	}

	return OK(c, map[string]any{
		"folder":  folder,
		"folders": folderList,
		"files":   fileList,
	})
}

// Tree returns the folder tree.
func (h *Folders) Tree(c *mizu.Ctx) error {
	id := c.Param("id")

	folder, err := h.folders.GetByID(c.Request().Context(), id)
	if err != nil {
		return NotFound(c, "Folder")
	}

	tree, err := h.folders.GetTree(c.Request().Context(), folder.OwnerID, id)
	if err != nil {
		return InternalError(c, "Failed to get folder tree")
	}

	return OK(c, tree)
}

// Update updates folder metadata.
func (h *Folders) Update(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := GetAccountID(c)

	var in folders.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	folder, err := h.folders.Update(c.Request().Context(), id, accountID, &in)
	if err != nil {
		switch err {
		case folders.ErrNotFound:
			return NotFound(c, "Folder")
		case folders.ErrNotOwner:
			return Forbidden(c, "Not folder owner")
		case folders.ErrNameTaken:
			return Conflict(c, "A folder with this name already exists")
		case folders.ErrCannotMove:
			return BadRequest(c, "Cannot modify root folder")
		default:
			return InternalError(c, "Failed to update folder")
		}
	}

	return OK(c, folder)
}

// Move moves a folder.
func (h *Folders) Move(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := GetAccountID(c)

	var in struct {
		ParentID string `json:"parent_id"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	folder, err := h.folders.Move(c.Request().Context(), id, accountID, in.ParentID)
	if err != nil {
		switch err {
		case folders.ErrNotFound:
			return NotFound(c, "Folder")
		case folders.ErrNotOwner:
			return Forbidden(c, "Not folder owner")
		case folders.ErrCannotMove:
			return BadRequest(c, "Cannot move folder into itself or its descendants")
		case folders.ErrInvalidParent:
			return BadRequest(c, "Invalid destination folder")
		case folders.ErrNameTaken:
			return Conflict(c, "A folder with this name already exists in destination")
		default:
			return InternalError(c, "Failed to move folder")
		}
	}

	return OK(c, folder)
}

// Delete moves a folder to trash.
func (h *Folders) Delete(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := GetAccountID(c)

	if err := h.folders.Delete(c.Request().Context(), id, accountID); err != nil {
		switch err {
		case folders.ErrNotFound:
			return NotFound(c, "Folder")
		case folders.ErrNotOwner:
			return Forbidden(c, "Not folder owner")
		case folders.ErrCannotMove:
			return BadRequest(c, "Cannot delete root folder")
		default:
			return InternalError(c, "Failed to delete folder")
		}
	}

	return OK(c, map[string]string{"message": "Folder moved to trash"})
}

// Star stars a folder.
func (h *Folders) Star(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := GetAccountID(c)

	if err := h.folders.Star(c.Request().Context(), id, accountID, true); err != nil {
		return InternalError(c, "Failed to star folder")
	}

	return OK(c, map[string]bool{"starred": true})
}

// Unstar unstars a folder.
func (h *Folders) Unstar(c *mizu.Ctx) error {
	id := c.Param("id")
	accountID := GetAccountID(c)

	if err := h.folders.Star(c.Request().Context(), id, accountID, false); err != nil {
		return InternalError(c, "Failed to unstar folder")
	}

	return OK(c, map[string]bool{"starred": false})
}

// Root returns the root folder contents.
func (h *Folders) Root(c *mizu.Ctx) error {
	accountID := GetAccountID(c)

	root, err := h.folders.EnsureRoot(c.Request().Context(), accountID)
	if err != nil {
		return InternalError(c, "Failed to get root folder")
	}

	// Redirect to contents
	return h.contentsForFolder(c, root)
}

// ListStarred lists starred items.
func (h *Folders) ListStarred(c *mizu.Ctx) error {
	accountID := GetAccountID(c)

	starred := true
	folderList, err := h.folders.List(c.Request().Context(), accountID, &folders.ListIn{
		Starred: &starred,
		Trashed: false,
	})
	if err != nil {
		return InternalError(c, "Failed to list starred folders")
	}

	fileList, err := h.files.List(c.Request().Context(), accountID, &files.ListIn{
		Starred: &starred,
		Trashed: false,
	})
	if err != nil {
		return InternalError(c, "Failed to list starred files")
	}

	return OK(c, map[string]any{
		"folders": folderList,
		"files":   fileList,
	})
}

func (h *Folders) contentsForFolder(c *mizu.Ctx, folder *folders.Folder) error {
	folderList, _ := h.folders.List(c.Request().Context(), folder.OwnerID, &folders.ListIn{
		ParentID: folder.ID,
		Trashed:  false,
	})

	fileList, _ := h.files.List(c.Request().Context(), folder.OwnerID, &files.ListIn{
		FolderID: folder.ID,
		Trashed:  false,
	})

	return OK(c, map[string]any{
		"folder":  folder,
		"folders": folderList,
		"files":   fileList,
	})
}
