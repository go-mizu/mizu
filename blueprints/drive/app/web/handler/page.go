// Package handler provides HTTP handlers for the Drive application.
package handler

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/drive/feature/accounts"
	"github.com/go-mizu/blueprints/drive/feature/activity"
	"github.com/go-mizu/blueprints/drive/feature/files"
	"github.com/go-mizu/blueprints/drive/feature/folders"
	"github.com/go-mizu/blueprints/drive/feature/meta"
	"github.com/go-mizu/blueprints/drive/feature/shares"
)

// Breadcrumb represents a navigation breadcrumb.
type Breadcrumb struct {
	Label string
	URL   string
	Icon  string
}

// FileView wraps a file with view-specific data.
type FileView struct {
	*files.File
	Icon        string
	KindDisplay string
	SizeDisplay string
	TimeDisplay string
	IsSelected  bool
}

// FolderView wraps a folder with view-specific data.
type FolderView struct {
	*folders.Folder
	ItemCount int
}

// ShareView wraps a share with view-specific data.
type ShareView struct {
	*shares.Share
	OwnerName  string
	TargetName string
	TargetType string
}

// ActivityView wraps an activity with view-specific data.
type ActivityView struct {
	*activity.Activity
	ActorName   string
	TargetName  string
	Description string
}

// ItemGroup groups items by time period.
type ItemGroup struct {
	Label string
	Key   string
	Files []*FileView
}

// LoginData holds data for the login page.
type LoginData struct {
	Title    string
	Subtitle string
	Error    string
}

// RegisterData holds data for the registration page.
type RegisterData struct {
	Title    string
	Subtitle string
	Error    string
}

// FilesData holds data for the files page.
type FilesData struct {
	Title         string
	User          *accounts.User
	CurrentFolder *folders.Folder
	Path          []*folders.Folder
	Folders       []*FolderView
	Files         []*FileView
	ViewMode      string // "grid" | "list"
	SortBy        string // "name" | "date" | "size"
	SortOrder     string // "asc" | "desc"
	SelectedCount int
	StorageUsed   int64
	StorageLimit  int64
	ActiveNav     string
	Breadcrumbs   []Breadcrumb
	Query         string // search query if any
}

// SharedData holds data for the shared page.
type SharedData struct {
	Title        string
	User         *accounts.User
	Shares       []*ShareView
	StorageUsed  int64
	StorageLimit int64
	ActiveNav    string
	Breadcrumbs  []Breadcrumb
	Query        string
}

// RecentData holds data for the recent files page.
type RecentData struct {
	Title        string
	User         *accounts.User
	Groups       []*ItemGroup
	StorageUsed  int64
	StorageLimit int64
	ActiveNav    string
	Breadcrumbs  []Breadcrumb
	Query        string
}

// StarredData holds data for the starred page.
type StarredData struct {
	Title        string
	User         *accounts.User
	Folders      []*FolderView
	Files        []*FileView
	ViewMode     string
	StorageUsed  int64
	StorageLimit int64
	ActiveNav    string
	Breadcrumbs  []Breadcrumb
	Query        string
}

// TrashData holds data for the trash page.
type TrashData struct {
	Title        string
	User         *accounts.User
	Folders      []*FolderView
	Files        []*FileView
	StorageUsed  int64
	StorageLimit int64
	ActiveNav    string
	Breadcrumbs  []Breadcrumb
	Query        string
}

// SearchData holds data for the search page.
type SearchData struct {
	Title        string
	User         *accounts.User
	Query        string
	Folders      []*FolderView
	Files        []*FileView
	TotalResults int
	StorageUsed  int64
	StorageLimit int64
	ActiveNav    string
	Breadcrumbs  []Breadcrumb
}

// SettingsData holds data for the settings page.
type SettingsData struct {
	Title        string
	User         *accounts.User
	StorageUsed  int64
	StorageLimit int64
	ActiveNav    string
	Breadcrumbs  []Breadcrumb
	Query        string
}

// ActivityData holds data for the activity page.
type ActivityData struct {
	Title        string
	User         *accounts.User
	Activities   []*ActivityView
	StorageUsed  int64
	StorageLimit int64
	ActiveNav    string
	Breadcrumbs  []Breadcrumb
	Query        string
}

// ShareLinkData holds data for the public share link page.
type ShareLinkData struct {
	Title       string
	Share       *shares.Share
	File        *files.File
	Folder      *folders.Folder
	Files       []*FileView
	IsProtected bool
	Error       string
}

// PreviewData holds data for the preview page.
type PreviewData struct {
	Title        string
	User         *accounts.User
	File         *files.File
	PreviewType  files.PreviewType
	PreviewURL   string
	ThumbnailURL string
	CanPreview   bool
	Language     string
	FileContent  string
	PrevFile     *files.File
	NextFile     *files.File
	SiblingIDs   []string
	CurrentIndex int
	StorageUsed  int64
	StorageLimit int64
	ActiveNav    string
	Breadcrumbs  []Breadcrumb
	Query        string
}

// Page handles page rendering.
type Page struct {
	templates   map[string]*template.Template
	accounts    accounts.API
	files       files.API
	folders     folders.API
	shares      shares.API
	activity    activity.API
	meta        *meta.Service
	getUserID   func(*mizu.Ctx) string
	storageRoot string
}

// NewPage creates a new Page handler.
func NewPage(
	templates map[string]*template.Template,
	accounts accounts.API,
	files files.API,
	folders folders.API,
	shares shares.API,
	activity activity.API,
	getUserID func(*mizu.Ctx) string,
	storageRoot string,
) *Page {
	return &Page{
		templates:   templates,
		accounts:    accounts,
		files:       files,
		folders:     folders,
		shares:      shares,
		activity:    activity,
		meta:        meta.New(),
		getUserID:   getUserID,
		storageRoot: storageRoot,
	}
}

func render[T any](h *Page, c *mizu.Ctx, name string, data T) error {
	tmpl, ok := h.templates[name]
	if !ok {
		return c.Text(http.StatusInternalServerError, "Template not found: "+name)
	}

	c.Writer().Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	// Template execution writes directly to response, so errors after this
	// point (like broken pipe) should be ignored since headers are already sent
	_ = tmpl.Execute(c.Writer(), data)
	return nil
}

// Login renders the login page.
func (h *Page) Login(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID != "" {
		http.Redirect(c.Writer(), c.Request(), "/files", http.StatusFound)
		return nil
	}
	return render(h, c, "login", LoginData{
		Title:    "Sign In",
		Subtitle: "Sign in to access your files",
	})
}

// Register renders the registration page.
func (h *Page) Register(c *mizu.Ctx) error {
	userID := h.getUserID(c)
	if userID != "" {
		http.Redirect(c.Writer(), c.Request(), "/files", http.StatusFound)
		return nil
	}
	return render(h, c, "register", RegisterData{
		Title:    "Create Account",
		Subtitle: "Create a new account to get started",
	})
}

// Files renders the main files page.
func (h *Page) Files(c *mizu.Ctx) error {
	ctx := c.Request().Context()

	// Get current user (for local mode, create default user)
	userID := h.getUserID(c)
	var user *accounts.User
	if userID != "" {
		user, _ = h.accounts.GetByID(ctx, userID)
	}

	// Get path from URL
	path := c.Param("path")
	if path == "" {
		path = ""
	}

	// Build breadcrumbs from path
	breadcrumbs := []Breadcrumb{
		{Label: "My Drive", URL: "/files", Icon: "home"},
	}

	var currentFolder *folders.Folder
	var pathFolders []*folders.Folder

	if path != "" {
		parts := strings.Split(path, "/")
		currentPath := ""
		for _, part := range parts {
			if part == "" {
				continue
			}
			currentPath = currentPath + "/" + part
			breadcrumbs = append(breadcrumbs, Breadcrumb{
				Label: part,
				URL:   "/files" + currentPath,
				Icon:  "folder",
			})
		}
	}

	// Get view mode from query or default to grid
	viewMode := c.Query("view")
	if viewMode == "" {
		viewMode = "grid"
	}

	sortBy := c.Query("sort")
	if sortBy == "" {
		sortBy = "name"
	}

	sortOrder := c.Query("order")
	if sortOrder == "" {
		sortOrder = "asc"
	}

	// Read files from local storage
	folderViews, fileViews := h.readLocalDirectory(path)

	// Sort files
	sortItems(folderViews, fileViews, sortBy, sortOrder)

	// Calculate storage usage
	storageUsed, storageLimit := h.calculateStorage()

	return render(h, c, "files", FilesData{
		Title:         "My Drive",
		User:          user,
		CurrentFolder: currentFolder,
		Path:          pathFolders,
		Folders:       folderViews,
		Files:         fileViews,
		ViewMode:      viewMode,
		SortBy:        sortBy,
		SortOrder:     sortOrder,
		StorageUsed:   storageUsed,
		StorageLimit:  storageLimit,
		ActiveNav:     "files",
		Breadcrumbs:   breadcrumbs,
	})
}

// Shared renders the shared with me page.
func (h *Page) Shared(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)
	var user *accounts.User
	if userID != "" {
		user, _ = h.accounts.GetByID(ctx, userID)
	}

	// Get shares (for now, empty list in local mode)
	var shareViews []*ShareView

	storageUsed, storageLimit := h.calculateStorage()

	return render(h, c, "shared", SharedData{
		Title:        "Shared with me",
		User:         user,
		Shares:       shareViews,
		StorageUsed:  storageUsed,
		StorageLimit: storageLimit,
		ActiveNav:    "shared",
		Breadcrumbs: []Breadcrumb{
			{Label: "Shared with me", URL: "/shared", Icon: "users"},
		},
	})
}

// Recent renders the recent files page.
func (h *Page) Recent(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)
	var user *accounts.User
	if userID != "" {
		user, _ = h.accounts.GetByID(ctx, userID)
	}

	// Get recent files from local storage
	_, fileViews := h.readLocalDirectory("")
	recentFiles := h.getRecentFiles(fileViews, 50)

	// Group by time
	groups := groupFilesByTime(recentFiles)

	storageUsed, storageLimit := h.calculateStorage()

	return render(h, c, "recent", RecentData{
		Title:        "Recent",
		User:         user,
		Groups:       groups,
		StorageUsed:  storageUsed,
		StorageLimit: storageLimit,
		ActiveNav:    "recent",
		Breadcrumbs: []Breadcrumb{
			{Label: "Recent", URL: "/recent", Icon: "clock"},
		},
	})
}

// Starred renders the starred files page.
func (h *Page) Starred(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)
	var user *accounts.User
	if userID != "" {
		user, _ = h.accounts.GetByID(ctx, userID)
	}

	viewMode := c.Query("view")
	if viewMode == "" {
		viewMode = "grid"
	}

	// For local mode, starred is stored in DB
	var folderViews []*FolderView
	var fileViews []*FileView

	if userID != "" {
		starredFiles, _ := h.files.ListStarred(ctx, userID)
		for _, f := range starredFiles {
			fileViews = append(fileViews, &FileView{
				File:        f,
				Icon:        getIconForMime(f.MimeType),
				KindDisplay: getKindForMime(f.MimeType),
				SizeDisplay: formatSize(f.Size),
				TimeDisplay: formatTime(f.UpdatedAt),
			})
		}

		starredFolders, _ := h.folders.ListStarred(ctx, userID)
		for _, f := range starredFolders {
			folderViews = append(folderViews, &FolderView{
				Folder: f,
			})
		}
	}

	storageUsed, storageLimit := h.calculateStorage()

	return render(h, c, "starred", StarredData{
		Title:        "Starred",
		User:         user,
		Folders:      folderViews,
		Files:        fileViews,
		ViewMode:     viewMode,
		StorageUsed:  storageUsed,
		StorageLimit: storageLimit,
		ActiveNav:    "starred",
		Breadcrumbs: []Breadcrumb{
			{Label: "Starred", URL: "/starred", Icon: "star"},
		},
	})
}

// Trash renders the trash page.
func (h *Page) Trash(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)
	var user *accounts.User
	if userID != "" {
		user, _ = h.accounts.GetByID(ctx, userID)
	}

	var folderViews []*FolderView
	var fileViews []*FileView

	if userID != "" {
		trashedFiles, _ := h.files.ListTrashed(ctx, userID)
		for _, f := range trashedFiles {
			fileViews = append(fileViews, &FileView{
				File:        f,
				Icon:        getIconForMime(f.MimeType),
				KindDisplay: getKindForMime(f.MimeType),
				SizeDisplay: formatSize(f.Size),
				TimeDisplay: formatTime(f.TrashedAt),
			})
		}

		trashedFolders, _ := h.folders.ListTrashed(ctx, userID)
		for _, f := range trashedFolders {
			folderViews = append(folderViews, &FolderView{
				Folder: f,
			})
		}
	}

	storageUsed, storageLimit := h.calculateStorage()

	return render(h, c, "trash", TrashData{
		Title:        "Trash",
		User:         user,
		Folders:      folderViews,
		Files:        fileViews,
		StorageUsed:  storageUsed,
		StorageLimit: storageLimit,
		ActiveNav:    "trash",
		Breadcrumbs: []Breadcrumb{
			{Label: "Trash", URL: "/trash", Icon: "trash-2"},
		},
	})
}

// Search renders the search page.
func (h *Page) Search(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)
	var user *accounts.User
	if userID != "" {
		user, _ = h.accounts.GetByID(ctx, userID)
	}

	query := c.Query("q")

	var folderViews []*FolderView
	var fileViews []*FileView

	if query != "" {
		// Search in local files
		folderViews, fileViews = h.searchLocalFiles(query)
	}

	storageUsed, storageLimit := h.calculateStorage()

	return render(h, c, "search", SearchData{
		Title:        "Search",
		User:         user,
		Query:        query,
		Folders:      folderViews,
		Files:        fileViews,
		TotalResults: len(folderViews) + len(fileViews),
		StorageUsed:  storageUsed,
		StorageLimit: storageLimit,
		ActiveNav:    "search",
		Breadcrumbs: []Breadcrumb{
			{Label: "Search", URL: "/search", Icon: "search"},
		},
	})
}

// Settings renders the settings page.
func (h *Page) Settings(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)
	var user *accounts.User
	if userID != "" {
		user, _ = h.accounts.GetByID(ctx, userID)
	}

	storageUsed, storageLimit := h.calculateStorage()

	return render(h, c, "settings", SettingsData{
		Title:        "Settings",
		User:         user,
		StorageUsed:  storageUsed,
		StorageLimit: storageLimit,
		ActiveNav:    "settings",
		Breadcrumbs: []Breadcrumb{
			{Label: "Settings", URL: "/settings", Icon: "settings"},
		},
	})
}

// Activity renders the activity page.
func (h *Page) Activity(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	userID := h.getUserID(c)
	var user *accounts.User
	if userID != "" {
		user, _ = h.accounts.GetByID(ctx, userID)
	}

	var activityViews []*ActivityView

	if userID != "" {
		activities, _ := h.activity.ListByUser(ctx, userID, 100)
		for _, a := range activities {
			activityViews = append(activityViews, &ActivityView{
				Activity:    a,
				Description: formatActivityDescription(a),
			})
		}
	}

	storageUsed, storageLimit := h.calculateStorage()

	return render(h, c, "activity", ActivityData{
		Title:        "Activity",
		User:         user,
		Activities:   activityViews,
		StorageUsed:  storageUsed,
		StorageLimit: storageLimit,
		ActiveNav:    "activity",
		Breadcrumbs: []Breadcrumb{
			{Label: "Activity", URL: "/activity", Icon: "activity"},
		},
	})
}

// ShareLink renders the public share link page.
func (h *Page) ShareLink(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	token := c.Param("token")

	share, err := h.shares.GetByToken(ctx, token)
	if err != nil {
		return render(h, c, "share", ShareLinkData{
			Title: "Share Not Found",
			Error: "This share link is invalid or has expired.",
		})
	}

	var file *files.File
	var folder *folders.Folder
	var fileViews []*FileView

	if share.ResourceType == "file" {
		file, _ = h.files.GetByID(ctx, share.ResourceID)
	} else {
		folder, _ = h.folders.GetByID(ctx, share.ResourceID)
		// List folder contents if it's a folder
	}

	return render(h, c, "share", ShareLinkData{
		Title:  "Shared File",
		Share:  share,
		File:   file,
		Folder: folder,
		Files:  fileViews,
	})
}

// Preview renders the file preview page.
func (h *Page) Preview(c *mizu.Ctx) error {
	ctx := c.Request().Context()
	fileID := c.Param("id")

	userID := h.getUserID(c)
	var user *accounts.User
	if userID != "" {
		user, _ = h.accounts.GetByID(ctx, userID)
	}

	// Get file from local storage
	fullPath := filepath.Join(h.storageRoot, fileID)
	info, err := os.Stat(fullPath)
	if err != nil {
		// Return empty preview if file not found
		return render(h, c, "preview", PreviewData{
			Title:      "File Not Found",
			User:       user,
			CanPreview: false,
		})
	}

	mimeType := getMimeType(fileID)
	file := &files.File{
		ID:        fileID,
		Name:      filepath.Base(fileID),
		MimeType:  mimeType,
		Size:      info.Size(),
		CreatedAt: info.ModTime(),
		UpdatedAt: info.ModTime(),
	}

	previewType := files.DetectPreviewType(mimeType, file.Name)
	language := files.GetLanguage(file.Name)
	canPreview := previewType != files.PreviewTypeUnsupported

	// Build breadcrumbs
	breadcrumbs := []Breadcrumb{
		{Label: "My Drive", URL: "/files", Icon: "home"},
	}

	// Add parent folders to breadcrumbs
	parentPath := filepath.Dir(fileID)
	if parentPath != "." && parentPath != "" {
		parts := strings.Split(parentPath, "/")
		currentPath := ""
		for _, part := range parts {
			if part == "" {
				continue
			}
			currentPath = currentPath + "/" + part
			breadcrumbs = append(breadcrumbs, Breadcrumb{
				Label: part,
				URL:   "/files" + currentPath,
				Icon:  "folder",
			})
		}
	}
	breadcrumbs = append(breadcrumbs, Breadcrumb{
		Label: file.Name,
		URL:   "",
		Icon:  "",
	})

	// Get siblings for navigation
	var prevFile, nextFile *files.File
	var siblingIDs []string
	var currentIndex int

	parentDir := filepath.Dir(fullPath)
	entries, _ := os.ReadDir(parentDir)
	var fileEntries []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			fileEntries = append(fileEntries, entry)
		}
	}

	for i, entry := range fileEntries {
		relPath := fileID
		if parentPath != "." && parentPath != "" {
			relPath = filepath.Join(parentPath, entry.Name())
		} else {
			relPath = entry.Name()
		}
		siblingIDs = append(siblingIDs, relPath)

		if entry.Name() == filepath.Base(fileID) {
			currentIndex = i
			if i > 0 {
				prevEntry := fileEntries[i-1]
				prevInfo, _ := prevEntry.Info()
				prevPath := relPath
				if parentPath != "." && parentPath != "" {
					prevPath = filepath.Join(parentPath, prevEntry.Name())
				} else {
					prevPath = prevEntry.Name()
				}
				prevMime := getMimeType(prevEntry.Name())
				prevFile = &files.File{
					ID:        prevPath,
					Name:      prevEntry.Name(),
					MimeType:  prevMime,
					Size:      prevInfo.Size(),
					UpdatedAt: prevInfo.ModTime(),
				}
			}
			if i < len(fileEntries)-1 {
				nextEntry := fileEntries[i+1]
				nextInfo, _ := nextEntry.Info()
				nextPath := relPath
				if parentPath != "." && parentPath != "" {
					nextPath = filepath.Join(parentPath, nextEntry.Name())
				} else {
					nextPath = nextEntry.Name()
				}
				nextMime := getMimeType(nextEntry.Name())
				nextFile = &files.File{
					ID:        nextPath,
					Name:      nextEntry.Name(),
					MimeType:  nextMime,
					Size:      nextInfo.Size(),
					UpdatedAt: nextInfo.ModTime(),
				}
			}
		}
	}

	// Get file content for text-based previews
	var fileContent string
	if previewType == files.PreviewTypeCode || previewType == files.PreviewTypeText || previewType == files.PreviewTypeMarkdown {
		// Limit to 5MB for text content
		if info.Size() <= 5*1024*1024 {
			content, err := os.ReadFile(fullPath)
			if err == nil {
				fileContent = string(content)
			}
		}
	}

	storageUsed, storageLimit := h.calculateStorage()

	return render(h, c, "preview", PreviewData{
		Title:        file.Name,
		User:         user,
		File:         file,
		PreviewType:  previewType,
		PreviewURL:   "/api/v1/content/" + fileID,
		ThumbnailURL: "/api/v1/thumbnail/" + fileID,
		CanPreview:   canPreview,
		Language:     language,
		FileContent:  fileContent,
		PrevFile:     prevFile,
		NextFile:     nextFile,
		SiblingIDs:   siblingIDs,
		CurrentIndex: currentIndex,
		StorageUsed:  storageUsed,
		StorageLimit: storageLimit,
		ActiveNav:    "files",
		Breadcrumbs:  breadcrumbs,
	})
}

// Content serves file content for preview.
func (h *Page) Content(c *mizu.Ctx) error {
	fileID := c.Param("id")

	// URL decode the file ID to handle special characters
	decodedID, err := url.PathUnescape(fileID)
	if err != nil {
		decodedID = fileID // Fallback to original if decode fails
	}

	fullPath := filepath.Join(h.storageRoot, decodedID)

	// Check file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return c.Text(404, "File not found")
	}

	// Set headers
	mimeType := getMimeType(decodedID)
	c.Writer().Header().Set("Content-Type", mimeType)
	c.Writer().Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filepath.Base(decodedID)))
	c.Writer().Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	c.Writer().Header().Set("Cache-Control", "private, max-age=3600")

	// Stream file
	file, err := os.Open(fullPath)
	if err != nil {
		return c.Text(500, "Failed to open file")
	}
	defer file.Close()

	// ServeContent writes headers directly, so mark response as started
	c.Status(200)
	http.ServeContent(c.Writer(), c.Request(), filepath.Base(decodedID), info.ModTime(), file)
	// Ignore errors from ServeContent (like broken pipe) since headers are already sent
	return nil
}

// Thumbnail serves file thumbnail.
func (h *Page) Thumbnail(c *mizu.Ctx) error {
	fileID := c.Param("id")

	// URL decode the file ID to handle special characters
	decodedID, err := url.PathUnescape(fileID)
	if err != nil {
		decodedID = fileID // Fallback to original if decode fails
	}

	fullPath := filepath.Join(h.storageRoot, decodedID)

	// Check file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return c.Text(404, "File not found")
	}

	mimeType := getMimeType(decodedID)

	// For images, serve the file directly (browser will scale)
	if strings.HasPrefix(mimeType, "image/") && mimeType != "image/svg+xml" {
		c.Writer().Header().Set("Content-Type", mimeType)
		c.Writer().Header().Set("Cache-Control", "public, max-age=86400")

		file, err := os.Open(fullPath)
		if err != nil {
			return c.Text(500, "Failed to open file")
		}
		defer file.Close()

		// ServeContent writes headers directly, so mark response as started
		c.Status(200)
		http.ServeContent(c.Writer(), c.Request(), filepath.Base(decodedID), info.ModTime(), file)
		// Ignore errors from ServeContent (like broken pipe) since headers are already sent
		return nil
	}

	// For other types, return a placeholder or 404
	return c.Text(404, "Thumbnail not available")
}

// Helper functions

func (h *Page) readLocalDirectory(path string) ([]*FolderView, []*FileView) {
	fullPath := filepath.Join(h.storageRoot, path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, nil
	}

	var folderViews []*FolderView
	var fileViews []*FileView

	for _, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if entry.IsDir() {
			folderViews = append(folderViews, &FolderView{
				Folder: &folders.Folder{
					ID:        filepath.Join(path, entry.Name()),
					Name:      entry.Name(),
					CreatedAt: info.ModTime(),
					UpdatedAt: info.ModTime(),
				},
			})
		} else {
			mimeType := getMimeType(entry.Name())
			fileViews = append(fileViews, &FileView{
				File: &files.File{
					ID:        filepath.Join(path, entry.Name()),
					Name:      entry.Name(),
					MimeType:  mimeType,
					Size:      info.Size(),
					CreatedAt: info.ModTime(),
					UpdatedAt: info.ModTime(),
				},
				Icon:        getIconForMime(mimeType),
				KindDisplay: getKindForMime(mimeType),
				SizeDisplay: formatSize(info.Size()),
				TimeDisplay: formatTime(info.ModTime()),
			})
		}
	}

	return folderViews, fileViews
}

func (h *Page) searchLocalFiles(query string) ([]*FolderView, []*FileView) {
	query = strings.ToLower(query)
	var folderViews []*FolderView
	var fileViews []*FileView

	filepath.Walk(h.storageRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if name matches query
		if !strings.Contains(strings.ToLower(info.Name()), query) {
			return nil
		}

		relPath, _ := filepath.Rel(h.storageRoot, path)

		if info.IsDir() {
			folderViews = append(folderViews, &FolderView{
				Folder: &folders.Folder{
					ID:        relPath,
					Name:      info.Name(),
					CreatedAt: info.ModTime(),
					UpdatedAt: info.ModTime(),
				},
			})
		} else {
			mimeType := getMimeType(info.Name())
			fileViews = append(fileViews, &FileView{
				File: &files.File{
					ID:        relPath,
					Name:      info.Name(),
					MimeType:  mimeType,
					Size:      info.Size(),
					CreatedAt: info.ModTime(),
					UpdatedAt: info.ModTime(),
				},
				Icon:        getIconForMime(mimeType),
				KindDisplay: getKindForMime(mimeType),
				SizeDisplay: formatSize(info.Size()),
				TimeDisplay: formatTime(info.ModTime()),
			})
		}

		return nil
	})

	return folderViews, fileViews
}

func (h *Page) getRecentFiles(allFiles []*FileView, limit int) []*FileView {
	// Sort by modified time descending
	sort.Slice(allFiles, func(i, j int) bool {
		return allFiles[i].UpdatedAt.After(allFiles[j].UpdatedAt)
	})

	if len(allFiles) > limit {
		return allFiles[:limit]
	}
	return allFiles
}

func (h *Page) calculateStorage() (used, limit int64) {
	// Calculate total size of storage directory
	var totalSize int64
	filepath.Walk(h.storageRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	// Default limit of 10GB
	return totalSize, 10 * 1024 * 1024 * 1024
}

func sortItems(folders []*FolderView, files []*FileView, sortBy, sortOrder string) {
	// Sort folders
	sort.Slice(folders, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "name":
			less = strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
		case "date":
			less = folders[i].UpdatedAt.Before(folders[j].UpdatedAt)
		default:
			less = strings.ToLower(folders[i].Name) < strings.ToLower(folders[j].Name)
		}
		if sortOrder == "desc" {
			return !less
		}
		return less
	})

	// Sort files
	sort.Slice(files, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "name":
			less = strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		case "date":
			less = files[i].UpdatedAt.Before(files[j].UpdatedAt)
		case "size":
			less = files[i].Size < files[j].Size
		default:
			less = strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
		}
		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

func groupFilesByTime(files []*FileView) []*ItemGroup {
	groups := map[string]*ItemGroup{
		"today":     {Label: "Today", Key: "today", Files: []*FileView{}},
		"yesterday": {Label: "Yesterday", Key: "yesterday", Files: []*FileView{}},
		"this_week": {Label: "This Week", Key: "this_week", Files: []*FileView{}},
		"older":     {Label: "Older", Key: "older", Files: []*FileView{}},
	}

	// Note: Using a simplified time check for local mode
	for _, f := range files {
		groups["older"].Files = append(groups["older"].Files, f)
	}

	result := []*ItemGroup{
		groups["today"],
		groups["yesterday"],
		groups["this_week"],
		groups["older"],
	}

	// Filter out empty groups
	var filtered []*ItemGroup
	for _, g := range result {
		if len(g.Files) > 0 {
			filtered = append(filtered, g)
		}
	}

	return filtered
}

func getMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	mimeTypes := map[string]string{
		// Text/Documents
		".txt":  "text/plain",
		".html": "text/html",
		".css":  "text/css",
		".js":   "text/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
		// Archives
		".zip": "application/zip",
		".tar": "application/x-tar",
		".gz":  "application/gzip",
		".rar": "application/vnd.rar",
		".7z":  "application/x-7z-compressed",
		// Office Documents
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		// Images
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",
		".bmp":  "image/bmp",
		".tiff": "image/tiff",
		".tif":  "image/tiff",
		".heic": "image/heic",
		".heif": "image/heif",
		// Audio - Extended support
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".oga":  "audio/ogg",
		".flac": "audio/flac",
		".aac":  "audio/aac",
		".m4a":  "audio/mp4",
		".wma":  "audio/x-ms-wma",
		".aiff": "audio/aiff",
		".aif":  "audio/aiff",
		".opus": "audio/opus",
		".mid":  "audio/midi",
		".midi": "audio/midi",
		// Video
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mov":  "video/quicktime",
		".avi":  "video/x-msvideo",
		".mkv":  "video/x-matroska",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".m4v":  "video/x-m4v",
		".3gp":  "video/3gpp",
		".ogv":  "video/ogg",
		// Code/Programming
		".go":   "text/x-go",
		".py":   "text/x-python",
		".rs":   "text/x-rust",
		".java": "text/x-java",
		".c":    "text/x-c",
		".cpp":  "text/x-c++",
		".h":    "text/x-c",
		".hpp":  "text/x-c++",
		".ts":   "text/typescript",
		".tsx":  "text/typescript",
		".jsx":  "text/javascript",
		".vue":  "text/html",
		".svelte": "text/html",
		".rb":   "text/x-ruby",
		".php":  "text/x-php",
		".swift": "text/x-swift",
		".kt":   "text/x-kotlin",
		".scala": "text/x-scala",
		".sh":   "text/x-shellscript",
		".bash": "text/x-shellscript",
		".zsh":  "text/x-shellscript",
		// Config/Data
		".md":   "text/markdown",
		".yaml": "text/yaml",
		".yml":  "text/yaml",
		".toml": "text/toml",
		".sql":  "text/x-sql",
		".ini":  "text/plain",
		".env":  "text/plain",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func getIconForMime(mimeType string) string {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "image"
	case strings.HasPrefix(mimeType, "video/"):
		return "video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "music"
	case strings.HasPrefix(mimeType, "text/"):
		return "file-text"
	case mimeType == "application/pdf":
		return "file-text"
	case strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "archive") || strings.Contains(mimeType, "compressed"):
		return "file-archive"
	case strings.Contains(mimeType, "json"):
		return "file-json"
	case strings.Contains(mimeType, "javascript") || strings.Contains(mimeType, "typescript"):
		return "file-code"
	case strings.Contains(mimeType, "word") || strings.Contains(mimeType, "document"):
		return "file-text"
	case strings.Contains(mimeType, "sheet") || strings.Contains(mimeType, "excel"):
		return "file-spreadsheet"
	case strings.Contains(mimeType, "presentation") || strings.Contains(mimeType, "powerpoint"):
		return "file-presentation"
	default:
		return "file"
	}
}

// getKindForMime returns a human-readable file kind description (like Finder).
func getKindForMime(mimeType string) string {
	// Specific MIME type mappings
	mimeKinds := map[string]string{
		// Images
		"image/jpeg":      "JPEG Image",
		"image/png":       "PNG Image",
		"image/gif":       "GIF Image",
		"image/webp":      "WebP Image",
		"image/svg+xml":   "SVG Image",
		"image/bmp":       "BMP Image",
		"image/tiff":      "TIFF Image",
		"image/heic":      "HEIC Image",
		"image/heif":      "HEIF Image",
		"image/x-icon":    "Icon File",
		"image/vnd.adobe.photoshop": "Photoshop Document",
		// Videos
		"video/mp4":         "MP4 Video",
		"video/webm":        "WebM Video",
		"video/quicktime":   "QuickTime Movie",
		"video/x-msvideo":   "AVI Video",
		"video/x-matroska":  "Matroska Video",
		"video/x-ms-wmv":    "WMV Video",
		"video/x-flv":       "Flash Video",
		"video/x-m4v":       "M4V Video",
		"video/3gpp":        "3GPP Video",
		"video/ogg":         "Ogg Video",
		"video/mpeg":        "MPEG Video",
		// Audio
		"audio/mpeg":        "MP3 Audio",
		"audio/mp3":         "MP3 Audio",
		"audio/wav":         "WAV Audio",
		"audio/x-wav":       "WAV Audio",
		"audio/flac":        "FLAC Audio",
		"audio/x-flac":      "FLAC Audio",
		"audio/ogg":         "Ogg Audio",
		"audio/aac":         "AAC Audio",
		"audio/mp4":         "M4A Audio",
		"audio/x-m4a":       "M4A Audio",
		"audio/x-ms-wma":    "WMA Audio",
		"audio/aiff":        "AIFF Audio",
		"audio/x-aiff":      "AIFF Audio",
		"audio/opus":        "Opus Audio",
		"audio/midi":        "MIDI Audio",
		"audio/x-midi":      "MIDI Audio",
		// Documents
		"application/pdf":   "PDF Document",
		"application/msword": "Word Document",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "Word Document",
		"application/vnd.ms-excel": "Excel Spreadsheet",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": "Excel Spreadsheet",
		"application/vnd.ms-powerpoint": "PowerPoint Presentation",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": "PowerPoint Presentation",
		"application/rtf":   "Rich Text Document",
		// Archives
		"application/zip":   "ZIP Archive",
		"application/x-rar-compressed": "RAR Archive",
		"application/x-7z-compressed": "7-Zip Archive",
		"application/gzip":  "Gzip Archive",
		"application/x-tar": "TAR Archive",
		"application/x-bzip2": "Bzip2 Archive",
		// Code/Data
		"application/json":  "JSON File",
		"application/xml":   "XML File",
		"application/javascript": "JavaScript File",
		"application/x-javascript": "JavaScript File",
		"text/javascript":   "JavaScript File",
		// Text
		"text/plain":        "Plain Text",
		"text/html":         "HTML Document",
		"text/css":          "CSS Stylesheet",
		"text/markdown":     "Markdown Document",
		"text/x-markdown":   "Markdown Document",
		"text/csv":          "CSV Document",
		"text/xml":          "XML Document",
		"text/x-python":     "Python Script",
		"text/x-go":         "Go Source",
		"text/x-java":       "Java Source",
		"text/x-c":          "C Source",
		"text/x-c++":        "C++ Source",
		"text/x-ruby":       "Ruby Script",
		"text/x-php":        "PHP Script",
		"text/x-shellscript": "Shell Script",
		"text/yaml":         "YAML File",
		"text/x-yaml":       "YAML File",
		// Fonts
		"font/ttf":          "TrueType Font",
		"font/otf":          "OpenType Font",
		"font/woff":         "Web Font",
		"font/woff2":        "Web Font 2",
		// Executables
		"application/x-msdownload": "Windows Executable",
		"application/x-mach-binary": "macOS Executable",
		"application/x-executable": "Executable",
	}

	if kind, ok := mimeKinds[mimeType]; ok {
		return kind
	}

	// Generic type mappings
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "Image"
	case strings.HasPrefix(mimeType, "video/"):
		return "Video"
	case strings.HasPrefix(mimeType, "audio/"):
		return "Audio"
	case strings.HasPrefix(mimeType, "text/"):
		return "Text File"
	case strings.Contains(mimeType, "zip") || strings.Contains(mimeType, "archive") || strings.Contains(mimeType, "compressed"):
		return "Archive"
	case strings.Contains(mimeType, "font"):
		return "Font"
	default:
		return "Document"
	}
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.1f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func formatTime(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

func formatActivityDescription(a *activity.Activity) string {
	switch a.Action {
	case "create":
		return "created"
	case "update":
		return "modified"
	case "delete":
		return "deleted"
	case "move":
		return "moved"
	case "copy":
		return "copied"
	case "star":
		return "starred"
	case "unstar":
		return "unstarred"
	case "share":
		return "shared"
	case "unshare":
		return "unshared"
	case "trash":
		return "moved to trash"
	case "restore":
		return "restored"
	default:
		return a.Action
	}
}

// Metadata returns file metadata as JSON.
func (h *Page) Metadata(c *mizu.Ctx) error {
	fileID := c.Param("id")

	// URL decode the file ID to handle special characters
	decodedID, err := url.PathUnescape(fileID)
	if err != nil {
		slog.Debug("metadata: failed to decode file id", "file_id", fileID, "error", err)
		decodedID = fileID // Fallback to original if decode fails
	}

	fullPath := filepath.Join(h.storageRoot, decodedID)

	// Check file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		slog.Debug("metadata: file not found", "file_id", decodedID, "path", fullPath)
		return c.JSON(404, map[string]string{"error": "File not found"})
	}

	// Extract metadata
	metadata, err := h.meta.Extract(c.Context(), fullPath)
	if err != nil {
		slog.Debug("metadata: extraction failed", "file_id", fileID, "error", err)
		return c.JSON(500, map[string]string{"error": "Failed to extract metadata"})
	}

	// Update file ID to be the relative path
	metadata.FileID = decodedID

	// Add file timestamps
	metadata.ModifiedAt = info.ModTime()
	metadata.CreatedAt = info.ModTime() // Use ModTime as fallback for created time

	// Debug log the extracted metadata
	slog.Debug("metadata: extracted",
		"file_id", fileID,
		"mime_type", metadata.MimeType,
		"size", metadata.Size,
		"has_image", metadata.Image != nil,
		"has_video", metadata.Video != nil,
		"has_audio", metadata.Audio != nil,
		"has_document", metadata.Document != nil,
	)

	// Log image-specific metadata if present
	if metadata.Image != nil {
		slog.Debug("metadata: image details",
			"file_id", fileID,
			"width", metadata.Image.Width,
			"height", metadata.Image.Height,
			"make", metadata.Image.Make,
			"model", metadata.Image.Model,
			"has_gps", metadata.Image.GPSLatitude != 0 || metadata.Image.GPSLongitude != 0,
		)
	}

	return c.JSON(200, metadata)
}

// FolderChildren returns folder contents for column view navigation.
func (h *Page) FolderChildren(c *mizu.Ctx) error {
	folderID := c.Param("id")

	// URL decode the folder ID to handle special characters
	if folderID != "" {
		decodedID, err := url.PathUnescape(folderID)
		if err == nil {
			folderID = decodedID
		}
	}

	folderViews, fileViews := h.readLocalDirectory(folderID)

	// Build path breadcrumbs
	var pathItems []map[string]string
	if folderID != "" {
		parts := strings.Split(folderID, "/")
		currentPath := ""
		for _, part := range parts {
			if part == "" {
				continue
			}
			currentPath = currentPath + "/" + part
			pathItems = append(pathItems, map[string]string{
				"id":   strings.TrimPrefix(currentPath, "/"),
				"name": part,
			})
		}
	}

	// Build response
	response := map[string]any{
		"folders": folderViews,
		"files":   fileViews,
		"path":    pathItems,
	}

	return c.JSON(200, response)
}
