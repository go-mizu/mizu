package web

// A Responder is a value that knows how to send itself.
//
//	type postView struct{ post Post }
//
//	func (v postView) Respond(c *web.Ctx) error {
//		return web.JSON(c, v.post)
//	}
//
// The point of the interface is that a handler can decide what the response is
// without sending it. What comes back from the handler is a value a test can
// look at, and the writing happens after the handler has returned, in one place
// [R] owns rather than at every point a handler is finished.
//
// It is one method and nothing else on purpose. A type that satisfies it is
// whatever an application already had, with a method on it, so a view struct, a
// domain type or a wrapper around one can all be returned from a handler
// without either package knowing about the other.
type Responder interface {
	// Respond writes the response. It is called after the handler returned and
	// it is the last thing that happens to the request, so the rules about a
	// Ctx not outliving its handler apply here as they do everywhere else.
	Respond(c *Ctx) error
}

// R adapts a handler that returns its response to a [Handler].
//
//	func show(c *web.Ctx) (postView, error) {
//		p, err := posts.Find(c.Context(), c.Param("id"))
//		if err != nil {
//			return postView{}, err
//		}
//		return postView{p}, nil
//	}
//
//	r.Handle("GET /posts/{id}", web.H(web.R(show)))
//
// The shape it buys is a handler that is a function from a request to a value.
// A test calls show and reads the value, without a recorder, a status or a
// parse in between, and the code that turns the value into bytes is written
// once in Respond rather than at the end of every handler that has one.
//
// The type parameter is the reason show can name its own return type rather
// than [Responder]. It is inferred from the function, so nothing at the call
// site says it, and a handler that returns the wrong view type does not compile
// where an interface would have taken it.
//
// A handler that returns an error returns it, and the response is whatever
// [Errors] decides, the same as for any other handler. A handler that returns a
// nil Responder and no error has already written the response itself, which is
// what returning nil means for a [Handler] too. That last part only applies to
// a nil interface: a nil pointer of a concrete type is a value the handler
// chose, and its own method is called with it.
func R[T Responder](fn func(c *Ctx) (T, error)) Handler {
	return func(c *Ctx) error {
		v, err := fn(c)
		if err != nil {
			return err
		}
		if any(v) == nil {
			return nil
		}
		return v.Respond(c)
	}
}
