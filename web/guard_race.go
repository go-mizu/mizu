//go:build race || mizudebug

package web

import (
	"fmt"
	"sync/atomic"
)

// The guarded build.
//
// A Ctx that outlives its handler is the one bug this package can have that is
// worse than a crash, because the symptom is one request reading another
// request's user. So in a build with the race detector on, or with -tags
// mizudebug, two things change.
//
// The pool is not used. Every request allocates a Ctx and no Ctx is ever handed
// out twice, so a stale pointer stays stale rather than becoming a pointer to
// somebody else's request. This is the part that matters: a generation counter
// on a pooled Ctx can only report the window between the release and the next
// acquire, and the window is where the harmless half of the bug lives.
//
// And every method checks the counter first. Release sets it to zero, so a
// stale pointer says which method was called and which route the Ctx used to
// belong to, and the process stops there.
//
// Both cost too much to leave on: an allocation per request, and a load and a
// branch per method call. Which is why they are here, in the build people run
// their tests under, rather than in the one they deploy.

// live panics when the request this Ctx belonged to has finished.
func (c *Ctx) live(method string) {
	if c.gen != 0 {
		return
	}
	panic(fmt.Sprintf("web.Ctx used after the request completed: Ctx.%s called on the Ctx for %s\n"+
		"a *web.Ctx does not outlive its handler, see the web package comment", method, c.was))
}

// stamped counts the Ctx values this build has made, which is what makes the
// generation of each one different from the last.
var stamped atomic.Uint64

// acquire makes a Ctx. There is no pool in this build.
func acquire() *Ctx {
	return &Ctx{gen: stamped.Add(1)}
}

// release retires a Ctx for good.
//
// It writes down what the Ctx was serving before it clears the fields, since
// that is the one thing a panic from a stale pointer can say that is any use to
// the person reading it.
func release(c *Ctx) {
	was := c.r.Method + " " + c.r.URL.Path
	if c.route != nil {
		was = c.route.Info().Pattern
	}
	c.reset()
	c.was = was
	c.gen = 0
}
