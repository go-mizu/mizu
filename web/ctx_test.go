package web

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/router"
)

// TestResetClearsEveryField is the test the comment on reset points at.
//
// It fills every field of a Ctx with something that is not the zero value,
// resets, and asks whether anything survived. A field added to the struct and
// forgotten in reset fails here rather than in production, where it would show
// up as one request seeing the last request's data.
func TestResetClearsEveryField(t *testing.T) {
	// gen and was belong to the guard, which sets them itself around reset.
	owned := map[string]bool{"gen": true, "was": true}

	c := new(Ctx)
	v := reflect.ValueOf(c).Elem()
	fill(t, v)

	for i := range v.NumField() {
		if name := v.Type().Field(i).Name; !owned[name] && at(v.Field(i)).IsZero() {
			t.Fatalf("the test could not put anything in Ctx.%s, so it is not being checked", name)
		}
	}

	c.reset()

	for i := range v.NumField() {
		name := v.Type().Field(i).Name
		if owned[name] {
			continue
		}
		if !at(v.Field(i)).IsZero() {
			t.Errorf("reset left Ctx.%s set, so it carries over to the next request", name)
		}
	}
}

// fill puts something that is not the zero value in every field of a struct,
// as deep as the struct goes.
func fill(t *testing.T, v reflect.Value) {
	t.Helper()
	for i := range v.NumField() {
		set(t, at(v.Field(i)))
	}
}

// set puts something that is not the zero value in one place.
func set(t *testing.T, f reflect.Value) {
	t.Helper()
	switch f.Kind() {
	case reflect.Struct:
		fill(t, f)
	case reflect.Array:
		for i := range f.Len() {
			set(t, f.Index(i))
		}
	case reflect.Bool:
		f.SetBool(true)
	case reflect.Int, reflect.Int64:
		f.SetInt(1)
	case reflect.Uint64:
		f.SetUint(1)
	case reflect.String:
		f.SetString("x")
	case reflect.Slice:
		f.Set(reflect.MakeSlice(f.Type(), 1, 1))
	case reflect.Pointer:
		f.Set(reflect.New(f.Type().Elem()))
	case reflect.Func:
		f.Set(reflect.MakeFunc(f.Type(), func([]reflect.Value) []reflect.Value { return nil }))
	case reflect.Interface:
		w := reflect.ValueOf(httptest.NewRecorder())
		if !w.Type().Implements(f.Type()) {
			t.Fatalf("Ctx reaches an interface of type %s and this test has nothing to put in it", f.Type())
		}
		f.Set(w)
	default:
		t.Fatalf("Ctx reaches a value of kind %s and this test has nothing to put in it", f.Kind())
	}
}

// at makes an unexported field settable, which is allowed here because the
// field and the test are in the same package and nothing escapes.
func at(f reflect.Value) reflect.Value {
	return reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem()
}

func TestTheAccessorsReadTheRequest(t *testing.T) {
	r := httptest.NewRequest("GET", "/things/7", nil)
	r.RemoteAddr = "203.0.113.9:5555"

	var got *Ctx
	serve(t, r, func(c *Ctx) error {
		got = c
		if c.Request() != c.r {
			t.Error("Request is not the request being served")
		}
		if c.Writer() == nil {
			t.Error("Writer is nil")
		}
		if c.Context() != c.r.Context() {
			t.Error("Context is not the request's context")
		}
		if _, ok := c.Deadline(); ok {
			t.Error("a request with no deadline reported one")
		}
		if c.Route() != nil {
			t.Error("a request that did not come through a mizu router reported a route")
		}
		if c.Params().Len() != 0 {
			t.Error("a request with no route reported parameters")
		}
		if c.Param("id") != "" {
			t.Error("a request with no route reported a parameter value")
		}
		if want := netip.MustParseAddr("203.0.113.9"); c.IP() != want {
			t.Errorf("IP is %v, want %v", c.IP(), want)
		}
		return nil
	})

	if got == nil {
		t.Fatal("the handler did not run")
	}
}

func TestIPAnswersWithTheZeroAddressForSomethingThatIsNotOne(t *testing.T) {
	for _, remote := range []string{"", "@", "/tmp/app.sock", "not an address"} {
		r := httptest.NewRequest("GET", "/", nil)
		r.RemoteAddr = remote
		serve(t, r, func(c *Ctx) error {
			if c.IP().IsValid() {
				t.Errorf("RemoteAddr %q gave the address %v", remote, c.IP())
			}
			return nil
		})
	}
}

func TestIPUnmapsAnIPv4AddressThatArrivedAsIPv6(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "[::ffff:203.0.113.9]:5555"
	serve(t, r, func(c *Ctx) error {
		if want := netip.MustParseAddr("203.0.113.9"); c.IP() != want {
			t.Errorf("IP is %v, want %v", c.IP(), want)
		}
		return nil
	})
}

func TestIPBelievesNothingInAHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.9:5555"
	r.Header.Set("X-Forwarded-For", "198.51.100.1")
	r.Header.Set("X-Real-IP", "198.51.100.2")
	serve(t, r, func(c *Ctx) error {
		if want := netip.MustParseAddr("203.0.113.9"); c.IP() != want {
			t.Errorf("IP is %v, want %v, so a header was believed", c.IP(), want)
		}
		return nil
	})
}

func TestParamsComeFromTheRoute(t *testing.T) {
	rt := router.New()
	rt.Handle("GET /things/{id:int}/{action}", H(func(c *Ctx) error {
		if c.Route() == nil {
			t.Fatal("a request through the router reported no route")
		}
		if want := "GET /things/{id:int}/{action}"; c.Route().Info().Pattern != want {
			t.Errorf("the pattern is %q, want %q", c.Route().Info().Pattern, want)
		}
		if c.Params().Len() != 2 {
			t.Errorf("there are %d parameters, want 2", c.Params().Len())
		}
		if c.Param("action") != "edit" {
			t.Errorf("action is %q, want edit", c.Param("action"))
		}
		if id, ok := c.ParamInt("id"); !ok || id != 7 {
			t.Errorf("ParamInt(id) is %d, %v, want 7, true", id, ok)
		}
		if _, ok := c.ParamInt("action"); ok {
			t.Error("ParamInt read a word as a number")
		}
		if _, ok := c.ParamInt("nothing"); ok {
			t.Error("ParamInt read a parameter the route does not have")
		}
		return nil
	}))

	w := httptest.NewRecorder()
	rt.ServeHTTP(w, httptest.NewRequest("GET", "/things/7/edit", nil))
	if w.Code != http.StatusOK {
		t.Errorf("the status is %d, want 200", w.Code)
	}
}

func TestRequestIDIsEmptyUntilSomethingSetsIt(t *testing.T) {
	serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		if c.RequestID() != "" {
			t.Errorf("the request id is %q before anything set one", c.RequestID())
		}
		return nil
	})

	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(ctxdata.With(r.Context(), RequestIDKey, "abc123"))
	serve(t, r, func(c *Ctx) error {
		if c.RequestID() != "abc123" {
			t.Errorf("the request id is %q, want abc123", c.RequestID())
		}
		return nil
	})
}

func TestLogCarriesTheRequestIDAndTheRoute(t *testing.T) {
	var buf strings.Builder
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	rt := router.New()
	rt.Handle("GET /things/{id}", H(func(c *Ctx) error {
		if c.Log() != c.Log() {
			t.Error("Log built a second logger on the second call")
		}
		c.Log().Info("hello")
		return nil
	})).Name("things.show")

	r := httptest.NewRequest("GET", "/things/7", nil)
	r = r.WithContext(ctxdata.With(r.Context(), RequestIDKey, "abc123"))
	rt.ServeHTTP(httptest.NewRecorder(), r)

	for _, want := range []string{"request_id=abc123", "route=things.show", "msg=hello"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("the log line has no %s in it:\n%s", want, buf.String())
		}
	}
}

func TestLogNamesTheRouteByItsPatternWhenItHasNoName(t *testing.T) {
	var buf strings.Builder
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(old) })

	rt := router.New()
	rt.Handle("GET /things/{id}", H(func(c *Ctx) error {
		c.Log().Info("hello")
		return nil
	}))
	rt.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/things/7", nil))

	if want := `route="GET /things/{id}"`; !strings.Contains(buf.String(), want) {
		t.Errorf("the log line has no %s in it:\n%s", want, buf.String())
	}
}

func TestDetachOutlivesTheRequest(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(ctxdata.With(r.Context(), RequestIDKey, "abc123"))
	ctx, cancel := context.WithCancel(r.Context())
	r = r.WithContext(ctx)

	var detached context.Context
	serve(t, r, func(c *Ctx) error {
		detached = c.Detach()
		return nil
	})

	cancel()
	if err := detached.Err(); err != nil {
		t.Errorf("the detached context ended with %v when the request did", err)
	}
	if id, _ := ctxdata.Get(detached, RequestIDKey); id != "abc123" {
		t.Errorf("the detached context lost the request id, it has %q", id)
	}
}

func TestDetachStopsForShutdown(t *testing.T) {
	stop, down := context.WithCancel(context.Background())
	defer down()

	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(WithShutdown(r.Context(), stop))

	var detached context.Context
	serve(t, r, func(c *Ctx) error {
		detached = c.Detach()
		return nil
	})

	if err := detached.Err(); err != nil {
		t.Fatalf("the detached context was already over: %v", err)
	}

	down()
	select {
	case <-detached.Done():
	case <-time.After(time.Second):
		t.Error("the detached context did not stop when the server went down")
	}
}

// serve runs one handler over one request and returns what it wrote.
func serve(t *testing.T, r *http.Request, h Handler) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	H(h).ServeHTTP(w, r)
	return w
}

// direct builds a Ctx around a writer, for a test that is looking at the
// response wrapper rather than at a handler.
//
// It goes through acquire and release like H does, since a Ctx built with a
// struct literal is a Ctx the guarded build considers already finished.
func direct(t *testing.T, w http.ResponseWriter) *Ctx {
	t.Helper()
	c := acquire()
	c.res.ResponseWriter = w
	c.r = httptest.NewRequest("GET", "/", nil)
	t.Cleanup(func() { release(c) })
	return c
}
