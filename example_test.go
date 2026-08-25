package mizu_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-mizu/mizu"
)

// A handler takes a [mizu.Ctx] and returns an error. The pattern is the one
// [net/http.ServeMux] understands, so a wildcard is written {id} and read with
// Param.
func Example() {
	app := mizu.New()
	app.ClearMiddleware() // the request logger, which writes a line to stderr

	app.Get("/posts/{id}", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "post "+c.Param("id"))
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/7", nil))

	fmt.Println(rec.Code, rec.Body)
	// Output: 200 post 7
}

func ExampleCtx_JSON() {
	app := mizu.New()
	app.ClearMiddleware() // the request logger, which writes a line to stderr

	app.Get("/posts/{id}", func(c *mizu.Ctx) error {
		return c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")})
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/7", nil))

	fmt.Println(rec.Header().Get("Content-Type"))
	fmt.Print(rec.Body)
	// Output:
	// application/json; charset=utf-8
	// {"id":"7"}
}

// Prefix writes a prefix in front of everything registered on the router it
// returns, and Group does the same for a block of routes.
func ExampleRouter_Group() {
	app := mizu.New()
	app.ClearMiddleware() // the request logger, which writes a line to stderr

	api := app.Prefix("/api")
	api.Group("/v1", func(g *mizu.Router) {
		g.Get("/ping", func(c *mizu.Ctx) error {
			return c.Text(http.StatusOK, "pong")
		})
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))

	fmt.Println(rec.Code, rec.Body)
	// Output: 200 pong
}

// Middleware is func(Handler) Handler. Use adds it to everything, outermost
// first, so the one passed first sees the request first and the response last.
func ExampleRouter_Use() {
	app := mizu.New()
	app.ClearMiddleware() // the request logger, which writes a line to stderr

	tag := func(name string) mizu.Middleware {
		return func(next mizu.Handler) mizu.Handler {
			return func(c *mizu.Ctx) error {
				fmt.Println("in", name)
				defer fmt.Println("out", name)
				return next(c)
			}
		}
	}
	app.Use(tag("outer"), tag("inner"))

	app.Get("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "hello")
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Code, rec.Body)
	// Output:
	// in outer
	// in inner
	// out inner
	// out outer
	// 200 hello
}

// A handler returns an error rather than writing a status and hoping every
// caller remembers to. ErrorHandler is the one place that turns one into a
// response.
func ExampleRouter_ErrorHandler() {
	app := mizu.New()
	app.ClearMiddleware() // the request logger, which writes a line to stderr

	app.ErrorHandler(func(c *mizu.Ctx, err error) {
		_ = c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	})

	app.Get("/posts/{id}", func(c *mizu.Ctx) error {
		return errors.New("the database is not answering")
	})

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/7", nil))

	fmt.Println(rec.Code)
	fmt.Print(rec.Body)
	// Output:
	// 500
	// {"error":"the database is not answering"}
}
