// Package app sits above both leaves and reaches the standard library only
// through them.
package app

import (
	"mizu.test/graph/store"
	"mizu.test/graph/web"
)

// Serve builds the handler for a record.
func Serve(id, body string) any {
	return web.Handler(store.Record{ID: id, Body: body})
}
