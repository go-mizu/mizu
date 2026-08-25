package mw

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-mizu/mizu/clock"
	"github.com/go-mizu/mizu/router"
	"github.com/go-mizu/mizu/web"
)

// collector keeps the records rather than formatting them, so a test reads an
// attribute by name instead of matching text.
type collector struct{ got []slog.Record }

func (c *collector) Enabled(context.Context, slog.Level) bool { return true }
func (c *collector) WithAttrs([]slog.Attr) slog.Handler       { return c }
func (c *collector) WithGroup(string) slog.Handler            { return c }

func (c *collector) Handle(_ context.Context, r slog.Record) error {
	c.got = append(c.got, r.Clone())
	return nil
}

// only is the one record the collector holds, and a failure when it holds any
// other number.
func (c *collector) only(tb testing.TB) slog.Record {
	tb.Helper()
	if len(c.got) != 1 {
		tb.Fatalf("the collector holds %d records, want one", len(c.got))
	}
	return c.got[0]
}

// attrs is one record's attributes by name.
func attrs(r slog.Record) map[string]string {
	out := map[string]string{}
	for a := range r.Attrs {
		out[a.Key] = a.Value.String()
	}
	return out
}

// logged serves one request through m and reports what came out.
func logged(tb testing.TB, build func(*slog.Logger) http.Handler, r *http.Request) map[string]string {
	tb.Helper()

	c := new(collector)
	build(slog.New(c)).ServeHTTP(httptest.NewRecorder(), r)
	return attrs(c.only(tb))
}

func TestTheRecordSaysWhatHappened(t *testing.T) {
	r := httptest.NewRequest("GET", "/posts/7?draft=1", nil)
	r.RemoteAddr = "203.0.113.7:44321"
	r.Header.Set("User-Agent", "curl/8.5.0")

	got := logged(t, func(l *slog.Logger) http.Handler {
		return Logger(l)(web.H(func(c *web.Ctx) error {
			return c.Status(http.StatusCreated).Text("made it")
		}))
	}, r)

	want := map[string]string{
		"method": "GET",
		"path":   "/posts/7",
		"status": "201",
		"bytes":  "7",
		"ip":     "203.0.113.7",
		"ua":     "curl/8.5.0",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("%s is %q, want %q", key, got[key], value)
		}
	}
	// The query string is on the URL and not on the path, because a log grouped
	// by path is a log with one group per query.
	if _, ok := got["route"]; ok {
		t.Errorf("route is %q, and nothing routed this request", got["route"])
	}
}

func TestAHandlerThatWroteNothingSentA200(t *testing.T) {
	got := logged(t, func(l *slog.Logger) http.Handler {
		return Logger(l)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	}, httptest.NewRequest("GET", "/", nil))

	if got["status"] != "200" {
		t.Errorf("status is %q, want 200", got["status"])
	}
	if got["bytes"] != "0" {
		t.Errorf("bytes is %q, want 0", got["bytes"])
	}
}

func TestTheLevelIsErrorOnlyForTheServersOwnFailures(t *testing.T) {
	cases := []struct {
		status int
		want   slog.Level
	}{
		{http.StatusOK, slog.LevelInfo},
		{http.StatusMovedPermanently, slog.LevelInfo},
		{http.StatusNotFound, slog.LevelInfo},
		{http.StatusTooManyRequests, slog.LevelInfo},
		{http.StatusInternalServerError, slog.LevelError},
		{http.StatusBadGateway, slog.LevelError},
	}

	for _, c := range cases {
		t.Run(http.StatusText(c.status), func(t *testing.T) {
			got := new(collector)
			Logger(slog.New(got))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(c.status)
			})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

			if level := got.only(t).Level; level != c.want {
				t.Errorf("a %d is logged at %s, want %s", c.status, level, c.want)
			}
		})
	}
}

// TestTheRequestIdIsOnTheRecord is the pair the package comment is about, in
// the order that works.
func TestTheRequestIdIsOnTheRecord(t *testing.T) {
	got := logged(t, func(l *slog.Logger) http.Handler {
		return web.Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), RequestID(), Logger(l))
	}, httptest.NewRequest("GET", "/", nil))

	if got["request_id"] == "" {
		t.Error("the record has no request_id on it")
	}
}

// TestTheRequestIdIsMissingInTheOtherOrder is the same pair the wrong way
// round. It is here so that the behaviour the package comment warns about is
// the behaviour, rather than something that changes without anybody noticing.
func TestTheRequestIdIsMissingInTheOtherOrder(t *testing.T) {
	got := logged(t, func(l *slog.Logger) http.Handler {
		return web.Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), Logger(l), RequestID())
	}, httptest.NewRequest("GET", "/", nil))

	if id, ok := got["request_id"]; ok {
		t.Errorf("request_id is %q, and the id is set inside the logger", id)
	}
}

func TestTheAddressIsWhateverRealIPLeftBehind(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.7")

	got := logged(t, func(l *slog.Logger) http.Handler {
		return web.Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), RealIP(Private()...), Logger(l))
	}, r)

	if got["ip"] != "203.0.113.7" {
		t.Errorf("ip is %q, want 203.0.113.7", got["ip"])
	}
}

func TestAnAddressThatIsNotOneIsReportedAsItIs(t *testing.T) {
	cases := map[string]string{
		"203.0.113.7:44321": "203.0.113.7",
		"203.0.113.7":       "203.0.113.7",
		"::ffff:10.0.0.1":   "10.0.0.1",
		"@":                 "@",
		"":                  "",
	}

	for in, want := range cases {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = in
		if got := from(r); got != want {
			t.Errorf("from(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTheDurationComesFromTheClock(t *testing.T) {
	fake := clock.Fake(time.Date(2026, 3, 31, 23, 55, 0, 0, time.UTC))

	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(clock.With(r.Context(), fake))

	got := logged(t, func(l *slog.Logger) http.Handler {
		return Logger(l)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			fake.Advance(250 * time.Millisecond)
		}))
	}, r)

	if got["dur"] != "250ms" {
		t.Errorf("dur is %q, want 250ms", got["dur"])
	}
}

// TestTheRouteIsThereWhenTheRouterHasMatched is the second of the two ways to
// place this middleware, and the one that fills in the key.
func TestTheRouteIsThereWhenTheRouterHasMatched(t *testing.T) {
	c := new(collector)
	l := slog.New(c)

	rt := router.New()
	rt.Handle("GET /posts/{id:int}", web.Chain(web.H(func(c *web.Ctx) error {
		return c.Text("post " + c.Param("id"))
	}), Logger(l)))

	rt.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/posts/7", nil))

	got := attrs(c.only(t))
	if got["route"] != "GET /posts/{id:int}" {
		t.Errorf("route is %q, want the pattern", got["route"])
	}
	if got["path"] != "/posts/7" {
		t.Errorf("path is %q, want /posts/7", got["path"])
	}
}

func TestANamedRouteIsLoggedByItsName(t *testing.T) {
	c := new(collector)
	l := slog.New(c)

	rt := router.New()
	rt.Handle("GET /posts/{id:int}", web.Chain(web.H(func(*web.Ctx) error { return nil }), Logger(l))).Name("posts.show")

	rt.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/posts/7", nil))

	if got := attrs(c.only(t)); got["route"] != "posts.show" {
		t.Errorf("route is %q, want posts.show", got["route"])
	}
}
