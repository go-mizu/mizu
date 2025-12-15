// File: router.go
package mizu

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
)

// Middleware wraps a Handler to add cross-cutting behavior like logging or auth.
type Middleware func(Handler) Handler

// PanicError is returned to the error handler when a panic is recovered.
type PanicError struct {
	Value any
	Stack []byte
}

func (e *PanicError) Error() string { return fmt.Sprintf("panic: %v", e.Value) }

// Router is a thin wrapper over Go's http.ServeMux (Go 1.22+ method patterns)
// that adds middleware and a small Handler/Ctx layer.
//
// Usage
//   - Register routes with Get/Post/.../Handle.
//   - Add global middleware with Use.
//   - Add scoped middleware with With.
//   - Build subtrees with Prefix/Group.
//   - Interop with net/http using Compat (Handle, HandleMethod, Use).
//
// Behavior
//   - One *Ctx is created per request in ServeHTTP and reused for the whole request.
//   - Global middleware runs before routing for every request.
//   - Scoped middleware runs only for handlers registered on that router instance.
//   - Handler errors are surfaced to global middleware by storing them in request context
//     during dispatch and returning them after mux routing completes.
//   - Panics are recovered at the ServeHTTP boundary and forwarded to ErrorHandler (if set),
//     otherwise logged and turned into a 500 if possible.
type Router struct {
	mux   *http.ServeMux
	base  string
	chain []Middleware

	// Global middleware chain. Runs for every request before routing.
	globalChain []Middleware

	// Optional central error handler.
	errh func(*Ctx, error)

	// Logger used by new contexts and default error logging.
	log *slog.Logger

	// Compat exposes a net/http-first facade for interop.
	Compat *httpRouter

	mu         sync.Mutex
	entry      Handler
	entryDirty bool
}

// NewRouter returns a new Router with a default logger and default middleware.
// It installs the request Logger middleware by default.
func NewRouter() *Router {
	var h slog.Handler
	if forceColorOn() || supportsColorEnv() {
		h = newColorTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	r := &Router{
		mux:        http.NewServeMux(),
		log:        slog.New(h),
		entryDirty: true,
	}
	r.Compat = &httpRouter{r: r}

	r.Use(Logger(LoggerOptions{Logger: r.Logger()}))
	return r
}

// ctxKey stores the request-scoped *Ctx in the request context.
type ctxKey struct{}

// routeErrorKey stores a route handler error in request context so it can be returned
// after mux dispatch (net/http handlers cannot return errors directly).
type routeErrorKey struct{}

// ServeHTTP is the net/http entry point.
// It creates one *Ctx per request, runs the cached global chain, and recovers panics.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := newCtx(w, req, r.log)

	// Attach *Ctx to the request context and keep Ctx synchronized.
	req2 := req.WithContext(context.WithValue(req.Context(), ctxKey{}, c))
	c.request = req2
	c.writer = w
	c.rc = http.NewResponseController(w)

	entry := r.cachedEntry()

	defer func() {
		if rec := recover(); rec != nil {
			perr := &PanicError{Value: rec, Stack: debug.Stack()}
			r.handleError(c, perr, "panic recovered", rec)
		}
	}()

	if err := entry(c); err != nil {
		r.handleError(c, err, "handler error", nil)
	}
}

// cachedEntry returns the compiled global chain.
// The chain is rebuilt only when Use(...) changes global middleware.
func (r *Router) cachedEntry() Handler {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.entry != nil && !r.entryDirty {
		return r.entry
	}

	// This delegate runs after global middleware and performs routing.
	// After the mux runs, it checks whether the route handler stored an error.
	var muxDelegate Handler = func(c *Ctx) error {
		r.mux.ServeHTTP(c.Writer(), c.Request())

		if req := c.Request(); req != nil {
			if err, ok := req.Context().Value(routeErrorKey{}).(error); ok && err != nil {
				return err
			}
		}
		return nil
	}

	hnd := muxDelegate
	for i := len(r.globalChain) - 1; i >= 0; i-- {
		hnd = r.globalChain[i](hnd)
	}

	r.entry = hnd
	r.entryDirty = false
	return r.entry
}

// handleError calls ErrorHandler if configured, otherwise logs and writes a 500
// if no response has started.
func (r *Router) handleError(c *Ctx, err error, msg string, panicValue any) {
	if err == nil {
		return
	}

	if r.errh != nil {
		r.errh(c, err)
		return
	}

	if r.log != nil {
		method := ""
		p := ""
		if c != nil && c.Request() != nil {
			method = c.Request().Method
			p = safePath(c.Request())
		}

		if panicValue != nil {
			r.log.Error(
				msg,
				slog.String("method", method),
				slog.String("path", p),
				slog.Any("error", err),
				slog.Any("value", panicValue),
			)
		} else {
			r.log.Error(
				msg,
				slog.String("method", method),
				slog.String("path", p),
				slog.Any("error", err),
			)
		}
	}

	if c != nil && !c.wroteHeader {
		http.Error(c.Writer(), http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func safePath(rq *http.Request) string {
	if rq == nil || rq.URL == nil {
		return ""
	}
	return rq.URL.Path
}

// Logger returns the router logger.
func (r *Router) Logger() *slog.Logger { return r.log }

// SetLogger updates the router logger used by new contexts.
func (r *Router) SetLogger(l *slog.Logger) *Router {
	if l != nil {
		r.log = l
	}
	return r
}

// ErrorHandler sets a central error handler for returned errors and recovered panics.
func (r *Router) ErrorHandler(h func(*Ctx, error)) { r.errh = h }

// Use appends global middleware.
// Global middleware runs for every request before routing.
func (r *Router) Use(mw ...Middleware) {
	if len(mw) == 0 {
		return
	}
	r.globalChain = append(r.globalChain, mw...)

	r.mu.Lock()
	r.entryDirty = true
	r.mu.Unlock()
}

// Group creates a prefixed child router and runs fn with it.
func (r *Router) Group(prefix string, fn func(g *Router)) {
	g := r.Prefix(prefix)
	fn(g)
}

// Prefix returns a child router rooted at prefix.
// It shares the underlying ServeMux but has its own base path and scoped chain.
func (r *Router) Prefix(prefix string) *Router {
	child := &Router{
		mux:         r.mux,
		base:        joinPath(r.base, prefix),
		chain:       slices.Clone(r.chain),
		globalChain: slices.Clone(r.globalChain),
		errh:        r.errh,
		log:         r.log,
		entryDirty:  true,
	}
	child.Compat = &httpRouter{r: child}
	return child
}

// With returns a child router that adds scoped middleware.
// Scoped middleware affects only routes registered via that router value.
func (r *Router) With(mw ...Middleware) *Router {
	child := &Router{
		mux:         r.mux,
		base:        r.base,
		chain:       append(slices.Clone(r.chain), mw...),
		globalChain: slices.Clone(r.globalChain),
		errh:        r.errh,
		log:         r.log,
		entryDirty:  true,
	}
	child.Compat = &httpRouter{r: child}
	return child
}

// Get registers a handler for GET.
func (r *Router) Get(p string, h Handler) { r.handle(http.MethodGet, p, h) }

// Head registers a handler for HEAD.
func (r *Router) Head(p string, h Handler) { r.handle(http.MethodHead, p, h) }

// Post registers a handler for POST.
func (r *Router) Post(p string, h Handler) { r.handle(http.MethodPost, p, h) }

// Put registers a handler for PUT.
func (r *Router) Put(p string, h Handler) { r.handle(http.MethodPut, p, h) }

// Patch registers a handler for PATCH.
func (r *Router) Patch(p string, h Handler) { r.handle(http.MethodPatch, p, h) }

// Delete registers a handler for DELETE.
func (r *Router) Delete(p string, h Handler) { r.handle(http.MethodDelete, p, h) }

// Connect registers a handler for CONNECT.
func (r *Router) Connect(p string, h Handler) { r.handle(http.MethodConnect, p, h) }

// Trace registers a handler for TRACE.
func (r *Router) Trace(p string, h Handler) { r.handle(http.MethodTrace, p, h) }

// Handle registers a handler for method and path.
func (r *Router) Handle(method, p string, h Handler) { r.handle(strings.ToUpper(method), p, h) }

// Static serves files from fsys at the given URL prefix.
// It registers GET and HEAD for the subtree and redirects "/prefix" to "/prefix/" (except "/").
//
// For "/" it does not call StripPrefix("/", ...) because FileServer expects a leading slash.
func (r *Router) Static(prefix string, fsys http.FileSystem) {
	if prefix == "" {
		prefix = "/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	base := r.fullPath(prefix)

	subtree := base
	if subtree != "/" && !strings.HasSuffix(subtree, "/") {
		subtree += "/"
	}

	fs := http.FileServer(fsys)

	serve := func(c *Ctx) error {
		if subtree == "/" {
			fs.ServeHTTP(c.Writer(), c.Request())
			return nil
		}
		http.StripPrefix(subtree, fs).ServeHTTP(c.Writer(), c.Request())
		return nil
	}

	r.handle(http.MethodGet, subtree, serve)
	r.handle(http.MethodHead, subtree, serve)

	if base != "/" && !strings.HasSuffix(base, "/") {
		redirect := func(c *Ctx) error {
			http.Redirect(c.Writer(), c.Request(), subtree, http.StatusMovedPermanently)
			return nil
		}
		r.handle(http.MethodGet, base, redirect)
		r.handle(http.MethodHead, base, redirect)
	}
}

// Mount mounts a net/http handler at a path.
func (r *Router) Mount(p string, h http.Handler) { r.Compat.Handle(p, h) }

func (r *Router) handle(method, p string, h Handler) {
	full := r.fullPath(p)
	composed := r.compose(h)
	adapted := r.adapt(composed)
	pattern := fmt.Sprintf("%s %s", strings.ToUpper(method), full)
	r.mux.Handle(pattern, adapted)
}

// compose applies scoped middleware from right to left.
func (r *Router) compose(h Handler) Handler {
	for i := len(r.chain) - 1; i >= 0; i-- {
		h = r.chain[i](h)
	}
	return h
}

// syncCtx keeps a request-scoped ctx consistent with the current writer/request.
// This is needed because standard net/http middleware can wrap ResponseWriter or
// replace the request to attach a new context.
func (r *Router) syncCtx(c *Ctx, w http.ResponseWriter, req *http.Request) *Ctx {
	if c == nil {
		return newCtx(w, req, r.log)
	}
	c.writer = w
	c.request = req
	c.rc = http.NewResponseController(w)
	return c
}

// ctxFromRequest fetches the request-scoped ctx (if present) and syncs it.
func (r *Router) ctxFromRequest(w http.ResponseWriter, req *http.Request) *Ctx {
	c, _ := req.Context().Value(ctxKey{}).(*Ctx)
	return r.syncCtx(c, w, req)
}

// adapt converts a Mizu Handler into an http.Handler.
// If the handler returns an error, it is stored on the request context so the mux delegate
// can return it to the global chain.
func (r *Router) adapt(h Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		c := r.ctxFromRequest(w, req)

		if err := h(c); err != nil {
			req2 := req.WithContext(context.WithValue(req.Context(), routeErrorKey{}, err))
			c.request = req2
		}
	})
}

// adaptStdMiddleware wraps a standard net/http middleware as a Mizu middleware.
// If the wrapped chain produces an error, it is stored on the request context so the
// mux delegate can return it to the global chain.
func (r *Router) adaptStdMiddleware(sm func(http.Handler) http.Handler) Middleware {
	return func(next Handler) Handler {
		base := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			c := r.ctxFromRequest(w, req)

			if err := next(c); err != nil {
				req2 := req.WithContext(context.WithValue(req.Context(), routeErrorKey{}, err))
				c.request = req2
			}
		})

		wrapped := sm(base)

		return func(c *Ctx) error {
			wrapped.ServeHTTP(c.Writer(), c.Request())
			return nil
		}
	}
}

// fullPath joins base and p.
// If p ends with "/" (and is not "/"), it preserves the trailing slash so subtree patterns
// like "/assets/" remain distinct from "/assets".
func (r *Router) fullPath(p string) string {
	if p == "" {
		p = "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	wantTrailing := p != "/" && strings.HasSuffix(p, "/")
	out := joinPath(r.base, p)

	if wantTrailing && out != "/" && !strings.HasSuffix(out, "/") {
		out += "/"
	}
	return out
}

// joinPath joins URL path segments and keeps a single leading slash.
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

// httpRouter is a compatibility facade for net/http style APIs.
type httpRouter struct{ r *Router }

// Use appends standard net/http middleware to the Router global chain.
func (h *httpRouter) Use(mw ...func(http.Handler) http.Handler) *httpRouter {
	for _, sm := range mw {
		h.r.Use(h.r.adaptStdMiddleware(sm))
	}
	return h
}

// Handle registers an http.Handler at a path using plain ServeMux matching.
func (h *httpRouter) Handle(p string, hh http.Handler) *httpRouter {
	full := h.r.fullPath(p)
	h.r.mux.Handle(full, hh)
	return h
}

// HandleMethod registers an http.Handler for a specific method and path using method patterns.
// On Go 1.22+ if the path matches but method does not, ServeMux responds 405.
func (h *httpRouter) HandleMethod(method, p string, hh http.Handler) *httpRouter {
	full := h.r.fullPath(p)
	pattern := fmt.Sprintf("%s %s", strings.ToUpper(method), full)
	h.r.mux.Handle(pattern, hh)
	return h
}

// Prefix returns a child compatibility router rooted at prefix.
func (h *httpRouter) Prefix(prefix string) *httpRouter { return &httpRouter{r: h.r.Prefix(prefix)} }

// Group creates a child compatibility router and runs fn with it.
func (h *httpRouter) Group(prefix string, fn func(g *httpRouter)) { fn(h.Prefix(prefix)) }

// Mount mounts an http.Handler at a path using the compatibility facade.
func (h *httpRouter) Mount(p string, hh http.Handler) *httpRouter { return h.Handle(p, hh) }
