package console

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// pruneCmd is the shape a generated command has: fields, a spec pointing at
// them, and a Run that reads them as ordinary Go values.
type pruneCmd struct {
	days    int
	dry     bool
	verbose int
	tags    []string
	timeout time.Duration
	secret  string

	tenant string
	reason string

	ran bool
	err error
}

func (c *pruneCmd) Spec() Spec {
	return Spec{
		Name: "users:prune",
		Desc: "Delete users who never verified their email",
		Long: "Accounts that never followed the link in the welcome email.\nThe tenant is the one to work on, or all for every one of them.",
		Flags: []Flag{
			{Name: "days", Short: 'd', Default: "30", Desc: "Delete accounts older than this", Value: Int(&c.days)},
			{Name: "dry-run", Desc: "Report without deleting", Value: Bool(&c.dry)},
			{Name: "verbose", Short: 'v', Desc: "Say more about what is happening", Value: Count(&c.verbose)},
			{Name: "tag", Desc: "Only these tags", Value: Strings(&c.tags, ",")},
			{Name: "timeout", Env: "MIZU_TIMEOUT", Value: Duration(&c.timeout)},
			{Name: "secret", Hidden: true, Value: String(&c.secret)},
		},
		Args: []Arg{
			{Name: "tenant", Required: true, Desc: "Tenant slug, or all", Value: String(&c.tenant)},
			{Name: "reason", Default: "cleanup", Desc: "Why, for the audit log", Value: String(&c.reason)},
		},
	}
}

func (c *pruneCmd) Run(ctx context.Context, io *IO) error {
	c.ran = true
	return c.err
}

// simple is a command with nothing on it, which is most of them.
type simple struct {
	name   string
	desc   string
	hidden bool
	ran    bool
}

func (c *simple) Spec() Spec { return Spec{Name: c.name, Desc: c.desc, Hidden: c.hidden} }

func (c *simple) Run(ctx context.Context, io *IO) error {
	c.ran = true
	return nil
}

// app builds a small CLI and hands back the buffers it writes to.
func app(cmds ...Command) (*App, *IO, *strings.Builder, *strings.Builder) {
	var out, err strings.Builder
	a := &App{Name: "mizu", Desc: "mizu builds and runs a mizu application.", Version: "0.3.1"}
	a.Add(cmds...)
	return a, New(strings.NewReader(""), &out, &err, Options{Color: ColorNever}), &out, &err
}

func run(t *testing.T, a *App, c *IO, argv ...string) error {
	t.Helper()
	return a.Run(context.Background(), c, argv)
}

func TestRunsTheCommand(t *testing.T) {
	cmd := &pruneCmd{}
	a, c, _, _ := app(cmd)

	if err := run(t, a, c, "users:prune", "--days", "7", "acme"); err != nil {
		t.Fatal(err)
	}
	if !cmd.ran {
		t.Error("the command did not run")
	}
	if cmd.days != 7 || cmd.tenant != "acme" {
		t.Errorf("days %d tenant %q, want 7 and acme", cmd.days, cmd.tenant)
	}
}

func TestReturnsWhatTheCommandReturns(t *testing.T) {
	want := errors.New("the database is not there")
	cmd := &pruneCmd{err: want}
	a, c, _, _ := app(cmd)

	if err := run(t, a, c, "users:prune", "acme"); !errors.Is(err, want) {
		t.Errorf("got %v, want %v", err, want)
	}
}

func TestABadCommandLineDoesNotRunTheCommand(t *testing.T) {
	cmd := &pruneCmd{}
	a, c, _, _ := app(cmd)

	err := run(t, a, c, "users:prune", "--days", "soon", "acme")
	var usage *UsageError
	if !errors.As(err, &usage) {
		t.Fatalf("got %v, want a UsageError", err)
	}
	if cmd.ran {
		t.Error("the command ran on a command line that did not parse")
	}
}

func TestUnknownCommand(t *testing.T) {
	a, c, _, _ := app(&simple{name: "migrate"}, &simple{name: "db:seed"})

	for _, tt := range []struct {
		argv []string
		want string
	}{
		{[]string{"migrat"}, `unknown command "migrat", did you mean migrate`},
		{[]string{"nothing:like:it"}, `unknown command "nothing:like:it", run "mizu help" for the list`},
		{[]string{"help", "migrat"}, `unknown command "migrat", did you mean migrate`},
		{[]string{"--nope"}, `unknown flag --nope, run "mizu help" for what this takes`},
	} {
		err := run(t, a, c, tt.argv...)
		if err == nil || err.Error() != tt.want {
			t.Errorf("%v: got %v, want %s", tt.argv, err, tt.want)
		}
	}
}

func TestHiddenCommandRunsAndIsNotSuggested(t *testing.T) {
	cmd := &simple{name: "migrate", hidden: true}
	a, c, out, _ := app(cmd)

	if err := run(t, a, c, "migrate"); err != nil {
		t.Fatal(err)
	}
	if !cmd.ran {
		t.Error("a hidden command did not run")
	}
	if err := run(t, a, c, "migrat"); err == nil || strings.Contains(err.Error(), "did you mean") {
		t.Errorf("got %v, want no suggestion", err)
	}

	a.Help(c)
	if strings.Contains(out.String(), "migrate") {
		t.Errorf("a hidden command is in help:\n%s", out)
	}
}

func TestHelpTakesOneCommand(t *testing.T) {
	a, c, _, _ := app(&simple{name: "migrate"})

	err := run(t, a, c, "help", "migrate", "again")
	if err == nil || err.Error() != "help takes one command" {
		t.Errorf("got %v", err)
	}
}

func TestVersion(t *testing.T) {
	a, c, out, _ := app(&simple{name: "migrate"})

	if err := run(t, a, c, "--version"); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != "mizu 0.3.1\n" {
		t.Errorf("got %q", got)
	}

	a.Version = ""
	if err := run(t, a, c, "--version"); err == nil {
		t.Error("a program with no version answered --version")
	}
}

func TestNothingAtAllIsHelp(t *testing.T) {
	a, c, out, _ := app(&simple{name: "migrate", desc: "Run the migrations"})

	if err := run(t, a, c); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Run the migrations") {
		t.Errorf("got:\n%s", out)
	}
}

func TestHelpGoesToStdout(t *testing.T) {
	a, c, out, errOut := app(&simple{name: "migrate", desc: "Run the migrations"})

	for _, argv := range [][]string{{"--help"}, {"-h"}, {"help"}} {
		out.Reset()
		if err := run(t, a, c, argv...); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("%v wrote nothing to stdout", argv)
		}
	}
	if errOut.String() != "" {
		t.Errorf("help wrote to stderr:\n%s", errOut)
	}
}

func TestCommandHelp(t *testing.T) {
	a, c, out, _ := app(&pruneCmd{})

	for _, argv := range [][]string{{"help", "users:prune"}, {"users:prune", "--help"}, {"users:prune", "-h"}} {
		out.Reset()
		if err := run(t, a, c, argv...); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "mizu users:prune [flags] <tenant>") {
			t.Errorf("%v printed:\n%s", argv, out)
		}
	}
}

func TestHelpDoesNotNeedTheRequiredArguments(t *testing.T) {
	cmd := &pruneCmd{}
	a, c, out, _ := app(cmd)

	// The tenant is required, and asking what the command takes is how somebody
	// finds that out. Parsing first would answer with the error instead.
	if err := run(t, a, c, "users:prune", "--help"); err != nil {
		t.Fatal(err)
	}
	if cmd.ran {
		t.Error("--help ran the command")
	}
	if !strings.Contains(out.String(), "tenant") {
		t.Errorf("got:\n%s", out)
	}
}

func TestHelpAfterDoubleDashIsAnArgument(t *testing.T) {
	cmd := &pruneCmd{}
	a, c, _, _ := app(cmd)

	if err := run(t, a, c, "users:prune", "--", "--help"); err != nil {
		t.Fatal(err)
	}
	if !cmd.ran {
		t.Error("the command did not run")
	}
	if cmd.tenant != "--help" {
		t.Errorf("tenant is %q, want --help", cmd.tenant)
	}
}

// helpful declares --help itself, which is unusual and has to keep working.
type helpful struct{ help string }

func (c *helpful) Spec() Spec {
	return Spec{
		Name:  "print",
		Flags: []Flag{{Name: "help", Short: 'h', Desc: "Which help page to print", Value: String(&c.help)}},
	}
}

func (c *helpful) Run(ctx context.Context, io *IO) error { return nil }

func TestACommandKeepsItsOwnHelpFlag(t *testing.T) {
	cmd := &helpful{}
	a, c, out, _ := app(cmd)

	if err := run(t, a, c, "print", "--help", "routing"); err != nil {
		t.Fatal(err)
	}
	if cmd.help != "routing" {
		t.Errorf("help is %q, want routing", cmd.help)
	}
	if out.String() != "" {
		t.Errorf("the command's own flag printed help:\n%s", out)
	}

	out.Reset()
	a.usage(c, cmd.Spec())
	if strings.Contains(out.String(), "Show what this command takes") {
		t.Errorf("help text shows a --help that is not there:\n%s", out)
	}
}

func TestAddPanics(t *testing.T) {
	for _, tt := range []struct {
		name string
		cmds []Command
		want string
	}{
		{"no name", []Command{&simple{}}, "console: a command has no name"},
		{"a space", []Command{&simple{name: "db seed"}}, `console: command name "db seed" has a space in it`},
		{"twice", []Command{&simple{name: "migrate"}, &simple{name: "migrate"}}, "console: command migrate is registered twice"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if got := recover(); got != tt.want {
					t.Errorf("got %v, want %v", got, tt.want)
				}
			}()
			new(App).Add(tt.cmds...)
		})
	}
}

// broken has two flags with the same letter, which is a mistake in the program
// and is worth finding when it starts rather than when somebody runs it.
type broken struct{ a, b int }

func (c *broken) Spec() Spec {
	return Spec{
		Name: "broken",
		Flags: []Flag{
			{Name: "alpha", Short: 'a', Value: Int(&c.a)},
			{Name: "beta", Short: 'a', Value: Int(&c.b)},
		},
	}
}

func (c *broken) Run(ctx context.Context, io *IO) error { return nil }

func TestAddChecksTheFlags(t *testing.T) {
	defer func() {
		if got := recover(); got != "console: flag -a is declared twice" {
			t.Errorf("got %v", got)
		}
	}()
	new(App).Add(&broken{})
}

func TestSpecIsAskedForOnce(t *testing.T) {
	cmd := &counted{}
	a, c, _, _ := app(cmd)

	if err := run(t, a, c, "count", "--n", "3"); err != nil {
		t.Fatal(err)
	}
	// A second Spec would hand out a second set of values, and the parse would
	// have filled the first set.
	if cmd.specs != 1 {
		t.Errorf("Spec was called %d times, want 1", cmd.specs)
	}
	if cmd.n != 3 {
		t.Errorf("n is %d, want 3", cmd.n)
	}
}

type counted struct {
	specs int
	n     int
}

func (c *counted) Spec() Spec {
	c.specs++
	return Spec{Name: "count", Flags: []Flag{{Name: "n", Value: Int(&c.n)}}}
}

func (c *counted) Run(ctx context.Context, io *IO) error { return nil }

func TestAppHelpLayout(t *testing.T) {
	a, c, out, _ := app(
		&simple{name: "db:wipe", desc: "Drop every table"},
		&simple{name: "new", desc: "Create an application"},
		&simple{name: "db:seed", desc: "Fill the database with test data"},
		&simple{name: "doctor", desc: "Check the project for problems"},
	)

	a.Help(c)

	want := `mizu builds and runs a mizu application.

Usage:
  mizu <command> [flags]

Commands:
  doctor   Check the project for problems
  new      Create an application

  db:seed  Fill the database with test data
  db:wipe  Drop every table

Run "mizu help <command>" for more about one.
`
	if got := out.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestCommandOrder(t *testing.T) {
	for _, tt := range []struct {
		x, y string
		want int
	}{
		{"new", "db:seed", -1},
		{"db:seed", "new", 1},
		{"db:seed", "make:model", -1},
		{"db:wipe", "db:seed", 1},
		{"new", "new", 0},
	} {
		x := listing{group: group(tt.x), spec: Spec{Name: tt.x}}
		y := listing{group: group(tt.y), spec: Spec{Name: tt.y}}
		if got := byGroupThenName(x, y); got != tt.want {
			t.Errorf("%s against %s: got %d, want %d", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestAnAppWithNoCommands(t *testing.T) {
	a, c, out, _ := app()

	a.Help(c)

	want := `mizu builds and runs a mizu application.

Usage:
  mizu <command> [flags]
`
	if got := out.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestCommandHelpLayout(t *testing.T) {
	a, c, out, _ := app(&pruneCmd{})

	if err := run(t, a, c, "help", "users:prune"); err != nil {
		t.Fatal(err)
	}

	want := `Delete users who never verified their email

Accounts that never followed the link in the welcome email.
The tenant is the one to work on, or all for every one of them.

Usage:
  mizu users:prune [flags] <tenant> [<reason>]

Arguments:
  tenant  Tenant slug, or all
  reason  Why, for the audit log (default cleanup)

Flags:
  -d, --days int          Delete accounts older than this (default 30)
      --dry-run           Report without deleting
  -v, --verbose           Say more about what is happening
      --tag list          Only these tags
      --timeout duration  [$MIZU_TIMEOUT]
  -h, --help              Show what this command takes
`
	if got := out.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestUsageLineShapes(t *testing.T) {
	for _, tt := range []struct {
		arg  Arg
		want string
	}{
		{Arg{Name: "tenant", Required: true}, "<tenant>"},
		{Arg{Name: "tenant"}, "[<tenant>]"},
		{Arg{Name: "file", Rest: true, Required: true}, "<file>..."},
		{Arg{Name: "file", Rest: true}, "[<file>...]"},
	} {
		if got := argShape(tt.arg); got != tt.want {
			t.Errorf("%+v: got %s, want %s", tt.arg, got, tt.want)
		}
	}
}

func TestFlagKinds(t *testing.T) {
	var (
		s    string
		n    int
		u    uint
		f    float64
		d    time.Duration
		t0   time.Time
		mode string
		list []string
		kv   map[string]string
		b    bool
		c    int
	)
	for _, tt := range []struct {
		value Value
		want  string
	}{
		{String(&s), "string"},
		{Int(&n), "int"},
		{Uint(&u), "uint"},
		{Float(&f), "float"},
		{Duration(&d), "duration"},
		{Time(&t0), "time"},
		{Enum(&mode, "fast", "slow"), "fast|slow"},
		{Strings(&list, ","), "list"},
		{KeyValues(&kv), "key=value"},
		{Text(&half{}), "value"},
		{Var(&n, parseInt), "value"},
		{Bool(&b), ""},
		{Count(&c), ""},
	} {
		if got := kind(tt.value); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

// plainValue implements Value and nothing else, which is what a type written
// against the standard library's flag package looks like.
type plainValue struct{}

func (plainValue) String() string   { return "" }
func (plainValue) Set(string) error { return nil }

func TestAValueThatSaysNothingTakesAValue(t *testing.T) {
	if got := kind(plainValue{}); got != "value" {
		t.Errorf("got %q, want value", got)
	}
}

func TestRequiredFlagSaysSo(t *testing.T) {
	var name string
	rows := flagRows(Spec{Flags: []Flag{
		{Name: "name", Desc: "Who to greet", Required: true, Value: String(&name)},
	}})

	want := row{"    --name string", "Who to greet (required)"}
	if rows[0] != want {
		t.Errorf("got %+v, want %+v", rows[0], want)
	}
}

func TestAFlagWithNoDescriptionStillLinesUp(t *testing.T) {
	var name string
	rows := flagRows(Spec{Flags: []Flag{
		{Name: "name", Env: "MIZU_NAME", Value: String(&name)},
	}})

	want := row{"    --name string", "[$MIZU_NAME]"}
	if rows[0] != want {
		t.Errorf("got %+v, want %+v", rows[0], want)
	}
}
