Below is a concrete adoption plan for `mizu/live` and what the developer experience should look like. I will show: package shape, the user-facing API, file conventions, how events bind in templates, how state is managed, and what “real-time updates” look like in day-to-day code.

## Design goals for Mizu Live DX

* Familiar to Go developers (interfaces, structs, methods, no magic metaprogramming)
* LiveView-like semantics (persistent session, sequential events, server push)
* Minimal client JS (a small runtime that sends events and applies patches)
* Works with `mizu/view` templates and partials
* Opt-in, route-by-route

---

## 1) Package layout

* `mizu/view` stays SSR template engine
* `mizu/live` adds stateful sessions + websocket + patch protocol
* Optional helpers:

  * `mizu/live/html` for template helpers and attributes
  * `mizu/live/pubsub` for in-memory broker

Suggested tree:

```
mizu/
  live/
    live.go          # public API, Router mounting
    session.go       # Session, Assigns, Dirty
    event.go         # Event decoding
    patch.go         # Patch encoding
    ws.go            # websocket transport
    runtime.js       # tiny client (served as asset)
    pubsub.go        # Broker interface + inmem
```

---

## 2) Developer-facing API

### The main interface: `live.Page[T]`

Use generics so the page state is typed.

```go
package live

type Page[T any] interface {
	// Mount is called once when the live session is established.
	Mount(ctx *Ctx, s *Session[T]) error

	// Render returns a page template and optional region templates.
	// Regions are rendered individually when dirty.
	Render(ctx *Ctx, s *Session[T]) (View, error)

	// Handle handles client-originated events.
	Handle(ctx *Ctx, s *Session[T], e Event) error

	// Info handles server-originated messages (pubsub, timers).
	Info(ctx *Ctx, s *Session[T], msg any) error
}
```

Supporting types:

```go
type View struct {
	Page    string            // e.g. "users/show"
	Regions map[string]string // region id -> partial template name
	Layout  string            // optional override
}

type Session[T any] struct {
	ID     string
	State  T
	Flash  Flash
	Dirty  DirtySet
	UserID string // optional, from auth middleware
}

func (s *Session[T]) Mark(ids ...string)
func (s *Session[T]) ReplaceState(next T)
```

Event:

```go
type Event struct {
	Name   string
	Target string            // optional component id
	Values map[string]string // data-lv-value-*
	Form   map[string][]string // optional form fields
}
```

### Mounting live pages

DX should look like mounting a normal handler:

```go
eng := view.Must(view.New(...))

lv := live.New(live.Options{
	View:   eng,
	PubSub: live.NewInmemPubSub(),
	Dev:    true,
})

app := mizu.New()
lv.Mount(app) // mounts /_live websocket + runtime assets

app.GET("/counter", lv.Page("/counter", &CounterPage{}))
```

`lv.Page(path, page)` returns a `mizu.Handler`.

---

## 3) Template DX: attributes instead of DSL

Avoid new template syntax. Use HTML `data-*` attributes.

### Events

* Click: `data-lv-click="inc"`
* Submit: `data-lv-submit="save"`
* Change: `data-lv-change="validate"`
* Values: `data-lv-value-by="1"`
* Target component: `data-lv-target="cid:cart"`

Example:

```html
<div id="lv-root" data-lv>
  <button data-lv-click="inc" data-lv-value-by="1">+</button>
  <button data-lv-click="dec" data-lv-value-by="1">-</button>

  <div id="count">{{.State.Count}}</div>
</div>
```

### Regions

Define live regions by stable IDs:

```html
<div id="lv-root" data-lv>
  <div id="stats">{{partial "counter/stats" .}}</div>
  <div id="log">{{partial "counter/log" .}}</div>
</div>
```

When the server marks `stats` dirty, it re-renders only `counter/stats` and sends a patch for `#stats`.

---

## 4) What a Live page looks like in Go

### Example: Counter (hello world)

State:

```go
type CounterState struct {
	Count int
	Log   []string
}
```

Page:

```go
type CounterPage struct{}

func (p *CounterPage) Mount(ctx *live.Ctx, s *live.Session[CounterState]) error {
	s.State = CounterState{Count: 0}
	s.Mark("lv-root") // initial render
	return nil
}

func (p *CounterPage) Render(ctx *live.Ctx, s *live.Session[CounterState]) (live.View, error) {
	return live.View{
		Page: "counter/index",
		Regions: map[string]string{
			"stats": "counter/stats",
			"log":   "counter/log",
		},
	}, nil
}

func (p *CounterPage) Handle(ctx *live.Ctx, s *live.Session[CounterState], e live.Event) error {
	switch e.Name {
	case "inc":
		s.State.Count++
		s.State.Log = append(s.State.Log, "inc")
		s.Mark("stats", "log")
	case "dec":
		s.State.Count--
		s.State.Log = append(s.State.Log, "dec")
		s.Mark("stats", "log")
	}
	return nil
}

func (p *CounterPage) Info(ctx *live.Ctx, s *live.Session[CounterState], msg any) error {
	return nil
}
```

Templates:

* `views/pages/counter/index.html` (full page)
* `views/partials/counter/stats.html` (region)
* `views/partials/counter/log.html` (region)

---

## 5) Real-time server push DX (pubsub)

Example: activity feed updates when a background job publishes an event.

### Publish

```go
lv.PubSub().Publish("user:123", ActivityEvent{Text: "New message"})
```

### Subscribe in Mount

```go
func (p *FeedPage) Mount(ctx *live.Ctx, s *live.Session[FeedState]) error {
	ctx.Subscribe("user:" + s.UserID)
	return nil
}

func (p *FeedPage) Info(ctx *live.Ctx, s *live.Session[FeedState], msg any) error {
	ev, ok := msg.(ActivityEvent)
	if !ok { return nil }
	s.State.Items = append([]string{ev.Text}, s.State.Items...)
	s.Mark("feed")
	return nil
}
```

Now the server pushes patches without any client polling.

---

## 6) Progressive enhancement DX

When JS is disabled or WS fails:

* the same route serves a normal SSR page
* form submits become normal POSTs
* links navigate normally

How:

* Initial HTTP GET always renders full HTML via `view.Engine`.
* The live runtime connects after load.
* If no connection, user still sees a working page.

This is a major selling point vs SPA.

---

## 7) How developers will build apps day-to-day

### Typical patterns

* Pages define structure and regions.
* Regions are partial templates.
* Go page code mutates typed state and marks dirty regions.
* Components are optional; start with regions.

### Common helpers to make DX excellent

Provide helpers so users do not hand-write data attributes:

```go
// In templates:
<button {{lv.Click "inc" (lv.V("by", 1))}}>+</button>
<form {{lv.Submit "save"}}>...</form>
```

Where `lv.Click` returns `template.HTMLAttr` such as:
`data-lv-click="inc" data-lv-value-by="1"`.

This keeps templates clean without inventing directives.

---

## 8) “Hello Live” user experience

A new Mizu app should have:

1. `mizu new --template live` scaffolds:

* a sample live page
* runtime JS file embedded/served
* make targets

2. Code looks like:

```go
app := mizu.New()
eng := view.Must(view.New(...))
lv := live.New(live.Options{View: eng, Dev: true})

lv.Mount(app)

app.GET("/counter", lv.Page("/counter", &CounterPage{}))
app.Run(":3000")
```

That is the full adoption story.

---

## 9) Why this DX is better than copying Livewire

* No per-event state serialization
* No hydration bugs
* Sequential server loop avoids data races
* Server push is natural
* Typed Go state is reliable

---

If you want, I can also show:

* the minimal client runtime responsibilities (event capture, patch apply, reconnect)
* the patch wire protocol
* how to integrate with CSRF/auth from `mizu` middleware cleanly
