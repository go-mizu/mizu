package validategen

import (
	"slices"
	"testing"
)

// This file is the tag reader copied from the validate package, and the whole
// value of the copy is that the two cut a tag the same way. A tag that means one
// thing to validate.Struct and another to the generator would make the marker a
// change to what the API answers, which is the one thing the generator must not
// be.
//
// The corpus in testdata reaches the tags a request struct carries. What is here
// is the punctuation, which is easier to be exhaustive about from a table.

func TestParseTag(t *testing.T) {
	for _, c := range []struct {
		tag  string
		want []tagRule
	}{
		{"", nil},
		{"required", []tagRule{{name: "required"}}},
		{"required,email", []tagRule{{name: "required"}, {name: "email"}}},
		{"min=3", []tagRule{{name: "min", params: []string{"3"}}}},
		{"between=3 10", []tagRule{{name: "between", params: []string{"3", "10"}}}},
		{"required, min=3", []tagRule{{name: "required"}, {name: "min", params: []string{"3"}}}},
		{"required,", []tagRule{{name: "required"}}},
		{",required", []tagRule{{name: "required"}}},
		{"max=5,dive,required", []tagRule{{name: "max", params: []string{"5"}}, {name: "dive"}, {name: "required"}}},

		// A backslash escapes the character behind it, which is how a parameter
		// carrying one of the separators is written.
		{`eq=a\,b`, []tagRule{{name: "eq", params: []string{"a,b"}}}},
		{`eq=a\=b`, []tagRule{{name: "eq", params: []string{"a=b"}}}},
		{`eq=a\ b`, []tagRule{{name: "eq", params: []string{"a b"}}}},
		{`eq=a\\b`, []tagRule{{name: "eq", params: []string{`a\b`}}}},
	} {
		got, err := parseTag(c.tag)
		if err != nil {
			t.Errorf("parseTag(%q): %v", c.tag, err)
			continue
		}
		if !slices.EqualFunc(got, c.want, sameRule) {
			t.Errorf("parseTag(%q) = %v, want %v", c.tag, got, c.want)
		}
	}
}

func TestParseTagRefuses(t *testing.T) {
	for _, c := range []struct{ tag, want string }{
		{"=3", "an equals sign with no rule name in front of it"},
		{"min=3=4", "min has a second equals sign in it"},
		{"min=", "min= has nothing after it"},
		{`min=3\`, "the tag ends in a backslash"},
	} {
		_, err := parseTag(c.tag)
		if err == nil {
			t.Errorf("parseTag(%q) was read as a tag", c.tag)
			continue
		}
		if err.Error() != c.want {
			t.Errorf("parseTag(%q) said %q, want %q", c.tag, err, c.want)
		}
	}
}

func sameRule(a, b tagRule) bool {
	return a.name == b.name && slices.Equal(a.params, b.params)
}
