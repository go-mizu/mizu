package handler

import (
	"io"
	"strconv"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/accounts"
	"github.com/go-mizu/blueprints/drive/feature/files"
)

// Files handles file endpoints.
type Files struct {
	files    files.API
	accounts accounts.API
}

// NewFiles creates a new files handler.
func NewFiles(files files.API, accounts accounts.API) *Files {
	return &Files{files: files, accounts: accounts}
}

// Upload handles file upload.
func (h *Files) Upload(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	// Parse multipart form (max 100MB in memory)
	if err := c.Request().ParseMultipartForm(100 << 20); err != nil {
		return BadRequest(c, "Failed to parse form")
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return BadRequest(c, "File is required")
	}
	defer file.Close()

	in := &files.UploadIn{
		Filename:    header.Filename,
		FolderID:    c.Request().FormValue("folder_id"),
		Description: c.Request().FormValue("description"),
		MimeType:    header.Header.Get("Content-Type"),
		Size:        header.Size,
		Reader:      file,
	}

	f, err := h.files.Upload(c.Request().Context(), accountID, in)
	if err != nil {
		if err == files.ErrQuotaExceeded {
			return BadRequest(c, "Storage quota exceeded")
		}
		return InternalError(c, "Failed to upload file")
	}

	return Created(c, f)
}

// Get retrieves file metadata.
func (h *Files) Get(c *mizu.Ctx) error {
	id := c.Param("id")

	f, err := h.files.GetByID(c.Request().Context(), id)
	if err != nil {
		return NotFound(c, "File")
	}

	return OK(c, f)
}

// Download handles file download.
func (h *Files) Download(c *mizu.Ctx) error {
	id := c.Param("id")

	reader, f, err := h.files.Open(c.Request().Context(), id)
	if err != nil {
		return NotFound(c, "File")
	}
	defer reader.Close()

	// Update accessed time
	h.files.UpdateAccessed(c.Request().Context(), id)

	c.Writer().Header().Set("Content-Type", f.MimeType)
	c.Writer().Header().Set("Content-Disposition", "attachment; filename=\""+f.Name+"\"")
	c.Writer().Header().Set("Content-Length", strconv.FormatInt(f.Size, 10))

	io.Copy(c.Writer(), reader)
	return nil
}

// Update updates file metadata.
func (h *Files) Update(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	id := c.Param("id")

	var in files.UpdateIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	f, err := h.files.Update(c.Request().Context(), id, accountID, &in)
	if err != nil {
		switch err {
		case files.ErrNotFound:
			return NotFound(c, "File")
		case files.ErrNotOwner:
			return Forbidden(c, "Not file owner")
		case files.ErrFileLocked:
			return Conflict(c, "File is locked")
		case files.ErrNameTaken:
			return Conflict(c, "A file with this name already exists")
		default:
			return InternalError(c, "Failed to update file")
		}
	}

	return OK(c, f)
}

// Move moves a file.
func (h *Files) Move(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	id := c.Param("id")

	var in struct {
		FolderID string `json:"folder_id"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	f, err := h.files.Move(c.Request().Context(), id, accountID, in.FolderID)
	if err != nil {
		switch err {
		case files.ErrNotFound:
			return NotFound(c, "File")
		case files.ErrNotOwner:
			return Forbidden(c, "Not file owner")
		case files.ErrNameTaken:
			return Conflict(c, "A file with this name already exists in destination")
		default:
			return InternalError(c, "Failed to move file")
		}
	}

	return OK(c, f)
}

// Copy copies a file.
func (h *Files) Copy(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	id := c.Param("id")

	var in struct {
		FolderID string `json:"folder_id"`
		Name     string `json:"name"`
	}
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	f, err := h.files.Copy(c.Request().Context(), id, accountID, in.FolderID, in.Name)
	if err != nil {
		switch err {
		case files.ErrNotFound:
			return NotFound(c, "File")
		case files.ErrQuotaExceeded:
			return BadRequest(c, "Storage quota exceeded")
		default:
			return InternalError(c, "Failed to copy file")
		}
	}

	return Created(c, f)
}

// Delete moves a file to trash.
func (h *Files) Delete(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	id := c.Param("id")

	if err := h.files.Delete(c.Request().Context(), id, accountID); err != nil {
		switch err {
		case files.ErrNotFound:
			return NotFound(c, "File")
		case files.ErrNotOwner:
			return Forbidden(c, "Not file owner")
		default:
			return InternalError(c, "Failed to delete file")
		}
	}

	return OK(c, map[string]string{"message": "File moved to trash"})
}

// Star stars a file.
func (h *Files) Star(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	id := c.Param("id")

	if err := h.files.Star(c.Request().Context(), id, accountID, true); err != nil {
		return InternalError(c, "Failed to star file")
	}

	return OK(c, map[string]bool{"starred": true})
}

// Unstar unstars a file.
func (h *Files) Unstar(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	id := c.Param("id")

	if err := h.files.Star(c.Request().Context(), id, accountID, false); err != nil {
		return InternalError(c, "Failed to unstar file")
	}

	return OK(c, map[string]bool{"starred": false})
}

// Lock locks a file.
func (h *Files) Lock(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	id := c.Param("id")

	var in struct {
		Duration int `json:"duration"` // seconds
	}
	c.BindJSON(&in, 1<<20)

	duration := time.Duration(in.Duration) * time.Second
	if duration == 0 {
		duration = time.Hour
	}

	if err := h.files.Lock(c.Request().Context(), id, accountID, duration); err != nil {
		switch err {
		case files.ErrNotFound:
			return NotFound(c, "File")
		case files.ErrNotOwner:
			return Forbidden(c, "Not file owner")
		case files.ErrFileLocked:
			return Conflict(c, "File already locked by another user")
		default:
			return InternalError(c, "Failed to lock file")
		}
	}

	return OK(c, map[string]bool{"locked": true})
}

// Unlock unlocks a file.
func (h *Files) Unlock(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	id := c.Param("id")

	if err := h.files.Unlock(c.Request().Context(), id, accountID); err != nil {
		switch err {
		case files.ErrNotFound:
			return NotFound(c, "File")
		case files.ErrNotOwner:
			return Forbidden(c, "Cannot unlock file")
		default:
			return InternalError(c, "Failed to unlock file")
		}
	}

	return OK(c, map[string]bool{"locked": false})
}

// CreateChunkedUpload creates a chunked upload session.
func (h *Files) CreateChunkedUpload(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	var in files.CreateChunkedUploadIn
	if err := c.BindJSON(&in, 1<<20); err != nil {
		return BadRequest(c, "Invalid request body")
	}

	upload, err := h.files.CreateChunkedUpload(c.Request().Context(), accountID, &in)
	if err != nil {
		if err == files.ErrQuotaExceeded {
			return BadRequest(c, "Storage quota exceeded")
		}
		return InternalError(c, "Failed to create upload session")
	}

	return Created(c, upload)
}

// UploadChunk uploads a chunk.
func (h *Files) UploadChunk(c *mizu.Ctx) error {
	uploadID := c.Param("id")
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil {
		return BadRequest(c, "Invalid chunk index")
	}

	checksum := c.Request().Header.Get("X-Chunk-Checksum")

	if err := h.files.UploadChunk(c.Request().Context(), uploadID, index, c.Request().Body, c.Request().ContentLength, checksum); err != nil {
		switch err {
		case files.ErrUploadNotFound:
			return NotFound(c, "Upload session")
		case files.ErrUploadExpired:
			return BadRequest(c, "Upload session expired")
		case files.ErrInvalidUpload:
			return BadRequest(c, "Invalid chunk index")
		default:
			return InternalError(c, "Failed to upload chunk")
		}
	}

	return OK(c, map[string]any{
		"chunk_index": index,
		"received":    true,
	})
}

// GetUploadProgress returns upload progress.
func (h *Files) GetUploadProgress(c *mizu.Ctx) error {
	uploadID := c.Param("id")

	progress, err := h.files.GetUploadProgress(c.Request().Context(), uploadID)
	if err != nil {
		return NotFound(c, "Upload session")
	}

	return OK(c, progress)
}

// CompleteUpload completes a chunked upload.
func (h *Files) CompleteUpload(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	uploadID := c.Param("id")

	var in struct {
		Checksum string `json:"checksum_sha256"`
	}
	c.BindJSON(&in, 1<<20)

	f, err := h.files.CompleteUpload(c.Request().Context(), uploadID, accountID, in.Checksum)
	if err != nil {
		switch err {
		case files.ErrUploadNotFound:
			return NotFound(c, "Upload session")
		case files.ErrNotOwner:
			return Forbidden(c, "Not upload owner")
		case files.ErrChunkMissing:
			return BadRequest(c, "Some chunks are missing")
		default:
			return InternalError(c, "Failed to complete upload")
		}
	}

	return Created(c, f)
}

// ListVersions lists file versions.
func (h *Files) ListVersions(c *mizu.Ctx) error {
	id := c.Param("id")

	versions, err := h.files.ListVersions(c.Request().Context(), id)
	if err != nil {
		return InternalError(c, "Failed to list versions")
	}

	return OK(c, versions)
}

// UploadVersion uploads a new version.
func (h *Files) UploadVersion(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	id := c.Param("id")

	if err := c.Request().ParseMultipartForm(100 << 20); err != nil {
		return BadRequest(c, "Failed to parse form")
	}

	file, header, err := c.Request().FormFile("file")
	if err != nil {
		return BadRequest(c, "File is required")
	}
	defer file.Close()

	comment := c.Request().FormValue("comment")

	version, err := h.files.UploadVersion(c.Request().Context(), id, accountID, file, header.Size, comment)
	if err != nil {
		switch err {
		case files.ErrNotFound:
			return NotFound(c, "File")
		case files.ErrNotOwner:
			return Forbidden(c, "Not file owner")
		case files.ErrQuotaExceeded:
			return BadRequest(c, "Storage quota exceeded")
		default:
			return InternalError(c, "Failed to upload version")
		}
	}

	return Created(c, version)
}

// DownloadVersion downloads a specific version.
func (h *Files) DownloadVersion(c *mizu.Ctx) error {
	id := c.Param("id")
	version, err := strconv.Atoi(c.Param("version"))
	if err != nil {
		return BadRequest(c, "Invalid version number")
	}

	reader, v, err := h.files.OpenVersion(c.Request().Context(), id, version)
	if err != nil {
		return NotFound(c, "Version")
	}
	defer reader.Close()

	c.Writer().Header().Set("Content-Disposition", "attachment")
	c.Writer().Header().Set("Content-Length", strconv.FormatInt(v.Size, 10))

	io.Copy(c.Writer(), reader)
	return nil
}

// ListRecent lists recently accessed files.
func (h *Files) ListRecent(c *mizu.Ctx) error {
	accountID := getAccountIDFromCookie(c, h.accounts)
	if accountID == "" {
		return Unauthorized(c, "Not authenticated")
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	files, err := h.files.ListRecent(c.Request().Context(), accountID, limit)
	if err != nil {
		return InternalError(c, "Failed to list recent files")
	}

	return OK(c, files)
}
