//go:build !race && !mizudebug

package web

import "github.com/go-mizu/mizu/conc"

// This is the check the package comment describes, in the build where it is
// not made. Every method on Ctx calls live, and here live does nothing, so the
// inliner removes the call and the generation counter is a field nobody reads.
//
// The guarded half is in guard_race.go, and the two files are the whole of the
// difference between the builds. The pool is here rather than beside H because
// it is one of the halves: the guarded build does not have one.

// pool is where a Ctx comes from between requests.
var pool = conc.Pool(func() *Ctx { return new(Ctx) })

// live reports that the request is still running, which it does not check.
func (c *Ctx) live(string) {}

// acquire takes a Ctx from the pool.
func acquire() *Ctx {
	c := pool.Get()
	c.gen++
	return c
}

// release puts one back.
func release(c *Ctx) {
	c.reset()
	c.gen = 0
	pool.Put(c)
}
