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

// PanicError describes a recovered panic with its value and stack for error handling.
type PanicError struct {
	Value any
	Stack []byte
}

func (e *PanicError) Error() string { return fmt.Sprintf("panic: %v", e.Value) }

// Router is a small wrapper over Go 1.22 ServeMux with method patterns and middleware.
type Router struct {
	mux   *http.ServeMux
	base  string
	chain []Middleware
	errh  func(*Ctx, error)
	log   *slog.Logger

	// Compat exposes a net/http-first facade for interop.
	Compat *httpRouter
}

// NewRouter creates a router with colored slog logging and default middleware.
func NewRouter() *Router {
	// Choose handler: color if environment or TTY supports it, else plain text.
	var h slog.Handler
	if forceColorOn() || supportsColorEnv() {
		h = newColorTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	logger := slog.New(h)
	slog.SetDefault(logger)

	r := &Router{
		mux: http.NewServeMux(),
		log: logger,
	}

	r.Compat = &httpRouter{r: r}

	// Always include request logger middleware so requests are logged by default.
	r.Use(Logger(LoggerOptions{
		Logger: r.Logger(),
	}))

	return r
}

// canonicalPath normalizes a request path so that "/s3" and "/s3/"
// are treated the same, and dot segments are cleaned.
// Root "/" is preserved as "/".
func canonicalPath(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	// Ensure leading slash for path.Clean to behave as expected.
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// path.Clean removes duplicate slashes and trailing slash (except for root).
	p = path.Clean(p)
	if p == "/" {
		return "/"
	}
	// Strip trailing slash if any, to unify "/s3" and "/s3/".
	p = strings.TrimRight(p, "/")
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

// ServeHTTP implements http.Handler and delegates to the internal ServeMux.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL != nil {
		cp := canonicalPath(req.URL.Path)
		if cp != req.URL.Path {
			clone := req.Clone(req.Context())
			clone.URL.Path = cp
			req = clone
		}
	}
	r.mux.ServeHTTP(w, req)
}

// Logger returns the logger used by this router.
func (r *Router) Logger() *slog.Logger {
	return r.log
}

// SetLogger sets the logger for this router and new Ctx values.
func (r *Router) SetLogger(l *slog.Logger) *Router {
	if l != nil {
		r.log = l
	}
	return r
}

// ErrorHandler sets a central error handler for returned errors or recovered panics.
func (r *Router) ErrorHandler(h func(*Ctx, error)) { r.errh = h }

// NotFound installs a simple fallback handler at "/".
// No 405 logic, no auto HEAD, just what you register.
func (r *Router) NotFound(h http.Handler) {
	if h == nil {
		return
	}
	r.mux.Handle("/", h)
}

// Use appends middleware so later items run closer to the handler.
func (r *Router) Use(mw ...Middleware) {
	r.chain = append(r.chain, mw...)
}

// UseFirst prepends middleware so they run outside the rest of the chain.
func (r *Router) UseFirst(mw ...Middleware) {
	if len(mw) == 0 {
		return
	}
	r.chain = append(append([]Middleware{}, mw...), r.chain...)
}

// Group creates a child router with a path prefix and runs the given function.
func (r *Router) Group(prefix string, fn func(g *Router)) {
	g := r.Prefix(prefix)
	fn(g)
}

// Prefix returns a child router that shares state and adds a URL prefix.
func (r *Router) Prefix(prefix string) *Router {
	child := &Router{
		mux:   r.mux,
		base:  joinPath(r.base, prefix),
		chain: slices.Clone(r.chain),
		errh:  r.errh,
		log:   r.log,
	}
	child.Compat = &httpRouter{r: child}
	return child
}

// Get registers a handler for HTTP GET.
func (r *Router) Get(p string, h Handler) { r.handle(http.MethodGet, p, h) }

// Head registers a handler for HTTP HEAD.
func (r *Router) Head(p string, h Handler) { r.handle(http.MethodHead, p, h) }

// Post registers a handler for HTTP POST.
func (r *Router) Post(p string, h Handler) { r.handle(http.MethodPost, p, h) }

// Put registers a handler for HTTP PUT.
func (r *Router) Put(p string, h Handler) { r.handle(http.MethodPut, p, h) }

// Patch registers a handler for HTTP PATCH.
func (r *Router) Patch(p string, h Handler) { r.handle(http.MethodPatch, p, h) }

// Delete registers a handler for HTTP DELETE.
func (r *Router) Delete(p string, h Handler) { r.handle(http.MethodDelete, p, h) }

// Connect registers a handler for HTTP CONNECT.
func (r *Router) Connect(p string, h Handler) { r.handle(http.MethodConnect, p, h) }

// Trace registers a handler for HTTP TRACE.
func (r *Router) Trace(p string, h Handler) { r.handle(http.MethodTrace, p, h) }

// Handle registers a handler for the given HTTP method and path.
func (r *Router) Handle(method, p string, h Handler) {
	r.handle(strings.ToUpper(method), p, h)
}

// Static serves files from a provided http.FileSystem at the given URL prefix.
// It goes through the middleware chain so logger and other middleware still run.
func (r *Router) Static(prefix string, fsys http.FileSystem) {
	if prefix == "" {
		prefix = "/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	// Canonical base path for the static subtree.
	base := r.fullPath(prefix)
	subtree := base
	if !strings.HasSuffix(subtree, "/") {
		subtree += "/"
	}

	fs := http.FileServer(fsys)

	handler := func(c *Ctx) error {
		// Example:
		//   request path:  /assets/img/logo.png
		//   subtree:       /assets/
		//   StripPrefix -> /img/logo.png
		http.StripPrefix(subtree, fs).ServeHTTP(c.Writer(), c.Request())
		return nil
	}

	// Register as a GET subtree handler, with middleware and adapt().
	composed := r.compose(handler)
	adapted := r.adapt(composed)

	// Go 1.22 ServeMux treats patterns ending with "/" as subtree matches.
	// For root, subtree is "/", which is also a subtree.
	pattern := http.MethodGet + " " + subtree
	r.mux.Handle(pattern, adapted)
}

// Mount mounts any http.Handler at a path.
func (r *Router) Mount(p string, h http.Handler) { r.Compat.Handle(p, h) }

func (r *Router) handle(method, p string, h Handler) {
	full := r.fullPath(p)
	composed := r.compose(h)
	adapted := r.adapt(composed)
	pattern := fmt.Sprintf("%s %s", strings.ToUpper(method), full)

	r.mux.Handle(pattern, adapted)
}

// compose wraps the handler with middleware from right to left.
func (r *Router) compose(h Handler) Handler {
	for i := len(r.chain) - 1; i >= 0; i-- {
		h = r.chain[i](h)
	}
	return h
}

// adapt converts a Mizu Handler into http.Handler with panic recovery and error handling.
func (r *Router) adapt(h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		c := newCtx(w, req, r.log)

		defer func() {
			if rec := recover(); rec != nil {
				perr := &PanicError{Value: rec, Stack: debug.Stack()}
				if r.errh != nil {
					r.errh(c, perr)
					return
				}
				if r.log != nil {
					r.log.Error("panic recovered",
						slog.Any("value", rec),
						slog.String("method", req.Method),
						slog.String("path", req.URL.Path),
					)
				}
				if !c.wroteHeader {
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}
		}()

		if err := h(c); err != nil {
			if r.errh != nil {
				r.errh(c, err)
				return
			}
			if r.log != nil {
				r.log.Error("handler error",
					slog.String("method", req.Method),
					slog.String("path", req.URL.Path),
					slog.Any("error", err),
				)
			}
			if !c.wroteHeader {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}
	})
}

// adaptStdMiddleware converts a standard net/http middleware into a Mizu Middleware.
func (r *Router) adaptStdMiddleware(sm func(http.Handler) http.Handler) Middleware {
	return func(next Handler) Handler {
		base := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			c := newCtx(w, req, r.log)
			if err := next(c); err != nil {
				if r.errh != nil {
					r.errh(c, err)
				} else {
					if r.log != nil {
						r.log.Error("handler error",
							slog.String("method", req.Method),
							slog.String("path", req.URL.Path),
							slog.Any("error", err),
						)
					}
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}
		})
		wrapped := sm(base)
		return func(c *Ctx) error {
			wrapped.ServeHTTP(c.Writer(), c.Request())
			return nil
		}
	}
}

// fullPath joins base and p without extra trailing-slash magic, then canonicalizes.
func (r *Router) fullPath(p string) string {
	if p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return canonicalPath(joinPath(r.base, p))
}

// joinPath joins URL path segments, cleans dot segments, and keeps a single leading slash.
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

type httpRouter struct{ r *Router }

// Use appends standard net/http middlewares to the Mizu chain.
func (h *httpRouter) Use(mw ...func(http.Handler) http.Handler) *httpRouter {
	for _, sm := range mw {
		h.r.chain = append(h.r.chain, h.r.adaptStdMiddleware(sm))
	}
	return h
}

// Handle registers an http.Handler at a path on the shared ServeMux.
func (h *httpRouter) Handle(p string, hh http.Handler) *httpRouter {
	full := h.r.fullPath(p)
	h.r.mux.Handle(full, hh)
	return h
}

// HandleMethod registers an http.Handler for a specific method and path.
func (h *httpRouter) HandleMethod(method, p string, hh http.Handler) *httpRouter {
	full := h.r.fullPath(p)
	pattern := fmt.Sprintf("%s %s", strings.ToUpper(method), full)

	h.r.mux.Handle(pattern, hh)
	return h
}

// Prefix returns a child httpRouter rooted at the given prefix.
func (h *httpRouter) Prefix(prefix string) *httpRouter { return &httpRouter{r: h.r.Prefix(prefix)} }

// Group creates a child httpRouter at prefix and runs the given function.
func (h *httpRouter) Group(prefix string, fn func(g *httpRouter)) {
	fn(h.Prefix(prefix))
}

// Mount mounts an http.Handler at a path using the compatibility facade.
func (h *httpRouter) Mount(p string, hh http.Handler) *httpRouter { return h.Handle(p, hh) }

// With returns a child router that shares state but has extra middleware
// appended to the chain. It is useful for route-specific middleware.
func (r *Router) With(mw ...Middleware) *Router {
	child := &Router{
		mux:   r.mux,
		base:  r.base,
		chain: append(slices.Clone(r.chain), mw...),
		errh:  r.errh,
		log:   r.log,
	}
	child.Compat = &httpRouter{r: child}
	return child
}
