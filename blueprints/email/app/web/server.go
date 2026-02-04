// Package web provides the HTTP server for the email application.
package web

import (
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/blueprints/email/app/web/handler/api"
	"github.com/go-mizu/mizu/blueprints/email/assets"
	"github.com/go-mizu/mizu/blueprints/email/pkg/email"
	"github.com/go-mizu/mizu/blueprints/email/store"
)

// NewServer creates an HTTP handler for the email application.
// In dev mode the frontend shows a text message; in production
// it serves embedded static files with SPA fallback.
func NewServer(st store.Store, driver email.Driver, fromAddr string, devMode bool) (http.Handler, error) {
	app := mizu.New()

	// Health check
	app.Get("/health", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Create handlers
	emailHandler := api.NewEmailHandler(st, driver, fromAddr)
	labelHandler := api.NewLabelHandler(st)
	contactHandler := api.NewContactHandler(st)
	settingsHandler := api.NewSettingsHandler(st)
	draftHandler := api.NewDraftHandler(st, driver, fromAddr)
	attachmentHandler := api.NewAttachmentHandler(st)

	// Register API routes
	app.Group("/api", func(r *mizu.Router) {
		// Emails
		r.Get("/emails", emailHandler.List)
		r.Get("/emails/{id}", emailHandler.Get)
		r.Post("/emails", emailHandler.Create)
		r.Put("/emails/{id}", emailHandler.Update)
		r.Delete("/emails/{id}", emailHandler.Delete)
		r.Post("/emails/{id}/reply", emailHandler.Reply)
		r.Post("/emails/{id}/reply-all", emailHandler.ReplyAll)
		r.Post("/emails/{id}/forward", emailHandler.Forward)
		r.Post("/emails/{id}/snooze", emailHandler.Snooze)
		r.Post("/emails/{id}/unsnooze", emailHandler.Unsnooze)
		r.Post("/emails/batch", emailHandler.Batch)

		// Attachments
		r.Get("/emails/{id}/attachments", attachmentHandler.List)
		r.Post("/emails/{id}/attachments", attachmentHandler.Upload)
		r.Get("/attachments/{id}", attachmentHandler.Download)
		r.Delete("/attachments/{id}", attachmentHandler.Delete)

		// Threads
		r.Get("/threads", threadList(st))
		r.Get("/threads/{id}", threadGet(st))

		// Labels
		r.Get("/labels", labelHandler.List)
		r.Post("/labels", labelHandler.Create)
		r.Put("/labels/{id}", labelHandler.Update)
		r.Delete("/labels/{id}", labelHandler.Delete)

		// Contacts
		r.Get("/contacts", contactHandler.List)
		r.Post("/contacts", contactHandler.Create)
		r.Put("/contacts/{id}", contactHandler.Update)
		r.Delete("/contacts/{id}", contactHandler.Delete)

		// Settings
		r.Get("/settings", settingsHandler.Get)
		r.Put("/settings", settingsHandler.Update)

		// Search
		r.Get("/search", emailHandler.Search)

		// Drafts
		r.Post("/drafts", draftHandler.Save)
		r.Put("/drafts/{id}", draftHandler.Update)
		r.Delete("/drafts/{id}", draftHandler.Delete)
		r.Post("/drafts/{id}/send", draftHandler.Send)

		// Driver status
		r.Get("/driver/status", func(c *mizu.Ctx) error {
			return c.JSON(http.StatusOK, map[string]any{
				"driver":     driver.Name(),
				"configured": driver.Name() != "noop",
				"from":       fromAddr,
			})
		})
	})

	// Serve frontend
	if devMode {
		app.Get("/{path...}", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "Email app running in development mode. Start the frontend dev server separately.")
		})
	} else {
		serveFrontend(app)
	}

	return app, nil
}

// threadList returns a handler that lists threads.
func threadList(st store.Store) mizu.Handler {
	return func(c *mizu.Ctx) error {
		filter := store.EmailFilter{
			LabelID: c.Query("label"),
			Query:   c.Query("q"),
			Page:    1,
			PerPage: 50,
		}
		if p := c.Query("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				filter.Page = v
			}
		}
		if pp := c.Query("per_page"); pp != "" {
			if v, err := strconv.Atoi(pp); err == nil && v > 0 && v <= 100 {
				filter.PerPage = v
			}
		}
		resp, err := st.ListThreads(c.Context(), filter)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list threads"})
		}
		return c.JSON(http.StatusOK, resp)
	}
}

// threadGet returns a handler that fetches a single thread.
func threadGet(st store.Store) mizu.Handler {
	return func(c *mizu.Ctx) error {
		id := c.Param("id")
		if id == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "thread id is required"})
		}
		thread, err := st.GetThread(c.Context(), id)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "thread not found"})
		}
		return c.JSON(http.StatusOK, thread)
	}
}

// serveFrontend serves embedded static files with SPA fallback.
// Any request that does not match a real file is served index.html
// so that client-side routing works correctly.
func serveFrontend(app *mizu.App) {
	staticFS, err := fs.Sub(assets.StaticFS, "static")
	if err != nil {
		app.Get("/{path...}", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "Email app. No frontend assets found.")
		})
		return
	}

	// Read index.html once at startup for the SPA fallback.
	indexHTML, readErr := fs.ReadFile(staticFS, "index.html")
	if readErr != nil {
		indexHTML = []byte("<html><body><h1>Email</h1><p>Frontend not built yet.</p></body></html>")
	}

	httpFS := http.FS(staticFS)
	fileServer := http.FileServer(httpFS)

	app.Get("/{path...}", func(c *mizu.Ctx) error {
		reqPath := c.Request().URL.Path
		if reqPath == "/" {
			c.Header().Set("Content-Type", "text/html; charset=utf-8")
			return c.HTML(http.StatusOK, string(indexHTML))
		}

		// Strip leading slash for fs.Open
		clean := strings.TrimPrefix(reqPath, "/")

		// Try to open the requested file
		if info, statErr := fs.Stat(staticFS, clean); statErr == nil && !info.IsDir() {
			fileServer.ServeHTTP(c.Writer(), c.Request())
			return nil
		}

		// SPA fallback: serve index.html for non-asset paths
		if !isStaticAsset(reqPath) {
			c.Header().Set("Content-Type", "text/html; charset=utf-8")
			return c.HTML(http.StatusOK, string(indexHTML))
		}

		return c.Text(http.StatusNotFound, "Not found")
	})
}

// isStaticAsset returns true if the path looks like a static asset.
func isStaticAsset(path string) bool {
	exts := []string{
		".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico",
		".woff", ".woff2", ".ttf", ".eot", ".map", ".json", ".webp",
		".avif", ".mp4", ".webm",
	}
	lower := strings.ToLower(path)
	for _, ext := range exts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}
