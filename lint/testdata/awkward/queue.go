// Package awkward is the shapes that are wrong through a type with no name of
// its own, or wrong in more than one way at once.
package awkward

import "github.com/go-mizu/mizu/web"

// A pending is what a worker was left to get through.
//
// The channel field is reported as a channel rather than as a field, because
// what is wrong with it is the sending and not the holding, and one report
// about one mistake is the whole of it. The counter is a pointer to something
// that is not a Ctx and is nobody's business here.
type pending struct {
	next  chan *web.Ctx
	count *int
}

// live is a channel of Ctx and a package level variable, and both are worth
// hearing about, since taking either half away leaves the other.
var live chan *web.Ctx
