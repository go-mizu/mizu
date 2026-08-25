// Package deep keeps a *web.Ctx inside a slice, an array and a map, which is
// the same mistake as keeping one in a field and is worth reporting the same
// way.
package deep

import "github.com/go-mizu/mizu/web"

// A batch is the requests something means to answer together.
type batch struct {
	all   []*web.Ctx
	first [2]*web.Ctx
	byID  map[string]*web.Ctx
	name  string
}
