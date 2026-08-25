package mw

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-mizu/mizu/web"
)

// The field a form uses to say which method it meant, which is the name Rails
// picked and everything since has copied.
const methodField = "_method"

// MethodOverride lets an HTML form send a PUT, a PATCH or a DELETE.
//
//	<form method="post" action="/posts/7">
//		<input type="hidden" name="_method" value="delete">
//		<button>Delete</button>
//	</form>
//
// A form element can send a GET or a POST and nothing else, which is why a route
// table full of REST verbs ends up with a POST route beside every one of them.
// With this in the chain, the route table says DELETE /posts/{id} and the form
// says so too.
//
// # What it will not do
//
// The request has to be a POST with a form encoded body. A GET carrying
// _method=delete is a link somebody can be sent, and a browser will follow it
// from an email, so upgrading one would turn every crawler into a problem.
//
// The new method has to be PUT, PATCH or DELETE. Those are the three a form
// cannot send and a handler might want. Anything else, including CONNECT and
// TRACE and a method nobody has heard of, is left alone.
//
// A cross origin request is left alone. When the request carries an Origin, its
// host has to be the host being served, and the Referer is used the same way
// when there is no Origin. A browser sends one or the other on a form post, so
// what this refuses is a form on somebody else's page aimed at yours. A request
// with neither is not from a browser and has no form to be tricked into
// submitting, so it goes through.
//
// None of that is a substitute for CSRF protection, which arrives with the
// session. The same origin check here stops an override, not a request.
func MethodOverride() web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, override(r))
		})
	}
}

// override is the request to serve, which is r itself unless the body has been
// read here.
//
// Reading it is what makes the copy necessary. ParseForm drains the body and
// leaves the fields on the request it was called with, so calling it on the
// server's request would change a value the handler was told not to expect
// changes to, and calling it on a copy that is then thrown away would leave the
// handler with a body somebody else has already read.
func override(r *http.Request) *http.Request {
	if r.Method != http.MethodPost || !formEncoded(r) || !sameOrigin(r) {
		return r
	}

	copied := *r
	if copied.ParseForm() != nil {
		// The body is drained either way, so the copy still has to go on. The
		// handler sees the same failure when it asks for a field.
		return &copied
	}

	switch method := strings.ToUpper(copied.PostForm.Get(methodField)); method {
	case http.MethodPut, http.MethodPatch, http.MethodDelete:
		copied.Method = method
	}
	return &copied
}

// formEncoded reports whether the body is the kind ParseForm reads.
//
// Multipart is not it. ParseForm leaves a multipart body alone, the field would
// be invisible here, and a file upload form that thought it was sending a DELETE
// would quietly send a POST instead.
func formEncoded(r *http.Request) bool {
	ct, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
	return strings.TrimSpace(ct) == "application/x-www-form-urlencoded"
}

// sameOrigin reports whether the request came from the site it is addressed to,
// as far as the headers a browser attaches can say.
func sameOrigin(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		return hostOf(origin) == r.Host
	}
	if ref := r.Header.Get("Referer"); ref != "" {
		return hostOf(ref) == r.Host
	}
	// Neither header means no browser, and no browser means no form on somebody
	// else's page.
	return true
}

// hostOf is the host in an absolute URL, or the empty string when it is not one.
func hostOf(s string) string {
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	return u.Host
}
