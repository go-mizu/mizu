// Package conc starts goroutines that somebody is waiting for.
//
//	g, ctx := conc.NewGroup(ctx, conc.Limit(8))
//	for _, id := range ids {
//		g.Go(func(ctx context.Context) error { return warm(ctx, id) })
//	}
//	if err := g.Wait(); err != nil {
//		return err
//	}
//
// A [Group] is a set of goroutines and the context they run under. The first
// one to fail cancels that context, so the rest stop instead of finishing work
// whose result is already going in the bin, and [Group.Wait] returns the error
// that started it.
//
// # Fanning out
//
// The loop above is common enough to have a name. [Map] runs one function over
// a slice, at most n at a time, and gives the results back in the order of the
// input:
//
//	prices, err := conc.Map(ctx, ids, 8, priceOf)
//
// [Each] is the same for work that has an effect rather than a result, and
// [All] runs a fixed list of different functions instead of one function over a
// list, so there is no limit to pass: the limit is how many were written. All
// three stop at the first error and return it on its own, because a half filled
// slice is a thing nobody can use safely.
//
// [Race] runs several ways of getting the same answer and takes whichever
// arrives first, cancelling the rest:
//
//	body, err := conc.Race(ctx, fromReplica, fromPrimary)
//
// [MapSeq] is [Map] over an [iter.Seq], handing each result over as it is ready
// rather than collecting them all first:
//
//	for row, err := range conc.MapSeq(ctx, rows, 8, enrich) {
//		if err != nil {
//			return err
//		}
//		w.Write(row)
//	}
//
// n elements are in flight at a time and nothing is read from the input ahead
// of that, so ten million rows cost what ten do. The results still come out in
// the order they went in, which means an element that takes a long time holds
// up the finished ones behind it. That is what an ordered answer costs, and the
// alternative is a sequence whose order has nothing to do with the input.
//
// An error from [MapSeq] arrives against the element it belongs to and does not
// end the sequence, since a sequence hands that decision to the caller one
// element at a time. Break to end it. Breaking out cancels everything in flight
// and waits for it before the loop is over, so nothing it started outlives it.
//
// # Doing it less often
//
// [Debounce] and [Throttle] wrap a function and hand back another one with the
// same shape, so the calling code does not change:
//
//	reindex := conc.Debounce(200*time.Millisecond, rebuildSearchIndex)
//	report := conc.Throttle(time.Second, publishProgress)
//
// Debounce waits for the calls to stop and then runs once, which is what work
// wants when only the latest version of it matters. Throttle runs the first
// call and drops the rest of that second, which is what work wants when the
// newest call is the one worth making. Neither queues anything: something that
// runs every call but spaces them out is a rate limiter, and it holds the
// caller back instead of letting it through.
//
// Throttle runs fn on the goroutine that called it, so a panic lands where the
// caller can see it. Debounce cannot, since running later is the whole point,
// which makes it the one thing here that starts something nobody is waiting
// for. Debounce work that is small and cannot fail, and hand the rest to a
// group from inside it.
//
// # Doing it once
//
// [Once] runs a function the first time somebody asks for the answer and hands
// the same one to everybody after that:
//
//	region := conc.Once(func() (string, error) { return metadata.Region(ctx) })
//
// Callers who arrive while the first one is still working wait for it. The
// error is part of the result, so a failure is remembered rather than letting
// the next caller try again, and something that should be attempted again is a
// retry rather than a Once.
//
// # Reuse
//
// [Pool] is [sync.Pool] with the type kept, which takes the assertion and the
// nil check off every call site:
//
//	buffers := conc.Pool(func() *bytes.Buffer { return new(bytes.Buffer) })
//
//	b := buffers.Get()
//	defer func() { b.Reset(); buffers.Put(b) }()
//
// Values come back in whatever state they were put back in, so resetting is the
// caller's job and belongs next to the Put it goes with. A pool is for reuse
// under churn and is not a cache: anything in it can disappear at the next
// collection.
//
// # No bare go statement
//
// The go statement has no return value, no owner and no end. Nothing collects
// the error, nothing notices the panic, and nothing waits for it, so a process
// shutting down can walk away in the middle of a write. mizu's own code
// contains no bare go statement and this package is the reason it does not need
// one.
//
// The package level [Go] and [Wait] are that rule at a distance. A group goes
// into the context once, near the top, and code far below starts a goroutine
// without having to be handed anything:
//
//	conc.Go(ctx, func(ctx context.Context) error {
//		return audit.Record(ctx, event)
//	})
//
// [Go] hands fn the context it was given rather than the group's own, so a user,
// a tenant and a trace picked up since the group was created are all still
// there. A context with no group in it panics, because a goroutine with nothing
// waiting for it is the failure this package exists to prevent and starting it
// anyway would hide that until something in production went quiet.
//
// # Panics
//
// A panic in a goroutine takes the process down with it, and no recover
// anywhere else can stop that. So every goroutine started here recovers, and
// the panic becomes the group's error:
//
//	err := g.Wait()
//	errs.KindOf(err)  // internal
//	errs.CodeOf(err)  // "panic"
//
// The stack is captured inside the deferred function, while the frames that
// panicked are still on the stack, so [github.com/go-mizu/mizu/errs.Stack]
// points at the line that panicked rather than at this package. A panic
// carrying an error keeps that error in the chain, so [errors.Is] still
// reaches it.
//
// This is containment and not forgiveness. A panic is a bug either way, and the
// only thing recovery buys is that one goroutine fails instead of every request
// in flight.
//
// # Limits
//
// [Limit] is how many goroutines run at once, and past it [Group.Go] blocks
// until a slot frees up:
//
//	g, ctx := conc.NewGroup(ctx, conc.Limit(8))
//
// A loop over ten thousand rows without one holds ten thousand goroutines and
// ten thousand connections, and the database finds out before the loop does.
// The limit is worth having when the work is expensive and costs a little when
// it is not: the semaphore is a channel send and receive per goroutine, which
// is measurable against a function that returns immediately and invisible
// against anything that touches a network.
//
// Waiting for a slot ends when the group does. A [Group.Go] parked on a full
// semaphore returns without starting fn once the group has been cancelled,
// rather than waiting for a slot that is not coming.
//
// # Testing
//
// Ordering questions belong to [testing/synctest], which this package is
// written against rather than around. Inside a bubble, synctest.Wait returns
// when every goroutine in the group is blocked, which turns "has it started yet"
// into a question with an answer:
//
//	synctest.Test(t, func(t *testing.T) {
//		g, _ := conc.NewGroup(t.Context(), conc.Limit(2))
//		for range 10 {
//			g.Go(hold)
//		}
//		synctest.Wait()
//		// Exactly two of them are running, every time.
//	})
//
// # Cost
//
// A group is about 75 nanoseconds and 240 bytes on an M4: a cancel context, the
// value that carries the group, and the group itself.
//
// One goroutine from start to finish, meaning [NewGroup], one [Group.Go] and
// one [Group.Wait], is about 390 nanoseconds against about 235 for the go
// statement and [sync.WaitGroup] written out by hand. The difference is around
// 150 nanoseconds and three allocations, and it buys the panic recovery, the
// error, and the cancellation of everything alongside it.
//
// Fanning out is about 230 nanoseconds and one allocation per goroutine, so the
// fixed cost stops mattering somewhere around the tenth one. A recovered panic
// is about 2.7 microseconds, nearly all of it capturing the stack, which is the
// right trade for something that should not be happening.
//
// The helpers add what their shape needs and nothing more. [Map] and [Each]
// cost two allocations per element against the one a bare [Group.Go] costs, the
// extra being the closure that carries the element. [MapSeq] costs four, since
// every element gets a channel of its own, which is what keeps the results in
// order without holding all of them at once, and it runs about half as long
// again per element as [Map] does. [All] over three functions is around 1.5
// microseconds and [Race] over three is around 2.5, the difference being that
// Race stops the losers and waits for them before it returns.
//
// The four that are not about goroutines are cheap enough to put anywhere. A
// [Debounce] call that runs nothing, which is most of them, is about 80
// nanoseconds and two allocations for the timer it replaces. A [Throttle] call
// that is turned away is about four nanoseconds and nothing. [Once] after the
// first call is about nine nanoseconds and nothing. [Pool] is a Get and a Put
// in about seven nanoseconds, which is what [sync.Pool] costs, since the type
// is the compiler's problem rather than the runtime's.
//
// Timings were taken on a machine with other work on it, so read them as
// ceilings. The allocation counts do not move.
//
// # What is not here yet
//
// Registration with the application, so that shutdown waits for a group that
// outlives the request it started in, needs the supervisor and arrives with it.
package conc
