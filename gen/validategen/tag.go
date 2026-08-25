package validategen

import (
	"errors"
	"fmt"
	"strings"
)

// A tagRule is one rule as a validate tag spelled it, before anything has
// looked at what the rule means.
//
// This file is the tag reader from the validate package, copied rather than
// shared because that one is unexported and the two have to cut a tag the same
// way for the generated validator to be the tag the struct is carrying.
type tagRule struct {
	name   string
	params []string
}

// parseTag splits a validate tag into the rules it lists.
//
// Rules are separated by commas, and a rule's parameters follow an equals sign
// and are separated by spaces:
//
//	validate:"required,min=3,max=200"
//	validate:"max=5,dive,email"
//
// A backslash escapes the character after it, which is how a parameter that
// carries a comma, an equals sign, a space or a backslash is written. An empty
// piece is nothing at all, so a tag may be spaced out to be read and a trailing
// comma is not a rule.
func parseTag(tag string) ([]tagRule, error) {
	pieces, err := lex(tag)
	if err != nil {
		return nil, err
	}

	var (
		rules []tagRule
		param bool // whether the next piece is a parameter rather than a name
	)
	for _, p := range pieces {
		switch {
		case p.text == "":
			// Space between rules, and the empty piece a trailing comma
			// leaves behind.
		case param:
			rules[len(rules)-1].params = append(rules[len(rules)-1].params, p.text)
		default:
			rules = append(rules, tagRule{name: p.text})
		}

		switch p.sep {
		case '=':
			switch {
			case len(rules) == 0:
				return nil, errors.New("an equals sign with no rule name in front of it")
			case param:
				return nil, fmt.Errorf("%s has a second equals sign in it", rules[len(rules)-1].name)
			}
			param = true
			// An empty non-nil slice is how the check below tells a rule that
			// was given no parameters from one that was written without an
			// equals sign at all.
			rules[len(rules)-1].params = []string{}
		case ',':
			param = false
		}
	}

	for _, r := range rules {
		if r.params != nil && len(r.params) == 0 {
			return nil, fmt.Errorf("%s= has nothing after it", r.name)
		}
	}
	return rules, nil
}

// A piece is one run of text from a tag and the separator that ended it, which
// is zero at the end of the tag.
type piece struct {
	text string
	sep  byte
}

// lex cuts a tag at every separator that is not escaped, and takes the
// backslashes out of the text it hands back.
func lex(tag string) ([]piece, error) {
	var (
		pieces []piece
		b      strings.Builder
		esc    bool
	)
	for i := range len(tag) {
		c := tag[i]
		switch {
		case esc:
			b.WriteByte(c)
			esc = false
		case c == '\\':
			esc = true
		case c == ',' || c == '=' || c == ' ':
			pieces = append(pieces, piece{text: b.String(), sep: c})
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if esc {
		return nil, errors.New("the tag ends in a backslash")
	}
	return append(pieces, piece{text: b.String()}), nil
}
