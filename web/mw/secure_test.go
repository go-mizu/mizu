package mw

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// sent serves one request through Secure and reports the response headers.
func sent(tb testing.TB, cfg SecureConfig) http.Header {
	tb.Helper()

	w := httptest.NewRecorder()
	Secure(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("a page"))
	})).ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	return w.Header()
}

func TestTheHeadersAreTheOnesTheConfigAskedFor(t *testing.T) {
	h := sent(t, SecureConfig{
		HSTS:               365 * 24 * time.Hour,
		HSTSSubdomains:     true,
		FrameOptions:       "DENY",
		ContentTypeOptions: true,
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		PermissionsPolicy:  "camera=(), microphone=()",
		CSP:                "default-src 'self'",
		COOP:               "same-origin",
		COEP:               "require-corp",
		CORP:               "same-origin",
	})

	want := map[string]string{
		"Strict-Transport-Security":    "max-age=31536000; includeSubDomains",
		"X-Frame-Options":              "DENY",
		"X-Content-Type-Options":       "nosniff",
		"Referrer-Policy":              "strict-origin-when-cross-origin",
		"Permissions-Policy":           "camera=(), microphone=()",
		"Content-Security-Policy":      "default-src 'self'",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Embedder-Policy": "require-corp",
		"Cross-Origin-Resource-Policy": "same-origin",
	}

	for name, value := range want {
		if got := h.Get(name); got != value {
			t.Errorf("%s is %q, want %q", name, got, value)
		}
	}
}

// TestTheZeroConfigSendsNothing is what makes this safe to put in a chain before
// anybody has decided what the policy is.
func TestTheZeroConfigSendsNothing(t *testing.T) {
	h := sent(t, SecureConfig{})

	for name := range h {
		switch name {
		case "Content-Type": // net/http sniffs one for the body.
		default:
			t.Errorf("the zero config sent %s: %q", name, h.Get(name))
		}
	}
}

func TestHSTS(t *testing.T) {
	cases := []struct {
		name string
		cfg  SecureConfig
		want string
	}{
		{"a year", SecureConfig{HSTS: 365 * 24 * time.Hour}, "max-age=31536000"},
		{"with subdomains", SecureConfig{HSTS: 365 * 24 * time.Hour, HSTSSubdomains: true}, "max-age=31536000; includeSubDomains"},
		{
			"the whole preload set",
			SecureConfig{HSTS: 365 * 24 * time.Hour, HSTSSubdomains: true, HSTSPreload: true},
			"max-age=31536000; includeSubDomains; preload",
		},
		{"an hour, which is how you start", SecureConfig{HSTS: time.Hour}, "max-age=3600"},
		{"none", SecureConfig{}, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sent(t, c.cfg).Get("Strict-Transport-Security"); got != c.want {
				t.Errorf("Strict-Transport-Security is %q, want %q", got, c.want)
			}
		})
	}
}

// TestAPreloadThatWouldBeRejectedIsRejectedHere is the one worth catching early,
// because a submission to the browser vendors' list is quick to make and slow to
// undo.
func TestAPreloadThatWouldBeRejectedIsRejectedHere(t *testing.T) {
	cases := map[string]SecureConfig{
		"no max age":    {HSTSPreload: true},
		"under a year":  {HSTS: 30 * 24 * time.Hour, HSTSSubdomains: true, HSTSPreload: true},
		"no subdomains": {HSTS: 365 * 24 * time.Hour, HSTSPreload: true},
	}

	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("it was accepted")
				}
			}()
			Secure(cfg)
		})
	}
}

func TestFrameOptionsIsACloseSet(t *testing.T) {
	for _, v := range []string{"deny", "ALLOW-FROM https://example.com", "SAME-ORIGIN", "none"} {
		t.Run(v, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("%q was accepted", v)
				}
			}()
			Secure(SecureConfig{FrameOptions: v})
		})
	}
}

func TestReportOnlyIsADifferentHeader(t *testing.T) {
	h := sent(t, SecureConfig{CSP: "default-src 'self'", CSPReportOnly: true})

	if got := h.Get("Content-Security-Policy-Report-Only"); got != "default-src 'self'" {
		t.Errorf("Content-Security-Policy-Report-Only is %q", got)
	}
	if got := h.Get("Content-Security-Policy"); got != "" {
		t.Errorf("a report only policy also went out as Content-Security-Policy: %q", got)
	}
}

// TestAHandlerCanOverrideOne is the way a route that serves an embeddable widget
// gets out of the site wide policy, and it works because the headers go on
// before the handler runs.
func TestAHandlerCanOverrideOne(t *testing.T) {
	w := httptest.NewRecorder()
	Secure(SecureConfig{FrameOptions: "DENY", ContentTypeOptions: true})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		}),
	).ServeHTTP(w, httptest.NewRequest("GET", "/embed", nil))

	if got := w.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options is %q, and the handler asked for SAMEORIGIN", got)
	}
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options is %q, and the handler said nothing about it", got)
	}
}

// TestHSTSGoesOutOverPlainHTTPToo pins the decision. A browser ignores it on a
// plain connection, and every service behind a proxy that terminates TLS sees
// plain connections.
func TestHSTSGoesOutOverPlainHTTPToo(t *testing.T) {
	h := sent(t, SecureConfig{HSTS: time.Hour})
	if h.Get("Strict-Transport-Security") == "" {
		t.Error("no Strict-Transport-Security on a plain request")
	}
}
