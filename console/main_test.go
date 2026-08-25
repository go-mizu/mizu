package console

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

// says is a command that reports what it was given, so a test can see what the
// global flags left behind and what the IO was built with.
type says struct {
	args []string
	err  error

	sawQuiet   bool
	sawJSON    bool
	sawColor   bool
	sawTimeout bool
}

func (c *says) Spec() Spec {
	return Spec{
		Name: "say",
		Desc: "Say what it was given",
		Args: []Arg{{Name: "word", Rest: true, Value: Strings(&c.args, "")}},
	}
}

func (c *says) Run(ctx context.Context, io *IO) error {
	c.sawQuiet = io.Verbosity() == Quiet
	c.sawJSON = io.JSONMode()
	c.sawColor = io.colorOut
	_, c.sawTimeout = ctx.Deadline()
	io.Line(strings.Join(c.args, " "))
	return c.err
}

// start runs a command line the way a process would, and hands back the exit
// code and what landed on each stream.
func start(a *App, argv ...string) (int, string, string) {
	var out, err strings.Builder
	code := a.Start(context.Background(), strings.NewReader(""), &out, &err, argv)
	return code, out.String(), err.String()
}

func TestStartRunsAndExitsZero(t *testing.T) {
	a := &App{Name: "mizu"}
	a.Add(&says{})

	code, out, _ := start(a, "say", "hello", "there")
	if code != CodeOK {
		t.Errorf("exit %d, want 0", code)
	}
	if out != "hello there\n" {
		t.Errorf("stdout is %q", out)
	}
}

func TestGlobalFlagsAreTakenOutOfTheCommandLine(t *testing.T) {
	for _, argv := range [][]string{
		{"--json", "say", "hello"},
		{"say", "--json", "hello"},
		{"say", "hello", "--json"},
	} {
		cmd := &says{}
		a := &App{Name: "mizu"}
		a.Add(cmd)

		if code, _, _ := start(a, argv...); code != CodeOK {
			t.Fatalf("%v exited %d", argv, code)
		}
		if !cmd.sawJSON {
			t.Errorf("%v did not switch the IO to JSON", argv)
		}
		if len(cmd.args) != 1 || cmd.args[0] != "hello" {
			t.Errorf("%v left the command with %q", argv, cmd.args)
		}
	}
}

func TestGlobalFlagsShapeTheIO(t *testing.T) {
	for _, tt := range []struct {
		argv  []string
		check func(*says) bool
	}{
		{[]string{"--quiet", "say"}, func(c *says) bool { return c.sawQuiet }},
		{[]string{"-q", "say"}, func(c *says) bool { return c.sawQuiet }},
		{[]string{"--color", "always", "say"}, func(c *says) bool { return c.sawColor }},
		{[]string{"--color=always", "say"}, func(c *says) bool { return c.sawColor }},
		{[]string{"--color=always", "--no-color", "say"}, func(c *says) bool { return !c.sawColor }},
		{[]string{"--timeout", "1h", "say"}, func(c *says) bool { return c.sawTimeout }},
		{[]string{"say"}, func(c *says) bool { return !c.sawTimeout }},
	} {
		cmd := &says{}
		a := &App{Name: "mizu"}
		a.Add(cmd)

		if code, _, errOut := start(a, tt.argv...); code != CodeOK {
			t.Fatalf("%v exited %d: %s", tt.argv, code, errOut)
		}
		if !tt.check(cmd) {
			t.Errorf("%v did not take", tt.argv)
		}
	}
}

func TestQuietBeatsVerbose(t *testing.T) {
	g := Globals{Verbose: 2, Quiet: true}
	if got := g.Options().Verbosity; got != Quiet {
		t.Errorf("verbosity is %d, want quiet", got)
	}
}

func TestAGlobalFlagAfterTheDoubleDashIsTheCommandsWord(t *testing.T) {
	cmd := &says{}
	a := &App{Name: "mizu"}
	a.Add(cmd)

	code, _, _ := start(a, "say", "--", "--json")
	if code != CodeOK {
		t.Fatalf("exit %d", code)
	}
	if cmd.sawJSON {
		t.Error("a word after -- was read as a global flag")
	}
	if len(cmd.args) != 1 || cmd.args[0] != "--json" {
		t.Errorf("the command got %q", cmd.args)
	}
}

// clusters has its own -f, which a global -v must not be pulled out of.
type clusters struct {
	force   bool
	verbose int
}

func (c *clusters) Spec() Spec {
	return Spec{
		Name: "build",
		Flags: []Flag{
			{Name: "force", Short: 'f', Value: Bool(&c.force)},
			{Name: "loud", Short: 'v', Value: Count(&c.verbose)},
		},
	}
}

func (c *clusters) Run(ctx context.Context, io *IO) error { return nil }

func TestAClusterIsGlobalOnlyWhenAllOfItIs(t *testing.T) {
	cmd := &clusters{}
	a := &App{Name: "mizu"}
	a.Add(cmd)

	// -vf is not all global, so it stays whole and the command reads both
	// letters, including its own -v.
	if code, _, errOut := start(a, "build", "-vf"); code != CodeOK {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if !cmd.force || cmd.verbose != 1 {
		t.Errorf("force %v verbose %d, want true and 1", cmd.force, cmd.verbose)
	}
}

func TestAGlobalClusterIsTaken(t *testing.T) {
	cmd := &says{}
	a := &App{Name: "mizu"}
	a.Add(cmd)

	if code, _, errOut := start(a, "say", "-qn"); code != CodeOK {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if !cmd.sawQuiet {
		t.Error("-qn did not turn quiet on")
	}
	if len(cmd.args) != 0 {
		t.Errorf("the command got %q", cmd.args)
	}
}

func TestExitCodes(t *testing.T) {
	for _, tt := range []struct {
		name string
		argv []string
		err  error
		want int
	}{
		{"ok", []string{"say"}, nil, CodeOK},
		{"failure", []string{"say"}, errors.New("the disk is full"), CodeFailure},
		{"usage", []string{"sya"}, nil, CodeUsage},
		{"a flag that does not parse", []string{"--timeout", "soon", "say"}, nil, CodeUsage},
		{"aborted", []string{"say"}, ErrAborted, CodeInterrupted},
		{"cancelled", []string{"say"}, context.Canceled, CodeInterrupted},
		{"timed out", []string{"say"}, context.DeadlineExceeded, CodeFailure},
		{"config", []string{"say"}, Exit(CodeConfig, errors.New("APP_KEY is not set")), CodeConfig},
		{"wrapped", []string{"say"}, fmt.Errorf("boot: %w", Exit(CodeUnavailable, errors.New("no database"))), CodeUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			a := &App{Name: "mizu"}
			a.Add(&says{err: tt.err})

			if code, _, _ := start(a, tt.argv...); code != tt.want {
				t.Errorf("exit %d, want %d", code, tt.want)
			}
		})
	}
}

func TestAnAbortedCommandSaysNothing(t *testing.T) {
	a := &App{Name: "mizu"}
	a.Add(&says{err: ErrAborted})

	code, _, errOut := start(a, "say")
	if code != CodeInterrupted {
		t.Errorf("exit %d", code)
	}
	if errOut != "" {
		t.Errorf("it said %q, and the person who typed n knows what happened", errOut)
	}
}

func TestAnErrorIsPrinted(t *testing.T) {
	a := &App{Name: "mizu"}
	a.Add(&says{err: errors.New("the disk is full")})

	_, _, errOut := start(a, "say")
	if !strings.Contains(errOut, "error: the disk is full") {
		t.Errorf("got %q", errOut)
	}
}

func TestTheCauseChainNeedsAFlag(t *testing.T) {
	inner := errors.New("no such file")
	err := fmt.Errorf("reading config/app.toml: %w", inner)

	a := &App{Name: "mizu"}
	a.Add(&says{err: err})

	_, _, quiet := start(a, "say")
	if strings.Contains(quiet, "caused by") {
		t.Errorf("the chain printed without -v:\n%s", quiet)
	}

	a = &App{Name: "mizu"}
	a.Add(&says{err: err})
	_, _, loud := start(a, "-v", "say")
	if !strings.Contains(loud, "caused by: no such file") {
		t.Errorf("-v did not print the chain:\n%s", loud)
	}
}

func TestTimeoutCancelsTheCommand(t *testing.T) {
	cmd := &waits{}
	a := &App{Name: "mizu"}
	a.Add(cmd)

	code, _, _ := start(a, "--timeout", "1ns", "wait")
	if code != CodeFailure {
		t.Errorf("exit %d, want %d", code, CodeFailure)
	}
}

// waits runs until its context is done, which is what a queue worker or a
// migration on a large table looks like from here.
type waits struct{}

func (c *waits) Spec() Spec { return Spec{Name: "wait"} }

func (c *waits) Run(ctx context.Context, io *IO) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestStartHonoursAContextThatIsAlreadyDone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	a := &App{Name: "mizu"}
	a.Add(&waits{})

	var out, err strings.Builder
	if code := a.Start(ctx, strings.NewReader(""), &out, &err, []string{"wait"}); code != CodeInterrupted {
		t.Errorf("exit %d, want %d", code, CodeInterrupted)
	}
}

func TestHelpListsTheGlobalFlags(t *testing.T) {
	a := &App{Name: "mizu", Desc: "mizu builds and runs a mizu application."}
	a.Add(&says{})
	a.Globals = []Flag{{Name: "env", Desc: "Which environment to run as", Value: String(new(string))}}

	_, out, _ := start(a, "help")
	for _, want := range []string{"Global flags:", "-v, --verbose", "--env string", "Which environment to run as"} {
		if !strings.Contains(out, want) {
			t.Errorf("help does not mention %q:\n%s", want, out)
		}
	}

	_, out, _ = start(a, "help", "say")
	if !strings.Contains(out, "Global flags:") || !strings.Contains(out, "--no-interaction") {
		t.Errorf("a command's help does not list the global flags:\n%s", out)
	}
}

func TestRunAloneDoesNotClaimGlobalFlags(t *testing.T) {
	a := &App{Name: "mizu"}
	a.Add(&says{})

	var out, err strings.Builder
	c := New(strings.NewReader(""), &out, &err, Options{})
	if err := a.Run(context.Background(), c, []string{"help"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Global flags:") {
		t.Errorf("help offered flags that nothing parses:\n%s", out.String())
	}
}

func TestStrip(t *testing.T) {
	var g Globals
	globals := g.Flags()

	for _, tt := range []struct {
		argv  []string
		kept  []string
		taken []string
	}{
		{[]string{"say", "hello"}, []string{"say", "hello"}, nil},
		{[]string{"-v", "say"}, []string{"say"}, []string{"-v"}},
		{[]string{"say", "--timeout", "30s"}, []string{"say"}, []string{"--timeout", "30s"}},
		{[]string{"say", "--timeout=30s"}, []string{"say"}, []string{"--timeout=30s"}},
		{[]string{"say", "--no-json"}, []string{"say"}, []string{"--no-json"}},
		{[]string{"say", "-5"}, []string{"say", "-5"}, nil},
		{[]string{"say", "-"}, []string{"say", "-"}, nil},
		{[]string{"say", "--", "-v"}, []string{"say", "--", "-v"}, nil},
		{[]string{"say", "--timeout"}, []string{"say"}, []string{"--timeout"}},
	} {
		kept, taken := strip(globals, tt.argv)
		if !slices.Equal(kept, tt.kept) || !slices.Equal(taken, tt.taken) {
			t.Errorf("%v: kept %v taken %v, want kept %v taken %v", tt.argv, kept, taken, tt.kept, tt.taken)
		}
	}
}

// silent does nothing and says nothing, which is what a test that runs against
// the process's own stdout wants.
type silent struct{}

func (c *silent) Spec() Spec                            { return Spec{Name: "silent"} }
func (c *silent) Run(ctx context.Context, io *IO) error { return nil }

func TestMainRunsOnTheProcessStreams(t *testing.T) {
	a := &App{Name: "mizu"}
	a.Add(&silent{})

	if code := a.Main([]string{"silent"}); code != CodeOK {
		t.Errorf("exit %d, want 0", code)
	}
}

func TestExitError(t *testing.T) {
	inner := errors.New("APP_KEY is not set")
	err := Exit(CodeConfig, inner)

	if err.Error() != inner.Error() {
		t.Errorf("message is %q, want the one it wraps", err)
	}
	if !errors.Is(err, inner) {
		t.Error("the error it wraps is not reachable")
	}
}

func TestGlobalsCoverTheSpecifiedFlags(t *testing.T) {
	var g Globals
	names := make(map[string]bool)
	for _, f := range g.Flags() {
		names[f.Name] = true
	}
	// The set from the specification, less the ones that belong to the CLI
	// rather than to what a command says: --env, --config, --profile, --trace.
	for _, want := range []string{"verbose", "quiet", "json", "no-color", "no-interaction", "timeout"} {
		if !names[want] {
			t.Errorf("--%s is not a global flag", want)
		}
	}
}

func TestDurationGlobalIsUsable(t *testing.T) {
	var g Globals
	if err := Parse(g.Flags(), nil, []string{"--timeout", "90s"}); err != nil {
		t.Fatal(err)
	}
	if g.Timeout != 90*time.Second {
		t.Errorf("timeout is %v", g.Timeout)
	}
}
