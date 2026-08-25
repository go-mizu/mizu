//go:build !race && !mizudebug

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTheCtxComesFromThePool is the half of the pooling contract this build
// keeps: a Ctx is reused, and it arrives clean.
func TestTheCtxComesFromThePool(t *testing.T) {
	var first *Ctx
	serve(t, httptest.NewRequest("GET", "/first", nil), func(c *Ctx) error {
		first = c
		c.status = http.StatusTeapot
		return c.Text("one")
	})

	var second *Ctx
	serve(t, httptest.NewRequest("GET", "/second", nil), func(c *Ctx) error {
		second = c
		if c.status != 0 {
			t.Errorf("the status carried over from the last request as %d", c.status)
		}
		if c.res.Status() != 0 {
			t.Error("the write flag carried over from the last request")
		}
		return c.Text("two")
	})

	if first != second {
		t.Skip("the pool handed out a different Ctx, which it is allowed to do under load")
	}
}

// TestAcquireAndReleaseDoNotAllocate is the budget row ctx/acquire, checked
// rather than measured.
func TestAcquireAndReleaseDoNotAllocate(t *testing.T) {
	r := httptest.NewRequest("GET", "/things/7", nil)
	w := httptest.NewRecorder()
	got := testing.AllocsPerRun(1000, func() {
		c := acquire()
		c.record(w)
		c.r = r
		release(c)
	})
	if got != 0 {
		t.Errorf("a pool round trip allocates %v times, want none", got)
	}
}

// TestLiveIsFreeInThisBuild is the other half: the check costs nothing when it
// is not being made.
func TestLiveIsFreeInThisBuild(t *testing.T) {
	c := acquire()
	c.record(httptest.NewRecorder())
	c.r = httptest.NewRequest("GET", "/", nil)
	defer release(c)

	if got := testing.AllocsPerRun(1000, func() { c.live("Request") }); got != 0 {
		t.Errorf("live allocates %v times in a build that does not check", got)
	}
}

// TestAReleasedCtxDoesNotPanicInThisBuild says out loud that the check is off
// here, so a test that relies on it has to run under -race or -tags mizudebug.
func TestAReleasedCtxDoesNotPanicInThisBuild(t *testing.T) {
	var stale *Ctx
	serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		stale = c
		return nil
	})

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("a released Ctx panicked with %v in the build that does not check", r)
		}
	}()
	stale.live("Request")
}
