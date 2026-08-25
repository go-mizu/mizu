package mizutest

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/go-mizu/mizu"
	"github.com/go-mizu/mizu/clock"
)

// An example function cannot take a parameter, and every fixture here needs the
// testing.TB of the test it belongs to. So the examples read one from here, and
// in a real test it is the t the test was handed. They are compiled rather than
// run, which is why none of them carries an Output comment.
var t *testing.T

func Example() {
	app := NewApp(t)
	app.Routes().Get("/posts/{id}", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]any{
			"id":    c.Param("id"),
			"title": "Hello",
		})
	})

	app.Get("/posts/1").Do().
		AssertOK().
		AssertJSONPath("$.title", "Hello")
}

// The fixture serves the handler in the same goroutine as the test, so a
// failure points at the line that made the request and a breakpoint in the
// handler is a breakpoint in the test.
func ExampleNewApp() {
	app := NewApp(t)
	app.Routes().Post("/posts", func(c *mizu.Ctx) error {
		var post struct {
			Title string `json:"title"`
		}
		if err := c.BindJSON(&post, 1<<20); err != nil {
			return err
		}
		return c.JSON(http.StatusCreated, map[string]any{"id": 1, "title": post.Title})
	})

	app.Post("/posts").JSON(map[string]any{"title": "Hello"}).Do().
		AssertCreated().
		AssertJSON(json.RawMessage(`{"id": 1, "title": "Hello"}`))
}

// Serve takes a handler the test built itself, which is what incremental
// adoption looks like: a service that is not a mizu application yet still gets
// the request builder, the assertions and the log.
func ExampleServe() {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"path": %q}`, r.URL.Path)
	})

	app := NewApp(t, Serve(h))
	app.Get("/anywhere").Do().AssertJSONPath("$.path", "/anywhere")
}

// At sets the fixture clock, so a handler that reads the time reads one the
// test chose. Anything that takes the clock from the context sees it.
func ExampleAt() {
	app := NewApp(t, At(time.Date(2026, 3, 14, 15, 9, 26, 0, time.UTC)))
	app.Routes().Get("/now", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, clock.Now(c.Context()).Format(time.RFC3339))
	})

	app.Get("/now").Do().AssertSee("2026-03-14T15:09:26Z")
}

// NoParallel is for a test that cannot run beside another one, usually because
// it calls t.Setenv or reaches something shared. Without it a fixture calls
// t.Parallel for you.
func ExampleNoParallel() {
	t.Setenv("APP_ENV", "testing")

	app := NewApp(t, NoParallel())
	app.Routes().Get("/env", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "testing")
	})

	app.Get("/env").Do().AssertOK()
}

// A request is built up a piece at a time and sent by Do. Nothing in the middle
// of the chain reports an error, so a mistake anywhere in it surfaces in one
// place.
func ExampleApp_Get() {
	app := NewApp(t)
	app.Routes().Get("/posts", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, []any{})
	})

	app.Get("/posts").
		WithQuery("page", "2").
		WithHeader("Accept", "application/json").
		WithCookie("locale", "ja").
		Do().
		AssertOK()
}

// AssertJSONSubset names the members the test is about and ignores the rest, so
// adding a field to a response does not break every test that reads it.
func ExampleResponse_AssertJSONSubset() {
	app := NewApp(t)
	app.Routes().Get("/posts/1", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]any{
			"id": 1, "title": "Hello", "created_at": "2026-01-01T00:00:00Z",
		})
	})

	app.Get("/posts/1").Do().
		AssertJSONSubset(map[string]any{"title": "Hello"})
}

// The path syntax is the part of JSONPath tests use: members, elements, and a
// negative index counting from the end.
func ExampleResponse_AssertJSONPath() {
	app := NewApp(t)
	app.Routes().Get("/posts", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]any{
			"data": []any{
				map[string]any{"id": 1, "title": "First"},
				map[string]any{"id": 2, "title": "Second"},
			},
			"meta": map[string]any{"total": 2},
		})
	})

	app.Get("/posts").Do().
		AssertJSONCount("$.data", 2).
		AssertJSONPath("$.data[0].title", "First").
		AssertJSONPath("$.data[-1].id", 2).
		AssertJSONPath("$.meta.total", 2).
		AssertJSONMissing("$.meta.next")
}

// A handler that is meant to say something about what it did can be asked
// whether it did. Nothing reaches stderr, so a test that passes stays quiet and
// a test that fails prints the log beside the response.
func ExampleApp_Log() {
	app := NewApp(t)
	app.Routes().Delete("/posts/{id}", func(c *mizu.Ctx) error {
		c.Logger().Warn("deleting a post", "id", c.Param("id"))
		return c.Text(http.StatusNoContent, "")
	})

	res := app.Delete("/posts/1").Do().AssertNoContent()

	for _, e := range res.Logs() {
		if e.Level == slog.LevelWarn && e.Attrs["id"] == "1" {
			return
		}
	}
	t.Error("the handler did not warn about the delete")
}

// The clock moves when the test moves it, so something that expires an hour
// from now can be checked an hour from now without the test taking an hour.
func ExampleApp_Clock() {
	app := NewApp(t)
	app.Routes().Get("/token", func(c *mizu.Ctx) error {
		c.SetCookie(&http.Cookie{
			Name:    "token",
			Value:   "abc",
			Expires: clock.Now(c.Context()).Add(time.Hour),
		})
		return c.Text(http.StatusOK, "ok")
	})

	app.Get("/token").Do().AssertCookie("token", "abc")

	app.Clock().Advance(2 * time.Hour)

	app.Get("/token").Do().AssertCookieExpired("token")
}

// Dump prints the whole exchange through the test log, for working something
// out. go test hides it unless the test fails or -v is on, so one left behind
// costs a passing run nothing.
func ExampleResponse_Dump() {
	app := NewApp(t)
	app.Routes().Get("/posts/1", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]any{"id": 1})
	})

	app.Get("/posts/1").Do().
		Dump().
		AssertOK()
}
