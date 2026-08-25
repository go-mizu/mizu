package web_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-mizu/mizu/errs"
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

// Chain wraps a handler in middleware, and the order it reads in is the order
// it runs in.
func ExampleChain() {
	mark := func(name string) web.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Println("in", name)
				next.ServeHTTP(w, r)
				fmt.Println("out", name)
			})
		}
	}

	r := router.New()
	r.Handle("GET /", web.H(func(c *web.Ctx) error { return c.Text("hello") }))

	h := web.Chain(r, mark("first"), mark("second"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	// Output:
	// in first
	// in second
	// out second
	// out first
}

// A Stack is a chain whose order can be declared separately from the order
// things were added to it, which is what middleware that has to run before
// other middleware needs.
func ExampleStack() {
	mark := func(name string) web.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Println(name)
				next.ServeHTTP(w, r)
			})
		}
	}

	// The order these were added in is wrong, and nothing here has to know
	// that: the priority list says what the right one is.
	var s web.Stack
	s.Add("csrf", mark("csrf"))
	s.Use(mark("request id"))
	s.Add("session", mark("session"))
	s.Priority("session", "csrf")

	h := s.Then(web.H(func(c *web.Ctx) error { return c.Text("hello") }))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	// Output:
	// session
	// request id
	// csrf
}

// Record is how middleware finds out how the request was answered, without
// wrapping the response writer itself.
func ExampleRecord() {
	logger := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := web.Record(w)
			next.ServeHTTP(rec, r)
			fmt.Println(r.Method, r.URL.Path, rec.Status(), rec.Written())
		})
	}

	gone := web.H(func(c *web.Ctx) error {
		return c.Status(http.StatusNotFound).Text("no such post")
	})

	web.Chain(gone, logger).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/posts/7", nil))

	// Output:
	// GET /posts/7 404 12
}

// Bind fills a struct from the request, taking each field from where its tag
// says and from the query string when nothing says otherwise.
func ExampleBind() {
	type search struct {
		Q       string `query:"q"`
		Tags    []string
		Page    int
		PerPage int
	}

	list := web.H(func(c *web.Ctx) error {
		in, err := web.Bind[search](c)
		if err != nil {
			return err
		}
		return c.Text(fmt.Sprintf("%q %v page %d of %d", in.Q, in.Tags, in.Page, in.PerPage))
	})

	w := httptest.NewRecorder()
	list.ServeHTTP(w, httptest.NewRequest("GET", "/?q=water&tags=go&tags=web&page=2&per_page=25", nil))

	fmt.Println(w.Body)
	// Output:
	// "water" [go web] page 2 of 25
}

// A JSON body binds through the same call and the same struct. What the body
// carries wins, and what it left out the query string still fills in.
func ExampleBind_body() {
	type post struct {
		Title string   `json:"title"`
		Tags  []string `json:"tags"`
		Draft bool     `json:"draft"`
	}

	create := web.H(func(c *web.Ctx) error {
		in, err := web.Bind[post](c)
		if err != nil {
			return err
		}
		return c.Text(fmt.Sprintf("%q %v draft %v", in.Title, in.Tags, in.Draft))
	})

	r := httptest.NewRequest("POST", "/posts?draft=true",
		strings.NewReader(`{"title":"water","tags":["go"]}`))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	create.ServeHTTP(w, r)

	fmt.Println(w.Body)
	// Output:
	// "water" [go] draft true
}

// A member the struct has no field for is a mistake, unless the struct embeds
// AllowUnknown to say that it is not. A webhook payload that grows a field is
// the case for it.
func ExampleAllowUnknown() {
	type hook struct {
		web.AllowUnknown

		Event string `json:"event"`
	}

	receive := web.H(func(c *web.Ctx) error {
		in, err := web.Bind[hook](c)
		if err != nil {
			return err
		}
		return c.Text(in.Event)
	})

	r := httptest.NewRequest("POST", "/hooks",
		strings.NewReader(`{"event":"paid","added_in_v3":true}`))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	receive.ServeHTTP(w, r)

	fmt.Println(w.Body)
	// Output:
	// paid
}

// Ctx.JSON reads the body and nothing else, for a payload that is not a struct
// or one whose signature was checked before it was decoded.
func ExampleCtx_JSON() {
	receive := web.H(func(c *web.Ctx) error {
		var in map[string]int
		if err := c.JSON(&in); err != nil {
			return err
		}
		return c.Text(fmt.Sprint(in["items"] * in["each"]))
	})

	r := httptest.NewRequest("POST", "/total", strings.NewReader(`{"items":3,"each":7}`))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	receive.ServeHTTP(w, r)

	fmt.Println(w.Body)
	// Output:
	// 21
}

// A *web.Upload field binds a file out of a multipart form. The type is
// sniffed from the file's first bytes rather than taken from what the client
// claimed it was sending.
func ExampleUpload() {
	type avatar struct {
		Name  string      `form:"name"`
		Image *web.Upload `form:"image"`
	}

	save := web.H(func(c *web.Ctx) error {
		in, err := web.Bind[avatar](c)
		if err != nil {
			return err
		}
		return c.Text(fmt.Sprintf("%s sent %s, %s, %d bytes",
			in.Name, in.Image.Filename, in.Image.MIME, in.Image.Size))
	})

	w := httptest.NewRecorder()
	save.ServeHTTP(w, avatarRequest())

	fmt.Println(w.Body)
	// Output:
	// water sent me.gif, image/gif, 35 bytes
}

// avatarRequest posts a name and a small GIF, which is the shortest real image
// there is and keeps the example about the binding.
func avatarRequest() *http.Request {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	w.WriteField("name", "water")
	f, _ := w.CreateFormFile("image", "me.gif")
	gif.Encode(f, image.NewPaletted(image.Rect(0, 0, 1, 1), color.Palette{color.Black}), nil)
	w.Close()

	r := httptest.NewRequest("POST", "/avatar", &body)
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r
}

// A value that will not decode comes back as one errs.Field per field, which is
// what a form redisplay and a JSON error document are both built from.
func ExampleBind_errors() {
	type search struct {
		Page    int `query:"page"`
		PerPage int `query:"per_page"`
	}

	list := web.H(func(c *web.Ctx) error {
		_, err := web.Bind[search](c)
		for _, f := range errs.Fields(err) {
			fmt.Println(f.Name, f.Code, f.Msg)
		}
		return nil
	})

	list.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/?page=one&per_page=999999999999999999999", nil))

	// Output:
	// page invalid_number Must be a whole number.
	// per_page out_of_range Is too large for this field.
}
