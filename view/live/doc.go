// Package live provides real-time, stateful, server-rendered pages for Mizu.
//
// Inspired by Phoenix LiveView, it enables rich interactive experiences without
// writing client-side JavaScript, while maintaining Go's type safety.
//
// Key concepts:
//   - Server-authoritative state: All state lives on the server
//   - Sequential event processing: Events processed one at a time per session
//   - Progressive enhancement: Pages work without JS, upgrade with WebSocket
//   - Type-safe sessions: Generic Session[T] keeps state strongly typed
//
// Basic usage:
//
//	// Define state
//	type CounterState struct {
//	    Count int
//	}
//
//	// Implement Page interface
//	type CounterPage struct{}
//
//	func (p *CounterPage) Mount(ctx *live.Ctx, s *live.Session[CounterState]) error {
//	    s.State = CounterState{Count: 0}
//	    return nil
//	}
//
//	func (p *CounterPage) Render(ctx *live.Ctx, s *live.Session[CounterState]) (live.View, error) {
//	    return live.View{Page: "counter/index"}, nil
//	}
//
//	func (p *CounterPage) Handle(ctx *live.Ctx, s *live.Session[CounterState], e live.Event) error {
//	    switch e.Name {
//	    case "inc":
//	        s.State.Count++
//	    case "dec":
//	        s.State.Count--
//	    }
//	    s.MarkAll()
//	    return nil
//	}
//
//	func (p *CounterPage) Info(ctx *live.Ctx, s *live.Session[CounterState], msg any) error {
//	    return nil
//	}
//
//	// Mount and run
//	eng := view.Must(view.New(view.Options{Dir: "views"}))
//	lv := live.New(live.Options{View: eng})
//	app := mizu.New()
//	lv.Mount(app)
//	app.Get("/counter", lv.Handle(&CounterPage{}))
//	app.Listen(":3000")
package live
