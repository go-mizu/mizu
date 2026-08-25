package web

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// trace is middleware that writes a mark on the way in and the same mark
// upper-cased on the way out, so one string says both what ran and in what
// order it unwound.
func trace(log *strings.Builder, mark string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.WriteString(mark)
			next.ServeHTTP(w, r)
			log.WriteString(strings.ToUpper(mark))
		})
	}
}

// handler is the end of a chain, which writes a mark of its own so a chain that
// never reaches it is told apart from one that reaches it in the wrong order.
func handler(log *strings.Builder) http.Handler {
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) { log.WriteString("h") })
}

// passthrough is middleware that has a frame of its own and does nothing in it,
// which is what the mw/chain budget measures.
func passthrough(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) })
}

func TestChainRunsOutermostFirst(t *testing.T) {
	var log strings.Builder
	h := Chain(handler(&log), trace(&log, "a"), trace(&log, "b"), trace(&log, "c"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	if got, want := log.String(), "abchCBA"; got != want {
		t.Errorf("the chain ran %q, want %q", got, want)
	}
}

func TestChainWithNothingToAddIsTheHandler(t *testing.T) {
	h := http.NotFoundHandler()
	if Chain(h) == nil {
		t.Fatal("Chain with no middleware returned nil")
	}

	var log strings.Builder
	Chain(handler(&log)).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if log.String() != "h" {
		t.Errorf("Chain with no middleware ran %q, want the handler alone", log.String())
	}
}

func TestStackRunsInTheOrderItWasBuilt(t *testing.T) {
	var log strings.Builder
	var s Stack
	s.Use(trace(&log, "a"))
	s.Add("named", trace(&log, "b"))
	s.Use(trace(&log, "c"), trace(&log, "d"))

	s.Then(handler(&log)).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got, want := log.String(), "abcdhDCBA"; got != want {
		t.Errorf("the stack ran %q, want %q", got, want)
	}
	if s.Len() != 4 {
		t.Errorf("the stack is %d layers, want 4", s.Len())
	}
}

// TestPriorityMovesOnlyTheSlotsItNames is the rule the Priority comment
// states, one row per way of getting it wrong.
func TestPriorityMovesOnlyTheSlotsItNames(t *testing.T) {
	for _, tc := range []struct {
		name  string
		add   []string // an upper-case mark is added unnamed
		order []string
		want  string
	}{
		{
			name:  "the order it is already in",
			add:   []string{"session", "auth", "csrf"},
			order: []string{"session", "auth", "csrf"},
			want:  "session auth csrf",
		},
		{
			name:  "backwards, which is the case the type exists for",
			add:   []string{"csrf", "auth", "session"},
			order: []string{"session", "auth", "csrf"},
			want:  "session auth csrf",
		},
		{
			name:  "unnamed layers stay where they were put",
			add:   []string{"A", "csrf", "session", "B"},
			order: []string{"session", "csrf"},
			want:  "A session csrf B",
		},
		{
			name:  "a name nothing has is ignored",
			add:   []string{"csrf", "session"},
			order: []string{"session", "locale", "csrf"},
			want:  "session csrf",
		},
		{
			name:  "a layer the order does not name stays put",
			add:   []string{"csrf", "locale", "session"},
			order: []string{"session", "csrf"},
			want:  "session locale csrf",
		},
		{
			name:  "two layers with one name keep the order they were added",
			add:   []string{"auth", "can", "can", "session"},
			order: []string{"session", "auth", "can"},
			want:  "session auth can can",
		},
		{
			name:  "one named layer has nothing to swap with",
			add:   []string{"A", "session", "B"},
			order: []string{"session", "csrf"},
			want:  "A session B",
		},
		{
			name:  "no order at all",
			add:   []string{"csrf", "session"},
			order: nil,
			want:  "csrf session",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var log strings.Builder
			var s Stack
			for _, mark := range tc.add {
				m := note(&log, mark)
				if strings.ToUpper(mark) == mark {
					s.Use(m)
				} else {
					s.Add(mark, m)
				}
			}
			s.Priority(tc.order...)

			s.Then(http.NotFoundHandler()).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
			if got := strings.TrimSpace(log.String()); got != tc.want {
				t.Errorf("the stack ran %q, want %q", got, tc.want)
			}
		})
	}
}

// note is middleware that writes its mark on the way in and nothing on the way
// out, for a test that is reading the order rather than the nesting.
func note(log *strings.Builder, mark string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.WriteString(mark + " ")
			next.ServeHTTP(w, r)
		})
	}
}

func TestPriorityReplacesTheOrderRatherThanAddingToIt(t *testing.T) {
	var log strings.Builder
	var s Stack
	s.Add("a", note(&log, "a")).Add("b", note(&log, "b"))
	s.Priority("b", "a")
	s.Priority("a", "b")

	s.Then(http.NotFoundHandler()).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got, want := strings.TrimSpace(log.String()), "a b"; got != want {
		t.Errorf("the stack ran %q, want %q, so the second Priority did not replace the first", got, want)
	}
}

func TestWithoutDropsByName(t *testing.T) {
	var log strings.Builder
	var s Stack
	s.Use(note(&log, "unnamed"))
	s.Add("auth", note(&log, "auth"))
	s.Add("csrf", note(&log, "csrf"))
	s.Add("auth", note(&log, "auth-again"))

	s.Without("auth", "nothing-has-this")

	s.Then(http.NotFoundHandler()).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got, want := strings.TrimSpace(log.String()), "unnamed csrf"; got != want {
		t.Errorf("the stack ran %q, want %q", got, want)
	}
}

func TestWithoutAnEmptyNameLeavesTheUnnamedLayers(t *testing.T) {
	var log strings.Builder
	var s Stack
	s.Use(note(&log, "unnamed"))

	s.Without("")

	s.Then(http.NotFoundHandler()).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got, want := strings.TrimSpace(log.String()), "unnamed"; got != want {
		t.Errorf("the stack ran %q, want %q, so an empty name matched a layer that has no name", got, want)
	}
}

func TestCloneDoesNotReachBackIntoItsParent(t *testing.T) {
	var log strings.Builder
	parent := new(Stack)
	parent.Add("a", note(&log, "a"))
	parent.Priority("a", "b")

	child := parent.Clone()
	child.Add("b", note(&log, "b"))
	child.Priority("b", "a")
	child.Without("a")

	parent.Then(http.NotFoundHandler()).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got, want := strings.TrimSpace(log.String()), "a"; got != want {
		t.Errorf("the parent ran %q, want %q, so the child changed it", got, want)
	}

	log.Reset()
	child.Then(http.NotFoundHandler()).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got, want := strings.TrimSpace(log.String()), "b"; got != want {
		t.Errorf("the child ran %q, want %q", got, want)
	}
}

func TestANilStackServesTheHandlerItIsGiven(t *testing.T) {
	var s *Stack
	if s.Len() != 0 {
		t.Errorf("a nil stack is %d layers", s.Len())
	}
	if s.Clone().Len() != 0 {
		t.Error("cloning a nil stack gave something with layers in it")
	}

	var log strings.Builder
	s.Then(handler(&log)).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if log.String() != "h" {
		t.Errorf("a nil stack ran %q, want the handler alone", log.String())
	}
}

func TestThenKeepsNothing(t *testing.T) {
	var log strings.Builder
	var s Stack
	s.Add("a", note(&log, "a"))

	built := s.Then(http.NotFoundHandler())
	s.Add("b", note(&log, "b"))

	built.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got, want := strings.TrimSpace(log.String()), "a"; got != want {
		t.Errorf("a chain that was already built ran %q, want %q", got, want)
	}
}

// TestTheStackDoesNotAllocatePerRequest is the budget row mw/chain, checked
// rather than measured. A layer is built once, at startup, and a request pays
// for the calls and nothing else.
func TestTheStackDoesNotAllocatePerRequest(t *testing.T) {
	var s Stack
	for i := range 8 {
		s.Add(string(rune('a'+i)), passthrough)
	}
	s.Priority("h", "g", "f", "e", "d", "c", "b", "a")

	h := s.Then(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	w, r := httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil)
	if got := testing.AllocsPerRun(1000, func() { h.ServeHTTP(w, r) }); got != 0 {
		t.Errorf("a request through eight middleware allocates %v times, want none", got)
	}
}

// TestSortedIsStableForOneNamedLayer keeps the shortcut in sorted honest: with
// fewer than two slots to fill there is nothing to reorder, and the answer has
// to be the stack it was given rather than a copy of it.
func TestSortedIsStableForOneNamedLayer(t *testing.T) {
	var s Stack
	s.Use(passthrough)
	s.Add("a", passthrough)
	s.Priority("a")

	if !slices.Equal(names(s.sorted()), names(s.layers)) {
		t.Errorf("sorted gave %v, want %v", names(s.sorted()), names(s.layers))
	}
}

func names(layers []layer) []string {
	out := make([]string, len(layers))
	for i, l := range layers {
		out[i] = l.name
	}
	return out
}
