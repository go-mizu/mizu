package micro

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/go-mizu/mizu/router"
)

func init() {
	register("router/match", benchRouterMatch)
	register("router/miss", benchRouterMiss)
}

// benchRouterMatch is the one thing every request pays for, so it is the number
// the rest of the HTTP path is measured against.
//
// The path has two wildcards in it and lands three segments down, which is what
// a route in an application looks like once it stops being an example. The
// table is four hundred routes, which is a medium application, and it matters
// because the cost of a match is the cost of walking past everything that did
// not match.
func benchRouterMatch(b *testing.B) {
	r := benchRoutes()

	b.ReportAllocs()
	for b.Loop() {
		_, params, ok := r.Lookup("GET", "", "/users/42/edit")
		if !ok || params.Len() != 2 {
			b.Fatal("the benchmark path stopped matching")
		}
	}
}

// benchRouterMiss is the same table and a path that matches nothing.
//
// It is here because a miss is not free and is not the same shape as a match. A
// miss walks the literal children down four levels, takes the constrained
// wildcard, and comes back up when the segment after it has nowhere to go, so
// this is the cost of the backtracking rather than the cost of a walk.
//
// It is also the request an unauthenticated scanner sends, thousands at a time,
// which is the load nobody profiles.
func benchRouterMiss(b *testing.B) {
	r := benchRoutes()

	b.ReportAllocs()
	for b.Loop() {
		if _, _, ok := r.Lookup("GET", "", "/api/v1/thing42/7/extra"); ok {
			b.Fatal("the benchmark path started matching")
		}
	}
}

// benchRoutes builds a four hundred route table.
//
// Ninety nine resources with four routes each is the bulk of it, which is what
// a REST API turns into, and the four written out at the end are the shapes the
// benchmarks above ask about.
func benchRoutes() *router.Router {
	r := router.New()
	h := http.NotFoundHandler()

	for i := range 99 {
		n := strconv.Itoa(i)
		r.Handle("GET /api/v1/thing"+n, h)
		r.Handle("POST /api/v1/thing"+n, h)
		r.Handle("GET /api/v1/thing"+n+"/{id:int}", h)
		r.Handle("DELETE /api/v1/thing"+n+"/{id:int}", h)
	}

	r.Handle("GET /health", h)
	r.Handle("GET /users/{id}", h)
	r.Handle("GET /users/{id}/{action}", h)
	r.Handle("GET /files/{path...}", h)
	return r
}
