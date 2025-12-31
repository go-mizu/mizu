package web

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/app/web/handler/obake"
	"github.com/go-mizu/blueprints/cms/app/web/handler/rest"
	"github.com/go-mizu/blueprints/cms/app/web/handler/site"
	"github.com/go-mizu/blueprints/cms/app/web/handler/wpadmin"
	"github.com/go-mizu/blueprints/cms/app/web/handler/wpapi"
	"github.com/go-mizu/blueprints/cms/assets"
	"github.com/go-mizu/blueprints/cms/feature/categories"
	"github.com/go-mizu/blueprints/cms/feature/comments"
	"github.com/go-mizu/blueprints/cms/feature/media"
	"github.com/go-mizu/blueprints/cms/feature/menus"
	"github.com/go-mizu/blueprints/cms/feature/pages"
	"github.com/go-mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/blueprints/cms/feature/settings"
	"github.com/go-mizu/blueprints/cms/feature/tags"
	"github.com/go-mizu/blueprints/cms/feature/users"
	"github.com/go-mizu/blueprints/cms/store/duckdb"
)

// Config holds server configuration.
type Config struct {
	Addr    string
	DataDir string
	Dev     bool
}

// Server is the HTTP server.
type Server struct {
	app *mizu.App
	cfg Config
	db  *sql.DB

	// Services
	users      users.API
	posts      posts.API
	pages      pages.API
	categories categories.API
	tags       tags.API
	media      media.API
	comments   comments.API
	settings   settings.API
	menus      menus.API

	// Handlers
	authHandlers       *rest.Auth
	usersHandlers      *rest.Users
	postsHandlers      *rest.Posts
	pagesHandlers      *rest.Pages
	categoriesHandlers *rest.Categories
	tagsHandlers       *rest.Tags
	mediaHandlers      *rest.Media
	commentsHandlers   *rest.Comments
	settingsHandlers   *rest.Settings
	menusHandlers      *rest.Menus

	// WordPress API Handler
	wpHandler *wpapi.Handler

	// WordPress Admin Handler
	wpAdminHandler *wpadmin.Handler

	// Ghost-compatible Admin Handler (Obake)
	obakeHandler *obake.Handler

	// Site frontend Handler
	siteHandler *site.Handler
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	// Create uploads directory
	uploadsDir := filepath.Join(cfg.DataDir, "uploads")
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		return nil, fmt.Errorf("create uploads dir: %w", err)
	}

	// Open database
	dbPath := filepath.Join(cfg.DataDir, "cms.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Create core store for schema initialization
	coreStore, err := duckdb.New(db)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	// Initialize schema
	if err := coreStore.Ensure(context.Background()); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	// Create feature stores
	usersStore := duckdb.NewUsersStore(db)
	postsStore := duckdb.NewPostsStore(db)
	pagesStore := duckdb.NewPagesStore(db)
	categoriesStore := duckdb.NewCategoriesStore(db)
	tagsStore := duckdb.NewTagsStore(db)
	mediaStore := duckdb.NewMediaStore(db)
	commentsStore := duckdb.NewCommentsStore(db)
	settingsStore := duckdb.NewSettingsStore(db)
	menusStore := duckdb.NewMenusStore(db)

	// Create storage for media
	storage := media.NewLocalStorage(uploadsDir, "/uploads")

	// Create services
	usersSvc := users.NewService(usersStore)
	postsSvc := posts.NewService(postsStore)
	pagesSvc := pages.NewService(pagesStore)
	categoriesSvc := categories.NewService(categoriesStore)
	tagsSvc := tags.NewService(tagsStore)
	mediaSvc := media.NewService(mediaStore, storage)
	commentsSvc := comments.NewService(commentsStore)
	settingsSvc := settings.NewService(settingsStore)
	menusSvc := menus.NewService(menusStore)

	s := &Server{
		app:        mizu.New(),
		cfg:        cfg,
		db:         db,
		users:      usersSvc,
		posts:      postsSvc,
		pages:      pagesSvc,
		categories: categoriesSvc,
		tags:       tagsSvc,
		media:      mediaSvc,
		comments:   commentsSvc,
		settings:   settingsSvc,
		menus:      menusSvc,
	}

	// Create handlers
	s.authHandlers = rest.NewAuth(usersSvc)
	s.usersHandlers = rest.NewUsers(usersSvc, s.getUserID)
	s.postsHandlers = rest.NewPosts(postsSvc, s.getUserID)
	s.pagesHandlers = rest.NewPages(pagesSvc, s.getUserID)
	s.categoriesHandlers = rest.NewCategories(categoriesSvc)
	s.tagsHandlers = rest.NewTags(tagsSvc)
	s.mediaHandlers = rest.NewMedia(mediaSvc, s.getUserID)
	s.commentsHandlers = rest.NewComments(commentsSvc, s.getUserID)
	s.settingsHandlers = rest.NewSettings(settingsSvc)
	s.menusHandlers = rest.NewMenus(menusSvc)

	// Create WordPress API handler
	baseURL := "http://localhost" + cfg.Addr
	s.wpHandler = wpapi.New(wpapi.Config{
		BaseURL:    baseURL,
		Users:      usersSvc,
		Posts:      postsSvc,
		Pages:      pagesSvc,
		Categories: categoriesSvc,
		Tags:       tagsSvc,
		Media:      mediaSvc,
		Comments:   commentsSvc,
		Settings:   settingsSvc,
		GetUserID:  s.getUserID,
		GetUser:    s.getUser,
	})

	// Create WordPress Admin handler
	wpAdminTemplates, err := assets.WPAdminTemplates()
	if err != nil {
		return nil, fmt.Errorf("parse wpadmin templates: %w", err)
	}
	s.wpAdminHandler = wpadmin.New(wpAdminTemplates, wpadmin.Config{
		BaseURL:    baseURL,
		Users:      usersSvc,
		Posts:      postsSvc,
		Pages:      pagesSvc,
		Categories: categoriesSvc,
		Tags:       tagsSvc,
		Media:      mediaSvc,
		Comments:   commentsSvc,
		Settings:   settingsSvc,
		Menus:      menusSvc,
		GetUserID:  s.getUserID,
		GetUser:    s.getUser,
	})

	// Create Ghost-compatible Admin handler (Obake)
	obakeTemplates, err := assets.ObakeTemplates()
	if err != nil {
		return nil, fmt.Errorf("parse obake templates: %w", err)
	}
	s.obakeHandler = obake.New(obakeTemplates, obake.Config{
		BaseURL:   baseURL,
		Users:     usersSvc,
		Posts:     postsSvc,
		Pages:     pagesSvc,
		Tags:      tagsSvc,
		Media:     mediaSvc,
		Settings:  settingsSvc,
		GetUserID: s.getUserID,
		GetUser:   s.getUser,
	})

	// Create Site frontend handler
	siteTemplates, err := assets.SiteTemplates()
	if err != nil {
		return nil, fmt.Errorf("parse site templates: %w", err)
	}
	s.siteHandler = site.New(siteTemplates, site.Config{
		BaseURL:    baseURL,
		Posts:      postsSvc,
		Pages:      pagesSvc,
		Categories: categoriesSvc,
		Tags:       tagsSvc,
		Users:      usersSvc,
		Media:      mediaSvc,
		Comments:   commentsSvc,
		Settings:   settingsSvc,
		Menus:      menusSvc,
		GetUserID:  s.getUserID,
		GetUser:    s.getUser,
	})

	s.setupRoutes()

	return s, nil
}

// Run starts the server.
func (s *Server) Run() error {
	log.Printf("Starting CMS server on %s", s.cfg.Addr)
	return s.app.Listen(s.cfg.Addr)
}

// Close shuts down the server.
func (s *Server) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() *mizu.App {
	return s.app
}

// getUserID extracts the user ID from the session.
func (s *Server) getUserID(c *mizu.Ctx) string {
	cookie, err := c.Cookie("session")
	if err != nil || cookie.Value == "" {
		return ""
	}
	user, err := s.users.GetBySession(c.Context(), cookie.Value)
	if err != nil || user == nil {
		return ""
	}
	return user.ID
}

// getUser extracts the user from the session.
func (s *Server) getUser(c *mizu.Ctx) *users.User {
	cookie, err := c.Cookie("session")
	if err != nil || cookie.Value == "" {
		return nil
	}
	user, err := s.users.GetBySession(c.Context(), cookie.Value)
	if err != nil {
		return nil
	}
	return user
}

// authRequired middleware requires authentication.
func (s *Server) authRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		userID := s.getUserID(c)
		if userID == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		return next(c)
	}
}

func (s *Server) setupRoutes() {
	// Health check
	s.app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Serve uploads
	uploadsDir := filepath.Join(s.cfg.DataDir, "uploads")
	s.app.Get("/uploads/{filename}", func(c *mizu.Ctx) error {
		filename := c.Param("filename")
		http.ServeFile(c.Writer(), c.Request(), filepath.Join(uploadsDir, filename))
		return nil
	})

	// API routes
	s.app.Group("/api/v1", func(api *mizu.Router) {
		// Auth
		api.Post("/auth/register", s.authHandlers.Register)
		api.Post("/auth/login", s.authHandlers.Login)
		api.Post("/auth/logout", s.authRequired(s.authHandlers.Logout))
		api.Get("/auth/me", s.authRequired(s.authHandlers.Me))

		// Users
		api.Get("/users", s.authRequired(s.usersHandlers.List))
		api.Get("/users/{id}", s.authRequired(s.usersHandlers.Get))
		api.Put("/users/{id}", s.authRequired(s.usersHandlers.Update))
		api.Delete("/users/{id}", s.authRequired(s.usersHandlers.Delete))

		// Posts
		api.Get("/posts", s.postsHandlers.List) // Public
		api.Post("/posts", s.authRequired(s.postsHandlers.Create))
		api.Get("/posts/{id}", s.postsHandlers.Get) // Public
		api.Get("/posts/by-slug/{slug}", s.postsHandlers.GetBySlug) // Public
		api.Put("/posts/{id}", s.authRequired(s.postsHandlers.Update))
		api.Delete("/posts/{id}", s.authRequired(s.postsHandlers.Delete))
		api.Post("/posts/{id}/publish", s.authRequired(s.postsHandlers.Publish))
		api.Post("/posts/{id}/unpublish", s.authRequired(s.postsHandlers.Unpublish))

		// Pages
		api.Get("/pages", s.pagesHandlers.List) // Public
		api.Post("/pages", s.authRequired(s.pagesHandlers.Create))
		api.Get("/pages/{id}", s.pagesHandlers.Get) // Public
		api.Get("/pages/by-slug/{slug}", s.pagesHandlers.GetBySlug) // Public
		api.Get("/pages/tree", s.pagesHandlers.GetTree) // Public
		api.Put("/pages/{id}", s.authRequired(s.pagesHandlers.Update))
		api.Delete("/pages/{id}", s.authRequired(s.pagesHandlers.Delete))

		// Categories
		api.Get("/categories", s.categoriesHandlers.List) // Public
		api.Post("/categories", s.authRequired(s.categoriesHandlers.Create))
		api.Get("/categories/{id}", s.categoriesHandlers.Get) // Public
		api.Get("/categories/tree", s.categoriesHandlers.GetTree) // Public
		api.Put("/categories/{id}", s.authRequired(s.categoriesHandlers.Update))
		api.Delete("/categories/{id}", s.authRequired(s.categoriesHandlers.Delete))

		// Tags
		api.Get("/tags", s.tagsHandlers.List) // Public
		api.Post("/tags", s.authRequired(s.tagsHandlers.Create))
		api.Get("/tags/{id}", s.tagsHandlers.Get) // Public
		api.Put("/tags/{id}", s.authRequired(s.tagsHandlers.Update))
		api.Delete("/tags/{id}", s.authRequired(s.tagsHandlers.Delete))

		// Media
		api.Get("/media", s.authRequired(s.mediaHandlers.List))
		api.Post("/media", s.authRequired(s.mediaHandlers.Upload))
		api.Get("/media/{id}", s.authRequired(s.mediaHandlers.Get))
		api.Put("/media/{id}", s.authRequired(s.mediaHandlers.Update))
		api.Delete("/media/{id}", s.authRequired(s.mediaHandlers.Delete))

		// Comments - actions use separate path to avoid route conflicts
		api.Get("/comments", s.authRequired(s.commentsHandlers.List))
		api.Get("/comments/for-post/{postID}", s.commentsHandlers.ListByPost) // Public
		api.Post("/comments/for-post/{postID}", s.commentsHandlers.Create) // Public (with captcha)
		api.Post("/comments/approve/{id}", s.authRequired(s.commentsHandlers.Approve))
		api.Post("/comments/spam/{id}", s.authRequired(s.commentsHandlers.MarkAsSpam))
		api.Get("/comments/{id}", s.authRequired(s.commentsHandlers.Get))
		api.Put("/comments/{id}", s.authRequired(s.commentsHandlers.Update))
		api.Delete("/comments/{id}", s.authRequired(s.commentsHandlers.Delete))

		// Settings
		api.Get("/settings", s.authRequired(s.settingsHandlers.GetAll))
		api.Get("/settings/public", s.settingsHandlers.GetPublic) // Public
		api.Put("/settings", s.authRequired(s.settingsHandlers.SetBulk))
		api.Get("/settings/{key}", s.authRequired(s.settingsHandlers.Get))
		api.Put("/settings/{key}", s.authRequired(s.settingsHandlers.Set))
		api.Delete("/settings/{key}", s.authRequired(s.settingsHandlers.Delete))

		// Menus
		api.Get("/menus", s.menusHandlers.List) // Public
		api.Post("/menus", s.authRequired(s.menusHandlers.Create))
		api.Get("/menus/{id}", s.menusHandlers.Get) // Public
		api.Get("/menus/by-location/{location}", s.menusHandlers.GetByLocation) // Public
		api.Put("/menus/{id}", s.authRequired(s.menusHandlers.Update))
		api.Delete("/menus/{id}", s.authRequired(s.menusHandlers.Delete))
		api.Post("/menus/{id}/items", s.authRequired(s.menusHandlers.CreateItem))
		api.Put("/menus/{id}/items/{itemID}", s.authRequired(s.menusHandlers.UpdateItem))
		api.Delete("/menus/{id}/items/{itemID}", s.authRequired(s.menusHandlers.DeleteItem))
		api.Post("/menus/{id}/reorder", s.authRequired(s.menusHandlers.ReorderItems))
	})

	// WordPress REST API routes
	s.app.Get("/wp-json", s.wpHandler.Discovery)
	s.app.Get("/wp-json/wp/v2", s.wpHandler.NamespaceDiscovery)

	s.app.Group("/wp-json/wp/v2", func(wp *mizu.Router) {
		// Posts
		wp.Get("/posts", s.wpHandler.ListPosts)
		wp.Post("/posts", s.wpHandler.CreatePost)
		wp.Get("/posts/{id}", s.wpHandler.GetPost)
		wp.Post("/posts/{id}", s.wpHandler.UpdatePost)
		wp.Put("/posts/{id}", s.wpHandler.UpdatePost)
		wp.Patch("/posts/{id}", s.wpHandler.UpdatePost)
		wp.Delete("/posts/{id}", s.wpHandler.DeletePost)

		// Pages
		wp.Get("/pages", s.wpHandler.ListPages)
		wp.Post("/pages", s.wpHandler.CreatePage)
		wp.Get("/pages/{id}", s.wpHandler.GetPage)
		wp.Post("/pages/{id}", s.wpHandler.UpdatePage)
		wp.Put("/pages/{id}", s.wpHandler.UpdatePage)
		wp.Patch("/pages/{id}", s.wpHandler.UpdatePage)
		wp.Delete("/pages/{id}", s.wpHandler.DeletePage)

		// Users
		wp.Get("/users", s.wpHandler.ListUsers)
		wp.Post("/users", s.wpHandler.CreateUser)
		wp.Get("/users/me", s.wpHandler.GetCurrentUser)
		wp.Post("/users/me", s.wpHandler.UpdateCurrentUser)
		wp.Put("/users/me", s.wpHandler.UpdateCurrentUser)
		wp.Patch("/users/me", s.wpHandler.UpdateCurrentUser)
		wp.Delete("/users/me", s.wpHandler.DeleteCurrentUser)
		wp.Get("/users/{id}", s.wpHandler.GetUser)
		wp.Post("/users/{id}", s.wpHandler.UpdateUser)
		wp.Put("/users/{id}", s.wpHandler.UpdateUser)
		wp.Patch("/users/{id}", s.wpHandler.UpdateUser)
		wp.Delete("/users/{id}", s.wpHandler.DeleteUser)

		// Categories
		wp.Get("/categories", s.wpHandler.ListCategories)
		wp.Post("/categories", s.wpHandler.CreateCategory)
		wp.Get("/categories/{id}", s.wpHandler.GetCategory)
		wp.Post("/categories/{id}", s.wpHandler.UpdateCategory)
		wp.Put("/categories/{id}", s.wpHandler.UpdateCategory)
		wp.Patch("/categories/{id}", s.wpHandler.UpdateCategory)
		wp.Delete("/categories/{id}", s.wpHandler.DeleteCategory)

		// Tags
		wp.Get("/tags", s.wpHandler.ListTags)
		wp.Post("/tags", s.wpHandler.CreateTag)
		wp.Get("/tags/{id}", s.wpHandler.GetTag)
		wp.Post("/tags/{id}", s.wpHandler.UpdateTag)
		wp.Put("/tags/{id}", s.wpHandler.UpdateTag)
		wp.Patch("/tags/{id}", s.wpHandler.UpdateTag)
		wp.Delete("/tags/{id}", s.wpHandler.DeleteTag)

		// Media
		wp.Get("/media", s.wpHandler.ListMedia)
		wp.Post("/media", s.wpHandler.UploadMedia)
		wp.Get("/media/{id}", s.wpHandler.GetMedia)
		wp.Post("/media/{id}", s.wpHandler.UpdateMedia)
		wp.Put("/media/{id}", s.wpHandler.UpdateMedia)
		wp.Patch("/media/{id}", s.wpHandler.UpdateMedia)
		wp.Delete("/media/{id}", s.wpHandler.DeleteMedia)

		// Comments
		wp.Get("/comments", s.wpHandler.ListComments)
		wp.Post("/comments", s.wpHandler.CreateComment)
		wp.Get("/comments/{id}", s.wpHandler.GetComment)
		wp.Post("/comments/{id}", s.wpHandler.UpdateComment)
		wp.Put("/comments/{id}", s.wpHandler.UpdateComment)
		wp.Patch("/comments/{id}", s.wpHandler.UpdateComment)
		wp.Delete("/comments/{id}", s.wpHandler.DeleteComment)

		// Settings
		wp.Get("/settings", s.wpHandler.GetSettings)
		wp.Post("/settings", s.wpHandler.UpdateSettings)
		wp.Put("/settings", s.wpHandler.UpdateSettings)
		wp.Patch("/settings", s.wpHandler.UpdateSettings)
	})

	// WordPress Admin static assets
	staticFS := assets.Static()
	s.app.Get("/wp-admin/css/{filename...}", func(c *mizu.Ctx) error {
		filename := c.Param("filename")
		subFS, _ := fs.Sub(staticFS, "css")
		http.StripPrefix("/wp-admin/css/", http.FileServer(http.FS(subFS))).ServeHTTP(c.Writer(), c.Request())
		_ = filename // unused but needed for pattern matching
		return nil
	})
	s.app.Get("/wp-admin/js/{filename...}", func(c *mizu.Ctx) error {
		filename := c.Param("filename")
		subFS, _ := fs.Sub(staticFS, "js")
		http.StripPrefix("/wp-admin/js/", http.FileServer(http.FS(subFS))).ServeHTTP(c.Writer(), c.Request())
		_ = filename
		return nil
	})

	// WordPress Admin routes - Login/Logout
	s.app.Get("/wp-login.php", s.wpAdminHandler.Login)
	s.app.Post("/wp-login.php", s.wpAdminHandler.LoginPost)
	s.app.Get("/wp-logout.php", s.wpAdminHandler.Logout)

	// Dashboard
	s.app.Get("/wp-admin/", s.wpAdminHandler.Dashboard)
	s.app.Get("/wp-admin/index.php", s.wpAdminHandler.Dashboard)

	// Posts
	s.app.Get("/wp-admin/edit.php", func(c *mizu.Ctx) error {
		postType := c.Query("post_type")
		if postType == "page" {
			return s.wpAdminHandler.PagesList(c)
		}
		return s.wpAdminHandler.PostsList(c)
	})
	s.app.Get("/wp-admin/post-new.php", func(c *mizu.Ctx) error {
		postType := c.Query("post_type")
		if postType == "page" {
			return s.wpAdminHandler.PageNew(c)
		}
		return s.wpAdminHandler.PostNew(c)
	})
	s.app.Get("/wp-admin/post.php", func(c *mizu.Ctx) error {
		postType := c.Query("post_type")
		if postType == "page" {
			return s.wpAdminHandler.PageEdit(c)
		}
		return s.wpAdminHandler.PostEdit(c)
	})

	// Media
	s.app.Get("/wp-admin/upload.php", s.wpAdminHandler.MediaLibrary)
	s.app.Get("/wp-admin/media-new.php", s.wpAdminHandler.MediaNew)

	// Comments
	s.app.Get("/wp-admin/edit-comments.php", s.wpAdminHandler.CommentsList)
	s.app.Get("/wp-admin/comment.php", func(c *mizu.Ctx) error {
		action := c.Query("action")
		if action == "editcomment" || action == "edit" {
			return s.wpAdminHandler.CommentEdit(c)
		}
		return s.wpAdminHandler.CommentAction(c)
	})

	// Taxonomies
	s.app.Get("/wp-admin/edit-tags.php", s.wpAdminHandler.TaxonomyList)

	// Appearance
	s.app.Get("/wp-admin/nav-menus.php", s.wpAdminHandler.MenusPage)

	// Users
	s.app.Get("/wp-admin/users.php", s.wpAdminHandler.UsersList)
	s.app.Get("/wp-admin/user-new.php", s.wpAdminHandler.UserNew)
	s.app.Get("/wp-admin/user-edit.php", s.wpAdminHandler.UserEdit)
	s.app.Get("/wp-admin/profile.php", s.wpAdminHandler.Profile)

	// Settings
	s.app.Get("/wp-admin/options-general.php", s.wpAdminHandler.SettingsGeneral)
	s.app.Get("/wp-admin/options-writing.php", s.wpAdminHandler.SettingsWriting)
	s.app.Get("/wp-admin/options-reading.php", s.wpAdminHandler.SettingsReading)
	s.app.Get("/wp-admin/options-discussion.php", s.wpAdminHandler.SettingsDiscussion)
	s.app.Get("/wp-admin/options-media.php", s.wpAdminHandler.SettingsMedia)
	s.app.Get("/wp-admin/options-permalink.php", s.wpAdminHandler.SettingsPermalinks)

	// Clean URL routes (without .php extensions) - aliases for modern URLs
	s.app.Get("/wp-admin/login", s.wpAdminHandler.Login)
	s.app.Post("/wp-admin/login", s.wpAdminHandler.LoginPost)
	s.app.Get("/wp-admin/logout", s.wpAdminHandler.Logout)

	// Posts (clean URLs)
	s.app.Get("/wp-admin/posts", s.wpAdminHandler.PostsList)
	s.app.Get("/wp-admin/posts/new", s.wpAdminHandler.PostNew)
	s.app.Get("/wp-admin/posts/{id}", s.wpAdminHandler.PostEdit)

	// Pages (clean URLs)
	s.app.Get("/wp-admin/pages", s.wpAdminHandler.PagesList)
	s.app.Get("/wp-admin/pages/new", s.wpAdminHandler.PageNew)
	s.app.Get("/wp-admin/pages/{id}", s.wpAdminHandler.PageEdit)

	// Media (clean URLs)
	s.app.Get("/wp-admin/media", s.wpAdminHandler.MediaLibrary)
	s.app.Get("/wp-admin/media/new", s.wpAdminHandler.MediaNew)
	s.app.Get("/wp-admin/media/{id}", s.wpAdminHandler.MediaEdit)

	// Comments (clean URLs)
	s.app.Get("/wp-admin/comments", s.wpAdminHandler.CommentsList)
	s.app.Get("/wp-admin/comments/{id}", s.wpAdminHandler.CommentEdit)

	// Taxonomies (clean URLs)
	s.app.Get("/wp-admin/categories", func(c *mizu.Ctx) error {
		c.Request().URL.RawQuery = "taxonomy=category"
		return s.wpAdminHandler.TaxonomyList(c)
	})
	s.app.Get("/wp-admin/tags", func(c *mizu.Ctx) error {
		c.Request().URL.RawQuery = "taxonomy=post_tag"
		return s.wpAdminHandler.TaxonomyList(c)
	})

	// Menus (clean URL)
	s.app.Get("/wp-admin/menus", s.wpAdminHandler.MenusPage)

	// Users (clean URLs)
	s.app.Get("/wp-admin/users/new", s.wpAdminHandler.UserNew)
	s.app.Get("/wp-admin/users/{id}", s.wpAdminHandler.UserEdit)

	// Settings (clean URLs)
	s.app.Get("/wp-admin/settings/general", s.wpAdminHandler.SettingsGeneral)
	s.app.Get("/wp-admin/settings/writing", s.wpAdminHandler.SettingsWriting)
	s.app.Get("/wp-admin/settings/reading", s.wpAdminHandler.SettingsReading)
	s.app.Get("/wp-admin/settings/discussion", s.wpAdminHandler.SettingsDiscussion)
	s.app.Get("/wp-admin/settings/media", s.wpAdminHandler.SettingsMedia)
	s.app.Get("/wp-admin/settings/permalinks", s.wpAdminHandler.SettingsPermalinks)

	// ============================================================
	// WordPress Admin POST routes - Form Submissions
	// ============================================================

	// Posts
	s.app.Post("/wp-admin/post.php", func(c *mizu.Ctx) error {
		action := c.Request().FormValue("action")
		postType := c.Query("post_type")
		if postType == "" {
			postType = c.Request().FormValue("post_type")
		}

		switch action {
		case "trash":
			if postType == "page" {
				return s.wpAdminHandler.PageTrash(c)
			}
			return s.wpAdminHandler.PostTrash(c)
		case "untrash":
			return s.wpAdminHandler.PostRestore(c)
		case "delete":
			return s.wpAdminHandler.PostDelete(c)
		case "post-quickdraft-save":
			return s.wpAdminHandler.QuickDraftSave(c)
		default:
			if postType == "page" {
				return s.wpAdminHandler.PageSave(c)
			}
			return s.wpAdminHandler.PostSave(c)
		}
	})

	// Bulk post actions
	s.app.Post("/wp-admin/edit.php", s.wpAdminHandler.BulkPostAction)

	// Media
	s.app.Post("/wp-admin/upload.php", s.wpAdminHandler.MediaUpload)
	s.app.Post("/wp-admin/media.php", s.wpAdminHandler.MediaSave)
	s.app.Get("/wp-admin/media.php", func(c *mizu.Ctx) error {
		action := c.Query("action")
		if action == "delete" {
			return s.wpAdminHandler.MediaDelete(c)
		}
		return s.wpAdminHandler.MediaEdit(c)
	})

	// Comments
	s.app.Post("/wp-admin/comment.php", s.wpAdminHandler.CommentSave)
	s.app.Post("/wp-admin/edit-comments.php", s.wpAdminHandler.BulkCommentAction)

	// Taxonomies
	s.app.Post("/wp-admin/edit-tags.php", func(c *mizu.Ctx) error {
		taxonomy := c.Query("taxonomy")
		if taxonomy == "" {
			taxonomy = c.Request().FormValue("taxonomy")
		}
		action := c.Query("action")
		if action == "" {
			action = c.Request().FormValue("action")
		}

		if action == "delete" {
			if taxonomy == "post_tag" {
				return s.wpAdminHandler.TagDelete(c)
			}
			return s.wpAdminHandler.CategoryDelete(c)
		}

		if taxonomy == "post_tag" {
			return s.wpAdminHandler.TagSave(c)
		}
		return s.wpAdminHandler.CategorySave(c)
	})

	// Users
	s.app.Post("/wp-admin/user-new.php", s.wpAdminHandler.UserSave)
	s.app.Post("/wp-admin/user-edit.php", s.wpAdminHandler.UserSave)
	s.app.Post("/wp-admin/profile.php", s.wpAdminHandler.ProfileSave)

	// Settings
	s.app.Post("/wp-admin/options.php", s.wpAdminHandler.SettingsSave)
	s.app.Post("/wp-admin/options-general.php", s.wpAdminHandler.SettingsSave)
	s.app.Post("/wp-admin/options-writing.php", s.wpAdminHandler.SettingsSave)
	s.app.Post("/wp-admin/options-reading.php", s.wpAdminHandler.SettingsSave)
	s.app.Post("/wp-admin/options-discussion.php", s.wpAdminHandler.SettingsSave)
	s.app.Post("/wp-admin/options-media.php", s.wpAdminHandler.SettingsSave)
	s.app.Post("/wp-admin/options-permalink.php", s.wpAdminHandler.SettingsSave)

	// Menus
	s.app.Post("/wp-admin/nav-menus.php", func(c *mizu.Ctx) error {
		action := c.Request().FormValue("action")
		if action == "add-item" {
			return s.wpAdminHandler.MenuItemSave(c)
		}
		if action == "delete-item" {
			return s.wpAdminHandler.MenuItemDelete(c)
		}
		return s.wpAdminHandler.MenuSave(c)
	})

	// ============================================================
	// Obake Admin Routes
	// ============================================================

	// Obake Admin static assets
	s.app.Get("/obake/css/{filename...}", func(c *mizu.Ctx) error {
		filename := c.Param("filename")
		subFS, _ := fs.Sub(staticFS, "css")
		http.StripPrefix("/obake/css/", http.FileServer(http.FS(subFS))).ServeHTTP(c.Writer(), c.Request())
		_ = filename
		return nil
	})
	s.app.Get("/obake/js/{filename...}", func(c *mizu.Ctx) error {
		filename := c.Param("filename")
		subFS, _ := fs.Sub(staticFS, "js")
		http.StripPrefix("/obake/js/", http.FileServer(http.FS(subFS))).ServeHTTP(c.Writer(), c.Request())
		_ = filename
		return nil
	})

	// Obake Auth routes
	s.app.Get("/obake/signin/", s.obakeHandler.Login)
	s.app.Post("/obake/signin/", s.obakeHandler.LoginPost)
	s.app.Get("/obake/signout/", s.obakeHandler.Logout)

	// Dashboard
	s.app.Get("/obake/", s.obakeHandler.Dashboard)
	s.app.Get("/obake/dashboard/", s.obakeHandler.Dashboard)

	// Posts
	s.app.Get("/obake/posts/", s.obakeHandler.PostsList)
	s.app.Get("/obake/editor/post/", s.obakeHandler.PostNew)
	s.app.Get("/obake/editor/post/{id}/", s.obakeHandler.PostEdit)
	s.app.Post("/obake/editor/post/", s.obakeHandler.PostSave)
	s.app.Post("/obake/editor/post/{id}/", s.obakeHandler.PostSave)

	// Pages
	s.app.Get("/obake/pages/", s.obakeHandler.PagesList)
	s.app.Get("/obake/editor/page/", s.obakeHandler.PageNew)
	s.app.Get("/obake/editor/page/{id}/", s.obakeHandler.PageEdit)
	s.app.Post("/obake/editor/page/", s.obakeHandler.PageSave)
	s.app.Post("/obake/editor/page/{id}/", s.obakeHandler.PageSave)

	// Tags
	s.app.Get("/obake/tags/", s.obakeHandler.TagsList)
	s.app.Get("/obake/tags/new/", s.obakeHandler.TagNew)
	s.app.Get("/obake/tags/{slug}/", s.obakeHandler.TagEdit)
	s.app.Post("/obake/tags/", s.obakeHandler.TagSave)
	s.app.Post("/obake/tags/{slug}/", s.obakeHandler.TagSave)
	s.app.Post("/obake/tags/{slug}/delete/", s.obakeHandler.TagDelete)

	// Members
	s.app.Get("/obake/members/", s.obakeHandler.MembersList)
	s.app.Get("/obake/members/{id}/", s.obakeHandler.MemberDetail)

	// Staff
	s.app.Get("/obake/settings/staff/", s.obakeHandler.StaffList)
	s.app.Get("/obake/settings/staff/{slug}/", s.obakeHandler.StaffEdit)
	s.app.Post("/obake/settings/staff/{slug}/", s.obakeHandler.StaffSave)
	s.app.Post("/obake/settings/staff/invite/", s.obakeHandler.StaffInvite)

	// Settings
	s.app.Get("/obake/settings/", s.obakeHandler.SettingsGeneral)
	s.app.Get("/obake/settings/general/", s.obakeHandler.SettingsGeneral)
	s.app.Get("/obake/settings/design/", s.obakeHandler.SettingsDesign)
	s.app.Get("/obake/settings/membership/", s.obakeHandler.SettingsMembership)
	s.app.Get("/obake/settings/email/", s.obakeHandler.SettingsEmail)
	s.app.Get("/obake/settings/advanced/", s.obakeHandler.SettingsAdvanced)
	s.app.Post("/obake/settings/save/", s.obakeHandler.SettingsSave)

	// Search
	s.app.Get("/obake/search/", s.obakeHandler.Search)

	// Media Library
	s.app.Get("/obake/media/", s.obakeHandler.MediaLibrary)
	s.app.Post("/obake/media/upload/", s.obakeHandler.MediaUpload)

	// Export
	s.app.Get("/obake/settings/export/", s.obakeHandler.Export)

	// ============================================================
	// Frontend Site Routes
	// ============================================================

	// Theme static assets
	themeAssetsFS := assets.ThemeAssets()
	s.app.Get("/theme/assets/{filepath...}", func(c *mizu.Ctx) error {
		filepath := c.Param("filepath")
		http.StripPrefix("/theme/assets/", http.FileServer(http.FS(themeAssetsFS))).ServeHTTP(c.Writer(), c.Request())
		_ = filepath
		return nil
	})

	// Homepage
	s.app.Get("/", s.siteHandler.Home)

	// RSS Feed
	s.app.Get("/feed", s.siteHandler.Feed)

	// Search
	s.app.Get("/search", s.siteHandler.Search)

	// Archive
	s.app.Get("/archive", s.siteHandler.Archive)

	// Category archives
	s.app.Get("/category/{slug}", s.siteHandler.Category)

	// Tag archives
	s.app.Get("/tag/{slug}", s.siteHandler.Tag)

	// Author archives
	s.app.Get("/author/{slug}", s.siteHandler.Author)

	// Static pages
	s.app.Get("/page/{slug}", s.siteHandler.Page)

	// Single posts (must be last to catch all slugs)
	s.app.Get("/{slug}", s.siteHandler.Post)
}

// Service accessors for CLI

// UserService returns the users service.
func (s *Server) UserService() users.API {
	return s.users
}

// PostService returns the posts service.
func (s *Server) PostService() posts.API {
	return s.posts
}

// PageService returns the pages service.
func (s *Server) PageService() pages.API {
	return s.pages
}

// CategoryService returns the categories service.
func (s *Server) CategoryService() categories.API {
	return s.categories
}

// TagService returns the tags service.
func (s *Server) TagService() tags.API {
	return s.tags
}

// MediaService returns the media service.
func (s *Server) MediaService() media.API {
	return s.media
}

// CommentService returns the comments service.
func (s *Server) CommentService() comments.API {
	return s.comments
}

// SettingsService returns the settings service.
func (s *Server) SettingsService() settings.API {
	return s.settings
}

// MenuService returns the menus service.
func (s *Server) MenuService() menus.API {
	return s.menus
}
