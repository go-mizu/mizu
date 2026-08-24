package str_test

import (
	"testing"

	"github.com/go-mizu/mizu/str"
)

func TestCamel(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"user", "user"},
		{"user_id", "userId"},
		{"user-id", "userId"},
		{"user id", "userId"},
		{"UserID", "userId"},
		{"HTTP server", "httpServer"},
		{"HTTPServer", "httpServer"},
		{"  spaced  out  ", "spacedOut"},
		{"already Camel", "alreadyCamel"},
		{"oauth2_token", "oauth2Token"},
	}

	for _, c := range cases {
		if got := str.Camel(c.in); got != c.want {
			t.Errorf("Camel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPascal(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"user_id", "UserId"},
		{"HTTPServer", "HttpServer"},
		{"a", "A"},
	}

	for _, c := range cases {
		if got := str.Pascal(c.in); got != c.want {
			t.Errorf("Pascal(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSnake(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"userID", "user_id"},
		{"UserID", "user_id"},
		{"HTTPServer", "http_server"},
		{"user id", "user_id"},
		{"user-id", "user_id"},
		{"user_id", "user_id"},
		{"OAuth2Token", "o_auth2_token"},
	}

	for _, c := range cases {
		if got := str.Snake(c.in); got != c.want {
			t.Errorf("Snake(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestKebab(t *testing.T) {
	if got := str.Kebab("userID"); got != "user-id" {
		t.Errorf("Kebab gave %q, want user-id", got)
	}
	if got := str.Kebab("HTTP Server"); got != "http-server" {
		t.Errorf("Kebab gave %q, want http-server", got)
	}
}

// TestSnakeAndKebabDifferOnlyInTheSeparator is worth pinning so the two cannot
// drift apart when one of them is changed.
func TestSnakeAndKebabDifferOnlyInTheSeparator(t *testing.T) {
	for _, in := range []string{"userID", "HTTP server", "a_b_c", "", "OneTwoThree"} {
		snake, kebab := str.Snake(in), str.Kebab(in)
		if len(snake) != len(kebab) {
			t.Errorf("Snake(%q) = %q and Kebab = %q, want the same shape", in, snake, kebab)
		}
	}
}

// TestTheAcronymRule is the one boundary rule that is hard to get right, so it
// gets a test of its own rather than a line in a table.
func TestTheAcronymRule(t *testing.T) {
	cases := []struct{ in, want string }{
		{"HTTPServer", "http_server"},
		{"userID", "user_id"},
		{"ID", "id"},
		{"IDs", "i_ds"},
		{"XMLHTTPRequest", "xmlhttp_request"},
		{"aB", "a_b"},
	}

	for _, c := range cases {
		if got := str.Snake(c.in); got != c.want {
			t.Errorf("Snake(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestHeadline(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"steve_jobs", "Steve Jobs"},
		{"email_notification_sent", "Email Notification Sent"},
		{"a", "A"},
		{"___", ""},
	}

	for _, c := range cases {
		if got := str.Headline(c.in); got != c.want {
			t.Errorf("Headline(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"a nice day", "A Nice Day"},
		{"ALL CAPS", "All Caps"},
	}

	for _, c := range cases {
		if got := str.Title(c.in); got != c.want {
			t.Errorf("Title(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestTitleKeepsTheSpacing is the difference from Headline, which cuts the
// string into words and rejoins them with single spaces.
func TestTitleKeepsTheSpacing(t *testing.T) {
	if got := str.Title("two  spaces"); got != "Two  Spaces" {
		t.Errorf("Title gave %q, want the two spaces kept", got)
	}
	if got := str.Headline("two  spaces"); got != "Two Spaces" {
		t.Errorf("Headline gave %q, want one space", got)
	}
}

func TestSentence(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"hello world", "Hello world"},
		{"hello world. goodbye.", "Hello world. Goodbye."},
		{"one! two? three.", "One! Two? Three."},
		{"already Fine", "Already Fine"},
	}

	for _, c := range cases {
		if got := str.Sentence(c.in); got != c.want {
			t.Errorf("Sentence(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestSentenceKeepsProperNouns is the reason this is not lower casing the
// string and raising the front of it.
func TestSentenceKeepsProperNouns(t *testing.T) {
	if got := str.Sentence("we ship on Tuesday"); got != "We ship on Tuesday" {
		t.Errorf("Sentence gave %q, want Tuesday left alone", got)
	}
}

func TestUpperFirst(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"hello world", "Hello world"},
		{"Hello", "Hello"},
		{"123abc", "123abc"},
		{"ünicode", "Ünicode"},
	}

	for _, c := range cases {
		if got := str.UpperFirst(c.in); got != c.want {
			t.Errorf("UpperFirst(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLowerFirst(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"Hello World", "hello World"},
		{"hello", "hello"},
	}

	for _, c := range cases {
		if got := str.LowerFirst(c.in); got != c.want {
			t.Errorf("LowerFirst(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestUpperFirstOnACombinedLetter is why this counts clusters rather than
// runes: the accent has to travel with the letter it belongs to.
func TestUpperFirstOnACombinedLetter(t *testing.T) {
	in := "étude" // e, combining acute, then tude
	want := "Étude"

	got := str.UpperFirst(in)
	if got != want {
		t.Errorf("UpperFirst(%q) = %q, want %q", in, got, want)
	}
	if str.Length(got) != str.Length(in) {
		t.Errorf("UpperFirst changed the length from %d to %d", str.Length(in), str.Length(got))
	}
}

func TestSwapCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"Hello World", "hELLO wORLD"},
		{"123", "123"},
		{"日本語", "日本語"},
	}

	for _, c := range cases {
		if got := str.SwapCase(c.in); got != c.want {
			t.Errorf("SwapCase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSwapCaseTwiceComesBack(t *testing.T) {
	in := "Hello, World 123"

	if got := str.SwapCase(str.SwapCase(in)); got != in {
		t.Errorf("swapping twice gave %q, want %q", got, in)
	}
}

// TestTheCasesOnAnEmptyStringAndOnPunctuation covers the two inputs that come
// out of a form and take every one of these down the short path.
func TestTheCasesOnAnEmptyStringAndOnPunctuation(t *testing.T) {
	fns := map[string]func(string) string{
		"Camel":      str.Camel,
		"Pascal":     str.Pascal,
		"Snake":      str.Snake,
		"Kebab":      str.Kebab,
		"Headline":   str.Headline,
		"Title":      str.Title,
		"Sentence":   str.Sentence,
		"UpperFirst": str.UpperFirst,
		"LowerFirst": str.LowerFirst,
		"SwapCase":   str.SwapCase,
	}

	for name, fn := range fns {
		if got := fn(""); got != "" {
			t.Errorf("%s of an empty string gave %q", name, got)
		}
	}

	// The name cases throw punctuation away; the text cases keep it.
	for _, name := range []string{"Camel", "Pascal", "Snake", "Kebab", "Headline"} {
		if got := fns[name]("!!!"); got != "" {
			t.Errorf("%s of punctuation gave %q, want nothing", name, got)
		}
	}
	for _, name := range []string{"Title", "Sentence", "UpperFirst", "LowerFirst", "SwapCase"} {
		if got := fns[name]("!!!"); got != "!!!" {
			t.Errorf("%s of punctuation gave %q, want it left alone", name, got)
		}
	}
}
