package web

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/cms/app/web/handler/rest"
	"github.com/go-mizu/mizu/blueprints/cms/feature/accounts"
	"github.com/go-mizu/mizu/blueprints/cms/feature/options"
	"github.com/go-mizu/mizu/blueprints/cms/feature/posts"
	"github.com/go-mizu/mizu/blueprints/cms/feature/terms"
	"github.com/go-mizu/mizu/blueprints/cms/store/duckdb"
)

// ServerConfig holds server configuration.
type ServerConfig struct {
	Addr    string
	Dev     bool
	DataDir string
}

// Server is the CMS web server.
type Server struct {
	app    *mizu.App
	store  *duckdb.Store
	config ServerConfig

	// Services
	accounts accounts.API
	posts    posts.API
	terms    terms.API
	options  options.API

	// Handlers
	restPosts      *rest.Posts
	restPages      *rest.Pages
	restUsers      *rest.Users
	restCategories *rest.Categories
	restTags       *rest.Tags
	restSettings   *rest.Settings
}

// NewServer creates a new server with the given store and config.
func NewServer(store *duckdb.Store, cfg ServerConfig) (*Server, error) {
	// Create services
	accountsSvc := accounts.NewService(store.Users(), store.Usermeta(), store.Sessions())
	optionsSvc := options.NewService(store.Options())
	termsSvc := terms.NewService(store.Terms(), store.TermTaxonomy(), store.Termmeta())
	postsSvc := posts.NewService(store.Posts(), store.Postmeta(), store.TermRelationships(), store.TermTaxonomy(), store.Options())

	// Create Mizu app
	app := mizu.New()

	s := &Server{
		app:      app,
		store:    store,
		config:   cfg,
		accounts: accountsSvc,
		posts:    postsSvc,
		terms:    termsSvc,
		options:  optionsSvc,
	}

	// Create handlers
	s.restPosts = rest.NewPosts(postsSvc, s.getUserID)
	s.restPages = rest.NewPages(postsSvc, s.getUserID)
	s.restUsers = rest.NewUsers(accountsSvc, s.getUserID)
	s.restCategories = rest.NewCategories(termsSvc, s.getUserID)
	s.restTags = rest.NewTags(termsSvc, s.getUserID)
	s.restSettings = rest.NewSettings(optionsSvc, s.getUserID)

	// Setup routes
	s.setupRoutes()

	return s, nil
}

// Start starts the server and blocks until context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.app.Listen(s.config.Addr)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return s.app.Router
}

// setupRoutes configures all routes.
func (s *Server) setupRoutes() {
	r := s.app.Router

	// WordPress REST API v2
	r.Group("/wp-json", func(api *mizu.Router) {
		// Discovery endpoint
		api.Get("/", s.apiDiscovery)

		// WP REST API v2
		api.Group("/wp/v2", func(v2 *mizu.Router) {
			// Posts
			v2.Get("/posts", s.restPosts.List)
			v2.Post("/posts", s.authRequired(s.restPosts.Create))
			v2.Get("/posts/{id}", s.restPosts.Get)
			v2.Put("/posts/{id}", s.authRequired(s.restPosts.Update))
			v2.Patch("/posts/{id}", s.authRequired(s.restPosts.Update))
			v2.Delete("/posts/{id}", s.authRequired(s.restPosts.Delete))

			// Pages
			v2.Get("/pages", s.restPages.List)
			v2.Post("/pages", s.authRequired(s.restPages.Create))
			v2.Get("/pages/{id}", s.restPages.Get)
			v2.Put("/pages/{id}", s.authRequired(s.restPages.Update))
			v2.Patch("/pages/{id}", s.authRequired(s.restPages.Update))
			v2.Delete("/pages/{id}", s.authRequired(s.restPages.Delete))

			// Users
			v2.Get("/users", s.restUsers.List)
			v2.Post("/users", s.authRequired(s.restUsers.Create))
			v2.Get("/users/{id}", s.restUsers.Get)
			v2.Get("/users/me", s.authRequired(s.restUsers.Me))
			v2.Put("/users/{id}", s.authRequired(s.restUsers.Update))
			v2.Patch("/users/{id}", s.authRequired(s.restUsers.Update))
			v2.Delete("/users/{id}", s.authRequired(s.restUsers.Delete))

			// Categories
			v2.Get("/categories", s.restCategories.List)
			v2.Post("/categories", s.authRequired(s.restCategories.Create))
			v2.Get("/categories/{id}", s.restCategories.Get)
			v2.Put("/categories/{id}", s.authRequired(s.restCategories.Update))
			v2.Patch("/categories/{id}", s.authRequired(s.restCategories.Update))
			v2.Delete("/categories/{id}", s.authRequired(s.restCategories.Delete))

			// Tags
			v2.Get("/tags", s.restTags.List)
			v2.Post("/tags", s.authRequired(s.restTags.Create))
			v2.Get("/tags/{id}", s.restTags.Get)
			v2.Put("/tags/{id}", s.authRequired(s.restTags.Update))
			v2.Patch("/tags/{id}", s.authRequired(s.restTags.Update))
			v2.Delete("/tags/{id}", s.authRequired(s.restTags.Delete))

			// Settings
			v2.Get("/settings", s.authRequired(s.restSettings.Get))
			v2.Put("/settings", s.authRequired(s.restSettings.Update))
			v2.Patch("/settings", s.authRequired(s.restSettings.Update))

			// Types
			v2.Get("/types", s.getTypes)
			v2.Get("/types/{type}", s.getType)

			// Statuses
			v2.Get("/statuses", s.getStatuses)
			v2.Get("/statuses/{status}", s.getStatus)

			// Taxonomies
			v2.Get("/taxonomies", s.getTaxonomies)
			v2.Get("/taxonomies/{taxonomy}", s.getTaxonomy)
		})
	})

	// XML-RPC endpoint (legacy)
	r.Post("/xmlrpc.php", s.handleXMLRPC)

	// Frontend routes
	r.Get("/", s.frontendHome)
	r.Get("/{year}/{month}/{slug}", s.frontendPost)
	r.Get("/page/{slug}", s.frontendPage)
	r.Get("/category/{slug}", s.frontendCategory)
	r.Get("/tag/{slug}", s.frontendTag)
	r.Get("/author/{slug}", s.frontendAuthor)

	// Admin routes (placeholder)
	r.Get("/wp-admin/", s.adminDashboard)
	r.Get("/wp-login.php", s.adminLogin)
}

// getUserID extracts the user ID from the request.
func (s *Server) getUserID(c *mizu.Ctx) string {
	// Check Authorization header
	auth := c.Header().Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		session, err := s.accounts.GetSession(c.Request().Context(), token)
		if err == nil && session != nil {
			return session.UserID
		}
	}

	// Check Basic auth (for application passwords)
	if strings.HasPrefix(auth, "Basic ") {
		// TODO: Implement application password verification
	}

	// Check cookie
	cookie, err := c.Request().Cookie("session")
	if err == nil && cookie != nil {
		session, err := s.accounts.GetSession(c.Request().Context(), cookie.Value)
		if err == nil && session != nil {
			return session.UserID
		}
	}

	return ""
}

// authRequired middleware requires authentication.
func (s *Server) authRequired(next mizu.Handler) mizu.Handler {
	return func(c *mizu.Ctx) error {
		userID := s.getUserID(c)
		if userID == "" {
			return rest.Unauthorized(c, "You are not currently logged in.")
		}
		return next(c)
	}
}

// API Discovery endpoint
func (s *Server) apiDiscovery(c *mizu.Ctx) error {
	return c.JSON(200, map[string]interface{}{
		"name":            "CMS",
		"description":     "WordPress-compatible CMS",
		"url":             s.config.Addr,
		"home":            s.config.Addr,
		"gmt_offset":      "0",
		"timezone_string": "UTC",
		"namespaces":      []string{"wp/v2"},
		"authentication": map[string]interface{}{
			"application-passwords": map[string]interface{}{
				"endpoints": map[string]interface{}{
					"authorization": s.config.Addr + "/wp-admin/authorize-application.php",
				},
			},
		},
		"routes": map[string]interface{}{
			"/wp/v2": map[string]interface{}{
				"namespace": "wp/v2",
				"methods":   []string{"GET"},
			},
			"/wp/v2/posts": map[string]interface{}{
				"namespace": "wp/v2",
				"methods":   []string{"GET", "POST"},
			},
			"/wp/v2/pages": map[string]interface{}{
				"namespace": "wp/v2",
				"methods":   []string{"GET", "POST"},
			},
			"/wp/v2/users": map[string]interface{}{
				"namespace": "wp/v2",
				"methods":   []string{"GET", "POST"},
			},
			"/wp/v2/categories": map[string]interface{}{
				"namespace": "wp/v2",
				"methods":   []string{"GET", "POST"},
			},
			"/wp/v2/tags": map[string]interface{}{
				"namespace": "wp/v2",
				"methods":   []string{"GET", "POST"},
			},
		},
	})
}

// Type endpoints
func (s *Server) getTypes(c *mizu.Ctx) error {
	return c.JSON(200, map[string]interface{}{
		"post": map[string]interface{}{
			"description":  "Blog posts",
			"hierarchical": false,
			"name":         "Posts",
			"slug":         "post",
			"rest_base":    "posts",
		},
		"page": map[string]interface{}{
			"description":  "Static pages",
			"hierarchical": true,
			"name":         "Pages",
			"slug":         "page",
			"rest_base":    "pages",
		},
		"attachment": map[string]interface{}{
			"description":  "Media attachments",
			"hierarchical": false,
			"name":         "Media",
			"slug":         "attachment",
			"rest_base":    "media",
		},
	})
}

func (s *Server) getType(c *mizu.Ctx) error {
	postType := c.Param("type")
	types := map[string]map[string]interface{}{
		"post": {
			"description":  "Blog posts",
			"hierarchical": false,
			"name":         "Posts",
			"slug":         "post",
			"rest_base":    "posts",
		},
		"page": {
			"description":  "Static pages",
			"hierarchical": true,
			"name":         "Pages",
			"slug":         "page",
			"rest_base":    "pages",
		},
	}

	if t, ok := types[postType]; ok {
		return c.JSON(200, t)
	}
	return rest.NotFound(c, "Type not found.")
}

// Status endpoints
func (s *Server) getStatuses(c *mizu.Ctx) error {
	return c.JSON(200, map[string]interface{}{
		"publish": map[string]interface{}{
			"name":   "Published",
			"public": true,
			"slug":   "publish",
		},
		"draft": map[string]interface{}{
			"name":   "Draft",
			"public": false,
			"slug":   "draft",
		},
		"pending": map[string]interface{}{
			"name":   "Pending Review",
			"public": false,
			"slug":   "pending",
		},
		"private": map[string]interface{}{
			"name":   "Private",
			"public": false,
			"slug":   "private",
		},
	})
}

func (s *Server) getStatus(c *mizu.Ctx) error {
	status := c.Param("status")
	statuses := map[string]map[string]interface{}{
		"publish": {"name": "Published", "public": true, "slug": "publish"},
		"draft":   {"name": "Draft", "public": false, "slug": "draft"},
		"pending": {"name": "Pending Review", "public": false, "slug": "pending"},
		"private": {"name": "Private", "public": false, "slug": "private"},
	}

	if st, ok := statuses[status]; ok {
		return c.JSON(200, st)
	}
	return rest.NotFound(c, "Status not found.")
}

// Taxonomy endpoints
func (s *Server) getTaxonomies(c *mizu.Ctx) error {
	return c.JSON(200, map[string]interface{}{
		"category": map[string]interface{}{
			"name":         "Categories",
			"slug":         "category",
			"description":  "Post categories",
			"hierarchical": true,
			"rest_base":    "categories",
			"types":        []string{"post"},
		},
		"post_tag": map[string]interface{}{
			"name":         "Tags",
			"slug":         "post_tag",
			"description":  "Post tags",
			"hierarchical": false,
			"rest_base":    "tags",
			"types":        []string{"post"},
		},
	})
}

func (s *Server) getTaxonomy(c *mizu.Ctx) error {
	taxonomy := c.Param("taxonomy")
	taxonomies := map[string]map[string]interface{}{
		"category": {
			"name":         "Categories",
			"slug":         "category",
			"description":  "Post categories",
			"hierarchical": true,
			"rest_base":    "categories",
			"types":        []string{"post"},
		},
		"post_tag": {
			"name":         "Tags",
			"slug":         "post_tag",
			"description":  "Post tags",
			"hierarchical": false,
			"rest_base":    "tags",
			"types":        []string{"post"},
		},
	}

	if t, ok := taxonomies[taxonomy]; ok {
		return c.JSON(200, t)
	}
	return rest.NotFound(c, "Taxonomy not found.")
}

// XML-RPC handler (placeholder)
func (s *Server) handleXMLRPC(c *mizu.Ctx) error {
	// TODO: Implement XML-RPC handling
	return c.Text(200, "XML-RPC endpoint")
}

// Frontend handlers (placeholders)
func (s *Server) frontendHome(c *mizu.Ctx) error {
	return c.HTML(200, "<html><head><title>CMS</title></head><body><h1>Welcome to CMS</h1><p>WordPress-compatible content management.</p></body></html>")
}

func (s *Server) frontendPost(c *mizu.Ctx) error {
	slug := c.Param("slug")
	return c.HTML(200, "<html><head><title>Post</title></head><body><h1>Post: "+slug+"</h1></body></html>")
}

func (s *Server) frontendPage(c *mizu.Ctx) error {
	slug := c.Param("slug")
	return c.HTML(200, "<html><head><title>Page</title></head><body><h1>Page: "+slug+"</h1></body></html>")
}

func (s *Server) frontendCategory(c *mizu.Ctx) error {
	slug := c.Param("slug")
	return c.HTML(200, "<html><head><title>Category</title></head><body><h1>Category: "+slug+"</h1></body></html>")
}

func (s *Server) frontendTag(c *mizu.Ctx) error {
	slug := c.Param("slug")
	return c.HTML(200, "<html><head><title>Tag</title></head><body><h1>Tag: "+slug+"</h1></body></html>")
}

func (s *Server) frontendAuthor(c *mizu.Ctx) error {
	slug := c.Param("slug")
	return c.HTML(200, "<html><head><title>Author</title></head><body><h1>Author: "+slug+"</h1></body></html>")
}

// Admin handlers (placeholders)
func (s *Server) adminDashboard(c *mizu.Ctx) error {
	return c.HTML(200, "<html><head><title>Admin</title></head><body><h1>Admin Dashboard</h1><p>Coming soon...</p></body></html>")
}

func (s *Server) adminLogin(c *mizu.Ctx) error {
	return c.HTML(200, "<html><head><title>Login</title></head><body><h1>Login</h1><form method='post'><input name='user' placeholder='Username'><input name='pass' type='password' placeholder='Password'><button>Login</button></form></body></html>")
}
