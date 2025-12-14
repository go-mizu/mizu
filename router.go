package mizu

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"slices"
	"strings"
)

// Middleware wraps a Handler to add cross-cutting behavior like logging or auth.
type Middleware func(Handler) Handler

// PanicError describes a recovered panic with its value and stack.
type PanicError struct {
	Value any
	Stack []byte
}

func (e *PanicError) Error() string { return fmt.Sprintf("panic: %v", e.Value) }

// Router is a thin wrapper over Go 1.22 ServeMux with scoped middleware.
type Router struct {
	mux   *http.ServeMux
	base  string
	chain []Middleware

	log *slog.Logger

	// Compat exposes a net/http-first facade.
	Compat *httpRouter
}

// NewRouter creates a router with slog logging and default middleware.
func NewRouter() *Router {
	var h slog.Handler
	if forceColorOn() || supportsColorEnv() {
		h = newColorTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	logger := slog.New(h)

	r := &Router{
		mux: http.NewServeMux(),
		log: logger,
	}
	r.Compat = &httpRouter{r: r}

	// Default logger middleware.
	r.Use(Logger(LoggerOptions{
		Logger: r.Logger(),
	}))

	return r
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) handleErr(c *Ctx, err error) {
	if r.log != nil {
		req := c.Request()
		if perr, ok := err.(*PanicError); ok {
			r.log.Error("panic recovered",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Any("error", perr),
				slog.String("stack", string(perr.Stack)),
			)
		} else {
			r.log.Error("handler error",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Any("error", err),
			)
		}
	}

	if !c.wroteHeader {
		http.Error(
			c.Writer(),
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
	}
}

// Logger returns the router logger.
func (r *Router) Logger() *slog.Logger { return r.log }

// SetLogger sets the router logger. It may be nil.
func (r *Router) SetLogger(l *slog.Logger) *Router {
	r.log = l
	return r
}

// Use appends scoped middleware.
func (r *Router) Use(mw ...Middleware) { r.chain = append(r.chain, mw...) }

// Group creates a prefixed router and executes fn.
func (r *Router) Group(prefix string, fn func(g *Router)) {
	if fn != nil {
		fn(r.Prefix(prefix))
	}
}

// Prefix returns a child router with inherited middleware.
func (r *Router) Prefix(prefix string) *Router {
	child := &Router{
		mux:   r.mux,
		base:  joinPath(r.base, prefix),
		chain: slices.Clone(r.chain),
		log:   r.log,
	}
	child.Compat = &httpRouter{r: child}
	return child
}

// With returns a child router with extra middleware.
func (r *Router) With(mw ...Middleware) *Router {
	child := &Router{
		mux:   r.mux,
		base:  r.base,
		chain: append(slices.Clone(r.chain), mw...),
		log:   r.log,
	}
	child.Compat = &httpRouter{r: child}
	return child
}

// HTTP method helpers.

func (r *Router) Get(p string, h Handler)     { r.handle(http.MethodGet, p, h) }
func (r *Router) Head(p string, h Handler)    { r.handle(http.MethodHead, p, h) }
func (r *Router) Post(p string, h Handler)    { r.handle(http.MethodPost, p, h) }
func (r *Router) Put(p string, h Handler)     { r.handle(http.MethodPut, p, h) }
func (r *Router) Patch(p string, h Handler)   { r.handle(http.MethodPatch, p, h) }
func (r *Router) Delete(p string, h Handler)  { r.handle(http.MethodDelete, p, h) }
func (r *Router) Connect(p string, h Handler) { r.handle(http.MethodConnect, p, h) }
func (r *Router) Trace(p string, h Handler)   { r.handle(http.MethodTrace, p, h) }

func (r *Router) Handle(method, p string, h Handler) {
	m := strings.TrimSpace(method)
	if m == "" {
		m = http.MethodGet
	}
	r.handle(strings.ToUpper(m), p, h)
}

// Static serves files from an http.FileSystem with the boring net/http pattern.
//
// Equivalent to:
//
//	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(fsys)))
//
// Notes:
//   - We always mount a subtree pattern ending with "/".
//   - net/http ServeMux will redirect "/assets" -> "/assets/" for GET/HEAD.
func (r *Router) Static(prefix string, fsys http.FileSystem) {
	if fsys == nil {
		return
	}
	if prefix == "" {
		prefix = "/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// fullPath uses path.Join which drops trailing slashes, so restore it.
	base := r.fullPath(prefix)
	if base != "/" && !strings.HasSuffix(base, "/") {
		base += "/"
	}

	fs := http.FileServer(fsys)

	h := func(c *Ctx) error {
		http.StripPrefix(base, fs).ServeHTTP(c.Writer(), c.Request())
		return nil
	}

	adapted := r.adapt(r.compose(h))
	r.mux.Handle(http.MethodGet+" "+base, adapted)
	r.mux.Handle(http.MethodHead+" "+base, adapted)
}

// Mount mounts a net/http handler through scoped middleware.
func (r *Router) Mount(p string, h http.Handler) { r.Compat.Handle(p, h) }

func (r *Router) handle(method, p string, h Handler) {
	full := r.fullPath(p)
	r.mux.Handle(fmt.Sprintf("%s %s", method, full), r.adapt(r.compose(h)))
}

// compose applies scoped middleware (right-to-left).
func (r *Router) compose(h Handler) Handler {
	for i := len(r.chain) - 1; i >= 0; i-- {
		h = r.chain[i](h)
	}
	return h
}

// adapt converts a Mizu handler to http.Handler.
// It owns panic recovery and centralized error handling for this route.
func (r *Router) adapt(h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		c := newCtx(w, req, r.log)

		defer func() {
			if rec := recover(); rec != nil {
				r.handleErr(c, &PanicError{Value: rec, Stack: debug.Stack()})
			}
		}()

		if err := h(c); err != nil {
			r.handleErr(c, err)
		}
	})
}

// Path helpers.

func (r *Router) fullPath(p string) string {
	if p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return joinPath(r.base, p)
}

func joinPath(base, add string) string {
	switch {
	case base == "" || base == "/":
		return cleanLeading(path.Join("/", add))
	case add == "" || add == "/":
		return cleanLeading(path.Join("/", base))
	default:
		return cleanLeading(path.Join("/", base, add))
	}
}

func cleanLeading(p string) string {
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

/* ---------------- net/http compatibility ---------------- */

type httpRouter struct{ r *Router }

// Handle mounts an http.Handler through scoped middleware.
//
// It registers a plain path pattern ("/x") to avoid conflicts with existing
// method-pattern routes like "GET /x". When both exist, ServeMux prefers method
// patterns for requests that match them.
func (h *httpRouter) Handle(p string, hh http.Handler) *httpRouter {
	full := h.r.fullPath(p)
	h.r.mux.Handle(full, h.r.wrapHTTPHandler(hh))
	return h
}

// HandleMethod mounts an http.Handler for a specific method through scoped middleware.
func (h *httpRouter) HandleMethod(method, p string, hh http.Handler) *httpRouter {
	m := strings.TrimSpace(method)
	if m == "" {
		m = http.MethodGet
	}
	full := h.r.fullPath(p)
	h.r.mux.Handle(fmt.Sprintf("%s %s", strings.ToUpper(m), full), h.r.wrapHTTPHandler(hh))
	return h
}

// Use adapts net/http middleware into scoped Mizu middleware.
func (h *httpRouter) Use(mw ...func(http.Handler) http.Handler) *httpRouter {
	for _, sm := range mw {
		h.r.Use(h.r.adaptStdMiddleware(sm))
	}
	return h
}

// Prefix returns a child httpRouter.
func (h *httpRouter) Prefix(prefix string) *httpRouter {
	return &httpRouter{r: h.r.Prefix(prefix)}
}

// Group executes fn with a prefixed router.
func (h *httpRouter) Group(prefix string, fn func(g *httpRouter)) {
	if fn != nil {
		fn(h.Prefix(prefix))
	}
}

func (r *Router) wrapHTTPHandler(hh http.Handler) http.Handler {
	asMizu := func(c *Ctx) error {
		if hh == nil {
			http.NotFound(c.Writer(), c.Request())
			return nil
		}
		hh.ServeHTTP(c.Writer(), c.Request())
		return nil
	}
	return r.adapt(r.compose(asMizu))
}

// adaptStdMiddleware adapts a net/http middleware into Mizu middleware.
//
// Many net/http middleware replace the request (e.g. req.WithContext()) before
// calling next. Since Mizu reads the request from *Ctx, update c.request to the
// request passed to next so downstream handlers observe it.
func (r *Router) adaptStdMiddleware(sm func(http.Handler) http.Handler) Middleware {
	return func(next Handler) Handler {
		return func(c *Ctx) error {
			var nextErr error

			base := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req != nil {
					c.request = req
				}
				nextErr = next(c)
			})

			sm(base).ServeHTTP(c.Writer(), c.Request())
			return nextErr
		}
	}
}
