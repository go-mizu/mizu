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
// # What is not here yet
//
// The fan-out helpers that the spec puts in this package, Map, MapSeq, Each,
// Race and All, are built on a Group and land next. Registration with the
// application, so that shutdown waits for a group that outlives the request it
// started in, needs the supervisor and arrives with it.
package conc
