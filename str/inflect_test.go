package str_test

import (
	"testing"

	"github.com/go-mizu/mizu/str"
)

// pairs is the word list both directions are checked against, so a rule that
// pluralises correctly and singularises wrongly cannot pass by having its own
// table. Everything here has to round trip.
var pairs = []struct{ one, many string }{
	// The plain rule, which is most words.
	{"user", "users"},
	{"post", "posts"},
	{"account", "accounts"},
	{"invoice", "invoices"},
	{"house", "houses"},
	{"day", "days"},
	{"key", "keys"},
	{"boy", "boys"},
	{"tie", "ties"},
	{"pie", "pies"},
	{"cry", "cries"},
	{"spy", "spies"},

	// Endings that take es rather than s.
	{"class", "classes"},
	{"glass", "glasses"},
	{"box", "boxes"},
	{"tax", "taxes"},
	{"buzz", "buzzes"},
	{"dish", "dishes"},
	{"batch", "batches"},
	{"match", "matches"},

	// A consonant before the y.
	{"city", "cities"},
	{"category", "categories"},
	{"company", "companies"},
	{"country", "countries"},
	{"entry", "entries"},

	// Old English.
	{"man", "men"},
	{"woman", "women"},
	{"child", "children"},
	{"person", "people"},
	{"tooth", "teeth"},
	{"foot", "feet"},
	{"goose", "geese"},
	{"mouse", "mice"},
	{"ox", "oxen"},

	// Latin and Greek endings English kept.
	{"cactus", "cacti"},
	{"focus", "foci"},
	{"radius", "radii"},
	{"datum", "data"},
	{"medium", "media"},
	{"criterion", "criteria"},
	{"phenomenon", "phenomena"},
	{"index", "indices"},
	{"matrix", "matrices"},
	{"vertex", "vertices"},
	{"analysis", "analyses"},
	{"thesis", "theses"},
	{"crisis", "crises"},
	{"axis", "axes"},

	// The f words.
	{"life", "lives"},
	{"knife", "knives"},
	{"wife", "wives"},
	{"leaf", "leaves"},
	{"half", "halves"},
	{"shelf", "shelves"},
	{"self", "selves"},
	{"thief", "thieves"},

	// Singular nouns that end in s, which the plain rule would take apart.
	{"bus", "buses"},
	{"gas", "gases"},
	{"status", "statuses"},
	{"virus", "viruses"},
	{"alias", "aliases"},
	{"lens", "lenses"},
	{"quiz", "quizzes"},

	// The o words that take es.
	{"potato", "potatoes"},
	{"tomato", "tomatoes"},
	{"hero", "heroes"},
	{"echo", "echoes"},
}

func TestPlural(t *testing.T) {
	for _, p := range pairs {
		if got := str.Plural(p.one); got != p.many {
			t.Errorf("Plural(%q) = %q, want %q", p.one, got, p.many)
		}
	}
}

func TestSingular(t *testing.T) {
	for _, p := range pairs {
		if got := str.Singular(p.many); got != p.one {
			t.Errorf("Singular(%q) = %q, want %q", p.many, got, p.one)
		}
	}
}

// TestInflectionIsIdempotent is the property that keeps a name safe to pass
// through twice, which happens whenever one generator feeds another.
func TestInflectionIsIdempotent(t *testing.T) {
	for _, p := range pairs {
		if got := str.Plural(p.many); got != p.many {
			t.Errorf("Plural of the plural %q = %q, want it left alone", p.many, got)
		}
		if got := str.Singular(p.one); got != p.one {
			t.Errorf("Singular of the singular %q = %q, want it left alone", p.one, got)
		}
	}
}

func TestUncountable(t *testing.T) {
	words := []string{"sheep", "fish", "series", "species", "equipment", "information", "news", "aircraft", "deer"}

	for _, w := range words {
		if got := str.Plural(w); got != w {
			t.Errorf("Plural(%q) = %q, want it unchanged", w, got)
		}
		if got := str.Singular(w); got != w {
			t.Errorf("Singular(%q) = %q, want it unchanged", w, got)
		}
	}
}

// TestInflectionKeepsTheCaseItWasGiven matters because the caller is usually a
// generator turning a type name into a table name, and the answer has to look
// like the question.
func TestInflectionKeepsTheCaseItWasGiven(t *testing.T) {
	cases := []struct{ in, plural, singular string }{
		{"user", "users", "user"},
		{"User", "Users", "User"},
		{"USER", "USERS", "USER"},
		{"person", "people", "person"},
		{"Person", "People", "Person"},
		{"PERSON", "PEOPLE", "PERSON"},
		{"City", "Cities", "City"},
		{"CITY", "CITIES", "CITY"},
		{"Box", "Boxes", "Box"},
		{"BOX", "BOXES", "BOX"},
	}

	for _, c := range cases {
		if got := str.Plural(c.in); got != c.plural {
			t.Errorf("Plural(%q) = %q, want %q", c.in, got, c.plural)
		}
		if got := str.Singular(c.in); got != c.singular {
			t.Errorf("Singular(%q) = %q, want %q", c.in, got, c.singular)
		}
	}
}

func TestPluralN(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"item", 0, "items"},
		{"item", 1, "item"},
		{"item", 2, "items"},
		{"items", 1, "item"},
		{"items", 5, "items"},
		{"person", 1, "person"},
		{"person", 3, "people"},
		{"people", 1, "person"},
		{"item", -1, "item"},
		{"item", -3, "items"},
	}

	for _, c := range cases {
		if got := str.PluralN(c.s, c.n); got != c.want {
			t.Errorf("PluralN(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
}

func TestInflectionOfNothing(t *testing.T) {
	if got := str.Plural(""); got != "" {
		t.Errorf("Plural of an empty string = %q", got)
	}
	if got := str.Singular(""); got != "" {
		t.Errorf("Singular of an empty string = %q", got)
	}
}

// TestTheLimitsAreWhereTheDocSaysTheyAre pins the two failures named in the doc
// comment. They are wrong answers, and the point of the test is that they stay
// the wrong answers the documentation warns about rather than quietly becoming
// different wrong answers.
func TestTheLimitsAreWhereTheDocSaysTheyAre(t *testing.T) {
	if got := str.Singular("chairmen"); got != "chairmen" {
		t.Errorf("Singular(chairmen) = %q, want it unchanged, which is what the doc promises", got)
	}
	if got := str.Singular("bases"); got != "basis" {
		t.Errorf("Singular(bases) = %q, want basis, which is the one the doc says it picks", got)
	}
}
