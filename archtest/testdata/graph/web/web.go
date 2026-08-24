// Package web reaches the standard library and one sibling.
package web

import (
	"net/http"

	"mizu.test/graph/store"
)

// Handler serves one record.
func Handler(r store.Record) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		b, err := store.Encode(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(b)
	})
}
