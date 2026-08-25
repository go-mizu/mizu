package consoletest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
)

// greet is the shortest command that still uses both streams.
type greet struct {
	Loud bool
	Name string
}

func (c *greet) Spec() console.Spec {
	return console.Spec{
		Name: "greet",
		Desc: "Say hello",
		Flags: []console.Flag{
			{Name: "loud", Short: 'l', Value: console.Bool(&c.Loud)},
		},
		Args: []console.Arg{
			{Name: "name", Default: "world", Value: console.String(&c.Name)},
		},
	}
}

func (c *greet) Run(ctx context.Context, out *console.IO) error {
	hello := "Hello, " + c.Name
	if c.Loud {
		hello = strings.ToUpper(hello)
	}
	out.Line(hello)
	out.Info("said hello")
	return nil
}

// setup asks all four kinds of question and writes down the answers.
type setup struct {
	Name  string
	DB    string
	Extra []string
	Tidy  bool
}

func (c *setup) Spec() console.Spec { return console.Spec{Name: "setup"} }

func (c *setup) Run(ctx context.Context, out *console.IO) error {
	databases := []string{"sqlite", "postgres", "mysql"}
	extras := []string{"queue", "cache", "mail"}

	name, err := out.Ask("Project name", "app")
	if err != nil {
		return err
	}
	c.Name = name

	db, err := out.Choice("Database", databases, 0)
	if err != nil {
		return err
	}
	c.DB = databases[db]

	picked, err := out.MultiChoice("What else", extras)
	if err != nil {
		return err
	}
	for _, i := range picked {
		c.Extra = append(c.Extra, extras[i])
	}

	c.Tidy, err = out.Confirm("Run go mod tidy", true)
	if err != nil {
		return err
	}
	out.Line(fmt.Sprintf("%s on %s with %v, tidy %v", c.Name, c.DB, c.Extra, c.Tidy))
	return nil
}

// fails is a command that goes wrong, for the exit codes and the dump.
type fails struct{ err error }

func (c *fails) Spec() console.Spec { return console.Spec{Name: "fails"} }

func (c *fails) Run(ctx context.Context, out *console.IO) error {
	out.Info("trying")
	return c.err
}

// echo is a command that reads its stdin instead of asking anything.
type echo struct{}

func (c *echo) Spec() console.Spec { return console.Spec{} }

func (c *echo) Run(ctx context.Context, out *console.IO) error {
	data, err := io.ReadAll(out.In())
	if err != nil {
		return err
	}
	out.Print("%s", data)
	return nil
}

// deaf keeps asking after the stream it reads has ended, which is what a
// command that drops the error from a prompt does.
type deaf struct{}

func (c *deaf) Spec() console.Spec { return console.Spec{Name: "deaf"} }

func (c *deaf) Run(ctx context.Context, out *console.IO) error {
	out.Ask("First", "")
	out.Ask("Second", "")
	return nil
}

func TestRunUsesTheFieldsItWasGiven(t *testing.T) {
	r := Run(t, &greet{Name: "Ada", Loud: true})

	r.AssertSuccess()
	r.AssertExitCode(console.CodeOK)
	r.AssertOutput("HELLO, ADA\n")
	r.AssertOutputContains("ADA")
	r.AssertErrorContains("said hello")
	if got := r.Prompts(); len(got) != 0 {
		t.Errorf("asked %v, want nothing", got)
	}
}

func TestArgsParseTheCommandLine(t *testing.T) {
	cmd := &greet{}
	r := Run(t, cmd, Args("-l", "Ada"))

	r.AssertSuccess()
	r.AssertOutput("HELLO, ADA\n")
	if !cmd.Loud || cmd.Name != "Ada" {
		t.Errorf("the command line left %+v", cmd)
	}
}

// An empty command line is not the same as no command line: it is parsed, so
// the defaults in the spec are applied.
func TestEmptyArgsApplyTheDefaults(t *testing.T) {
	cmd := &greet{Name: "Ada"}
	Run(t, cmd, Args()).AssertOutput("Hello, world\n")

	if cmd.Name != "world" {
		t.Errorf("the name is %q, want the default", cmd.Name)
	}
}

func TestArgsThatDoNotParse(t *testing.T) {
	r := Run(t, &greet{}, Args("--nope"))

	if err := r.AssertFailure(); err == nil {
		t.Fatal("a flag that does not exist parsed")
	}
	r.AssertExitCode(console.CodeUsage)
	r.AssertNoOutput()
	r.AssertErrorContains("--nope")
}

func TestAnsweringQuestions(t *testing.T) {
	cmd := &setup{}
	r := Run(t, cmd,
		Answer("Project name", "blog"),
		Choose("Database", "postgres"),
		ChooseAll("What else", "mail", "queue"),
		Confirm("Run go mod tidy", false),
	)

	r.AssertSuccess()
	r.AssertOutput("blog on postgres with [queue mail], tidy false\n")
	r.AssertAsked("Project name", "Database", "What else", "Run go mod tidy")

	if cmd.DB != "postgres" {
		t.Errorf("the database is %q", cmd.DB)
	}
}

// An empty answer is the user pressing enter, and a prompt with a default takes
// it.
func TestAnEmptyAnswerTakesTheDefault(t *testing.T) {
	r := Run(t, &setup{},
		Answer("Project name", ""),
		Choose("Database", "sqlite"),
		ChooseAll("What else"),
		Confirm("tidy", true),
	)

	r.AssertSuccess()
	r.AssertOutput("app on sqlite with [], tidy true\n")
}

func TestPromptsRecordWhatWasOnTheScreen(t *testing.T) {
	r := Run(t, &setup{},
		Answer("Project name", "blog"),
		Choose("Database", "mysql"),
		ChooseAll("What else"),
		Confirm("tidy", true),
	)

	got := r.Prompts()
	if len(got) != 4 {
		t.Fatalf("asked %d questions, want 4", len(got))
	}
	if got[0].Hint != "app" {
		t.Errorf("the first hint is %q, want the default", got[0].Hint)
	}
	if want := []string{"sqlite", "postgres", "mysql"}; !equal(got[1].Options, want) {
		t.Errorf("the database options are %q, want %q", got[1].Options, want)
	}
	if got[3].Hint != "Y/n" {
		t.Errorf("the confirmation hint is %q", got[3].Hint)
	}
	if s := got[0].String(); s != "Project name [app]" {
		t.Errorf("the prompt reads %q", s)
	}
	if s := (Prompt{Question: "Name"}).String(); s != "Name" {
		t.Errorf("a prompt with no hint reads %q", s)
	}
}

// Colour is off by default, and a test that turns it on still matches the
// questions, because the escapes come off before anything is compared.
func TestColouredPromptsStillMatch(t *testing.T) {
	r := Run(t, &setup{},
		With(console.Options{Color: console.ColorAlways}),
		Answer("Project name", "blog"),
		Choose("Database", "postgres"),
		ChooseAll("What else"),
		Confirm("tidy", true),
	)

	r.AssertSuccess()
	r.AssertAsked("Project name", "Database", "What else", "Run go mod tidy")
	if !strings.Contains(r.Stderr(), "\x1b[") {
		t.Error("nothing on stderr is coloured")
	}
}

func TestWithOptions(t *testing.T) {
	r := Run(t, &greet{Name: "Ada"}, With(console.Options{Verbosity: console.Quiet}))

	r.AssertOutput("Hello, Ada\n")
	r.AssertNoErrorOutput()
}

func TestInputIsReadInsteadOfPrompts(t *testing.T) {
	r := Run(t, &echo{}, Input("one\ntwo\n"))

	r.AssertSuccess()
	r.AssertOutput("one\ntwo\n")
	r.AssertNoErrorOutput()
	if got := r.Prompts(); got != nil {
		t.Errorf("recorded %v, want nothing", got)
	}
}

func TestContextIsHandedToTheCommand(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := Run(t, &waits{}, Context(ctx))

	if err := r.AssertFailure(); !errors.Is(err, context.Canceled) {
		t.Errorf("the command got %v", err)
	}
	r.AssertExitCode(console.CodeInterrupted)
}

// waits returns whatever its context has to say, which for a cancelled one is
// the cancellation.
type waits struct{}

func (c *waits) Spec() console.Spec { return console.Spec{Name: "waits"} }

func (c *waits) Run(ctx context.Context, out *console.IO) error { return ctx.Err() }

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
	}{
		{"a plain error", errors.New("no"), console.CodeFailure},
		{"a classified one", console.Exit(console.CodeConfig, errors.New("no")), console.CodeConfig},
		{"walking away", console.ErrAborted, console.CodeInterrupted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Run(t, &fails{err: tt.err})

			if err := r.AssertFailure(); !errors.Is(err, tt.err) {
				t.Errorf("the error is %v, want %v", err, tt.err)
			}
			r.AssertExitCode(tt.code)
			if r.Code() != tt.code {
				t.Errorf("the code is %d, want %d", r.Code(), tt.code)
			}
		})
	}
}

// A command with no name is still worth naming in a failure, since the test has
// to be told which of the two it wrote is failing.
func TestACommandWithNoName(t *testing.T) {
	tb := &recorder{}
	Run(tb, &echo{}, Input("x")).AssertNoOutput()

	if len(tb.msgs) != 1 || !strings.HasPrefix(tb.msgs[0], "the command: ") {
		t.Errorf("the failure reads %q", tb.msgs)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
