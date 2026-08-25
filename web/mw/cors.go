package mw

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/web"
)

// CORSConfig is the policy [CORS] enforces.
//
// The zero value allows nothing, which is the same as not having the middleware
// at all, except that a preflight gets an answer saying no rather than whatever
// the route table would have done with an OPTIONS.
type CORSConfig struct {
	// AllowedOrigins is the origins a browser may make a cross origin request
	// from, written the way an Origin header is: scheme, host, and a port when
	// it is not the default one, with no trailing slash.
	//
	// A single "*" allows every origin. The leftmost label of the host may be a
	// "*", so https://*.example.com covers https://app.example.com and does not
	// cover https://example.com or https://a.b.example.com. A "*" anywhere else
	// is a mistake and panics at construction.
	AllowedOrigins []string

	// AllowedOriginFunc decides about an origin the list does not cover, for a
	// service whose set of tenants lives in a database rather than in a
	// constant. It runs on the request goroutine, once per cross origin
	// request, so whatever it reads should already be in memory.
	AllowedOriginFunc func(origin string) bool

	// AllowedMethods is the methods a cross origin request may use.
	//
	// Empty means GET, HEAD and POST, which are the three a plain form could
	// send without asking permission, so allowing them takes nothing away.
	AllowedMethods []string

	// AllowedHeaders is the request headers a cross origin request may carry
	// beyond the ones the fetch specification safelists.
	//
	// Empty means none, so a request with an Authorization header is refused
	// until somebody says otherwise. A single "*" allows every header, and is
	// refused alongside AllowCredentials for the same reason a "*" origin is.
	AllowedHeaders []string

	// ExposedHeaders is the response headers the calling script may read.
	// Without this a script sees the fetch safelist and nothing else, which is
	// why the request id header is worth naming here.
	ExposedHeaders []string

	// AllowCredentials lets the browser send cookies and HTTP authentication,
	// and lets the script read the response when it did.
	//
	// It cannot be combined with a "*" in AllowedOrigins or AllowedHeaders. The
	// fetch specification refuses that pair and a browser that receives it
	// fails the request, so it panics at construction rather than at three in
	// the morning.
	AllowCredentials bool

	// MaxAge is how long a browser may reuse a preflight answer. Zero leaves
	// the header off and the browser falls back to a few seconds.
	//
	// Chrome caps it at two hours and Firefox at twenty four, so asking for a
	// week gets you two hours.
	MaxAge time.Duration
}

// CORS answers preflight requests and marks up cross origin responses.
//
//	mw.CORS(mw.CORSConfig{
//		AllowedOrigins:   []string{"https://app.example.com"},
//		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
//		AllowedHeaders:   []string{"Authorization", "Content-Type"},
//		ExposedHeaders:   []string{mw.RequestIDHeader},
//		AllowCredentials: true,
//		MaxAge:           24 * time.Hour,
//	})
//
// A preflight, which is an OPTIONS carrying Access-Control-Request-Method, is
// answered here and never reaches the handler. That is why this goes near the
// outside: an OPTIONS that runs the whole chain touches the session store and
// the database to answer a question about the route table.
//
// Every other request is passed along, with the headers a browser needs in order
// to hand the response to the script that asked for it. A request with no Origin
// is not a cross origin request and nothing is added to it.
//
// Vary: Origin goes on every response, including the ones this refuses, because
// a cache in front of the service must not hand one origin's answer to another.
// A preflight also varies on the two Access-Control-Request headers.
//
// # What it does not do
//
// It is not an authorization check. CORS is a rule browsers follow about which
// script may read a response, and nothing except a browser follows it. A request
// from curl, from another service, or from somebody's own server arrives with
// whatever Origin they typed, is refused the CORS headers, and is still served,
// because refusing to serve it would be a different middleware answering a
// different question. Anything that must not happen without permission needs
// that permission checked where the work is done.
func CORS(cfg CORSConfig) web.Middleware {
	p := compile(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			preflight := r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""

			h := w.Header()
			h.Add("Vary", "Origin")
			if preflight {
				h.Add("Vary", "Access-Control-Request-Method")
				h.Add("Vary", "Access-Control-Request-Headers")
			}

			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if p.allows(origin) {
				if preflight {
					p.preflight(h, r, origin)
				} else {
					p.actual(h, origin)
				}
			}

			// A preflight is answered here whether or not it was allowed. The
			// browser decides from the headers that are or are not on it, and an
			// OPTIONS that fell through to the route table would come back a 405
			// that tells it nothing.
			if preflight {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// wildcard is an origin pattern with a "*" for the leftmost label, held apart so
// the match does not have to find the pieces again on every request.
type wildcard struct {
	scheme string // "https://"
	suffix string // ".example.com"
}

// policy is a CORSConfig with the string work done once.
type policy struct {
	anyOrigin bool
	exact     []string
	wild      []wildcard
	fn        func(string) bool

	methods    []string
	methodList string

	anyHeader  bool
	headers    []string
	headerList string

	expose      string
	credentials bool
	maxAge      string
}

// compile turns a config into a policy and refuses the combinations a browser
// would.
func compile(cfg CORSConfig) *policy {
	p := &policy{
		fn:          cfg.AllowedOriginFunc,
		credentials: cfg.AllowCredentials,
		expose:      strings.Join(cfg.ExposedHeaders, ", "),
	}

	for _, o := range cfg.AllowedOrigins {
		switch {
		case o == "*":
			p.anyOrigin = true
		case !strings.Contains(o, "*"):
			p.exact = append(p.exact, o)
		default:
			p.wild = append(p.wild, parseWildcard(o))
		}
	}

	p.methods = cfg.AllowedMethods
	if len(p.methods) == 0 {
		p.methods = []string{http.MethodGet, http.MethodHead, http.MethodPost}
	}
	p.methodList = strings.Join(p.methods, ", ")

	p.anyHeader = slices.Contains(cfg.AllowedHeaders, "*")
	p.headers = cfg.AllowedHeaders
	p.headerList = strings.Join(cfg.AllowedHeaders, ", ")

	if cfg.MaxAge > 0 {
		p.maxAge = strconv.Itoa(int(cfg.MaxAge / time.Second))
	}

	if p.credentials && p.anyOrigin {
		panic("mw.CORS: AllowCredentials cannot go with a * in AllowedOrigins, because a browser refuses the pair")
	}
	if p.credentials && p.anyHeader {
		panic("mw.CORS: AllowCredentials cannot go with a * in AllowedHeaders, because a browser refuses the pair")
	}
	return p
}

// parseWildcard splits https://*.example.com, and panics on a "*" anywhere the
// match would not mean what it looks like.
func parseWildcard(o string) wildcard {
	scheme, host, ok := strings.Cut(o, "://")
	if !ok || !strings.HasPrefix(host, "*.") || strings.Contains(host[1:], "*") {
		panic("mw.CORS: " + o + " has a * somewhere other than the leftmost label of the host")
	}
	return wildcard{scheme: scheme + "://", suffix: host[1:]}
}

// matches reports whether origin is scheme, one label, then the suffix.
func (w wildcard) matches(origin string) bool {
	host, ok := strings.CutPrefix(origin, w.scheme)
	if !ok {
		return false
	}
	label, ok := strings.CutSuffix(host, w.suffix)
	return ok && label != "" && !strings.Contains(label, ".")
}

// allows reports whether origin is one the policy admits.
func (p *policy) allows(origin string) bool {
	if p.anyOrigin || slices.Contains(p.exact, origin) {
		return true
	}
	if slices.ContainsFunc(p.wild, func(w wildcard) bool { return w.matches(origin) }) {
		return true
	}
	return p.fn != nil && p.fn(origin)
}

// actual writes the headers an ordinary cross origin response carries.
func (p *policy) actual(h http.Header, origin string) {
	h.Set("Access-Control-Allow-Origin", p.echo(origin))
	if p.credentials {
		h.Set("Access-Control-Allow-Credentials", "true")
	}
	if p.expose != "" {
		h.Set("Access-Control-Expose-Headers", p.expose)
	}
}

// preflight writes the answer to an OPTIONS that is asking permission.
//
// Nothing is written when the method or one of the headers is refused, which is
// how a browser is told no: the answer arrives without the header it was looking
// for and the real request is never sent.
func (p *policy) preflight(h http.Header, r *http.Request, origin string) {
	if !slices.Contains(p.methods, r.Header.Get("Access-Control-Request-Method")) {
		return
	}

	want := r.Header.Get("Access-Control-Request-Headers")
	if !p.permits(want) {
		return
	}

	h.Set("Access-Control-Allow-Origin", p.echo(origin))
	h.Set("Access-Control-Allow-Methods", p.methodList)
	if p.credentials {
		h.Set("Access-Control-Allow-Credentials", "true")
	}
	switch {
	case p.anyHeader && want != "":
		// Echoing beats sending a "*" here. A browser matches the "*" only when
		// credentials are off, and echoing gives the same answer either way.
		h.Set("Access-Control-Allow-Headers", want)
	case p.headerList != "":
		h.Set("Access-Control-Allow-Headers", p.headerList)
	}
	if p.maxAge != "" {
		h.Set("Access-Control-Max-Age", p.maxAge)
	}
}

// echo is what goes in Access-Control-Allow-Origin.
//
// A "*" is only sent when the policy really is every origin without credentials,
// where it is both true and cacheable. Anything narrower gets the origin back,
// since a cache holding a "*" that was computed for one origin would be wrong
// for the rest.
func (p *policy) echo(origin string) string {
	if p.anyOrigin && !p.credentials {
		return "*"
	}
	return origin
}

// permits reports whether every header a preflight asked about is allowed.
func (p *policy) permits(want string) bool {
	if p.anyHeader || want == "" {
		return true
	}
	for name := range strings.SplitSeq(want, ",") {
		name = strings.TrimSpace(name)
		if name == "" || safelisted(name) {
			continue
		}
		if !slices.ContainsFunc(p.headers, func(a string) bool { return strings.EqualFold(a, name) }) {
			return false
		}
	}
	return true
}

// safelisted is the request headers the fetch specification lets through without
// permission, which a browser may still name in a preflight.
func safelisted(name string) bool {
	switch strings.ToLower(name) {
	case "accept", "accept-language", "content-language", "content-type", "range":
		return true
	}
	return false
}
