package web

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/go-mizu/mizu"

	"github.com/go-mizu/blueprints/cms/app/web/handler/rest"
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
