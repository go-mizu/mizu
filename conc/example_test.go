package conc_test

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/go-mizu/mizu/conc"
	"github.com/go-mizu/mizu/errs"
)

func Example() {
	prices := map[string]int{"apple": 3, "pear": 5, "plum": 2}
	names := []string{"apple", "pear", "plum"}
	found := make([]int, len(names))

	g, _ := conc.NewGroup(context.Background())
	for i, name := range names {
		// Each goroutine writes to its own slot, so the results keep the order
		// of the input and nothing needs a lock.
		g.Go(func(context.Context) error {
			p, ok := prices[name]
			if !ok {
				return errs.NotFoundf("no price for %s", name)
			}
			found[i] = p
			return nil
		})
	}

	fmt.Println(g.Wait(), found)

	// Output:
	// <nil> [3 5 2]
}

// ExampleNewGroup shows what the group's context is for. The goroutine that was
// still running can say what went wrong elsewhere, rather than reporting that
// it was cancelled and leaving somebody to work out why.
func ExampleNewGroup() {
	g, _ := conc.NewGroup(context.Background())

	g.Go(func(ctx context.Context) error {
		<-ctx.Done()
		fmt.Println("gave up because:", context.Cause(ctx))
		return nil
	})
	g.Go(func(context.Context) error {
		return errs.New(errs.Unavailable, "pricing.down", "the pricing service is down")
	})

	fmt.Println("Wait returned:", g.Wait())

	// Output:
	// gave up because: the pricing service is down
	// Wait returned: the pricing service is down
}

// ExampleLimit uses a limit of one, which makes the output worth printing: each
// goroutine waits for a slot, and the only slot belongs to the one before it.
func ExampleLimit() {
	g, _ := conc.NewGroup(context.Background(), conc.Limit(1))

	var mu sync.Mutex
	var order []int
	for i := range 4 {
		g.Go(func(context.Context) error {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, i)
			return nil
		})
	}

	fmt.Println(g.Wait(), order)

	// Output:
	// <nil> [0 1 2 3]
}

// ExampleGo is the shape inside a request, where the group was created by
// something further out and the handler only wants a goroutine.
func ExampleGo() {
	// A middleware does this once and puts ctx on the request.
	_, ctx := conc.NewGroup(context.Background())

	// A handler, somewhere well below it.
	conc.Go(ctx, func(ctx context.Context) error {
		fmt.Println("recording the view")
		return nil
	})

	// The same middleware, on the way back out.
	fmt.Println("Wait returned:", conc.Wait(ctx))

	// Output:
	// recording the view
	// Wait returned: <nil>
}

// ExampleGroup_Wait shows a panic arriving as an ordinary error. It has the
// stack from the line that panicked, so it reads like any other failure and the
// process stays up.
func ExampleGroup_Wait() {
	g, _ := conc.NewGroup(context.Background())
	g.Go(func(context.Context) error {
		var prices map[string]int
		prices["apple"] = 3 // A nil map, and a panic.
		return nil
	})

	err := g.Wait()
	fmt.Println(errs.KindOf(err), errs.CodeOf(err))
	fmt.Println(err)

	// Output:
	// internal panic
	// a goroutine panicked: assignment to entry in nil map
}

// ExampleMap is the group above with the loop and the slot bookkeeping taken
// out, which is what most fan-outs turn out to be.
func ExampleMap() {
	prices := map[string]int{"apple": 3, "pear": 5, "plum": 2}
	names := []string{"apple", "pear", "plum"}

	found, err := conc.Map(context.Background(), names, 4, func(_ context.Context, name string) (int, error) {
		p, ok := prices[name]
		if !ok {
			return 0, errs.NotFoundf("no price for %s", name)
		}
		return p, nil
	})

	fmt.Println(err, found)

	// Output:
	// <nil> [3 5 2]
}

// ExampleEach is [conc.Map] for work that has an effect rather than a result.
// The limit is what keeps a list of any length from opening a connection per
// element, so the results arrive in whatever order they finish.
func ExampleEach() {
	users := []string{"ana", "ben", "cleo"}

	var mu sync.Mutex
	var sent []string
	err := conc.Each(context.Background(), users, 2, func(_ context.Context, u string) error {
		mu.Lock()
		defer mu.Unlock()
		sent = append(sent, u)
		return nil
	})

	slices.Sort(sent)
	fmt.Println(err, sent)

	// Output:
	// <nil> [ana ben cleo]
}

// ExampleAll runs a fixed list of different jobs rather than one job over a
// list, so there is no limit to pass. The limit is how many were written.
func ExampleAll() {
	page, err := conc.All(context.Background(),
		func(context.Context) (string, error) { return "header", nil },
		func(context.Context) (string, error) { return "body", nil },
		func(context.Context) (string, error) { return "footer", nil },
	)

	fmt.Println(err, page)

	// Output:
	// <nil> [header body footer]
}

// ExampleMapSeq hands each result over as it is ready rather than collecting
// them all first, so a sequence of ten million costs what a sequence of ten
// does. The order still follows the input.
func ExampleMapSeq() {
	words := slices.Values([]string{"mizu", "kaze", "hi"})

	for n, err := range conc.MapSeq(context.Background(), words, 4, func(_ context.Context, w string) (int, error) {
		return len(w), nil
	}) {
		fmt.Println(n, err)
	}

	// Output:
	// 4 <nil>
	// 4 <nil>
	// 2 <nil>
}

// ExampleRace tries two ways of getting the same answer and takes whichever
// works. The one that lost is told why it was cancelled, so it can tell that
// apart from the caller giving up.
func ExampleRace() {
	fromCache := func(context.Context) (string, error) {
		return "", errs.NotFoundf("not in the cache")
	}
	fromOrigin := func(context.Context) (string, error) {
		return "the front page", nil
	}
	fromMirror := func(ctx context.Context) (string, error) {
		<-ctx.Done()
		fmt.Println("the mirror stopped because:", context.Cause(ctx))
		return "", ctx.Err()
	}

	body, err := conc.Race(context.Background(), fromCache, fromOrigin, fromMirror)
	fmt.Println(body, err)

	// Output:
	// the mirror stopped because: conc: something else finished first
	// the front page <nil>
}

// ExampleOnce works out the answer the first time somebody asks for it and
// hands the same one to everybody after that, including when the answer is a
// failure.
func ExampleOnce() {
	calls := 0
	region := conc.Once(func() (string, error) {
		calls++
		return "eu-west-1", nil
	})

	for range 3 {
		v, err := region()
		fmt.Println(v, err)
	}
	fmt.Println("fn ran", calls, "time")

	// Output:
	// eu-west-1 <nil>
	// eu-west-1 <nil>
	// eu-west-1 <nil>
	// fn ran 1 time
}

// ExampleDebounce coalesces a burst of calls into one run, fifty milliseconds
// after the last of them. The channel is the example waiting for a run it can
// print. Real code does not wait, which is the point of debouncing.
func ExampleDebounce() {
	done := make(chan struct{})
	reindex := conc.Debounce(50*time.Millisecond, func() {
		fmt.Println("rebuilt the index")
		close(done)
	})

	for range 100 {
		reindex()
	}
	<-done

	// Output:
	// rebuilt the index
}

// ExampleThrottle shows the shape rather than the timing, since a window of an
// hour makes the answer the same every run. The first call goes through and the
// rest of that window does not.
func ExampleThrottle() {
	runs := 0
	report := conc.Throttle(time.Hour, func() { runs++ })

	for range 100 {
		report()
	}
	fmt.Println("100 calls, ", runs, "run")

	// Output:
	// 100 calls,  1 run
}

// ExamplePool is the reuse pattern in full. Reset then Put, in that order and
// in the same defer, so a buffer never goes back with somebody else's bytes in
// it.
func ExamplePool() {
	buffers := conc.Pool(func() *bytes.Buffer { return new(bytes.Buffer) })

	greet := func(name string) string {
		b := buffers.Get()
		defer func() {
			b.Reset()
			buffers.Put(b)
		}()

		b.WriteString("hello, ")
		b.WriteString(name)
		return b.String()
	}

	fmt.Println(greet("mizu"))
	fmt.Println(greet("kaze"))

	// Output:
	// hello, mizu
	// hello, kaze
}
