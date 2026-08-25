package web

import (
	"net/http"
	"slices"
)

// Chain wraps h in mw, outermost first.
//
//	web.Chain(routes, mw.RequestID(), mw.Logger(l), mw.Recover())
//
// RequestID runs first, then Logger, then Recover, then whatever routes
// dispatches to, and they unwind in the opposite order. That is the order the
// call site reads in, which is the point: a chain that runs backwards from the
// way it is written is a chain somebody gets wrong once a year.
//
// Chain with no middleware returns h.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// A Stack is middleware being collected, in the order it will run.
//
// It is [Chain] with two things added: a layer can have a name, and the names
// can be put in an order that does not depend on when they were added. The
// second is why the type exists. Some middleware has to run before other
// middleware or the result is wrong rather than merely different, and getting
// that wrong is quiet: a session started after the thing that reads it looks
// like a user who is not logged in.
//
//	var s web.Stack
//	s.Add("session", mw.Session(store))
//	s.Add("auth", mw.Auth(guard))
//	s.Add("csrf", mw.CSRF(cfg))
//	s.Priority("session", "auth", "csrf")
//
//	srv := s.Then(routes)
//
// The zero Stack is an empty one and is ready to use. A Stack is written once
// at startup and read on every request, so it is not safe to change one that is
// already serving.
type Stack struct {
	layers []layer
	order  []string
}

// A layer is one middleware in a [Stack], with the name [Stack.Priority] and
// [Stack.Without] refer to it by.
type layer struct {
	name string
	mw   Middleware
}

// Use adds middleware nothing needs to refer to later.
//
// It runs in the order given, after whatever is already in the stack.
func (s *Stack) Use(mw ...Middleware) *Stack {
	for _, m := range mw {
		s.layers = append(s.layers, layer{mw: m})
	}
	return s
}

// Add is [Stack.Use] with a name, so that [Stack.Priority] and [Stack.Without]
// have something to call it.
//
// Two layers may share a name. Without drops both and Priority keeps them
// together in the order they were added, which is what a group that adds a
// second authorization check wants and is one fewer error to explain than
// refusing it.
func (s *Stack) Add(name string, mw Middleware) *Stack {
	s.layers = append(s.layers, layer{name: name, mw: mw})
	return s
}

// Priority fixes the relative order of the named layers.
//
// The layers that have one of these names run in the order named here,
// whenever they were added, and everything else stays where it was put. A name
// no layer has is ignored, because the list is written next to the application
// and the middleware it names is registered per group.
//
// Concretely, with a stack of a, csrf, session, b and a priority of session
// then csrf, the second and third slots belong to the named layers, so what
// runs is a, session, csrf, b. The unnamed layers have not moved.
//
// Calling it twice replaces the order rather than adding to it.
func (s *Stack) Priority(names ...string) *Stack {
	s.order = slices.Clone(names)
	return s
}

// Without removes every layer with one of these names.
//
// It is how a route drops something the group it is in added, which is the
// common shape: one endpoint in an authenticated group that the login form
// posts to.
func (s *Stack) Without(names ...string) *Stack {
	s.layers = slices.DeleteFunc(s.layers, func(l layer) bool {
		return l.name != "" && slices.Contains(names, l.name)
	})
	return s
}

// Clone is a copy that can be changed without changing this one.
//
// A group inherits its parent's middleware and then adds its own, and this is
// what makes the second half not reach back into the first.
func (s *Stack) Clone() *Stack {
	if s == nil {
		return new(Stack)
	}
	return &Stack{layers: slices.Clone(s.layers), order: slices.Clone(s.order)}
}

// Len is how many layers the stack has.
func (s *Stack) Len() int {
	if s == nil {
		return 0
	}
	return len(s.layers)
}

// Then wraps h in the stack and hands back what serves the request.
//
// It applies [Stack.Priority] here rather than as layers are added, so the
// order can be declared before or after the middleware it names. Nothing is
// kept: calling it twice builds two chains, and changing the stack afterwards
// does not change one that was already built.
func (s *Stack) Then(h http.Handler) http.Handler {
	if s == nil {
		return h
	}
	ordered := s.sorted()
	for i := len(ordered) - 1; i >= 0; i-- {
		h = ordered[i].mw(h)
	}
	return h
}

// sorted is the layers in the order they will run.
//
// The slots holding a layer the priority list names are the only ones that
// move, and they are filled with those layers in priority order. Everything
// else is where it was, which is what makes the rule one sentence: naming a
// middleware in the priority list says what it runs before, not where in the
// stack it goes.
func (s *Stack) sorted() []layer {
	if len(s.order) == 0 {
		return s.layers
	}

	var slots []int
	for i, l := range s.layers {
		if l.name != "" && slices.Contains(s.order, l.name) {
			slots = append(slots, i)
		}
	}
	if len(slots) < 2 {
		return s.layers
	}

	picked := make([]layer, len(slots))
	for i, at := range slots {
		picked[i] = s.layers[at]
	}
	slices.SortStableFunc(picked, func(a, b layer) int {
		return slices.Index(s.order, a.name) - slices.Index(s.order, b.name)
	})

	out := slices.Clone(s.layers)
	for i, at := range slots {
		out[at] = picked[i]
	}
	return out
}
