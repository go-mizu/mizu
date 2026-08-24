package str

import (
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Camel returns s in camelCase.
//
//	str.Camel("user_id")       // userId
//	str.Camel("HTTP server")   // httpServer
//
// The string is cut into words first, on punctuation, on spaces, and on the
// case changes inside a name that already runs its words together. See
// [Headline] for what counts as a word boundary.
func Camel(s string) string {
	words := splitName(s)
	if len(words) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(strings.ToLower(words[0]))
	for _, w := range words[1:] {
		b.WriteString(capitalize(w))
	}
	return b.String()
}

// Pascal returns s in PascalCase, which is [Camel] with the first word
// capitalised too.
//
//	str.Pascal("user_id")   // UserId
func Pascal(s string) string {
	var b strings.Builder
	for _, w := range splitName(s) {
		b.WriteString(capitalize(w))
	}
	return b.String()
}

// Snake returns s in snake_case.
//
//	str.Snake("userID")      // user_id
//	str.Snake("HTTPServer")  // http_server
func Snake(s string) string { return joinLower(splitName(s), "_") }

// Kebab returns s in kebab-case, which is [Snake] with hyphens.
//
//	str.Kebab("userID")   // user-id
func Kebab(s string) string { return joinLower(splitName(s), "-") }

// Headline returns s as words separated by spaces, each one capitalised.
//
//	str.Headline("steve_jobs")           // Steve Jobs
//	str.Headline("email_notification")   // Email Notification
//
// A word ends at anything that is not a letter or a digit, at a change from
// lower case to upper case, and at the last capital of a run of them when a
// lower case letter follows. That last rule is what keeps HTTPServer from
// becoming H T T P Server and turns it into HTTP Server, or Http Server once
// each word is capitalised.
//
// Those rules are all the information a string carries, so two acronyms in a
// row cannot be told apart: XMLHTTPRequest gives XMLHTTP and Request, and IDs
// gives I and Ds. Telling those apart needs a list of acronyms, and a list that
// is right for one codebase is wrong for the next one.
//
// [Title] is the one to use on a sentence, since it leaves the words where they
// are rather than cutting the string up.
func Headline(s string) string {
	words := splitName(s)
	for i, w := range words {
		words[i] = capitalize(w)
	}
	return strings.Join(words, " ")
}

// titler is built once because [cases.Title] does real work to set itself up.
// It is safe for concurrent use, which the cases package documents.
var titler = cases.Title(language.English)

// Title returns s with the first letter of every word in upper case, using the
// Unicode casing rules through [cases.Title] rather than the byte tricks that
// go wrong outside English.
//
//	str.Title("a nice day")   // A Nice Day
//
// This keeps the words and the spacing of s. [Headline] is the one that cuts a
// name into words first.
func Title(s string) string { return titler.String(s) }

// Sentence returns s with the first letter of each sentence in upper case.
//
//	str.Sentence("hello world. goodbye.")   // Hello world. Goodbye.
//
// A sentence starts at the beginning of the string and after a full stop, a
// question mark or an exclamation mark. Everything else is left alone, so a
// proper noun in the middle of a sentence keeps its capital, which is the
// difference from lower casing the string and capitalising the front of it.
func Sentence(s string) string {
	out := []rune(s)
	start := true

	for i, r := range out {
		switch {
		case start && unicode.IsLetter(r):
			out[i] = unicode.ToUpper(r)
			start = false
		case r == '.' || r == '?' || r == '!':
			start = true
		case unicode.IsSpace(r):
			// Space neither starts a sentence nor ends one, so the flag
			// carries across the gap after a full stop.
		default:
			start = false
		}
	}
	return string(out)
}

// UpperFirst returns s with its first letter in upper case and the rest of it
// untouched.
//
//	str.UpperFirst("hello world")   // Hello world
//
// The spec called this Ucfirst, which is a name from PHP. It counts in grapheme
// clusters, so a letter written as a base plus a combining accent is upper
// cased whole.
func UpperFirst(s string) string { return mapFirst(s, unicode.ToUpper) }

// LowerFirst returns s with its first letter in lower case and the rest of it
// untouched.
//
//	str.LowerFirst("Hello World")   // hello World
func LowerFirst(s string) string { return mapFirst(s, unicode.ToLower) }

// SwapCase returns s with every upper case letter lowered and every lower case
// letter raised.
//
//	str.SwapCase("Hello World")   // hELLO wORLD
//
// A letter with no case, which is most of the world's letters, comes back as it
// was.
func SwapCase(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsUpper(r):
			return unicode.ToLower(r)
		case unicode.IsLower(r):
			return unicode.ToUpper(r)
		}
		return r
	}, s)
}

// mapFirst applies fn to the first cluster of s and leaves the rest alone.
func mapFirst(s string, fn func(rune) rune) string {
	if s == "" {
		return s
	}

	n := clusterLen(s)
	// The cluster is mapped whole rather than only its first rune, so that a
	// base letter carries its accents through the change of case.
	return strings.Map(fn, s[:n]) + s[n:]
}

// capitalize returns w with its first letter in upper case and the rest in
// lower case, which is what every one of the name cases wants for the words
// after the first.
func capitalize(w string) string {
	n := clusterLen(w)
	return strings.ToUpper(w[:n]) + strings.ToLower(w[n:])
}

// joinLower lowers every word and puts sep between them.
func joinLower(words []string, sep string) string {
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, sep)
}

// splitName cuts s into the words a name is made of. The rules are in the doc
// comment on [Headline], which is the exported function that shows them off.
func splitName(s string) []string {
	var words []string
	var word []rune

	flush := func() {
		if len(word) > 0 {
			words = append(words, string(word))
			word = word[:0]
		}
	}

	runes := []rune(s)
	for i, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}

		if len(word) > 0 && unicode.IsUpper(r) {
			prev := word[len(word)-1]
			// A capital after anything that is not a capital starts a word, so
			// userID gives user and ID.
			if !unicode.IsUpper(prev) {
				flush()
			} else if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				// A capital inside a run of capitals starts a word only when a
				// small letter comes next, so HTTPServer gives HTTP and Server
				// rather than HTTPS and erver.
				flush()
			}
		}

		word = append(word, r)
	}

	flush()
	return words
}
