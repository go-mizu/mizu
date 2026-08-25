// Package web is the one that cannot be used on its own. Its constructor takes
// a store.Record, so anybody calling it has two packages to import rather than
// one.
package web

import (
	"net/http"

	"mizu.test/api/store"
)

// Handler is the plain case: one parameter from another package.
func Handler(r store.Record) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write(r.Body)
	})
}

// New takes nothing, so it is callable by a caller that imports web alone.
func New() *Server { return &Server{} }

// A Server is here for its methods.
type Server struct{ records []store.Record }

// Add takes a slice, which a caller cannot write without naming store.
func (s *Server) Add(rs []store.Record) { s.records = append(s.records, rs...) }

// Filter takes a function, and the same goes for its parameters.
func (s *Server) Filter(keep func(store.Record) bool) int {
	n := 0
	for _, r := range s.records {
		if keep(r) {
			n++
		}
	}
	return n
}

// drop is unexported, so what it takes is nobody's problem.
func (s *Server) drop(r store.Record) {}

// Index has the map and the pointer.
func Index(m map[string]*store.Record) int { return len(m) }

// Wrap stays inside the standard library.
func Wrap(next http.Handler) http.Handler { return next }
