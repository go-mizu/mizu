package console

import (
	"errors"
	"testing"
	"time"
)

// prune is a command's fields, close enough to a real one that the tests read
// like command lines somebody would type.
type prune struct {
	days    int
	name    string
	dry     bool
	verbose int
	tags    []string

	tenant string
	rest   []string
}

func (p *prune) flags() []Flag {
	return []Flag{
		{Name: "days", Short: 'd', Default: "30", Desc: "Delete accounts older than this", Value: Int(&p.days)},
		{Name: "name", Short: 'n', Value: String(&p.name)},
		{Name: "dry-run", Value: Bool(&p.dry)},
		{Name: "verbose", Short: 'v', Value: Count(&p.verbose)},
		{Name: "tag", Short: 't', Value: Strings(&p.tags, ",")},
	}
}

func (p *prune) args() []Arg {
	return []Arg{{Name: "tenant", Desc: "Tenant slug, or all", Value: String(&p.tenant)}}
}

// noenv is an environment with nothing in it, so that a test does not depend on
// what the machine running it happens to export.
func noenv(string) string { return "" }

// run parses a command line and fails the test if it could not be understood.
func (p *prune) run(t *testing.T, argv ...string) {
	t.Helper()
	if err := parse(p.flags(), p.args(), argv, noenv); err != nil {
		t.Fatalf("parse %v: %v", argv, err)
	}
}

// refuse parses a command line that should not have been understood, and
// returns what it said about it.
func refuse(t *testing.T, argv ...string) string {
	t.Helper()
	var p prune
	err := parse(p.flags(), p.args(), argv, noenv)
	if err == nil {
		t.Fatalf("parse %v was accepted", argv)
	}
	var usage *UsageError
	if !errors.As(err, &usage) {
		t.Fatalf("parse %v returned %T, want a *UsageError so the caller can exit 2", argv, err)
	}
	return err.Error()
}

func TestParseLongFlags(t *testing.T) {
	for _, argv := range [][]string{
		{"--days", "7", "--name", "Ada"},
		{"--days=7", "--name=Ada"},
		{"--name=Ada", "--days", "7"},
	} {
		var p prune
		p.run(t, argv...)
		if p.days != 7 || p.name != "Ada" {
			t.Errorf("%v gave days=%d name=%q", argv, p.days, p.name)
		}
	}
}

func TestParseShortFlags(t *testing.T) {
	for _, argv := range [][]string{
		{"-d", "7"},
		{"-d7"},
		{"-d=7"},
	} {
		var p prune
		p.run(t, argv...)
		if p.days != 7 {
			t.Errorf("%v gave days=%d", argv, p.days)
		}
	}
}

// TestParseClustersShortFlags is -vvv, and the reason the letters are walked
// rather than looked up whole.
func TestParseClustersShortFlags(t *testing.T) {
	var p prune
	p.run(t, "-vvv")
	if p.verbose != 3 {
		t.Errorf("-vvv counted %d, want 3", p.verbose)
	}

	// A letter that takes a value ends the cluster and takes the rest of it.
	var q prune
	q.run(t, "-vd7")
	if q.verbose != 1 || q.days != 7 {
		t.Errorf("-vd7 gave verbose=%d days=%d", q.verbose, q.days)
	}
}

func TestParseBooleans(t *testing.T) {
	for _, tt := range []struct {
		argv []string
		want bool
	}{
		{[]string{"--dry-run"}, true},
		{[]string{"--dry-run=true"}, true},
		{[]string{"--dry-run=false"}, false},
		{[]string{"--no-dry-run"}, false},
		{[]string{"--dry-run", "--no-dry-run"}, false},
	} {
		var p prune
		p.run(t, tt.argv...)
		if p.dry != tt.want {
			t.Errorf("%v gave dry-run=%v, want %v", tt.argv, p.dry, tt.want)
		}
	}
}

// TestParseNoOnlyTurnsOffBooleans keeps --no- from becoming a prefix that means
// something different on every flag. --name takes a string, so --no-name is not
// a flag at all, and the suggestion is the flag whose name is in there.
func TestParseNoOnlyTurnsOffBooleans(t *testing.T) {
	if got, want := refuse(t, "--no-name", "x"), "unknown flag --no-name, did you mean --name?"; got != want {
		t.Errorf("says %q, want %q", got, want)
	}
	if got, want := refuse(t, "--no-dry-run=true"), "--no-dry-run takes no value"; got != want {
		t.Errorf("says %q, want %q", got, want)
	}
}

// TestParseCountWithAValue is --verbose=2 saying the same thing as -vv, which
// is what a script writes when the number came from somewhere else.
func TestParseCountWithAValue(t *testing.T) {
	var p prune
	p.run(t, "--verbose=2")
	if p.verbose != 2 {
		t.Errorf("counted %d, want 2", p.verbose)
	}
}

// TestParseCountsTheLongForm covers --verbose --verbose, which a script writes
// when it builds its arguments in a loop.
func TestParseCountsTheLongForm(t *testing.T) {
	var p prune
	p.run(t, "--verbose", "--verbose")
	if p.verbose != 2 {
		t.Errorf("counted %d, want 2", p.verbose)
	}
}

// TestParseDoesNotSuggestHiddenFlags, because a suggestion is help text, and a
// hidden flag is one that has been kept out of the help.
func TestParseDoesNotSuggestHiddenFlags(t *testing.T) {
	var secret string
	flags := []Flag{{Name: "internal", Hidden: true, Value: String(&secret)}}

	err := parse(flags, nil, []string{"--internl"}, noenv)
	if err == nil {
		t.Fatal("--internl was accepted")
	}
	if want := "unknown flag --internl"; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

func TestParseRepeatedFlags(t *testing.T) {
	var p prune
	p.run(t, "--days", "7", "--days", "9", "--tag", "a,b", "-t", "c")
	if p.days != 9 {
		t.Errorf("days=%d, want the last one to win", p.days)
	}
	if len(p.tags) != 3 {
		t.Errorf("tags=%v, want a slice to collect all three", p.tags)
	}
}

// TestParseStopsAtDoubleDash is how a value that looks like a flag is passed.
func TestParseStopsAtDoubleDash(t *testing.T) {
	var p prune
	p.run(t, "--", "--days")
	if p.tenant != "--days" {
		t.Errorf("tenant=%q, want the flag-looking word to be an argument", p.tenant)
	}
	if p.days != 30 {
		t.Errorf("days=%d, want the default", p.days)
	}
}

// TestParseTakesDashAsAnArgument, because that is how a program is told to read
// from stdin, and it is not a flag with no letters.
func TestParseTakesDashAsAnArgument(t *testing.T) {
	var p prune
	p.run(t, "-")
	if p.tenant != "-" {
		t.Errorf("tenant=%q, want -", p.tenant)
	}
}

// TestParseTakesNegativeNumbers, so that a command taking an offset does not
// have to explain that -5 is a flag called 5.
func TestParseTakesNegativeNumbers(t *testing.T) {
	var p prune
	p.run(t, "-5")
	if p.tenant != "-5" {
		t.Errorf("tenant=%q, want -5", p.tenant)
	}
}

// TestParseAllowsFlagsAfterArguments, because that is where people put them.
func TestParseAllowsFlagsAfterArguments(t *testing.T) {
	var p prune
	p.run(t, "acme", "--days", "7")
	if p.tenant != "acme" || p.days != 7 {
		t.Errorf("tenant=%q days=%d", p.tenant, p.days)
	}
}

func TestParseUnknownFlag(t *testing.T) {
	for _, tt := range []struct {
		argv []string
		want string
	}{
		{[]string{"--dayz", "7"}, "unknown flag --dayz, did you mean --days?"},
		{[]string{"--nmae=Ada"}, "unknown flag --nmae, did you mean --name?"},
		{[]string{"--elephant"}, "unknown flag --elephant"},
		{[]string{"-z"}, "unknown flag -z"},
		{[]string{"--=7"}, "--=7 is not a flag name"},
	} {
		if got := refuse(t, tt.argv...); got != tt.want {
			t.Errorf("%v says %q, want %q", tt.argv, got, tt.want)
		}
	}
}

func TestParseFlagNeedsAValue(t *testing.T) {
	if got, want := refuse(t, "--days"), "--days needs a value"; got != want {
		t.Errorf("says %q, want %q", got, want)
	}
	if got, want := refuse(t, "-d"), "-d needs a value"; got != want {
		t.Errorf("says %q, want %q", got, want)
	}
}

// TestParseNamesTheFlagThatWasWrong, since "not a number" on its own leaves
// somebody with four numeric flags to check by hand.
func TestParseNamesTheFlagThatWasWrong(t *testing.T) {
	for _, tt := range []struct {
		argv []string
		want string
	}{
		{[]string{"--days=x"}, `--days: "x" is not a number`},
		{[]string{"--days", "x"}, `--days: "x" is not a number`},
		{[]string{"-dx"}, `-d: "x" is not a number`},
		{[]string{"--dry-run=maybe"}, `--dry-run: "maybe" is not true or false`},
	} {
		if got := refuse(t, tt.argv...); got != tt.want {
			t.Errorf("%v says %q, want %q", tt.argv, got, tt.want)
		}
	}
}

// stubborn takes no argument and refuses anyway, which a Value with a rule of
// its own is allowed to do.
type stubborn struct{}

func (stubborn) String() string   { return "" }
func (stubborn) Set(string) error { return errors.New("the build was not made with it") }
func (stubborn) IsBoolFlag() bool { return true }

func TestParseNamesABooleanThatRefused(t *testing.T) {
	flags := []Flag{{Name: "force", Short: 'f', Value: stubborn{}}}

	for _, tt := range []struct {
		argv []string
		want string
	}{
		{[]string{"-f"}, "-f: the build was not made with it"},
		{[]string{"--force"}, "--force: the build was not made with it"},
	} {
		err := parse(flags, nil, tt.argv, noenv)
		if err == nil {
			t.Errorf("%v was accepted", tt.argv)
			continue
		}
		if err.Error() != tt.want {
			t.Errorf("%v says %q, want %q", tt.argv, err, tt.want)
		}
	}
}

func TestParseAppliesDefaults(t *testing.T) {
	var p prune
	p.run(t)
	if p.days != 30 {
		t.Errorf("days=%d, want the default of 30", p.days)
	}
}

func TestParseAppliesTheEnvironment(t *testing.T) {
	env := func(name string) string {
		if name == "PRUNE_DAYS" {
			return "90"
		}
		return ""
	}
	flags := func(p *prune) []Flag {
		return []Flag{{Name: "days", Default: "30", Env: "PRUNE_DAYS", Value: Int(&p.days)}}
	}

	var p prune
	if err := parse(flags(&p), nil, nil, env); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.days != 90 {
		t.Errorf("days=%d, want the environment to beat the default", p.days)
	}

	var q prune
	if err := parse(flags(&q), nil, []string{"--days=7"}, env); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if q.days != 7 {
		t.Errorf("days=%d, want the command line to beat the environment", q.days)
	}
}

func TestParseRejectsABadEnvironment(t *testing.T) {
	env := func(string) string { return "soon" }

	var d time.Duration
	err := parse([]Flag{{Name: "timeout", Env: "MIZU_TIMEOUT", Value: Duration(&d)}}, nil, nil, env)
	if err == nil {
		t.Fatal("soon was accepted as a timeout")
	}
	if want := `MIZU_TIMEOUT: "soon" is not a length of time, try 30s or 5m`; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

func TestParseRequiredFlag(t *testing.T) {
	var name string
	flags := []Flag{{Name: "name", Required: true, Desc: "Who to greet", Value: String(&name)}}

	err := parse(flags, nil, nil, noenv)
	if err == nil {
		t.Fatal("the required flag was not missed")
	}
	if want := "--name is required: who to greet"; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}

	if err := parse(flags, nil, []string{"--name=Ada"}, noenv); err != nil {
		t.Errorf("parse: %v", err)
	}
}

func TestParseRequiredFlagWithNoDescription(t *testing.T) {
	var name string
	err := parse([]Flag{{Name: "name", Required: true, Value: String(&name)}}, nil, nil, noenv)
	if err == nil || err.Error() != "--name is required" {
		t.Errorf("says %v", err)
	}
}

// TestParseRejectsABadDefault is the mistake belonging to whoever declared the
// flag, caught the first time the command runs rather than never.
func TestParseRejectsABadDefault(t *testing.T) {
	var days int
	err := parse([]Flag{{Name: "days", Default: "soon", Value: Int(&days)}}, nil, nil, noenv)
	if err == nil {
		t.Fatal("soon was accepted as a default")
	}
	if want := `the default for --days does not parse: "soon" is not a number`; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

func TestParseArguments(t *testing.T) {
	var p prune
	p.run(t, "acme")
	if p.tenant != "acme" {
		t.Errorf("tenant=%q", p.tenant)
	}
}

func TestParseRequiredArgument(t *testing.T) {
	var tenant string
	args := []Arg{{Name: "tenant", Required: true, Desc: "Tenant slug, or all", Value: String(&tenant)}}

	err := parse(nil, args, nil, noenv)
	if err == nil {
		t.Fatal("the required argument was not missed")
	}
	if want := "tenant is required: tenant slug, or all"; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

func TestParseArgumentDefault(t *testing.T) {
	var tenant string
	args := []Arg{{Name: "tenant", Default: "all", Value: String(&tenant)}}

	if err := parse(nil, args, nil, noenv); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if tenant != "all" {
		t.Errorf("tenant=%q, want the default", tenant)
	}
}

func TestParseRestArgument(t *testing.T) {
	var p prune
	args := []Arg{
		{Name: "tenant", Value: String(&p.tenant)},
		{Name: "files", Rest: true, Value: Strings(&p.rest, "")},
	}

	if err := parse(nil, args, []string{"acme", "a.sql", "b.sql"}, noenv); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.tenant != "acme" {
		t.Errorf("tenant=%q", p.tenant)
	}
	if len(p.rest) != 2 || p.rest[1] != "b.sql" {
		t.Errorf("files=%v, want the two after the tenant", p.rest)
	}
}

// TestParseRequiredRestArgument is the one a loop over the arguments would
// walk straight past, because there is no slot to find empty.
func TestParseRequiredRestArgument(t *testing.T) {
	var files []string
	args := []Arg{{Name: "files", Rest: true, Required: true, Value: Strings(&files, "")}}

	err := parse(nil, args, nil, noenv)
	if err == nil {
		t.Fatal("no files at all was accepted")
	}
	if want := "files is required"; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

func TestParseRejectsABadArgumentDefault(t *testing.T) {
	var days int
	args := []Arg{{Name: "days", Default: "soon", Value: Int(&days)}}

	err := parse(nil, args, nil, noenv)
	if err == nil {
		t.Fatal("soon was accepted as a default")
	}
	if want := `the default for days does not parse: "soon" is not a number`; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

func TestParseRejectsABadWordInTheRest(t *testing.T) {
	var days []int
	args := []Arg{{Name: "days", Rest: true, Value: Slice(&days, parseInt, "")}}

	err := parse(nil, args, []string{"7", "soon"}, noenv)
	if err == nil {
		t.Fatal("soon was accepted")
	}
	if want := `days: "soon" is not a number`; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

func TestParseTooManyArguments(t *testing.T) {
	for _, tt := range []struct {
		args []Arg
		argv []string
		want string
	}{
		{nil, []string{"acme"}, `unexpected argument "acme", this command takes none`},
		{[]Arg{{Name: "a", Value: String(new(string))}}, []string{"one", "two"}, `unexpected argument "two", this command takes one`},
		{[]Arg{
			{Name: "a", Value: String(new(string))},
			{Name: "b", Value: String(new(string))},
		}, []string{"one", "two", "three"}, `unexpected argument "three", this command takes 2`},
	} {
		err := parse(nil, tt.args, tt.argv, noenv)
		if err == nil {
			t.Errorf("%v was accepted", tt.argv)
			continue
		}
		if err.Error() != tt.want {
			t.Errorf("%v says %q, want %q", tt.argv, err, tt.want)
		}
	}
}

func TestParseRejectsABadArgument(t *testing.T) {
	var days int
	args := []Arg{{Name: "days", Value: Int(&days)}}

	err := parse(nil, args, []string{"soon"}, noenv)
	if err == nil {
		t.Fatal("soon was accepted")
	}
	if want := `days: "soon" is not a number`; err.Error() != want {
		t.Errorf("says %q, want %q", err, want)
	}
}

// TestParseRejectsABadDefinition is the half of the mistakes that belong to
// whoever wrote the command rather than to whoever ran it. A person typing at a
// terminal can do nothing about a duplicate flag letter, so it is a panic and
// the test suite is where it lands.
func TestParseRejectsABadDefinition(t *testing.T) {
	s := new(string)
	b := new(bool)

	for _, tt := range []struct {
		name  string
		flags []Flag
		args  []Arg
		want  string
	}{
		{
			"no name",
			[]Flag{{Value: String(s)}}, nil,
			"console: a flag has no name",
		},
		{
			"no value",
			[]Flag{{Name: "days"}}, nil,
			"console: flag --days has no value to parse into",
		},
		{
			"twice",
			[]Flag{{Name: "days", Value: String(s)}, {Name: "days", Value: String(s)}}, nil,
			"console: flag --days is declared twice",
		},
		{
			"same letter",
			[]Flag{{Name: "days", Short: 'd', Value: String(s)}, {Name: "dry", Short: 'd', Value: Bool(b)}}, nil,
			"console: flag -d is declared twice",
		},
		{
			"required with a default",
			[]Flag{{Name: "days", Required: true, Default: "30", Value: String(s)}}, nil,
			"console: flag --days is required and has a default",
		},
		{
			"argument with no name",
			nil, []Arg{{Value: String(s)}},
			"console: an argument has no name",
		},
		{
			"argument with no value",
			nil, []Arg{{Name: "tenant"}},
			"console: argument tenant has no value to parse into",
		},
		{
			"the rest is not last",
			nil, []Arg{{Name: "files", Rest: true, Value: String(s)}, {Name: "tenant", Value: String(s)}},
			"console: argument files takes the rest and is not last",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if got == nil {
					t.Fatal("it was accepted")
				}
				if got != tt.want {
					t.Errorf("panicked with %q, want %q", got, tt.want)
				}
			}()
			parse(tt.flags, tt.args, nil, noenv)
		})
	}
}

// TestParseUsesTheRealEnvironment covers the one line the rest of these skip by
// passing their own lookup.
func TestParseUsesTheRealEnvironment(t *testing.T) {
	t.Setenv("MIZU_TEST_DAYS", "90")

	var days int
	if err := Parse([]Flag{{Name: "days", Env: "MIZU_TEST_DAYS", Value: Int(&days)}}, nil, nil); err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if days != 90 {
		t.Errorf("days=%d, want 90 from the environment", days)
	}
}
