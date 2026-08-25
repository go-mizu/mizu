package router_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-mizu/mizu/router"
)

func Example() {
	r := router.New()
	r.HandleFunc("GET /posts", index)
	r.HandleFunc("GET /posts/{id:int}", show).Name("posts.show")
	r.HandleFunc("GET /posts/latest", latest)

	for _, path := range []string{"/posts", "/posts/7", "/posts/latest", "/posts/nope"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		fmt.Print(w.Code, " ", w.Body.String())
	}

	// Output:
	// 200 every post
	// 200 post 7
	// 200 the latest post
	// 404 404 page not found
}

func index(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "every post") }

func latest(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, "the latest post") }

func show(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(w, "post", router.PathValue(req, "id"))
}

// A route carries whatever the rest of the program needs to know about it, so
// middleware and metrics can ask which route ran rather than parsing the path a
// second time.
func ExampleMatched() {
	r := router.New()
	r.Handle("GET /posts/{id:int}", audited(http.HandlerFunc(show))).
		Name("posts.show").
		Meta("scope", "posts:read")

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/posts/7", nil))

	// Output:
	// posts.show wants posts:read and matched id=7
}

func audited(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if rt, params, ok := router.Matched(req); ok {
			fmt.Println(rt.Info().Name, "wants", rt.Value("scope"),
				"and matched", params.At(0).Name+"="+params.At(0).Value)
		}
		next.ServeHTTP(w, req)
	})
}

// A constraint is a plain function, so anything that can say yes or no about a
// segment can be one.
func ExampleConstrain() {
	even := func(s string) bool { return s != "" && (s[len(s)-1]-'0')%2 == 0 }

	r := router.New(router.Constrain("even", even))
	r.HandleFunc("GET /n/{v:even}", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, router.PathValue(req, "v"), "is even")
	})
	r.HandleFunc("GET /n/{v}", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, router.PathValue(req, "v"), "is not")
	})

	for _, path := range []string{"/n/4", "/n/7"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		fmt.Print(w.Body.String())
	}

	// Output:
	// 4 is even
	// 7 is not
}

// Routes is the table as data, which is what prints a route list and what a
// generator reads.
func ExampleRouter_Routes() {
	r := router.New()
	r.Handle("GET /posts", http.NotFoundHandler()).Name("posts.index")
	r.Handle("POST /posts", http.NotFoundHandler()).Name("posts.store")
	r.Handle("GET /files/{path...}", http.NotFoundHandler())

	for _, rt := range r.Routes() {
		fmt.Printf("%-6s %-20s %-12s %v\n", rt.Method, rt.Path, rt.Name, rt.Params)
	}

	// Output:
	// GET    /posts               posts.index  []
	// POST   /posts               posts.store  []
	// GET    /files/{path...}                  [path]
}
