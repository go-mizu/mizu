//go:build race || mizudebug

package web

import (
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-mizu/mizu/router"
)

// TestAStaleCtxPanics is the check this build exists for.
func TestAStaleCtxPanics(t *testing.T) {
	var stale *Ctx
	serve(t, httptest.NewRequest("GET", "/things/7", nil), func(c *Ctx) error {
		stale = c
		return nil
	})

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a Ctx used after its handler returned did not panic")
		}
		for _, want := range []string{
			"web.Ctx used after the request completed",
			"Ctx.Request",
			"GET /things/7",
			"does not outlive its handler",
		} {
			if !strings.Contains(r.(string), want) {
				t.Errorf("the panic does not mention %q:\n%s", want, r)
			}
		}
	}()

	stale.Request()
}

// TestAStalePanicNamesTheRoute is the same thing for a request that came
// through the router, where the pattern says more than the path does.
func TestAStalePanicNamesTheRoute(t *testing.T) {
	var stale *Ctx
	rt := router.New()
	rt.Handle("GET /things/{id:int}", H(func(c *Ctx) error {
		stale = c
		return nil
	}))
	rt.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/things/7", nil))

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a Ctx used after its handler returned did not panic")
		}
		if want := "GET /things/{id:int}"; !strings.Contains(r.(string), want) {
			t.Errorf("the panic does not name the route %q:\n%s", want, r)
		}
	}()

	stale.Param("id")
}

// TestEveryMethodChecks walks the exported methods and asks each one whether it
// notices. A method added without a live call fails here.
func TestEveryMethodChecks(t *testing.T) {
	for _, call := range staleCalls() {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("Ctx.%s does not check whether the request is still running", call.name)
					return
				}
				if !strings.Contains(r.(string), "Ctx."+call.name) {
					t.Errorf("Ctx.%s panicked with %v, which names another method", call.name, r)
				}
			}()

			var stale *Ctx
			serve(t, httptest.NewRequest("POST", "/things/7", strings.NewReader("a=b")), func(c *Ctx) error {
				stale = c
				return nil
			})
			call.fn(stale)
		}()
	}
}

// staleCalls is one call of every exported method, so that a method with no
// live call in it has somewhere to be caught.
//
// It is written out rather than found by reflection because a method needs
// arguments that mean something, and because a list somebody has to add to is
// the point: the addition is where the live call gets remembered.
func staleCalls() []struct {
	name string
	fn   func(*Ctx)
} {
	return []struct {
		name string
		fn   func(*Ctx)
	}{
		{"Request", func(c *Ctx) { c.Request() }},
		{"Writer", func(c *Ctx) { c.Writer() }},
		{"Context", func(c *Ctx) { c.Context() }},
		{"Detach", func(c *Ctx) { c.Detach() }},
		{"Deadline", func(c *Ctx) { c.Deadline() }},
		{"Route", func(c *Ctx) { c.Route() }},
		{"Params", func(c *Ctx) { c.Params() }},
		{"Param", func(c *Ctx) { c.Param("id") }},
		{"ParamInt", func(c *Ctx) { c.ParamInt("id") }},
		{"RequestID", func(c *Ctx) { c.RequestID() }},
		{"Log", func(c *Ctx) { c.Log() }},
		{"IP", func(c *Ctx) { c.IP() }},
		{"Query", func(c *Ctx) { c.Query("q") }},
		{"QueryDefault", func(c *Ctx) { c.QueryDefault("q", "") }},
		{"QueryAll", func(c *Ctx) { c.QueryAll("q") }},
		{"Has", func(c *Ctx) { c.Has("q") }},
		{"Filled", func(c *Ctx) { c.Filled("q") }},
		{"Form", func(c *Ctx) { c.Form("a") }},
		{"Header", func(c *Ctx) { c.Header("Accept") }},
		{"Cookie", func(c *Ctx) { c.Cookie("session") }},
		{"Bearer", func(c *Ctx) { c.Bearer() }},
		{"IsAJAX", func(c *Ctx) { c.IsAJAX() }},
		{"WantsJSON", func(c *Ctx) { c.WantsJSON() }},
		{"Body", func(c *Ctx) { c.Body() }},
		{"BodyBytes", func(c *Ctx) { c.BodyBytes() }},
		{"Status", func(c *Ctx) { c.Status(200) }},
		{"StatusCode", func(c *Ctx) { c.StatusCode() }},
		{"SetHeader", func(c *Ctx) { c.SetHeader("X-Thing", "v") }},
		{"SetCookie", func(c *Ctx) { c.SetCookie(nil) }},
		{"Write", func(c *Ctx) { c.Write(nil) }},
		{"Text", func(c *Ctx) { c.Text("") }},
		{"HTML", func(c *Ctx) { c.HTML("") }},
		{"Bytes", func(c *Ctx) { c.Bytes("", nil) }},
		{"Stream", func(c *Ctx) { c.Stream("", strings.NewReader("")) }},
		{"File", func(c *Ctx) { c.File("nothing") }},
		{"FileFS", func(c *Ctx) { c.FileFS(fstest.MapFS{}, "nothing") }},
		{"Download", func(c *Ctx) { c.Download("a.txt", strings.NewReader("")) }},
		{"Attachment", func(c *Ctx) { c.Attachment("nothing", "a.txt") }},
		{"NoContent", func(c *Ctx) { c.NoContent() }},
		{"Redirect", func(c *Ctx) { c.Redirect("/") }},
	}
}

// TestNoCtxIsHandedOutTwice is the part a generation counter on its own cannot
// do. Every request in this build gets a Ctx of its own, so a pointer kept past
// the handler stays stale rather than turning into somebody else's request.
func TestNoCtxIsHandedOutTwice(t *testing.T) {
	seen := map[*Ctx]bool{}
	for range 100 {
		serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
			if seen[c] {
				t.Fatal("a Ctx was handed out twice in the build that is meant to allocate")
			}
			seen[c] = true
			return nil
		})
	}
}
