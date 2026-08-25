package mw

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// cross serves one request through CORS and reports the response, along with
// whether the handler behind it ran.
func cross(tb testing.TB, cfg CORSConfig, r *http.Request) (*httptest.ResponseRecorder, bool) {
	tb.Helper()

	ran := false
	w := httptest.NewRecorder()
	CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		ran = true
		w.Write([]byte("the answer"))
	})).ServeHTTP(w, r)
	return w, ran
}

// preflightTo is an OPTIONS asking permission for method from origin.
func preflightTo(origin, method string, headers ...string) *http.Request {
	r := httptest.NewRequest("OPTIONS", "/things", nil)
	r.Header.Set("Origin", origin)
	r.Header.Set("Access-Control-Request-Method", method)
	if len(headers) > 0 {
		r.Header.Set("Access-Control-Request-Headers", strings.Join(headers, ", "))
	}
	return r
}

// getFrom is an ordinary cross origin GET.
func getFrom(origin string) *http.Request {
	r := httptest.NewRequest("GET", "/things", nil)
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	return r
}

var app = CORSConfig{
	AllowedOrigins: []string{"https://app.example.com"},
	AllowedMethods: []string{"GET", "POST", "DELETE"},
	AllowedHeaders: []string{"Authorization", "X-Tenant"},
}

func TestARequestWithNoOriginIsNotACORSRequest(t *testing.T) {
	w, ran := cross(t, app, getFrom(""))

	if !ran {
		t.Error("the handler did not run")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("a request with no Origin got Access-Control-Allow-Origin: %q", got)
	}
	// Vary still goes on, since the answer would have been different with one.
	if got := w.Header().Values("Vary"); len(got) != 1 || got[0] != "Origin" {
		t.Errorf("Vary is %q, want Origin", got)
	}
}

func TestAnAllowedOriginGetsTheHeaderBack(t *testing.T) {
	w, ran := cross(t, app, getFrom("https://app.example.com"))

	if !ran {
		t.Error("the handler did not run")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin is %q", got)
	}
	if w.Body.String() != "the answer" {
		t.Errorf("the body is %q", w.Body)
	}
}

// TestARefusedOriginIsStillServed is the difference between CORS and an
// authorization check, and it is the one people are surprised by.
func TestARefusedOriginIsStillServed(t *testing.T) {
	w, ran := cross(t, app, getFrom("https://evil.example.net"))

	if !ran {
		t.Error("the handler did not run, and CORS is not an access check")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("a refused origin got Access-Control-Allow-Origin: %q", got)
	}
	if w.Body.String() != "the answer" {
		t.Errorf("the body is %q, and the response is the same one anybody else gets", w.Body)
	}
}

func TestAPreflightIsAnsweredHere(t *testing.T) {
	w, ran := cross(t, app, preflightTo("https://app.example.com", "DELETE"))

	if ran {
		t.Error("the preflight reached the handler")
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("the preflight is a %d, want 204", w.Code)
	}

	h := w.Header()
	if got := h.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin is %q", got)
	}
	if got := h.Get("Access-Control-Allow-Methods"); got != "GET, POST, DELETE" {
		t.Errorf("Access-Control-Allow-Methods is %q", got)
	}
	if got := h.Get("Access-Control-Allow-Headers"); got != "Authorization, X-Tenant" {
		t.Errorf("Access-Control-Allow-Headers is %q", got)
	}
}

func TestAPreflightVariesOnWhatItAsked(t *testing.T) {
	w, _ := cross(t, app, preflightTo("https://app.example.com", "GET"))

	want := "Origin, Access-Control-Request-Method, Access-Control-Request-Headers"
	if got := strings.Join(w.Header().Values("Vary"), ", "); got != want {
		t.Errorf("Vary is %q, want %q", got, want)
	}
}

// TestAnOPTIONSThatIsNotAPreflightGoesThrough keeps a route that answers OPTIONS
// itself working, which is what a discovery endpoint is.
func TestAnOPTIONSThatIsNotAPreflightGoesThrough(t *testing.T) {
	r := httptest.NewRequest("OPTIONS", "/things", nil)
	r.Header.Set("Origin", "https://app.example.com")

	if _, ran := cross(t, app, r); !ran {
		t.Error("an OPTIONS with no Access-Control-Request-Method was answered here")
	}
}

func TestAPreflightForAMethodThatIsNotAllowedIsRefused(t *testing.T) {
	w, ran := cross(t, app, preflightTo("https://app.example.com", "PUT"))

	if ran {
		t.Error("the preflight reached the handler")
	}
	if w.Code != http.StatusNoContent {
		t.Errorf("the preflight is a %d, want 204", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Errorf("a refused method got Access-Control-Allow-Methods: %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("a refused method got Access-Control-Allow-Origin: %q", got)
	}
}

func TestAPreflightForAHeaderThatIsNotAllowedIsRefused(t *testing.T) {
	w, _ := cross(t, app, preflightTo("https://app.example.com", "GET", "X-Secret"))

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("a refused header got Access-Control-Allow-Origin: %q", got)
	}
}

func TestTheHeaderCheckIgnoresCase(t *testing.T) {
	w, _ := cross(t, app, preflightTo("https://app.example.com", "GET", "authorization", "x-tenant"))

	if got := w.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Error("a preflight naming the allowed headers in lower case was refused")
	}
}

// TestASafelistedHeaderNeedsNoPermission covers the headers a browser may send
// without asking, which it still lists in a preflight when the request has them.
func TestASafelistedHeaderNeedsNoPermission(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://app.example.com"}}
	w, _ := cross(t, cfg, preflightTo("https://app.example.com", "POST", "Content-Type", "Accept"))

	if got := w.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Error("a preflight naming only safelisted headers was refused")
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got != "" {
		t.Errorf("Access-Control-Allow-Headers is %q, and nothing beyond the safelist was allowed", got)
	}
}

func TestTheDefaultMethodsAreTheOnesAFormCouldSendAnyway(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://app.example.com"}}
	w, _ := cross(t, cfg, preflightTo("https://app.example.com", "POST"))

	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, HEAD, POST" {
		t.Errorf("Access-Control-Allow-Methods is %q, want GET, HEAD, POST", got)
	}
}

func TestWildcardSubdomains(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://*.example.com"}}

	cases := []struct {
		origin string
		want   bool
	}{
		{"https://app.example.com", true},
		{"https://api.example.com", true},
		{"https://example.com", false},
		{"https://a.b.example.com", false},
		{"http://app.example.com", false},
		{"https://app.example.com.evil.net", false},
		{"https://.example.com", false},
		{"https://evilexample.com", false},
	}

	for _, c := range cases {
		t.Run(c.origin, func(t *testing.T) {
			w, _ := cross(t, cfg, getFrom(c.origin))
			if got := w.Header().Get("Access-Control-Allow-Origin") != ""; got != c.want {
				t.Errorf("allowed is %v, want %v", got, c.want)
			}
		})
	}
}

func TestAStarInTheWrongPlaceIsATypo(t *testing.T) {
	for _, o := range []string{"https://*.*.example.com", "https://app.*.com", "*.example.com", "https://*"} {
		t.Run(o, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%q was accepted", o)
				}
			}()
			CORS(CORSConfig{AllowedOrigins: []string{o}})
		})
	}
}

func TestEveryOriginGetsAStarBack(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"*"}}
	w, _ := cross(t, cfg, getFrom("https://anywhere.example.net"))

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin is %q, want *", got)
	}
}

// TestCredentialsWithAStarIsRefusedAtConstruction is the mistake this exists to
// catch. A browser fails the request outright, so the symptom is a fetch that
// does not work and a response that looks fine.
func TestCredentialsWithAStarIsRefusedAtConstruction(t *testing.T) {
	cases := map[string]CORSConfig{
		"a star origin": {AllowedOrigins: []string{"*"}, AllowCredentials: true},
		"a star header": {AllowedOrigins: []string{"https://app.example.com"}, AllowedHeaders: []string{"*"}, AllowCredentials: true},
	}

	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("it was accepted")
				}
			}()
			CORS(cfg)
		})
	}
}

func TestCredentialsAreAnnouncedOnBothKinds(t *testing.T) {
	cfg := app
	cfg.AllowCredentials = true

	for name, r := range map[string]*http.Request{
		"a request":   getFrom("https://app.example.com"),
		"a preflight": preflightTo("https://app.example.com", "GET"),
	} {
		t.Run(name, func(t *testing.T) {
			w, _ := cross(t, cfg, r)
			if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
				t.Errorf("Access-Control-Allow-Credentials is %q", got)
			}
		})
	}
}

func TestExposedHeadersAreOnTheResponseAndNotOnThePreflight(t *testing.T) {
	cfg := app
	cfg.ExposedHeaders = []string{RequestIDHeader, "RateLimit-Remaining"}

	w, _ := cross(t, cfg, getFrom("https://app.example.com"))
	if got, want := w.Header().Get("Access-Control-Expose-Headers"), "X-Request-Id, RateLimit-Remaining"; got != want {
		t.Errorf("Access-Control-Expose-Headers is %q, want %q", got, want)
	}

	// The preflight is about the request, so the answer says nothing about what
	// the response will let the script read.
	pre, _ := cross(t, cfg, preflightTo("https://app.example.com", "GET"))
	if got := pre.Header().Get("Access-Control-Expose-Headers"); got != "" {
		t.Errorf("the preflight carries Access-Control-Expose-Headers: %q", got)
	}
}

func TestMaxAgeIsInSeconds(t *testing.T) {
	cfg := app
	cfg.MaxAge = 24 * time.Hour

	w, _ := cross(t, cfg, preflightTo("https://app.example.com", "GET"))
	if got := w.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("Access-Control-Max-Age is %q, want 86400", got)
	}
}

func TestNoMaxAgeMeansNoHeader(t *testing.T) {
	w, _ := cross(t, app, preflightTo("https://app.example.com", "GET"))
	if got := w.Header().Get("Access-Control-Max-Age"); got != "" {
		t.Errorf("Access-Control-Max-Age is %q with no MaxAge set", got)
	}
}

func TestAnOriginFuncDecidesWhatTheListDoesNot(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:    []string{"https://app.example.com"},
		AllowedOriginFunc: func(o string) bool { return strings.HasSuffix(o, ".tenants.example.net") },
	}

	for origin, want := range map[string]bool{
		"https://app.example.com":          true,
		"https://acme.tenants.example.net": true,
		"https://evil.example.net":         false,
	} {
		t.Run(origin, func(t *testing.T) {
			w, _ := cross(t, cfg, getFrom(origin))
			if got := w.Header().Get("Access-Control-Allow-Origin") != ""; got != want {
				t.Errorf("allowed is %v, want %v", got, want)
			}
		})
	}
}

func TestAStarHeaderEchoesWhatWasAskedFor(t *testing.T) {
	cfg := CORSConfig{AllowedOrigins: []string{"https://app.example.com"}, AllowedHeaders: []string{"*"}}
	w, _ := cross(t, cfg, preflightTo("https://app.example.com", "GET", "X-Anything", "X-Else"))

	if got := w.Header().Get("Access-Control-Allow-Headers"); got != "X-Anything, X-Else" {
		t.Errorf("Access-Control-Allow-Headers is %q", got)
	}
}

func TestTheZeroConfigAllowsNothing(t *testing.T) {
	w, ran := cross(t, CORSConfig{}, getFrom("https://app.example.com"))

	if !ran {
		t.Error("the handler did not run")
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("the zero config allowed %q", got)
	}
}
