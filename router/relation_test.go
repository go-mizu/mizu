package router

import "testing"

func TestConflicts(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
		why  string
	}{
		// The same pattern written twice, and the same pattern written two ways.
		{"/a", "/a", true, "the same pattern"},
		{"GET /a", "GET /a", true, "the same pattern with a method"},
		{"/a/", "/a/{rest...}", true, "a trailing slash is a nameless trailing wildcard"},
		{"/a/{x}", "/a/{y}", true, "the name of a wildcard is not part of what it matches"},

		// One is more specific than the other, so the more specific one wins.
		{"/a/b", "/a/{x}", false, "a literal beats a wildcard"},
		{"/a/{x}", "/a/{rest...}", false, "one segment beats the rest of the path"},
		{"GET /a", "/a", false, "a method beats no method"},
		{"GET /a", "HEAD /a", false, "GET answers HEAD, so GET is the wider"},
		{"example.com/a", "/a", false, "a host beats no host"},
		{"/a/{$}", "/a/{rest...}", false, "the end of the path is more specific than the rest of it"},

		// Nothing they both match.
		{"/a", "/b", false, "different literals"},
		{"/a", "/a/b", false, "different lengths, neither takes the rest"},
		{"GET /a", "POST /a", false, "different methods"},
		{"example.com/a", "example.org/a", false, "different hosts"},
		{"/a/{$}", "/a/{x}", false, "a single wildcard does not match the end of a path"},

		// Each matches something the other does not.
		{"/a/{x}/c", "/a/b/{y}", true, "wider in one segment and narrower in the next"},
		{"/posts/{id}/edit", "/{kind}/latest/edit", true, "the pair from the package comment"},
		{"GET /a/{x}", "/a/b", true, "one answers more methods, the other matches fewer paths"},
		{"GET /a/{rest...}", "/a/b/c", true, "the same, with a trailing wildcard"},
		{"/a/{rest...}", "GET /a/b/c", false, "no method and the rest of the path is wider on both counts"},

		// A constraint tells them apart, which is the one rule mizu adds.
		{"/a/{x:int}", "/a/{y}", false, "a constraint against no constraint"},
		{"/a/{x:int}", "/a/{y:uuid}", false, "two different constraints"},
		{"/a/{x:int}", "/a/{y:int}", true, "the same constraint is no help"},
		{"/a/{x:int}/c", "/a/b/{y}", false, "int turns down the literal b"},
		{"/a/{x:int}/c", "/a/7/{y}", true, "and accepts the literal 7"},
		{"/a/{x:int}", "/a/{y:int}/b", false, "different lengths, so nothing they both match"},
		// A trailing wildcard carries no constraint, so it never tells anything
		// apart.
		{"/a/{x:int}", "/a/{rest...}", false, "the single wildcard is the more specific either way"},
	}

	for _, c := range cases {
		t.Run(c.a+" vs "+c.b, func(t *testing.T) {
			a, b := mustParse(t, c.a), mustParse(t, c.b)
			if got := conflicts(a, b); got != c.want {
				t.Errorf("conflicts(%q, %q) = %v, want %v: %s", c.a, c.b, got, c.want, c.why)
			}
			// The answer cannot depend on which of the two was registered
			// first, or a route table would mean two things.
			if got := conflicts(b, a); got != c.want {
				t.Errorf("conflicts(%q, %q) = %v, want %v: %s", c.b, c.a, got, c.want, c.why)
			}
		})
	}
}

func TestPathsRelation(t *testing.T) {
	cases := []struct {
		a, b string
		want relation
	}{
		{"/a", "/a", same},
		{"/{x}", "/{y}", same},
		{"/a/", "/a/{r...}", same},
		{"/{x}", "/a", wider},
		{"/a", "/{x}", narrower},
		{"/a/", "/a/b", wider},
		{"/a/b", "/a/", narrower},
		{"/a/", "/a/b/c/d", wider},
		{"/a", "/b", apart},
		{"/a", "/a/b", apart},
		{"/a/{x}/c", "/a/b/{y}", overlapping},
		{"/{x}/b", "/a/{y}", overlapping},
		{"/{$}", "/{x}", apart},
		{"/a/{$}", "/a/", narrower},
		{"/a/b/{r...}", "/a", apart},
		{"/x/{p}/{q}/{s}", "/{m}/y/{r...}", overlapping},
	}
	for _, c := range cases {
		a, b := mustParse(t, c.a), mustParse(t, c.b)
		if got := paths(a, b); got != c.want {
			t.Errorf("paths(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
		if got := paths(b, a); got != inverse(c.want) {
			t.Errorf("paths(%q, %q) = %v, want %v", c.b, c.a, got, inverse(c.want))
		}
	}
}

func TestMethodsRelation(t *testing.T) {
	cases := []struct {
		a, b string
		want relation
	}{
		{"GET /", "GET /", same},
		{"/", "/", same},
		{"/", "GET /", wider},
		{"GET /", "/", narrower},
		{"GET /", "HEAD /", wider},
		{"HEAD /", "GET /", narrower},
		{"GET /", "POST /", apart},
		{"POST /", "HEAD /", apart},
	}
	for _, c := range cases {
		a, b := mustParse(t, c.a), mustParse(t, c.b)
		if got := methods(a, b); got != c.want {
			t.Errorf("methods(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// combine folds one part of a pattern into another, and the fold has to give the
// same answer whichever way round the two parts come.
func TestCombineIsSymmetric(t *testing.T) {
	all := []relation{same, wider, narrower, apart, overlapping}
	for _, a := range all {
		for _, b := range all {
			if combine(a, b) != combine(b, a) {
				t.Errorf("combine(%v, %v) = %v and combine(%v, %v) = %v",
					a, b, combine(a, b), b, a, combine(b, a))
			}
			// Turning both parts around turns the answer around with them.
			if got, want := combine(inverse(a), inverse(b)), inverse(combine(a, b)); got != want {
				t.Errorf("combine(inverse(%v), inverse(%v)) = %v, want %v", a, b, got, want)
			}
		}
	}
}

func TestSharedPath(t *testing.T) {
	cases := []struct{ a, b, want string }{
		{"/posts/{id}/edit", "/{kind}/latest/edit", "/posts/latest/edit"},
		{"/a/{x}/c", "/a/b/{y}", "/a/b/c"},
		{"GET /a/{x}", "/a/b", "/a/b"},
		{"/a/{rest...}", "GET /a/b/c", "/a/b/c"},
		{"/a/{x}", "/a/{y}", "/a/x"},
		{"/a/", "/a/{rest...}", "/a/"},
		{"/x/{p}/{q}/{s}", "/{m}/y/{r...}", "/x/y/q/s"},
	}
	for _, c := range cases {
		a, b := mustParse(t, c.a), mustParse(t, c.b)
		if got := shared(a, b); got != c.want {
			t.Errorf("shared(%q, %q) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
}

func mustParse(t *testing.T, s string) *pattern {
	t.Helper()
	p, err := parse(s, builtin)
	if err != nil {
		t.Fatalf("parse(%q): %v", s, err)
	}
	return p
}
