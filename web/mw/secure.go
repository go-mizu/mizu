package mw

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-mizu/mizu/web"
)

// SecureConfig is the set of response headers [Secure] sends.
//
// Every field is off in the zero value, so a config says what it wants and
// nothing arrives by surprise. There is no "sensible defaults" mode, because a
// header that changes what a browser will load is not something to acquire
// without noticing.
type SecureConfig struct {
	// HSTS is the max-age in Strict-Transport-Security, which is how long a
	// browser refuses to speak plain HTTP to this host. Zero leaves the header
	// off.
	//
	// A browser ignores it over a plain connection, so it is sent on every
	// response rather than only on the TLS ones. That is what makes it work
	// behind a proxy that terminates TLS, which is where most services sit.
	HSTS time.Duration

	// HSTSSubdomains adds includeSubDomains, which applies the rule to every
	// name under this one. It is the right answer and it is also the one that
	// takes an intranet host on a subdomain offline, so it is opt in.
	HSTSSubdomains bool

	// HSTSPreload adds preload, which is the flag the browser vendors' list
	// requires before it will accept a submission.
	//
	// It needs HSTS of at least a year and HSTSSubdomains, and panics without
	// them, because getting onto that list is quick and getting off it is not.
	HSTSPreload bool

	// FrameOptions is X-Frame-Options: either DENY or SAMEORIGIN. Empty leaves
	// the header off, and anything else panics.
	//
	// The modern spelling is the frame-ancestors directive in CSP, which this
	// does not replace: a browser old enough to need X-Frame-Options is a
	// browser that ignores frame-ancestors.
	FrameOptions string

	// ContentTypeOptions sends X-Content-Type-Options: nosniff, which stops a
	// browser guessing a type the response did not declare. An upload served
	// back as text/plain and sniffed as HTML is stored cross site scripting.
	ContentTypeOptions bool

	// ReferrerPolicy is Referrer-Policy. Empty leaves the header off.
	// strict-origin-when-cross-origin is the usual answer and is also what
	// browsers now do without being asked.
	ReferrerPolicy string

	// PermissionsPolicy is Permissions-Policy, which is the successor to
	// Feature-Policy and says which browser features this document and anything
	// it embeds may use. Empty leaves the header off.
	//
	//	"camera=(), microphone=(), geolocation=()"
	PermissionsPolicy string

	// CSP is Content-Security-Policy, written out as the header value. Empty
	// leaves the header off.
	//
	// A typed builder with a per request nonce arrives with the secure package,
	// which is where the template side of it has to live. Until then this takes
	// the policy as a string, so a policy without a nonce works now and the
	// typed one can be handed to this field later.
	CSP string

	// CSPReportOnly sends the policy as Content-Security-Policy-Report-Only,
	// which reports what it would have blocked and blocks nothing. It is how a
	// policy gets turned on for a week before it is turned on.
	CSPReportOnly bool

	// COOP is Cross-Origin-Opener-Policy, usually same-origin, which cuts the
	// link between this document and whatever opened it.
	COOP string

	// COEP is Cross-Origin-Embedder-Policy, usually require-corp. It is what
	// SharedArrayBuffer needs and it is also what breaks every third party embed
	// that has not opted in, so it is worth turning on deliberately.
	COEP string

	// CORP is Cross-Origin-Resource-Policy, usually same-origin, which says who
	// may load this response as a subresource.
	CORP string
}

// Secure puts the response headers a browser reads as policy on every response.
//
//	mw.Secure(mw.SecureConfig{
//		HSTS:               365 * 24 * time.Hour,
//		HSTSSubdomains:     true,
//		FrameOptions:       "DENY",
//		ContentTypeOptions: true,
//		ReferrerPolicy:     "strict-origin-when-cross-origin",
//		PermissionsPolicy:  "camera=(), microphone=(), geolocation=()",
//		COOP:               "same-origin",
//		CORP:               "same-origin",
//	})
//
// The headers go on before the handler runs, so a handler that sets one of them
// itself wins. That is what a route serving an embeddable widget needs, and it
// is one Set call rather than a way of turning the middleware off for a route.
//
// The values are not checked beyond the two that panic, because the list of
// valid CSP directives and permission names changes faster than a release
// cycle, and a middleware that refuses a header a browser has just learned is
// worse than one that sends a header a browser ignores.
//
// Whether HSTS is on belongs to the configuration rather than to this. It is off
// in development because a browser that learns to refuse plain HTTP for
// localhost has learned it for every project on that machine.
func Secure(cfg SecureConfig) web.Middleware {
	fixed := headers(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			for _, kv := range fixed {
				h.Set(kv[0], kv[1])
			}
			next.ServeHTTP(w, r)
		})
	}
}

// headers is the config flattened into the pairs that go on every response, so
// the request path is a loop over a slice and no string building.
func headers(cfg SecureConfig) [][2]string {
	var out [][2]string
	add := func(name, value string) {
		if value != "" {
			out = append(out, [2]string{name, value})
		}
	}

	add("Strict-Transport-Security", hsts(cfg))
	add("X-Frame-Options", frameOptions(cfg.FrameOptions))
	if cfg.ContentTypeOptions {
		add("X-Content-Type-Options", "nosniff")
	}
	add("Referrer-Policy", cfg.ReferrerPolicy)
	add("Permissions-Policy", cfg.PermissionsPolicy)

	if cfg.CSPReportOnly {
		add("Content-Security-Policy-Report-Only", cfg.CSP)
	} else {
		add("Content-Security-Policy", cfg.CSP)
	}

	add("Cross-Origin-Opener-Policy", cfg.COOP)
	add("Cross-Origin-Embedder-Policy", cfg.COEP)
	add("Cross-Origin-Resource-Policy", cfg.CORP)
	return out
}

// hsts builds the Strict-Transport-Security value and refuses a preload that the
// browser vendors' list would refuse.
func hsts(cfg SecureConfig) string {
	if cfg.HSTS <= 0 {
		if cfg.HSTSPreload {
			panic("mw.Secure: HSTSPreload needs an HSTS of at least a year, and HSTS is not set")
		}
		return ""
	}

	v := "max-age=" + strconv.Itoa(int(cfg.HSTS/time.Second))
	if cfg.HSTSSubdomains {
		v += "; includeSubDomains"
	}
	if cfg.HSTSPreload {
		if cfg.HSTS < 365*24*time.Hour {
			panic("mw.Secure: HSTSPreload needs an HSTS of at least a year")
		}
		if !cfg.HSTSSubdomains {
			panic("mw.Secure: HSTSPreload needs HSTSSubdomains")
		}
		v += "; preload"
	}
	return v
}

// frameOptions checks the one header whose values are a closed set worth
// checking, since a typo there is a header the browser drops and a page that
// can be framed.
func frameOptions(v string) string {
	switch v {
	case "", "DENY", "SAMEORIGIN":
		return v
	default:
		panic("mw.Secure: FrameOptions is " + v + ", and the only values a browser reads are DENY and SAMEORIGIN")
	}
}
