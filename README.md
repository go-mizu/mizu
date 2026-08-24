# mizu

[![Go Reference](https://pkg.go.dev/badge/github.com/go-mizu/mizu.svg)](https://pkg.go.dev/github.com/go-mizu/mizu)
[![CI](https://github.com/go-mizu/mizu/actions/workflows/test.yml/badge.svg)](https://github.com/go-mizu/mizu/actions/workflows/test.yml)

Mizu (水, water) is a Go toolkit with the developer experience of Laravel and the shape of the standard library.
Take one package or take all of them.

## Where this is right now

What is published today is `v0.1.0`: a thin layer over `net/http` with a router, a request context, middleware, and app lifecycle.
It works, it is tested, and it is a small fraction of what is planned.

The plan is written down in full, milestone by milestone, in the [tracking issues](https://github.com/go-mizu/mizu/issues?q=is%3Aissue+is%3Aopen+label%3Atracking).
There are 226 items across 11 milestones, each with an effort estimate, acceptance criteria, and the risks that could sink it.
Read [M0](https://github.com/go-mizu/mizu/issues/2) first, since everything else rests on it.

Two things follow from that, and both are worth saying plainly.

The API will change.
`v0.1.0` predates the design the milestones describe, so the shape you see below is the starting point rather than the destination.
Nothing is under a compatibility promise until 1.0, and the version number stays at 0.x until the criteria in [M9](https://github.com/go-mizu/mizu/issues/11) are met rather than until a date arrives.

Documentation comes before code, on purpose.
[go-mizu.dev](https://go-mizu.dev) is built from the same artefacts the toolkit publishes, and a page that describes something unbuilt says so on the page.
Seeing the result the way a user will see it is what tells you whether the design is worth building, and it is much cheaper to change a page than a package.

## What is here today

```bash
go get github.com/go-mizu/mizu@latest
```

```go
package main

import "github.com/go-mizu/mizu"

func main() {
	app := mizu.New()

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(200, "Hello, mizu!")
	})

	app.Listen(":3000")
}
```

**Router.** A thin wrapper over `http.ServeMux` with method helpers, groups, and prefixes.

```go
app.Get("/users", listUsers)
app.Post("/users", createUser)
app.Put("/users/:id", updateUser)

api := app.Prefix("/api")
api.Group("/v1", func(g *mizu.Router) {
	g.Get("/ping", func(c *mizu.Ctx) error { return c.Text(200, "pong") })
})
```

**Context.** Params, query, forms, JSON binding, and responses as text, JSON, HTML, a stream, or server-sent events.

```go
id := c.Param("id")
q := c.Query("q")

var in CreateUser
c.BindJSON(&in, 1<<20)

c.JSON(201, map[string]any{"id": 1})
c.SSE(ch)
```

**Middleware.** The `func(Handler) Handler` shape, and standard `net/http` middleware through `Compat`.

```go
app.Use(requestID, auditLog)
app.Compat.Use(recoverer, gzipMiddleware)
```

**Lifecycle.** Structured startup, readiness, and graceful shutdown.

```go
app := mizu.New(
	mizu.WithPreShutdownDelay(1*time.Second),
	mizu.WithShutdownTimeout(15*time.Second),
)

app.Listen(":8080")
app.ListenTLS(":8443", "cert.pem", "key.pem")

http.Handle("/healthz", app.HealthzHandler())
```

**Logging.** A structured request logger on `slog`.

## Where it is going

The design is a toolkit rather than a framework, and that is the decision everything else follows from.

Sixty-five packages, ten generators, and one CLI.
No package imports the composition root, which is enforced in CI, and that single rule is what makes the rest true: every package works on its own, has a compiled standalone example, and can be adopted without adopting anything else.

Three levels, and you choose which one you are on.

1. **Libraries.** Import `cache`, or `errs`, or `validate`, and nothing else. It behaves like any other Go library.
2. **Wired.** Several packages sharing config, logging, and tracing through six small seams: `ctxdata`, `errs`, `clock`, `log`, `otelx`, and `codec`.
3. **Composed.** `mizu.App`, which is the Laravel-shaped experience, with scaffolding, generators, and the whole stack.

`mizu eject` moves you down a level and deletes itself on the way out.
There is no `mizu inject`, because a one-way exit you can prove works is worth more than a two-way door nobody trusts.

The parts that make this a real choice rather than a slogan are the ones on the critical path: high performance with budgets enforced in CI, RPC and gRPC as a first-class path rather than an afterthought, code generation instead of reflection so that the compiler and your editor both know what is going on, and frontend interoperability that does not care which framework you picked.

## Repositories

| Repository | What is in it |
|---|---|
| [go-mizu/mizu](https://github.com/go-mizu/mizu) | The toolkit, the CLI, and the roadmap tracking issues |
| [go-mizu/docs](https://github.com/go-mizu/docs) | go-mizu.dev, built from the artefacts each release publishes |
| [go-mizu/shizuku](https://github.com/go-mizu/shizuku) | Shizuku (雫, droplet), the design system the site is built from |

## Contributing

Start with [CONTRIBUTING.md](CONTRIBUTING.md).
If you are looking for something to pick up, the [`good first issue`](https://github.com/go-mizu/mizu/labels/good%20first%20issue) and [`help wanted`](https://github.com/go-mizu/mizu/labels/help%20wanted) labels are kept honest, and any unticked item on a tracking issue is fair game if you say so in the thread first.

If you are an AI coding agent, read [AGENTS.md](AGENTS.md).

## Licence

MIT.
Copyright 2025 the mizu contributors.
