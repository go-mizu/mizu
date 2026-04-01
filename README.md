# Mizu

[![Go Reference](https://pkg.go.dev/badge/github.com/go-mizu/mizu.svg)](https://pkg.go.dev/github.com/go-mizu/mizu)
[![CI](https://github.com/go-mizu/mizu/actions/workflows/test.yml/badge.svg)](https://github.com/go-mizu/mizu/actions/workflows/test.yml)

> A lightweight, composable web framework for Go.

Mizu builds on `net/http` (Go 1.22+ ServeMux patterns) and adds a thin context layer, middleware support, and app lifecycle management. No custom DSLs, no hidden global state — just plain Go.

### Quickstart

```bash
go get github.com/go-mizu/mizu@latest
```

```go
package main

import "github.com/go-mizu/mizu"

func main() {
	app := mizu.New()

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(200, "Hello, Mizu!")
	})

	app.Listen(":3000")
}
```

### Features

- **Router** — thin wrapper over `http.ServeMux` with method helpers (`Get`, `Post`, `Put`, `Delete`), groups, and prefixes
- **Ctx** — request context with helpers for params, query, forms, JSON binding, and response writing (Text, JSON, HTML, Stream, SSE)
- **Middleware** — `func(Handler) Handler` composable pattern, compatible with standard `net/http` middleware via `Compat`
- **App lifecycle** — structured startup, readiness checks, graceful shutdown with configurable timeouts
- **Logging** — structured request logger built on `slog`

### Routing

```go
app.Get("/users", listUsers)
app.Post("/users", createUser)
app.Put("/users/:id", updateUser)
app.Delete("/users/:id", deleteUser)

api := app.Prefix("/api")
api.Group("/v1", func(g *mizu.Router) {
    g.Get("/ping", func(c *mizu.Ctx) error { return c.Text(200, "pong") })
})
```

### Middleware

```go
app.Use(requestID, auditLog)

// Standard net/http middleware works too
app.Compat.Use(recoverer, gzipMiddleware)
```

### Context helpers

```go
id := c.Param("id")
q := c.Query("q")

var in CreateUser
c.BindJSON(&in, 1<<20)

c.Text(200, "ok")
c.JSON(201, map[string]any{"id": 1})
c.HTML(200, "<h1>Hello</h1>")
c.Stream(func(w io.Writer) error { _, _ = io.WriteString(w, "chunk"); return nil })
c.SSE(ch)
```

### App lifecycle

```go
app := mizu.New(
    mizu.WithPreShutdownDelay(1*time.Second),
    mizu.WithShutdownTimeout(15*time.Second),
)

app.Listen(":8080")
app.ListenTLS(":8443", "cert.pem", "key.pem")

http.Handle("/healthz", app.HealthzHandler())
```

### Contributing

Contributions are welcome — report issues, suggest improvements, or submit pull requests.

### License

MIT License © 2025 Mizu Contributors
