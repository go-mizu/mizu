package diagtest

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/go-mizu/mizu/errs/diag"
)

// Check holds l to the parts of doc 36 section 2.1 a machine can hold it to.
//
// It runs inside [Run] on every corpus entry, and it is exported for a test
// that produces a diagnostic without a corpus directory around it.
//
// It is deliberately not a style checker. Whether a message names the thing
// that is wrong, whether it says what was expected as well as what was found,
// and whether the fix it offers is the right one are the parts a person reads
// the golden file for. What is here is the part that has an answer: the shape
// of the message, the words that mean nothing wherever they appear, and the
// fields that promise something the reader can act on and have to keep it.
func Check(tb testing.TB, l diag.List) {
	tb.Helper()
	for i, d := range l {
		checkOne(tb, i, d)
	}
}

func checkOne(tb testing.TB, i int, d diag.Diagnostic) {
	tb.Helper()

	where := func(format string, a ...any) {
		tb.Helper()
		tb.Errorf("diagnostic %d: "+format, append([]any{i}, a...)...)
	}

	switch {
	case d.Message == "":
		where("has no message")
	case strings.ContainsAny(d.Message, "\n\r"):
		// The first line is the whole message. A second one is a detail, which
		// has a field, or a fix, which has a field, or noise, which does not.
		where("message runs to more than one line: %q", d.Message)
	case strings.HasSuffix(d.Message, ".") || strings.HasSuffix(d.Message, "!"):
		// The convention the compiler and the go command use, and the one Go
		// error strings use, because a message is quoted inside other messages
		// as often as it is printed on its own.
		where("message ends in punctuation: %q", d.Message)
	case startsWrong(d.Message):
		where("message starts with a capital letter: %q", d.Message)
	}

	if s := empty(d.Message); s != "" {
		where("message says %q, which tells the reader nothing: %q", s, d.Message)
	}
	if strings.Contains(d.Message, "\x1b") || strings.Contains(d.Detail, "\x1b") {
		// Colour is decided by the renderer, which knows whether anything is
		// watching. A message carrying its own escapes prints them into a log.
		where("message carries terminal escapes")
	}

	if d.Detail != "" && d.Detail == d.Message {
		// The label under the carets says what was expected where the message
		// said what was found. Saying it twice is rule 5 going the other way.
		where("detail repeats the message: %q", d.Message)
	}

	if d.Code != "" {
		switch _, known := diag.Lookup(d.Code); {
		case !d.Code.Valid():
			where("code %q is not a code", d.Code)
		case !known:
			// The explain line and the docs URL are computed from the code, so
			// an unregistered one sends the reader somewhere that is not there.
			where("code %s is not in the registry, so its explain line points at nothing", d.Code)
		}
	}

	if d.Range.IsValid() && d.File == "" {
		where("has a line and column but no file, so there is nowhere to look")
	}

	for j, s := range d.Suggestions {
		if s.Message == "" {
			// A suggestion with no message prints nothing at all, so an edit
			// under it is applied by a program and invisible to a person.
			where("suggestion %d has no message", j)
		}
		for k, e := range s.Edits {
			switch {
			case e.File == "":
				where("suggestion %d edit %d names no file", j, k)
			case !e.Range.IsValid():
				where("suggestion %d edit %d has no range, so there is nothing to replace", j, k)
			}
		}
	}
}

// startsWrong reports whether a message begins with a capital that is not part
// of a name.
//
// TOML, APP_KEY and Config.Database all start a message fine. A sentence does
// not, because a message is read in the middle of other text as often as it is
// read on its own.
func startsWrong(msg string) bool {
	r, _ := utf8.DecodeRuneInString(msg)
	if !unicode.IsUpper(r) {
		return false
	}
	word, _, _ := strings.Cut(msg, " ")
	for _, r := range word[utf8.RuneLen(r):] {
		if unicode.IsUpper(r) || unicode.IsDigit(r) || r == '_' || r == '.' || r == '-' {
			return false
		}
	}
	return true
}

// vague are the phrases that fill the space where the answer goes.
//
// Each one is a message that could have been printed by any program about any
// problem, which is the property that makes it useless. They are checked as
// substrings and in lower case, so the wording around them does not matter.
var vague = []string{
	"something went wrong",
	"an error occurred",
	"unknown error",
	"unexpected error",
	"check your",
	"try again",
	"invalid input",
	"failed to",
	"unable to",
	"please ",
}

// empty returns the first phrase from vague that msg contains, or "".
func empty(msg string) string {
	lower := strings.ToLower(msg)
	for _, v := range vague {
		if strings.Contains(lower, v) {
			return v
		}
	}
	return ""
}
