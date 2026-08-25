package diag_test

import (
	"slices"
	"strconv"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
)

// The settings of a database section, which is the candidate set the published
// example in doc 36 is measured against.
var settings = []string{
	"driver", "url", "max_open_conns", "max_idle_conns", "conn_max_lifetime",
	"connect_timeout", "read_timeout", "ssl_mode", "log_queries",
}

func suggest(t *testing.T, want string, candidates []string) []string {
	t.Helper()
	return diag.Suggest(want, slices.Values(candidates))
}

func TestSuggestFindsTheTypo(t *testing.T) {
	for _, tt := range []struct {
		want string
		then string
	}{
		{"max_open_conn", "max_open_conns"},
		{"max_open_cons", "max_open_conns"},
		{"max_open_connss", "max_open_conns"},
		{"max_open_ocnns", "max_open_conns"},
		{"ssl_moed", "ssl_mode"},
		{"drivr", "driver"},
	} {
		got := suggest(t, tt.want, settings)
		if len(got) == 0 || got[0] != tt.then {
			t.Errorf("Suggest(%q) = %v, want %q first", tt.want, got, tt.then)
		}
	}
}

// The case of a name is the commonest mistake against a list somebody read
// somewhere, and it comes out first.
func TestSuggestIgnoresCase(t *testing.T) {
	for _, want := range []string{"URL", "Url", "SSL_MODE", "Max_Open_Conns"} {
		got := suggest(t, want, settings)
		if len(got) == 0 {
			t.Errorf("Suggest(%q) found nothing", want)
			continue
		}
		if got[0] != "url" && got[0] != "ssl_mode" && got[0] != "max_open_conns" {
			t.Errorf("Suggest(%q) = %v", want, got)
		}
	}
}

// The half-remembered name, where edit distance is no use because the two are
// five mistakes apart.
func TestSuggestFindsTheHalfRememberedName(t *testing.T) {
	got := suggest(t, "timeout", settings)
	want := []string{"connect_timeout", "read_timeout"}
	if !slices.Equal(got, want) {
		t.Errorf("Suggest(\"timeout\") = %v, want %v", got, want)
	}
}

// A typo comes before a half-remembered name, because a name one mistake away
// is a stronger answer than a name that merely starts the same.
func TestSuggestPutsACorrectionBeforeAPrefix(t *testing.T) {
	got := suggest(t, "read_timeou", settings)
	if len(got) == 0 || got[0] != "read_timeout" {
		t.Errorf("Suggest(\"read_timeou\") = %v, want read_timeout first", got)
	}
}

// Nothing rather than the least bad thing available. A wrong suggestion sends
// the reader down a false path with confidence.
func TestSuggestSaysNothingWhenNothingIsClose(t *testing.T) {
	for _, want := range []string{"pool_size", "hostname", "xyzzy", "port"} {
		if got := suggest(t, want, settings); len(got) != 0 {
			t.Errorf("Suggest(%q) = %v, want nothing", want, got)
		}
	}
}

// One mistake in a short name is usually a different word, so a short name gets
// a tighter limit than a long one.
func TestSuggestIsStricterAboutShortNames(t *testing.T) {
	short := []string{"add", "get", "put", "list"}
	if got := suggest(t, "adds", short); !slices.Equal(got, []string{"add"}) {
		t.Errorf("Suggest(\"adds\") = %v, want add", got)
	}
	// Two mistakes from add, and also two from put. Neither is an answer.
	if got := suggest(t, "apt", short); len(got) != 0 {
		t.Errorf("Suggest(\"apt\") = %v, want nothing", got)
	}
	// The same two mistakes in a longer name is a typo worth answering.
	if got := suggest(t, "max_open_cnos", settings); len(got) == 0 {
		t.Error("Suggest(\"max_open_cnos\") found nothing")
	}
}

// One name nearer than the rest is the answer on its own. A section with a
// misspelled setting in it has siblings that are closer to the mistake than
// they are to anything, and listing them beside the one that is right asks the
// reader to weigh three names when mizu already knows which it is.
func TestSuggestKeepsOnlyTheClosest(t *testing.T) {
	paths := []string{
		"queue.connections.name", "queue.connections.tries",
		"queue.connections.node",
	}
	got := suggest(t, "queue.connections.tires", paths)
	if want := []string{"queue.connections.tries"}; !slices.Equal(got, want) {
		t.Errorf("Suggest(\"queue.connections.tires\") = %v, want %v", got, want)
	}
}

// Six names are all one mistake away from what was typed. Three is the ceiling,
// because a list long enough to read through is a list that has stopped being
// an answer.
var tied = []string{"max_a", "max_b", "max_c", "max_d", "max_e", "max_f"}

// A long name is allowed more mistakes than a short one, which is what lets a
// setting somebody wrote from memory find the one it was meant to be. Five
// mistakes in eighteen characters is a typo; five in five is a different word.
func TestSuggestAllowsMoreMistakesInALongerName(t *testing.T) {
	paths := []string{
		"database.driver", "database.url", "database.max_open_conns",
		"database.max_idle_conns", "app.name", "log.level",
	}
	got := suggest(t, "database.max_conns", paths)
	if want := []string{"database.max_idle_conns", "database.max_open_conns"}; !slices.Equal(got, want) {
		t.Errorf("Suggest(\"database.max_conns\") = %v, want %v", got, want)
	}
}

// Sharing a section with a setting is not being that setting. Every name under
// database. has nine characters in common with every other one, and answering
// an unknown setting with three arbitrary siblings is worse than saying
// nothing.
func TestSuggestDoesNotOfferEverySettingInASection(t *testing.T) {
	paths := []string{
		"database.driver", "database.url", "database.max_open_conns",
		"database.max_idle_conns",
	}
	if got := suggest(t, "database.replica", paths); len(got) != 0 {
		t.Errorf("Suggest(\"database.replica\") = %v, want nothing", got)
	}
}

// Two characters in common is most of the alphabet squared, so a name that
// short is offered only when it is a correction and never because of what it
// starts with.
func TestSuggestWillNotOfferATwoLetterName(t *testing.T) {
	if got := suggest(t, "logs", []string{"lo", "db"}); len(got) != 0 {
		t.Errorf("Suggest(\"logs\") = %v, want nothing", got)
	}
	if got := suggest(t, "lo", []string{"log"}); !slices.Equal(got, []string{"log"}) {
		t.Errorf("Suggest(\"lo\") = %v, want log", got)
	}
}

func TestSuggestOffersAtMostThree(t *testing.T) {
	got := suggest(t, "max_", tied)
	if want := []string{"max_a", "max_b", "max_c"}; !slices.Equal(got, want) {
		t.Errorf("Suggest(\"max_\") = %v, want %v", got, want)
	}
}

// Same answer every run, whatever order the candidates arrive in. Six names at
// the same distance is where an unstable sort would show.
func TestSuggestIsStable(t *testing.T) {
	forwards := suggest(t, "max_", tied)
	backwards := suggest(t, "max_", reversed(tied))
	if !slices.Equal(forwards, backwards) {
		t.Errorf("order of the candidates changed the answer: %v then %v", forwards, backwards)
	}
}

func reversed(s []string) []string {
	out := slices.Clone(s)
	slices.Reverse(out)
	return out
}

func TestSuggestOfNothing(t *testing.T) {
	if got := suggest(t, "", settings); got != nil {
		t.Errorf("Suggest(\"\") = %v, want nil", got)
	}
	if got := suggest(t, "url", nil); got != nil {
		t.Errorf("Suggest with no candidates = %v, want nil", got)
	}
	if got := suggest(t, "url", []string{"", "", ""}); got != nil {
		t.Errorf("Suggest over empty candidates = %v, want nil", got)
	}
}

// A candidate identical to what was typed is the honest answer even though it
// means the caller has a bug somewhere else.
func TestSuggestOfSomethingThatExists(t *testing.T) {
	got := suggest(t, "url", settings)
	if len(got) == 0 || got[0] != "url" {
		t.Errorf("Suggest(\"url\") = %v, want url first", got)
	}
}

func TestDistance(t *testing.T) {
	for _, tt := range []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"log", "log", 0},
		{"", "log", 3},
		{"log", "", 3},
		{"lgo", "log", 1},
		{"lg", "log", 1},
		{"loog", "log", 1},
		{"lox", "log", 1},
		{"xyz", "log", 3},
		{"kitten", "sitting", 3},
		{"max_open_conns", "max_idle_conns", 4},
	} {
		if got := diag.Distance(tt.a, tt.b); got != tt.want {
			t.Errorf("Distance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// Two adjacent characters the wrong way round is one mistake and not two, which
// is the whole reason this is not plain Levenshtein.
func TestDistanceCountsASwapAsOneMistake(t *testing.T) {
	if got := diag.Distance("lgo", "log"); got != 1 {
		t.Errorf("Distance(\"lgo\", \"log\") = %d, want 1", got)
	}
	if got := diag.Distance("cnos", "cons"); got != 1 {
		t.Errorf("Distance(\"cnos\", \"cons\") = %d, want 1", got)
	}
}

// Characters rather than bytes, so a mistake in a name that is not ASCII costs
// what a mistake costs anywhere else.
func TestDistanceCountsCharactersNotBytes(t *testing.T) {
	if got := diag.Distance("naive", "naïve"); got != 1 {
		t.Errorf("Distance(\"naive\", \"naïve\") = %d, want 1", got)
	}
	if got := diag.Distance("æther", "ether"); got != 1 {
		t.Errorf("Distance(\"æther\", \"ether\") = %d, want 1", got)
	}
}

func TestDistanceIsSymmetric(t *testing.T) {
	for _, tt := range [][2]string{{"lgo", "log"}, {"kitten", "sitting"}, {"", "log"}} {
		a, b := diag.Distance(tt[0], tt[1]), diag.Distance(tt[1], tt[0])
		if a != b {
			t.Errorf("Distance(%q, %q) = %d but the other way round is %d", tt[0], tt[1], a, b)
		}
	}
}

func TestDid(t *testing.T) {
	for _, tt := range []struct {
		names []string
		want  string
	}{
		{nil, ""},
		{[]string{}, ""},
		{[]string{"url"}, `did you mean "url"?`},
		{[]string{"url", "driver"}, `did you mean "url" or "driver"?`},
		{[]string{"a", "b", "c"}, `did you mean "a", "b" or "c"?`},
	} {
		if got := diag.Did(tt.names, strconv.Quote); got != tt.want {
			t.Errorf("Did(%v) = %q, want %q", tt.names, got, tt.want)
		}
	}
}

// A flag is written with its two dashes rather than in quotes, which is the
// reason the wrapping is the caller's to choose.
func TestDidWithAnotherWrapping(t *testing.T) {
	got := diag.Did([]string{"dry-run"}, func(s string) string { return "--" + s })
	if want := "did you mean --dry-run?"; got != want {
		t.Errorf("Did() = %q, want %q", got, want)
	}
}

func TestDidWithNoWrapping(t *testing.T) {
	got := diag.Did([]string{"url", "driver"}, nil)
	if want := "did you mean url or driver?"; got != want {
		t.Errorf("Did() = %q, want %q", got, want)
	}
}
