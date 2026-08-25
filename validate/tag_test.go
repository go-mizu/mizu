package validate

import (
	"slices"
	"strings"
	"testing"
)

func TestParseTag(t *testing.T) {
	cases := []struct {
		tag  string
		want []tagRule
	}{
		{"", nil},
		{"required", []tagRule{{name: "required"}}},
		{"required,email", []tagRule{{name: "required"}, {name: "email"}}},
		{"min=3", []tagRule{{name: "min", params: []string{"3"}}}},
		{"between=3 10", []tagRule{{name: "between", params: []string{"3", "10"}}}},
		{
			"required,min=3,max=200",
			[]tagRule{
				{name: "required"},
				{name: "min", params: []string{"3"}},
				{name: "max", params: []string{"200"}},
			},
		},
		{
			"max=5,dive,email",
			[]tagRule{
				{name: "max", params: []string{"5"}},
				{name: "dive"},
				{name: "email"},
			},
		},
		{
			"required_if=Publish true",
			[]tagRule{{name: "required_if", params: []string{"Publish", "true"}}},
		},

		// Space between rules is nothing, and so is an empty piece, which is
		// what a trailing or a doubled comma leaves behind.
		{"required, min=3", []tagRule{{name: "required"}, {name: "min", params: []string{"3"}}}},
		{"required,", []tagRule{{name: "required"}}},
		{",,required", []tagRule{{name: "required"}}},
		{"  required  ", []tagRule{{name: "required"}}},
		{"min=3 ", []tagRule{{name: "min", params: []string{"3"}}}},

		// A backslash is how a parameter carries a separator.
		{`oneof=a\,b c`, []tagRule{{name: "oneof", params: []string{"a,b", "c"}}}},
		{`starts=a\ b`, []tagRule{{name: "starts", params: []string{"a b"}}}},
		{`eq=x\=y`, []tagRule{{name: "eq", params: []string{"x=y"}}}},
		{`eq=c:\\tmp`, []tagRule{{name: "eq", params: []string{`c:\tmp`}}}},
	}

	for _, c := range cases {
		got, err := parseTag(c.tag)
		if err != nil {
			t.Errorf("parseTag(%q): %v", c.tag, err)
			continue
		}
		if !sameRules(got, c.want) {
			t.Errorf("parseTag(%q) = %v, want %v", c.tag, got, c.want)
		}
	}
}

// A tag is written by a programmer, so what is wrong with one is said out loud
// rather than guessed at.
func TestParseTagRefusesATagThatMakesNoSense(t *testing.T) {
	cases := []struct {
		tag  string
		says string
	}{
		{`required\`, "ends in a backslash"},
		{"=3", "no rule name in front of it"},
		{"min==3", "second equals sign"},
		{"min=3=4", "second equals sign"},
		{"min=", "has nothing after it"},
		{"min= ", "has nothing after it"},
		{"required,min=,max=9", "has nothing after it"},
	}

	for _, c := range cases {
		_, err := parseTag(c.tag)
		if err == nil {
			t.Errorf("parseTag(%q) came back with no error", c.tag)
			continue
		}
		if !strings.Contains(err.Error(), c.says) {
			t.Errorf("parseTag(%q) said %q, want it to mention %q", c.tag, err, c.says)
		}
	}
}

func TestTagSafe(t *testing.T) {
	for _, name := range []string{"vat", "credit_card", "required-ish", "e164"} {
		if !tagSafe(name) {
			t.Errorf("tagSafe(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"", "two words", "a,b", "a=b", `a\b`} {
		if tagSafe(name) {
			t.Errorf("tagSafe(%q) = true, want false", name)
		}
	}
}

func sameRules(a, b []tagRule) bool {
	return slices.EqualFunc(a, b, func(x, y tagRule) bool {
		return x.name == y.name && slices.Equal(x.params, y.params)
	})
}
