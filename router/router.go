package router

import (
	"context"
	"fmt"
	"maps"
	"net"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

// A Router matches requests to handlers.
//
// The zero Router is not usable. [New] makes one.
//
// Routes are registered before the router serves, which is how a route table is
// written: once, at startup, in one place. Registering while requests are in
// flight is safe, and setting a name or a piece of metadata on a route that is
// already serving is not, since the handler reads it without a lock.
type Router struct {
	mu     sync.Mutex
	routes []*Route
	names  map[string]*Route

	// built is the decision tree, thrown away by every registration and made
	// again by the next request. Matching reads it with one atomic load and no
	// lock, which is what keeps a match free of the contention a lock around
	// the route table would add to every request on every core.
	built atomic.Pointer[node]

	cons       constraints
	notFound   http.Handler
	notAllowed http.Handler
	slash      bool
	clean      bool
}

// New returns a router.
//
// It panics when an option is wrong, since the options are written once where
// the router is made and a mistake in one is a mistake in the program.
func New(opts ...Option) *Router {
	r, err := open(opts...)
	if err != nil {
		panic("router: " + err.Error())
	}
	return r
}

func open(opts ...Option) (*Router, error) {
	s := settings{cons: maps.Clone(builtin)}
	for _, opt := range opts {
		if err := opt(&s); err != nil {
			return nil, err
		}
	}
	r := &Router{
		names:      map[string]*Route{},
		cons:       s.cons,
		notFound:   s.notFound,
		notAllowed: s.notAllowed,
		slash:      s.slash,
		clean:      s.clean,
	}
	if r.notFound == nil {
		r.notFound = http.NotFoundHandler()
	}
	if r.notAllowed == nil {
		r.notAllowed = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		})
	}
	return r, nil
}

// An Option is something [New] is told at the point the router is made.
type Option func(*settings) error

// settings is what the options write to, kept apart from the Router so that
// nothing can be changed once requests are being served.
type settings struct {
	cons       constraints
	notFound   http.Handler
	notAllowed http.Handler
	slash      bool
	clean      bool
}

// NotFound sets the handler for a request no pattern matches. The default
// writes a plain 404.
func NotFound(h http.Handler) Option {
	return func(s *settings) error {
		s.notFound = h
		return nil
	}
}

// MethodNotAllowed sets the handler for a request whose path matches a pattern
// but whose method does not. The Allow header is already set when it runs, so a
// handler that only writes a body does not have to work out what to put there.
// The default writes a plain 405.
func MethodNotAllowed(h http.Handler) Option {
	return func(s *settings) error {
		s.notAllowed = h
		return nil
	}
}

// RedirectTrailingSlash answers a request that would match with a slash added
// or taken off the end with a 308 to that path.
//
// It is off by default. A redirect changes the method of a POST for some
// clients, turns one request into two for every client, and hides a route table
// that does not say what somebody thought it said. Off, /posts/ is a 404 when
// only /posts is registered, which is the answer that leads to the fix.
func RedirectTrailingSlash() Option {
	return func(s *settings) error {
		s.slash = true
		return nil
	}
}

// RedirectCleanPath answers a request whose path has dot segments or repeated
// slashes in it with a 308 to the cleaned path, when the cleaned path matches.
//
// It is off by default, for the reasons under [RedirectTrailingSlash]. Off, the
// path is matched as it arrived, so a handler always sees the path the client
// sent.
//
// It does not fold case. A case-insensitive route table is a second index over
// every pattern, and a URL that works in two casings is two URLs for one page,
// which is a thing to fix rather than to serve.
func RedirectCleanPath() Option {
	return func(s *settings) error {
		s.clean = true
		return nil
	}
}

// Constrain adds a constraint under a name, which patterns then write after a
// colon: Constrain("even", even) makes {n:even} a wildcard that only matches
// what even accepts.
//
// A built-in name cannot be redefined. A pattern means the same thing in every
// file of a program, and a package that quietly changed what {id:int} accepts
// would be the hardest kind of bug to find.
func Constrain(name string, c Constraint) Option {
	return func(s *settings) error {
		if _, ok := builtin[name]; ok {
			return fmt.Errorf("%q is a constraint this package defines, and redefining one would change what every pattern that names it means", name)
		}
		if _, ok := s.cons[name]; ok {
			return fmt.Errorf("%q is already a constraint on this router", name)
		}
		if c == nil {
			return fmt.Errorf("%q has nothing to check with", name)
		}
		s.cons[name] = c
		return nil
	}
}

// A Route is one registered pattern, what it runs, and what has been hung off
// it.
//
// Registering returns one, a match hands it back, and [Router.Routes] lists
// them all. Everything that wants to know which route ran rather than which
// path was requested reads it: a metric labelled by route rather than by URL, a
// rate limit that is per route, a span name that is not a cardinality problem.
type Route struct {
	r       *Router
	pat     *pattern
	handler http.Handler
	names   []string
	loc     string

	name string
	meta map[string]any
}

// Handle registers a pattern.
//
// The pattern is [net/http.ServeMux] syntax with constraints added, and the
// package comment has the whole of it. It panics when the pattern will not
// parse or when it conflicts with one already registered, since both are
// mistakes in a route table rather than conditions to handle at startup.
func (r *Router) Handle(pattern string, h http.Handler) *Route {
	rt, err := r.register(pattern, h, where(2))
	if err != nil {
		panic("router: " + err.Error())
	}
	return rt
}

// HandleFunc registers a pattern with a function.
func (r *Router) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) *Route {
	rt, err := r.register(pattern, http.HandlerFunc(h), where(2))
	if err != nil {
		panic("router: " + err.Error())
	}
	return rt
}

func (r *Router) register(pat string, h http.Handler, loc string) (*Route, error) {
	if h == nil {
		return nil, fmt.Errorf("%s has no handler, and a route with nothing to run is a route nobody meant to write", pat)
	}
	p, err := parse(pat, r.cons)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, old := range r.routes {
		if conflicts(p, old.pat) {
			return nil, clash(p, old)
		}
	}

	rt := &Route{r: r, pat: p, handler: h, names: p.names(), loc: loc}
	r.routes = append(r.routes, rt)
	r.built.Store(nil)
	return rt, nil
}

// clash says why two patterns cannot both be registered, in the terms the
// person reading it is thinking in: which two, where the other one is, and a
// path they both answer.
//
// There are three ways for it to happen and they read differently, so there are
// three messages. Two patterns can be the same thing written twice, they can
// each match something the other does not, or one can answer more methods while
// the other matches more paths, which is the one that surprises people.
func clash(p *pattern, old *Route) error {
	mrel, prel := methods(p, old.pat), paths(p, old.pat)
	if combine(mrel, prel) == same {
		return fmt.Errorf("%q matches the same requests as %q, registered at %s", p, old.pat, old.loc)
	}
	if prel == overlapping {
		return fmt.Errorf("%q and %q, registered at %s, both match %q, and neither of them is the more specific",
			p, old.pat, old.loc, shared(p, old.pat))
	}

	// One answers more methods and the other matches more paths. The pattern
	// being registered goes first either way, so that the location in the
	// message is always the other one, which is the one the reader has to go
	// and look at.
	if mrel == narrower {
		return fmt.Errorf("%q answers fewer methods than %q, registered at %s, and matches more paths than it, so neither of them is the one to prefer",
			p, old.pat, old.loc)
	}
	return fmt.Errorf("%q answers more methods than %q, registered at %s, and matches fewer paths than it, so neither of them is the one to prefer",
		p, old.pat, old.loc)
}

// where is the file and line of the call that registered a route, which is what
// a message about a conflict points at.
func where(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "an unknown place"
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// tree is the decision tree, made from the routes the first time a request
// needs it after a registration.
func (r *Router) tree() *node {
	if t := r.built.Load(); t != nil {
		return t
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if t := r.built.Load(); t != nil {
		return t
	}
	t := &node{}
	for _, rt := range r.routes {
		t.add(rt)
	}
	r.built.Store(t)
	return t
}

// Lookup is the route that answers a method, a host and a path, with the
// wildcard values it matched.
//
// The path is the path as it arrives on the wire, with its percent escapes
// still in it, which is what [net/url.URL.EscapedPath] returns. The host is a
// name with no port on it. Nothing is written and no handler is run, so this is
// also the way to ask what would happen without it happening.
func (r *Router) Lookup(method, host, path string) (*Route, Params, bool) {
	var v values
	n := r.tree().match(host, method, path, &v)
	if n == nil {
		return nil, Params{}, false
	}
	return n.route, Params{names: n.route.names, vals: v}, true
}

// Routes is every registered route, in the order they were registered, which is
// the order the route table was written in.
func (r *Router) Routes() []RouteInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]RouteInfo, 0, len(r.routes))
	for _, rt := range r.routes {
		out = append(out, rt.Info())
	}
	return out
}

// Named is the route registered under a name, and whether there is one.
func (r *Router) Named(name string) (*Route, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rt, ok := r.names[name]
	return rt, ok
}

// ServeHTTP matches the request and runs what it found.
//
// A path that matches nothing is a 404. A path that matches a pattern under
// another method is a 405 with an Allow header listing the methods that would
// have worked. An OPTIONS request that no route registered is answered from
// that same list.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	host := hostOf(req)
	path := req.URL.EscapedPath()

	if rt, ps, ok := r.Lookup(req.Method, host, path); ok {
		rt.handler.ServeHTTP(w, matched(req, rt, ps))
		return
	}
	if to, ok := r.elsewhere(req.Method, host, path); ok {
		http.Redirect(w, req, to, http.StatusPermanentRedirect)
		return
	}

	allow := r.tree().methods(host, path, nil)
	if len(allow) == 0 {
		r.notFound.ServeHTTP(w, req)
		return
	}
	if !contains(allow, http.MethodOptions) {
		allow = append(allow, http.MethodOptions)
	}
	slices.Sort(allow)
	w.Header().Set("Allow", strings.Join(allow, ", "))

	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	r.notAllowed.ServeHTTP(w, req)
}

// elsewhere is the path this request would have matched, when one of the
// redirect options is on and there is one.
func (r *Router) elsewhere(method, host, path string) (string, bool) {
	if r.clean {
		if c := cleanPath(path); c != path {
			if _, _, ok := r.Lookup(method, host, c); ok {
				return c, true
			}
		}
	}
	if r.slash {
		alt := strings.TrimSuffix(path, "/")
		if alt == path {
			alt = path + "/"
		}
		if alt != "" {
			if _, _, ok := r.Lookup(method, host, alt); ok {
				return alt, true
			}
		}
	}
	return "", false
}

// hostOf is the host a request is for, which is the Host header with any port
// taken off. A pattern with a host in it is matched against this.
func hostOf(req *http.Request) string {
	h := req.Host
	if !strings.ContainsRune(h, ':') {
		return h
	}
	if name, _, err := net.SplitHostPort(h); err == nil {
		return name
	}
	return h
}

// Name gives the route a name, which is what a URL is generated from and what a
// metric or a span is labelled with.
//
// Names are unique, so it panics on one that is taken. Two routes under one
// name means a generated URL points at whichever of them was registered last,
// which is a bug that shows up as a wrong link in production.
func (rt *Route) Name(name string) *Route {
	if err := rt.setName(name); err != nil {
		panic("router: " + err.Error())
	}
	return rt
}

func (rt *Route) setName(name string) error {
	rt.r.mu.Lock()
	defer rt.r.mu.Unlock()

	if old, ok := rt.r.names[name]; ok && old != rt {
		return fmt.Errorf("%q is already the name of %q, registered at %s", name, old.pat, old.loc)
	}
	delete(rt.r.names, rt.name)
	rt.name = name
	rt.r.names[name] = rt
	return nil
}

// Meta hangs a value off the route under a key, for whatever reads routes and
// needs something this package has no opinion about.
//
// It is set where the route is registered and read while the route is serving,
// so setting one on a router that is already answering requests is a race.
func (rt *Route) Meta(key string, value any) *Route {
	if rt.meta == nil {
		rt.meta = map[string]any{}
	}
	rt.meta[key] = value
	return rt
}

// Value is what [Route.Meta] put under a key, or nil.
func (rt *Route) Value(key string) any { return rt.meta[key] }

// Handler is what the route runs.
func (rt *Route) Handler() http.Handler { return rt.handler }

// Info is the route as data, which is what a route table prints and what a
// generator reads.
func (rt *Route) Info() RouteInfo {
	return RouteInfo{
		Method:  rt.pat.method,
		Host:    rt.pat.host,
		Path:    rt.pat.path(),
		Pattern: rt.pat.str,
		Name:    rt.name,
		Params:  slices.Clone(rt.names),
		Source:  rt.loc,
	}
}

// A RouteInfo is one route taken apart, for printing and for generating from.
type RouteInfo struct {
	// Method is the method the pattern named, or empty for a pattern that
	// named none and so answers every method.
	Method string

	// Host is the host the pattern named, or empty for a pattern that answers
	// whatever host the request carried.
	Host string

	// Path is the pattern without the method and the host.
	Path string

	// Pattern is the whole of what was registered.
	Pattern string

	// Name is what [Route.Name] set, or empty.
	Name string

	// Params is the wildcard names, left to right. The nameless wildcard a
	// trailing slash makes is not one of them.
	Params []string

	// Source is the file and line the route was registered at.
	Source string
}

// matchKey is the context key the matched route is carried under. It is a type
// of this package's own, so nothing else can collide with it or read it by
// accident.
type matchKey struct{}

type match struct {
	route  *Route
	params Params
}

func matched(req *http.Request, rt *Route, ps Params) *http.Request {
	return req.WithContext(context.WithValue(req.Context(), matchKey{}, &match{rt, ps}))
}

// Matched is the route that is running and the wildcard values it matched, for
// a request inside a handler or a middleware.
//
// It reports false for a request the router did not route, which is what a
// handler mounted somewhere else or called from a test gets.
func Matched(req *http.Request) (*Route, Params, bool) {
	m, ok := req.Context().Value(matchKey{}).(*match)
	if !ok {
		return nil, Params{}, false
	}
	return m.route, m.params, true
}

// PathValue is the value of one wildcard of the route that is running.
//
// This is where [net/http.Request.PathValue] would be, and it is not there:
// filling that in costs a map allocation on every request that has a parameter,
// and this package is trying to cost nothing. A handler that would rather have
// the standard shape can call req.SetPathValue itself.
func PathValue(req *http.Request, name string) string {
	_, ps, _ := Matched(req)
	return ps.Get(name)
}
