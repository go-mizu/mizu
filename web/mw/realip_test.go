package mw

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/go-mizu/mizu/web"
)

// remote serves one request through m and reports the RemoteAddr the handler
// was given.
func remote(tb testing.TB, m web.Middleware, r *http.Request) string {
	tb.Helper()

	var addr string
	m(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		addr = r.RemoteAddr
	})).ServeHTTP(httptest.NewRecorder(), r)
	return addr
}

func TestRealIP(t *testing.T) {
	cases := []struct {
		name    string
		peer    string
		headers map[string][]string
		want    string
	}{{
		name: "no forwarding header, so nothing to rewrite",
		peer: "10.0.0.1:1234",
		want: "10.0.0.1:1234",
	}, {
		name:    "one hop, which is the client",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"X-Forwarded-For": {"203.0.113.7"}},
		want:    "203.0.113.7:0",
	}, {
		name:    "the client, then a proxy of ours that recorded it",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"X-Forwarded-For": {"203.0.113.7, 10.0.0.5"}},
		want:    "203.0.113.7:0",
	}, {
		name:    "the client sent a hop of their own in front of the real one",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"X-Forwarded-For": {"1.2.3.4, 203.0.113.7, 10.0.0.5"}},
		want:    "203.0.113.7:0",
	}, {
		name:    "one header per hop rather than one list",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"X-Forwarded-For": {"203.0.113.7", "10.0.0.5"}},
		want:    "203.0.113.7:0",
	}, {
		name:    "the hop carries a port, which some proxies write",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"X-Forwarded-For": {"203.0.113.7:5544"}},
		want:    "203.0.113.7:0",
	}, {
		name:    "the connection is not one of ours, so the header is a stranger's",
		peer:    "198.51.100.9:443",
		headers: map[string][]string{"X-Forwarded-For": {"203.0.113.7"}},
		want:    "198.51.100.9:443",
	}, {
		name:    "every hop is a proxy of ours, so the client is not in the list",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"X-Forwarded-For": {"10.0.0.5, 10.0.0.6"}},
		want:    "10.0.0.1:1234",
	}, {
		name:    "a hop that is not an address is a wall",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"X-Forwarded-For": {"unknown, 10.0.0.5"}},
		want:    "10.0.0.1:1234",
	}, {
		name:    "the connection is loopback over IPv6",
		peer:    "[::1]:8080",
		headers: map[string][]string{"X-Forwarded-For": {"203.0.113.7"}},
		want:    "203.0.113.7:0",
	}, {
		name:    "the connection is a v4 address written as a v6 one",
		peer:    "[::ffff:10.0.0.1]:1234",
		headers: map[string][]string{"X-Forwarded-For": {"203.0.113.7"}},
		want:    "203.0.113.7:0",
	}, {
		name:    "the client is on IPv6",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"X-Forwarded-For": {"2001:db8::1, 10.0.0.5"}},
		want:    "[2001:db8::1]:0",
	}, {
		name:    "Forwarded, with the parameters nothing here reads",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"Forwarded": {"for=203.0.113.7;proto=https;by=10.0.0.5"}},
		want:    "203.0.113.7:0",
	}, {
		name: "Forwarded is read and X-Forwarded-For is not",
		peer: "10.0.0.1:1234",
		headers: map[string][]string{
			"Forwarded":       {"for=203.0.113.7"},
			"X-Forwarded-For": {"198.51.100.9"},
		},
		want: "203.0.113.7:0",
	}, {
		name: "Forwarded with no for parameter falls through to the other header",
		peer: "10.0.0.1:1234",
		headers: map[string][]string{
			"Forwarded":       {"proto=https;host=example.com"},
			"X-Forwarded-For": {"198.51.100.9"},
		},
		want: "198.51.100.9:0",
	}, {
		name:    "Forwarded writes IPv6 quoted and in brackets, with a port",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"Forwarded": {`for="[2001:db8::1]:4711"`}},
		want:    "[2001:db8::1]:0",
	}, {
		name:    "Forwarded writes IPv6 quoted and in brackets, without a port",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"Forwarded": {`for="[2001:db8::1]"`}},
		want:    "[2001:db8::1]:0",
	}, {
		name:    "Forwarded lists two elements",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"Forwarded": {"for=203.0.113.7;proto=http, for=10.0.0.5"}},
		want:    "203.0.113.7:0",
	}, {
		name:    "Forwarded hides the address, which RFC 7239 allows and this cannot check",
		peer:    "10.0.0.1:1234",
		headers: map[string][]string{"Forwarded": {"for=_hidden, for=10.0.0.5"}},
		want:    "10.0.0.1:1234",
	}, {
		name:    "the connection is not an address at all, which is what a Unix socket looks like",
		peer:    "@",
		headers: map[string][]string{"X-Forwarded-For": {"203.0.113.7"}},
		want:    "@",
	}}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = c.peer
			for name, values := range c.headers {
				for _, v := range values {
					r.Header.Add(name, v)
				}
			}

			if got := remote(t, RealIP(Private()...), r); got != c.want {
				t.Errorf("the handler saw %q, want %q", got, c.want)
			}
		})
	}
}

// TestTrustingNobodyTrustsNothing is the default somebody gets by writing
// RealIP() and thinking it means turn this on.
func TestTrustingNobodyTrustsNothing(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.7")

	if got := remote(t, RealIP(), r); got != "10.0.0.1:1234" {
		t.Errorf("the handler saw %q, and an empty list trusts nothing", got)
	}
}

func TestOnlyTheProxiesNamedAreTrusted(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.7")

	only := netip.MustParsePrefix("10.0.4.0/24")
	if got := remote(t, RealIP(only), r); got != "10.0.0.1:1234" {
		t.Errorf("the handler saw %q, and 10.0.0.1 is not in 10.0.4.0/24", got)
	}
}

// TestTheRequestTheServerOwnsIsNotTouched is the rule in net/http.Handler's
// documentation, and the reason the rewrite goes on a copy.
func TestTheRequestTheServerOwnsIsNotTouched(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.7")

	if got := remote(t, RealIP(Private()...), r); got != "203.0.113.7:0" {
		t.Fatalf("the handler saw %q, want 203.0.113.7:0", got)
	}
	if r.RemoteAddr != "10.0.0.1:1234" {
		t.Errorf("the request the server handed over now says %q", r.RemoteAddr)
	}
}

func TestTheAddressInAHopIsReadHoweverItIsWritten(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"203.0.113.7", "203.0.113.7"},
		{"  203.0.113.7  ", "203.0.113.7"},
		{`"203.0.113.7"`, "203.0.113.7"},
		{"203.0.113.7:5544", "203.0.113.7"},
		{"::ffff:203.0.113.7", "203.0.113.7"},
		{"2001:db8::1", "2001:db8::1"},
		{"[2001:db8::1]", "2001:db8::1"},
		{"[2001:db8::1]:4711", "2001:db8::1"},
		{"unknown", "invalid IP"},
		{"_obfuscated", "invalid IP"},
		{"", "invalid IP"},
		{"[not an address]", "invalid IP"},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := address(c.in).String(); got != c.want {
				t.Errorf("address(%q) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}

func TestPrivateCoversTheRangesAProxyRunsIn(t *testing.T) {
	in := []string{"127.0.0.1", "10.255.0.1", "172.16.0.1", "172.31.255.255", "192.168.1.1", "::1", "fd00::1"}
	out := []string{"203.0.113.7", "172.32.0.1", "8.8.8.8", "169.254.169.254", "2001:db8::1"}

	nets := Private()
	for _, s := range in {
		if !within(netip.MustParseAddr(s), nets) {
			t.Errorf("%s is not in the private list and it is a private address", s)
		}
	}
	for _, s := range out {
		if within(netip.MustParseAddr(s), nets) {
			t.Errorf("%s is in the private list and it is not a private address", s)
		}
	}
}
