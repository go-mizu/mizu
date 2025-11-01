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
	"sync"
)

// Middleware wraps a Handler to add cross cutting behavior like logging or auth.
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

	// NotFound handler is composed with middleware and bound once on first serve.
	notFoundCore     Handler
	notFoundBindOnce sync.Once

	allow      map[string]map[string]struct{} // path -> set(method)
	allowMu    *sync.RWMutex
	registerMu *sync.Mutex

	// Compat exposes a net/http-first facade for interop.
	Compat *httpRouter
}

// NewRouter creates a root Router with its own ServeMux and sane defaults.
func NewRouter() *Router {
	r := &Router{
		mux:        http.NewServeMux(),
		allow:      make(map[string]map[string]struct{}),
		allowMu:    &sync.RWMutex{},
		registerMu: &sync.Mutex{},
		log:        slog.Default(),
	}
	r.notFoundCore = func(c *Ctx) error {
		return c.Text(http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}
	r.Compat = &httpRouter{r: r}
	return r
}

// ServeHTTP implements http.Handler and delegates to the internal ServeMux.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.ensureNotFoundBound()
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

// NotFound sets the handler used when no route matches.
func (r *Router) NotFound(h http.Handler) {
	if h == nil {
		return
	}
	r.notFoundCore = func(c *Ctx) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
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
		mux:          r.mux,
		base:         joinPath(r.base, prefix),
		chain:        slices.Clone(r.chain),
		errh:         r.errh,
		log:          r.log,
		notFoundCore: r.notFoundCore,
		allow:        r.allow,
		allowMu:      r.allowMu,
		registerMu:   r.registerMu,
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
func (r *Router) Static(prefix string, fsys http.FileSystem) {
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	fs := http.FileServer(fsys)
	r.Compat.Handle(prefix, http.StripPrefix(prefix, fs))
}

// Mount mounts any http.Handler at a path.
func (r *Router) Mount(p string, h http.Handler) { r.Compat.Handle(p, h) }

func (r *Router) handle(method, p string, h Handler) {
	full := r.fullPath(p)
	composed := r.compose(h)
	adapted := r.adapt(composed)
	pattern := fmt.Sprintf("%s %s", method, full)

	r.registerMu.Lock()
	r.mux.Handle(pattern, adapted)

	// Install 405 guard once per path if not present yet.
	needGuard := false
	r.allowMu.RLock()
	if _, ok := r.allow[full]; !ok {
		needGuard = true
	}
	r.allowMu.RUnlock()
	if needGuard {
		guard := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			allow := r.allowHeader(full)
			if allow == "" {
				allow = "OPTIONS"
			}
			w.Header().Set("Allow", allow)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		})
		// Path-only pattern, does not shadow "METHOD path" entries.
		r.mux.Handle(full, guard)
	}
	r.registerMu.Unlock()

	r.allowAdd(full, method)
	if method == http.MethodGet {
		r.allowAdd(full, http.MethodHead)
	}
	// No auto OPTIONS registration to keep behavior simple and explicit.
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
			wrapped.ServeHTTP(c.Response(), c.Request())
			return nil
		}
	}
}

func (r *Router) ensureNotFoundBound() {
	r.notFoundBindOnce.Do(func() {
		adapted := r.adapt(r.compose(r.notFoundCore))
		r.registerMu.Lock()
		// Bind "/" once to act as NotFound while letting method patterns win.
		r.mux.Handle("/", adapted)
		r.registerMu.Unlock()
	})
}

func (r *Router) fullPath(p string) string {
	if p == "" {
		p = "/"
	}
	// Preserve subtree intent by remembering a trailing slash.
	wantTrailing := strings.HasSuffix(p, "/") && p != "/"

	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	joined := joinPath(r.base, p)

	// Re-apply trailing slash for subtree patterns so ServeMux matches subpaths.
	if wantTrailing && joined != "/" && !strings.HasSuffix(joined, "/") {
		joined += "/"
	}
	return joined
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

func (r *Router) allowAdd(full, method string) {
	r.allowMu.Lock()
	if _, ok := r.allow[full]; !ok {
		r.allow[full] = make(map[string]struct{})
	}
	r.allow[full][strings.ToUpper(method)] = struct{}{}
	r.allowMu.Unlock()
}

func (r *Router) allowHeader(full string) string {
	r.allowMu.RLock()
	set, ok := r.allow[full]
	r.allowMu.RUnlock()

	if !ok {
		return "OPTIONS"
	}
	list := make([]string, 0, len(set)+1)
	for m := range set {
		list = append(list, m)
	}
	// Ensure OPTIONS appears so 405 responses are predictable.
	hasOptions := false
	for _, m := range list {
		if m == http.MethodOptions {
			hasOptions = true
			break
		}
	}
	if !hasOptions {
		list = append(list, http.MethodOptions)
	}
	slices.Sort(list)
	return strings.Join(list, ", ")
}

// Dev enables a friendly colored logger and installs the request logger middleware.
func (r *Router) Dev(enable bool) *Router {
	if !enable {
		return r
	}

	// Dev writes to Stderr, color controlled by env.
	var h slog.Handler
	if forceColorOn() || supportsColorEnv() {
		h = newColorTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	}

	r.SetLogger(slog.New(h))
	slog.SetDefault(r.Logger())

	r.Use(Logger(LoggerOptions{
		Logger: r.Logger(),
	}))

	return r
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
	h.r.registerMu.Lock()
	h.r.mux.Handle(full, hh)
	h.r.registerMu.Unlock()
	return h
}

// HandleMethod registers an http.Handler for a specific method and path.
func (h *httpRouter) HandleMethod(method, p string, hh http.Handler) *httpRouter {
	full := h.r.fullPath(p)
	pattern := fmt.Sprintf("%s %s", strings.ToUpper(method), full)

	h.r.registerMu.Lock()
	h.r.mux.Handle(pattern, hh)
	// 405 guard once per path.
	needGuard := false
	h.r.allowMu.RLock()
	if _, ok := h.r.allow[full]; !ok {
		needGuard = true
	}
	h.r.allowMu.RUnlock()
	if needGuard {
		guard := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			allow := h.r.allowHeader(full)
			if allow == "" {
				allow = "OPTIONS"
			}
			w.Header().Set("Allow", allow)
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		})
		h.r.mux.Handle(full, guard)
	}
	h.r.registerMu.Unlock()

	h.r.allowAdd(full, strings.ToUpper(method))
	if strings.EqualFold(method, http.MethodGet) {
		h.r.allowAdd(full, http.MethodHead)
	}
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
