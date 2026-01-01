package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/files"
	"github.com/go-mizu/blueprints/drive/feature/folders"
	"github.com/go-mizu/blueprints/drive/feature/shares"
)

// Shares handles share endpoints.
type Shares struct {
	shares  shares.API
	files   files.API
	folders folders.API
}

// NewShares creates a new shares handler.
func NewShares(shares shares.API, files files.API, folders folders.API) *Shares {
	return &Shares{shares: shares, files: files, folders: folders}
}

// Create creates a share.
func (h *Shares) Create(c *mizu.Ctx) error {
	accountID := GetAccountID(c)

	var in shares.CreateShareIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	share, err := h.shares.Create(c.Request().Context(), accountID, &in)
	if err != nil {
		switch err {
		case shares.ErrAlreadyShared:
			return Conflict(c, "Already shared with this user")
		default:
			return InternalError(c, "Failed to create share")
		}
	}

	return Created(c, share)
}

// ListByOwner lists shares created by the user.
func (h *Shares) ListByOwner(c *mizu.Ctx) error {
	accountID := GetAccountID(c)

	shareList, err := h.shares.ListByOwner(c.Request().Context(), accountID)
	if err != nil {
		return InternalError(c, "Failed to list shares")
	}

	return OK(c, shareList)
}

// ListSharedWithMe lists items shared with the user.
func (h *Shares) ListSharedWithMe(c *mizu.Ctx) error {
	accountID := GetAccountID(c)

	items, err := h.shares.ListSharedWithMe(c.Request().Context(), accountID)
	if err != nil {
		return InternalError(c, "Failed to list shared items")
	}

	return OK(c, items)
}

// Delete deletes a share.
func (h *Shares) Delete(c *mizu.Ctx) error {
	accountID := GetAccountID(c)
	id := c.Param("id")

	if err := h.shares.Delete(c.Request().Context(), id, accountID); err != nil {
		switch err {
		case shares.ErrNotFound:
			return NotFound(c, "Share")
		case shares.ErrNotOwner:
			return Forbidden(c, "Not share owner")
		default:
			return InternalError(c, "Failed to delete share")
		}
	}

	return OK(c, map[string]string{"message": "Share deleted"})
}

// CreateLink creates a share link.
func (h *Shares) CreateLink(c *mizu.Ctx) error {
	accountID := GetAccountID(c)

	var in shares.CreateLinkIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	link, err := h.shares.CreateLink(c.Request().Context(), accountID, &in)
	if err != nil {
		return InternalError(c, "Failed to create share link")
	}

	return Created(c, link)
}

// ListLinksForItem lists links for an item.
func (h *Shares) ListLinksForItem(c *mizu.Ctx) error {
	itemType := c.Param("type")
	itemID := c.Param("id")

	links, err := h.shares.ListLinksForItem(c.Request().Context(), itemID, itemType)
	if err != nil {
		return InternalError(c, "Failed to list links")
	}

	return OK(c, links)
}

// UpdateLink updates a share link.
func (h *Shares) UpdateLink(c *mizu.Ctx) error {
	accountID := GetAccountID(c)
	id := c.Param("id")

	var in shares.UpdateLinkIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	link, err := h.shares.UpdateLink(c.Request().Context(), id, accountID, &in)
	if err != nil {
		switch err {
		case shares.ErrLinkNotFound:
			return NotFound(c, "Share link")
		case shares.ErrNotOwner:
			return Forbidden(c, "Not link owner")
		default:
			return InternalError(c, "Failed to update link")
		}
	}

	return OK(c, link)
}

// DeleteLink deletes a share link.
func (h *Shares) DeleteLink(c *mizu.Ctx) error {
	accountID := GetAccountID(c)
	id := c.Param("id")

	if err := h.shares.DeleteLink(c.Request().Context(), id, accountID); err != nil {
		switch err {
		case shares.ErrLinkNotFound:
			return NotFound(c, "Share link")
		case shares.ErrNotOwner:
			return Forbidden(c, "Not link owner")
		default:
			return InternalError(c, "Failed to delete link")
		}
	}

	return OK(c, map[string]string{"message": "Link deleted"})
}

// AccessLink handles public link access.
func (h *Shares) AccessLink(c *mizu.Ctx) error {
	token := c.Param("token")

	link, err := h.shares.GetLinkByToken(c.Request().Context(), token)
	if err != nil {
		switch err {
		case shares.ErrLinkNotFound:
			return NotFound(c, "Share link")
		case shares.ErrLinkExpired:
			return c.JSON(http.StatusGone, Response{
				Error: &Error{Code: "LINK_EXPIRED", Message: "This link has expired"},
			})
		case shares.ErrLinkDisabled:
			return c.JSON(http.StatusGone, Response{
				Error: &Error{Code: "LINK_DISABLED", Message: "This link has been disabled"},
			})
		case shares.ErrDownloadLimit:
			return c.JSON(http.StatusGone, Response{
				Error: &Error{Code: "DOWNLOAD_LIMIT", Message: "Download limit reached"},
			})
		default:
			return InternalError(c, "Failed to access link")
		}
	}

	// Check if password protected
	if link.HasPassword {
		// Check for verification cookie
		cookie, _ := c.Request().Cookie("share_" + token)
		if cookie == nil || cookie.Value != "verified" {
			return c.JSON(http.StatusForbidden, Response{
				Error: &Error{Code: "PASSWORD_REQUIRED", Message: "This link requires a password"},
			})
		}
	}

	// Record access
	h.shares.RecordLinkAccess(c.Request().Context(), token)

	// Return item info
	if link.ItemType == "file" {
		file, err := h.files.GetByID(c.Request().Context(), link.ItemID)
		if err != nil {
			return NotFound(c, "File")
		}
		return OK(c, map[string]any{
			"type":       "file",
			"file":       file,
			"permission": link.Permission,
		})
	} else {
		folder, err := h.folders.GetByID(c.Request().Context(), link.ItemID)
		if err != nil {
			return NotFound(c, "Folder")
		}
		return OK(c, map[string]any{
			"type":       "folder",
			"folder":     folder,
			"permission": link.Permission,
		})
	}
}

// DownloadLink handles download via share link.
func (h *Shares) DownloadLink(c *mizu.Ctx) error {
	token := c.Param("token")

	link, err := h.shares.GetLinkByToken(c.Request().Context(), token)
	if err != nil {
		return NotFound(c, "Share link")
	}

	if !link.AllowDownload {
		return Forbidden(c, "Download not allowed for this link")
	}

	// Check password
	if link.HasPassword {
		cookie, _ := c.Request().Cookie("share_" + token)
		if cookie == nil || cookie.Value != "verified" {
			return Forbidden(c, "Password verification required")
		}
	}

	if link.ItemType != "file" {
		return BadRequest(c, "Cannot download folder via this endpoint")
	}

	reader, file, err := h.files.Open(c.Request().Context(), link.ItemID)
	if err != nil {
		return NotFound(c, "File")
	}
	defer reader.Close()

	c.Writer().Header().Set("Content-Type", file.MimeType)
	c.Writer().Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	c.Writer().Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))

	io.Copy(c.Writer(), reader)
	return nil
}

// VerifyLinkPassword verifies share link password.
func (h *Shares) VerifyLinkPassword(c *mizu.Ctx) error {
	token := c.Param("token")

	var in struct {
		Password string `json:"password"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	valid, err := h.shares.VerifyLinkPassword(c.Request().Context(), token, in.Password)
	if err != nil {
		return NotFound(c, "Share link")
	}

	if !valid {
		return Unauthorized(c, "Invalid password")
	}

	// Set verification cookie
	http.SetCookie(c.Writer(), &http.Cookie{
		Name:     "share_" + token,
		Value:    "verified",
		Path:     "/s/" + token,
		HttpOnly: true,
		Secure:   c.Request().TLS != nil,
		SameSite: http.SameSiteStrictMode,
	})

	return OK(c, map[string]bool{"verified": true})
}
