// Command router matches requests against a route table and reads back the
// route that matched, which is the thing net/http.ServeMux does not hand over.
package main

import (
	"fmt"
	"net/http"

	"github.com/go-mizu/mizu/router"
)

func main() {
	r := router.New()
	r.HandleFunc("GET /posts/{id:int}", show).Name("posts.show")
	r.HandleFunc("GET /posts/{slug:slug}", show).Name("posts.bySlug")

	for _, path := range []string{"/posts/7", "/posts/hello-world", "/posts/Nope"} {
		if rt, params, ok := r.Lookup("GET", "", path); ok {
			fmt.Println(path, rt.Info().Name, params.At(0).Value)
		} else {
			fmt.Println(path, "matched nothing")
		}
	}
}

func show(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintln(w, "post", router.PathValue(req, "id"))
}
