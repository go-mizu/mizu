package diag

import "strings"

// A Code is the permanent name of one kind of diagnostic, written MZ followed
// by four digits.
//
// MZ1042 means the same thing in every version of mizu there will ever be.
// Codes are allocated from a registry, never reused, and a retired one stays
// documented saying what it used to mean. Somebody who learned a code, or an
// agent that learned one, should not have to learn it again.
//
// The empty Code is allowed and means the diagnostic has none. That is what a
// producer starts with and it is a thing to fix rather than a thing to ship,
// since a diagnostic with no code has nowhere to point the reader who wants
// the reason.
type Code string

// Valid reports whether c is MZ followed by four digits.
//
// The empty Code is not valid, and callers that allow one check for it first.
func (c Code) Valid() bool {
	if len(c) != 6 || !strings.HasPrefix(string(c), "MZ") {
		return false
	}
	for i := 2; i < 6; i++ {
		if c[i] < '0' || c[i] > '9' {
			return false
		}
	}
	return true
}

// Number is the four digit part of c, or zero when c is not valid.
//
// It is what the subsystem ranges are stated against: MZ1xxx is configuration,
// MZ2xxx is the database, and so on.
func (c Code) Number() int {
	if !c.Valid() {
		return 0
	}
	n := 0
	for i := 2; i < 6; i++ {
		n = n*10 + int(c[i]-'0')
	}
	return n
}

// Explain is the command that prints the long form, or empty when c has no
// code to explain.
func (c Code) Explain() string {
	if !c.Valid() {
		return ""
	}
	return "mizu explain " + string(c)
}

// Docs is the page for c, or empty when c is not valid.
//
// The URL is short and it is built out of the code rather than stored, so a
// diagnostic cannot point at a page for a different one.
func (c Code) Docs() string {
	if !c.Valid() {
		return ""
	}
	return "https://mizu.dev/e/" + string(c)
}
