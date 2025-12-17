package live

// Page defines a live page with typed state.
// Implement this interface to create interactive server-rendered pages.
type Page[T any] interface {
	// Mount initializes the session state when the live connection is established.
	// Called once per session after WebSocket connects.
	Mount(ctx *Ctx, s *Session[T]) error

	// Render returns the view configuration for rendering.
	// Called after Mount and after each state change.
	Render(ctx *Ctx, s *Session[T]) (View, error)

	// Handle processes client-originated events (clicks, form submits, etc.).
	Handle(ctx *Ctx, s *Session[T], e Event) error

	// Info processes server-originated messages (pubsub, timers).
	Info(ctx *Ctx, s *Session[T], msg any) error
}

// View specifies what to render.
type View struct {
	// Page is the main page template name (e.g., "counter/index").
	Page string

	// Regions maps region IDs to partial template names.
	// Only dirty regions are re-rendered on updates.
	Regions map[string]string

	// Layout overrides the default layout.
	// Empty string uses page's layout or engine default.
	Layout string
}

// pageWrapper wraps a typed Page[T] to work with the untyped session handler.
type pageWrapper[T any] struct {
	page Page[T]
}

func (w *pageWrapper[T]) mount(ctx *Ctx, s *sessionBase) error {
	return w.page.Mount(ctx, s.typed().(*Session[T]))
}

func (w *pageWrapper[T]) render(ctx *Ctx, s *sessionBase) (View, error) {
	return w.page.Render(ctx, s.typed().(*Session[T]))
}

func (w *pageWrapper[T]) handle(ctx *Ctx, s *sessionBase, e Event) error {
	return w.page.Handle(ctx, s.typed().(*Session[T]), e)
}

func (w *pageWrapper[T]) info(ctx *Ctx, s *sessionBase, msg any) error {
	return w.page.Info(ctx, s.typed().(*Session[T]), msg)
}

func (w *pageWrapper[T]) newSession(id string) *sessionBase {
	sess := &Session[T]{
		ID:      id,
		dirty:   newDirtySet(),
		regions: make(map[string]string),
	}
	return &sessionBase{session: sess}
}

// pageHandler is the interface used internally by the session handler.
type pageHandler interface {
	mount(ctx *Ctx, s *sessionBase) error
	render(ctx *Ctx, s *sessionBase) (View, error)
	handle(ctx *Ctx, s *sessionBase, e Event) error
	info(ctx *Ctx, s *sessionBase, msg any) error
	newSession(id string) *sessionBase
}

// Wrap wraps a typed Page[T] for registration with RegisterPage.
// This is used to register pages for WebSocket connections.
func Wrap[T any](page Page[T]) pageHandler {
	return &pageWrapper[T]{page: page}
}
