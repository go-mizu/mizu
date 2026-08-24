package conc_test

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/go-mizu/mizu/conc"
)

func TestPoolBuildsWhenItIsEmpty(t *testing.T) {
	var built atomic.Int64
	p := conc.Pool(func() *bytes.Buffer {
		built.Add(1)
		return new(bytes.Buffer)
	})

	b := p.Get()
	if b == nil {
		t.Fatal("Get returned nil")
	}
	if n := built.Load(); n != 1 {
		t.Errorf("an empty pool built %d values, want 1", n)
	}
}

// TestPoolReusesWhatComesBack is the whole point, and the loop is because a
// pool is free to drop anything it is handed. It happens at a collection, and
// under the race detector it happens to one Put in four on purpose, so that
// nobody writes a test that treats a pool as a container. This asks a few times
// and wants one of them back.
func TestPoolReusesWhatComesBack(t *testing.T) {
	p := conc.Pool(func() *bytes.Buffer { return new(bytes.Buffer) })

	for range 20 {
		b := p.Get()
		p.Put(b)
		if p.Get() == b {
			return
		}
	}
	t.Error("nothing came back out of the pool in twenty tries")
}

// TestPoolKeepsWhatWasPutIn says out loud that resetting is the caller's job. A
// pool that emptied buffers on the way in would be a different thing with a
// different name.
func TestPoolKeepsWhatWasPutIn(t *testing.T) {
	p := conc.Pool(func() *bytes.Buffer { return new(bytes.Buffer) })

	for range 20 {
		b := p.Get()
		b.Reset()
		b.WriteString("left over")
		p.Put(b)

		if got := p.Get(); got == b {
			if s := got.String(); s != "left over" {
				t.Errorf("the value came back as %q, want it untouched", s)
			}
			return
		}
	}
	t.Error("nothing came back out of the pool in twenty tries")
}

func TestPoolFromManyGoroutines(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := conc.Pool(func() *bytes.Buffer { return new(bytes.Buffer) })

		g, _ := conc.NewGroup(t.Context(), conc.Limit(8))
		for i := range 100 {
			g.Go(func(context.Context) error {
				b := p.Get()
				defer func() {
					b.Reset()
					p.Put(b)
				}()

				b.WriteString("hello")
				if b.Len() != 5 {
					t.Errorf("goroutine %d got a buffer holding %q", i, b)
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPoolWithoutAFunction(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a pool with nothing to build with was accepted")
		}
	}()
	conc.Pool[*bytes.Buffer](nil)
}

// TestPoolHoldsOneType is the type assertion that is not there any more, said
// as a test rather than as a comment. A value of another type cannot be put in,
// so Get has nothing to check for.
func TestPoolHoldsOneType(t *testing.T) {
	p := conc.Pool(func() int { return 42 })
	p.Put(1)
	if got := p.Get(); got != 1 && got != 42 {
		t.Errorf("Get returned %d, want a value this pool could hold", got)
	}
}
