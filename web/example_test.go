package web_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-mizu/mizu/router"
	"github.com/go-mizu/mizu/web"
)

// A handler takes a *web.Ctx and returns an error, and web.H turns it into
// something a router can hold.
func Example() {
	show := func(c *web.Ctx) error {
		id, ok := c.ParamInt("id")
		if !ok {
			return errors.New("that is not a post")
		}
		return c.Text(fmt.Sprintf("post %d", id))
	}

	r := router.New()
	r.Handle("GET /posts/{id:int}", web.H(show))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/posts/7", nil))

	fmt.Println(w.Code, w.Body)
	// Output:
	// 200 post 7
}

// Status goes in front of the write it applies to, and the write is the last
// thing the handler does.
func ExampleCtx_Status() {
	create := func(c *web.Ctx) error {
		return c.Status(http.StatusCreated).
			SetHeader("Location", "/posts/7").
			Text("made")
	}

	w := httptest.NewRecorder()
	web.H(create).ServeHTTP(w, httptest.NewRequest("POST", "/posts", nil))

	fmt.Println(w.Code, w.Header().Get("Location"), w.Body)
	// Output:
	// 201 /posts/7 made
}

// Errors decides once what a handler that failed looks like, and everything
// under it uses that.
func ExampleErrors() {
	render := func(c *web.Ctx, err error) {
		c.Status(http.StatusBadGateway).Text("sorry: " + err.Error())
	}

	fetch := func(c *web.Ctx) error {
		return errors.New("upstream said no")
	}

	h := web.Errors(render)(web.H(fetch))

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	fmt.Println(w.Code, w.Body)
	// Output:
	// 502 sorry: upstream said no
}

// Detach is the context for work that outlives the response. What goes into
// the goroutine is that context and a copy of the values it needs, never the
// Ctx itself.
func ExampleCtx_Detach() {
	archived := make(chan string, 1)

	record := func(c *web.Ctx) error {
		ctx, id := c.Detach(), c.Param("id")
		go func() {
			// ctx and id, never c
			_ = ctx
			archived <- id
		}()
		return c.NoContent()
	}

	r := router.New()
	r.Handle("GET /posts/{id}", web.H(record))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/posts/7", nil))

	fmt.Println(w.Code, "archived", <-archived)
	// Output:
	// 204 archived 7
}

// FromContext reaches the Ctx from code that was handed nothing but a context.
func ExampleFromContext() {
	// audit takes a context because most things do. It does not need to take a
	// *web.Ctx to find out which request it is in.
	audit := func(ctx context.Context, what string) {
		if c, ok := web.FromContext(ctx); ok {
			fmt.Println(what, "on", c.Request().URL.Path, "for", c.Param("id"))
			return
		}
		fmt.Println(what, "outside a request")
	}

	r := router.New()
	r.Handle("DELETE /posts/{id}", web.H(func(c *web.Ctx) error {
		audit(c.Context(), "delete")
		return c.NoContent()
	}))

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("DELETE", "/posts/7", nil))
	audit(context.Background(), "delete")
	// Output:
	// delete on /posts/7 for 7
	// delete outside a request
}
