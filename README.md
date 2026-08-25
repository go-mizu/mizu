# mizu

[![Go Reference](https://pkg.go.dev/badge/github.com/go-mizu/mizu.svg)](https://pkg.go.dev/github.com/go-mizu/mizu)
[![CI](https://github.com/go-mizu/mizu/actions/workflows/test.yml/badge.svg)](https://github.com/go-mizu/mizu/actions/workflows/test.yml)

Mizu (水, water) is a Go toolkit with the developer experience of Laravel and the shape of the standard library.
Take one package or take all of them.

## Where this is right now

The latest tag is `v0.6.0`, which is a thin layer over `net/http`: a router, a request context, middleware, and app lifecycle.
The packages and the command line tool listed below are on `main` and have not been tagged yet.
All of it works and is tested, and all of it is a small fraction of what is planned.

The plan is written down in full, milestone by milestone, in the [tracking issues](https://github.com/go-mizu/mizu/issues?q=is%3Aissue+is%3Aopen+label%3Atracking).
There are 226 items across 11 milestones, each with an effort estimate, acceptance criteria, and the risks that could sink it.
Read [M0](https://github.com/go-mizu/mizu/issues/2) first, since everything else rests on it.

Two things follow from that, and both are worth saying plainly.

The API will change.
The tagged root package predates the design the milestones describe, so the shape you see below is the starting point rather than the destination.
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
The patterns are the ones `net/http.ServeMux` understands, so a wildcard is written `{id}` and read with `Param`.

```go
app.Get("/users", listUsers)
app.Post("/users", createUser)
app.Put("/users/{id}", updateUser)

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
A signal flips readiness first, so a load balancer takes the instance out before the connections go, and then in-flight requests get the drain window to finish.

```go
app := mizu.New()
app.ShutdownTimeout = 15 * time.Second

app.Listen(":8080")
app.ListenTLS(":8443", "cert.pem", "key.pem")

// Handlers rather than routes, so they can be served on a port
// the traffic is not on.
http.Handle("/livez", app.LivezHandler())
http.Handle("/readyz", app.ReadyzHandler())
```

**Logging.** A structured request logger on `slog`.

## The packages

None of these are in a tag yet, so it is `go get github.com/go-mizu/mizu@main` until there is one, and the links go to the source rather than to pkg.go.dev.

Each one imports the standard library, `golang.org/x`, and at most two of the others.
None of them import the root package, which is the rule the rest of this rests on, and a test asserts it.

| Package | What it is |
| --- | --- |
| [`clock`](https://github.com/go-mizu/mizu/tree/main/clock) | What time it is, taken from the context, so a test can decide |
| [`conc`](https://github.com/go-mizu/mizu/tree/main/conc) | Goroutines that somebody is waiting for |
| [`config`](https://github.com/go-mizu/mizu/tree/main/config) | The value of a setting, and where it came from |
| [`console`](https://github.com/go-mizu/mizu/tree/main/console) | What a command line program says and how it says it |
| [`crypt`](https://github.com/go-mizu/mizu/tree/main/crypt) | The keys an application keeps, the values it hides, the random it draws |
| [`ctxdata`](https://github.com/go-mizu/mizu/tree/main/ctxdata) | A few named values carried with a request, so a log record says which tenant it was for |
| [`errs`](https://github.com/go-mizu/mizu/tree/main/errs) | A failure classified once, so every transport answers for it the same way |
| [`errs/diag`](https://github.com/go-mizu/mizu/tree/main/errs/diag) | One thing wrong, said once, for a person and for a program |
| [`gen`](https://github.com/go-mizu/mizu/tree/main/gen) | The code generation harness, and the two generators on it |
| [`golden`](https://github.com/go-mizu/mizu/tree/main/golden) | Output compared against a file in `testdata`, rewritten with `-update` |
| [`hash`](https://github.com/go-mizu/mizu/tree/main/hash) | Storing passwords, and moving a column of old ones as people sign in |
| [`log`](https://github.com/go-mizu/mizu/tree/main/log) | `slog` handlers, one a person reads and one a machine reads |
| [`str`](https://github.com/go-mizu/mizu/tree/main/str) | The string handling that `strings` leaves out |
| [`toml`](https://github.com/go-mizu/mizu/tree/main/toml) | TOML 1.0.0 |
| [`try`](https://github.com/go-mizu/mizu/tree/main/try) | Running something again when it is worth running again |
| [`xs`](https://github.com/go-mizu/mizu/tree/main/xs), [`xm`](https://github.com/go-mizu/mizu/tree/main/xm) | Sequences and maps, one element at a time and without collecting them |

The fixtures ship with the thing they test rather than as a separate download: [`mizutest`](https://github.com/go-mizu/mizu/tree/main/mizutest), [`console/consoletest`](https://github.com/go-mizu/mizu/tree/main/console/consoletest), [`errs/diag/diagtest`](https://github.com/go-mizu/mizu/tree/main/errs/diag/diagtest), and [`archtest`](https://github.com/go-mizu/mizu/tree/main/archtest), which is what asserts the paragraph above.
`mizutest` is the one exception to it, since a fixture for an application that has already been wired hands back the router the test registers routes on.

## The command line tool

```bash
go install github.com/go-mizu/mizu/cmd/mizu@main
```

`mizu new blog` writes a project that builds, tests, and runs, along with an `AGENTS.md` describing what is in it.
After that there are two commands you run all day:

```bash
mizu check    # type check and vet, the fastest answer there is
mizu verify   # gen, fmt, vet, build, test, doctor, in that order
```

`mizu gen` writes what the markers in the source ask for, `mizu about` prints what a project is made of, and `mizu doctor` says what is wrong with the environment and what to do about it.
Every command takes `--json` and writes a document a program can read, which is the same document the diagnostics come out in.

`go doc github.com/go-mizu/mizu/cmd/mizu` is the reference, and three tests read it back out of the source, so a new command or a renamed exit code fails the build rather than ageing the page.

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
