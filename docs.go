/*
Package mizu provides a small, idiomatic foundation for building HTTP servers in Go.
It builds on the standard library router and http package, adds a tiny request
context (Ctx), a friendly Router API, a structured request logger, and an App
type that owns startup, readiness, and graceful shutdown.

# Design
  - Stay close to net/http so code feels familiar.
  - Make common tasks short and clear without hiding control.
  - Interoperate cleanly with existing http.Handler and middleware.
  - Give a simple App that does the boring but important lifecycle work.

# Quick start

	import (
		"log"
		"net/http"

		"github.com/go-mizu/mizu"
	)

	func main() {
		app := mizu.New()          // App with a Router embedded
		app.Dev(true)              // color logs on stderr + request logger
		app.Get("/hello/:name", func(c *mizu.Ctx) error {
			return c.Text(200, "hi "+c.Param("name"))
		})
		app.Static("/assets/", http.Dir("./public"))

		// Health endpoint returns 200, then flips to 503 during shutdown
		http.Handle("/healthz", app.HealthzHandler())

		// Serve the App as your main handler
		log.Fatal(app.Listen(":8080"))
	}

# Routing

Use method helpers to register handlers. Paths follow Go 1.22 ServeMux rules
including trailing slash patterns for subtrees.

	app.Get("/users", listUsers)
	app.Post("/users", createUser)
	app.Put("/users/:id", updateUser)
	app.Delete("/users/:id", deleteUser)

Use Prefix and Group to organize routes.

	api := app.Prefix("/api")
	api.Group("/v1", func(g *mizu.Router) {
		g.Get("/ping", func(c *mizu.Ctx) error { return c.Text(200, "pong") })
	})

# Middleware

Middleware is a simple function: func(Handler) Handler. Order is the order you
pass to Use, from outside to inside.

	app.Use(requestID, auditLog)
	app.Get("/me", requireAuth, func(c *mizu.Ctx) error {
		return c.JSON(200, map[string]string{"hello": "you"})
	})

If you already have net/http middleware, adapt it with the Compat facade.

	// Apply standard middleware around Mizu handlers
	app.Compat.Use(recoverer, gzipMiddleware)

# Context

Ctx wraps http.Request and http.ResponseWriter with small helpers. You can read
path params, query, forms, and JSON, and write text, HTML, JSON, files, and
streams.

	id := c.Param("id")
	q := c.Query("q")
	form, _ := c.Form()

	var in CreateUser
	_ = c.BindJSON(&in, 1<<20) // 1 MiB limit

	_ = c.Text(200, "ok")
	_ = c.HTML(200, "<h1>Hello</h1>")
	_ = c.JSON(201, map[string]any{"id": 1})
	_ = c.File("./public/logo.png")
	_ = c.Download("./report.csv", "report-2025-10.csv")

	_ = c.Stream(func(w io.Writer) error { _, _ = io.WriteString(w, "chunk"); return nil })

	ch := make(chan any, 1)
	ch <- map[string]int{"n": 1}
	close(ch)
	_ = c.SSE(ch)

# Errors

Handlers return error. If a handler returns an error or panics, Router calls
the central ErrorHandler when set. Otherwise Router writes 500.

	app.ErrorHandler(func(c *mizu.Ctx, err error) {
		c.Logger().Error("handler failed", slog.Any("error", err))
		_ = c.JSON(500, map[string]string{"error": "internal"})
	})

# Not found and method not allowed

If no route matches, Router writes 404. You can replace it.

	app.NotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))

If the path exists but the method does not, Router writes 405 and sets the
Allow header, which always includes OPTIONS.

# Static files

Serve a directory under a URL prefix.

	// GET /assets/app.js serves ./public/app.js
	app.Static("/assets/", http.Dir("./public"))

# Logging

Use the Logger middleware for one structured log per request. Pass your own
slog.Logger or let it choose a mode.

	// Custom logger
	lg := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	app.Use(mizu.Logger(mizu.LoggerOptions{Logger: lg, UserAgent: true}))

	// Automatic mode selection and extra fields
	app.Use(mizu.Logger(mizu.LoggerOptions{
		Mode:   mizu.Auto,
		UserAgent: true,
		RequestIDGen: func() string { return uuid.NewString() },
		TraceExtractor: func(ctx context.Context) (string, string, bool) {
			// return traceID, spanID, sampled
			return "", "", false
		},
	}))

# App lifecycle

App owns the server lifetime, structured startup logs, readiness, and graceful
shutdown. It flips readiness to 503 before shutdown, sleeps a short delay so
load balancers drain, disables keep alives, runs http.Server.Shutdown with a
timeout for in flight requests, then forces close if needed.

	// Basic
	app := mizu.New()
	_ = app.Listen(":3000")

	// TLS
	_ = app.ListenTLS(":3443", "server.crt", "server.key")

	// External listener
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	_ = app.Serve(ln)

	// Custom shutdown tuning
	app = mizu.New(
		mizu.WithPreShutdownDelay(1*time.Second),
		mizu.WithShutdownTimeout(15*time.Second),
		mizu.WithForceCloseDelay(3*time.Second),
	)

# Health checks

Mount the readiness handler on your server. It returns 200 when ready and 503
after shutdown begins.

	http.Handle("/healthz", app.HealthzHandler())

Interop with net/http

Mount any http.Handler, or register a standard handler for a specific method.

	app.Mount("/ui", myMux)
	app.Compat.HandleMethod("POST", "/hook", myStdHandler)

# Testing

Use httptest to call routes in unit tests.

	app := mizu.New()
	app.Get("/ping", func(c *mizu.Ctx) error { return c.Text(200, "pong") })

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rr := httptest.NewRecorder()
	app.ServeHTTP(rr, req)

	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "pong" {
		t.Fatal("unexpected response")
	}

# Version and compatibility

Mizu targets Go 1.22 and newer. It follows semver for public symbols. Keep your
toolchain patched since many security fixes arrive in standard library updates.
*/
package mizu
