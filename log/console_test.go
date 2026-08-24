package log

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/errs"
)

// at is the time every record in these tests was made, so that the output is
// the same on every run and can be written down.
var at = time.Date(2026, 8, 24, 10, 44, 2, 113_000_000, time.UTC)

// consoleOf logs through a console handler and returns what it wrote.
func consoleOf(t *testing.T, o ConsoleOptions, f func(*slog.Logger)) string {
	t.Helper()
	var buf bytes.Buffer
	if o.Color == ColorAuto {
		o.Color = ColorNever
	}
	f(slog.New(&fixedTime{NewConsoleHandler(&buf, o)}))
	return buf.String()
}

// fixedTime gives every record the same timestamp, since a test cannot compare
// output it cannot predict.
type fixedTime struct{ slog.Handler }

func (h *fixedTime) Handle(ctx context.Context, r slog.Record) error {
	r.Time = at
	return h.Handler.Handle(ctx, r)
}

func (h *fixedTime) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &fixedTime{h.Handler.WithAttrs(attrs)}
}

func (h *fixedTime) WithGroup(name string) slog.Handler {
	return &fixedTime{h.Handler.WithGroup(name)}
}

func TestConsole(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.Info("request", "method", "GET", "path", "/posts", "status", 200, "dur", 4200*time.Microsecond)
	})
	want := "10:44:02.113 INF request                      method=GET path=/posts status=200 dur=4.2ms\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestConsoleLevels(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.Debug("a")
		log.Info("b")
		log.Warn("c")
		log.Error("d")
		log.Log(context.Background(), slog.LevelWarn+1, "e")
	})
	for i, want := range []string{"DBG a", "INF b", "WRN c", "ERR d", "WRN e"} {
		line := strings.Split(got, "\n")[i]
		if !strings.Contains(line, want) {
			t.Errorf("line %d is %q, want it to contain %q", i, line, want)
		}
	}
}

func TestConsoleBelowLevel(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{Level: slog.LevelWarn}, func(log *slog.Logger) {
		log.Info("not this one")
		log.Warn("this one")
	})
	if strings.Contains(got, "not this one") {
		t.Errorf("a record below the level was written: %q", got)
	}
	if !strings.Contains(got, "this one") {
		t.Errorf("a record at the level was not written: %q", got)
	}
}

// TestConsoleLevelVar is the level a running process can turn up.
func TestConsoleLevelVar(t *testing.T) {
	var level slog.LevelVar
	level.Set(slog.LevelWarn)

	var buf bytes.Buffer
	log := slog.New(NewConsoleHandler(&buf, ConsoleOptions{Level: &level, Color: ColorNever}))
	log.Info("quiet")
	level.Set(slog.LevelInfo)
	log.Info("loud")

	if strings.Contains(buf.String(), "quiet") || !strings.Contains(buf.String(), "loud") {
		t.Errorf("the level did not change under the handler: %q", buf.String())
	}
}

func TestConsoleLongMessage(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.Info("a message longer than the column it is meant to line up in", "n", 1)
	})
	want := "10:44:02.113 INF a message longer than the column it is meant to line up in n=1\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestConsoleWithAndGroup(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.With("service", "api").WithGroup("db").Info("query", "rows", 12)
	})
	want := "10:44:02.113 INF query                        service=api db.rows=12\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// TestConsoleWithDoesNotShare is the copy on write. Two loggers built from the
// same one must not write into each other's attributes.
func TestConsoleWithDoesNotShare(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		base := log.With("service", "api")
		left := base.With("side", "left")
		right := base.With("side", "right")
		left.Info("one")
		right.Info("two")
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if !strings.HasSuffix(lines[0], "service=api side=left") {
		t.Errorf("the first line is %q", lines[0])
	}
	if !strings.HasSuffix(lines[1], "service=api side=right") {
		t.Errorf("the second line is %q", lines[1])
	}
}

func TestConsoleValues(t *testing.T) {
	cases := []struct {
		name string
		attr slog.Attr
		want string
	}{
		{"a string", slog.String("s", "x"), "s=x"},
		{"a string with a space", slog.String("s", "two words"), `s="two words"`},
		{"an empty string", slog.String("s", ""), `s=""`},
		{"a string with an equals sign", slog.String("s", "a=b"), `s="a=b"`},
		{"a string with a newline", slog.String("s", "a\nb"), `s="a\nb"`},
		{"a string with a control character", slog.String("s", "a\x07b"), `s="a\ab"`},
		{"an integer", slog.Int("n", -3), "n=-3"},
		{"an unsigned integer", slog.Uint64("n", 3), "n=3"},
		{"a float", slog.Float64("f", 1.5), "f=1.5"},
		{"a boolean", slog.Bool("b", true), "b=true"},
		{"a duration", slog.Duration("d", 812*time.Millisecond), "d=812ms"},
		{"a time", slog.Time("t", at), "t=2026-08-24T10:44:02.113Z"},
		{"anything else", slog.Any("v", []int{1, 2}), `v="[1 2]"`},
		{"nothing", slog.Any("v", nil), "v=<nil>"},
	}
	for _, c := range cases {
		got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
			log.LogAttrs(context.Background(), slog.LevelInfo, "m", c.attr)
		})
		if !strings.Contains(got, " "+c.want+"\n") {
			t.Errorf("%s: logged %q, want it to end with %q", c.name, got, c.want)
		}
	}
}

func TestConsoleEmptyAttrs(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.LogAttrs(context.Background(), slog.LevelInfo, "m",
			slog.Attr{},
			slog.Group("empty"),
			slog.Group("outer", slog.String("in", "x")),
		)
	})
	want := "10:44:02.113 INF m                            outer.in=x\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestConsoleRedaction(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.Info("login", "user", "kim", "Password", "hunter2", "token", "sk_live_7f3a")
	})
	if strings.Contains(got, "hunter2") || strings.Contains(got, "sk_live_7f3a") {
		t.Errorf("a secret was written: %q", got)
	}
	if !strings.Contains(got, "Password="+Mask) {
		t.Errorf("the key was not matched without regard to case: %q", got)
	}
}

func TestConsoleRedactNothing(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{Redact: []string{}}, func(log *slog.Logger) {
		log.Info("login", "password", "hunter2")
	})
	if !strings.Contains(got, "password=hunter2") {
		t.Errorf("an empty list masked something: %q", got)
	}
}

func TestConsoleRedactList(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{Redact: []string{"dsn"}}, func(log *slog.Logger) {
		log.Info("connecting", "dsn", "postgres://u:p@h/db", "password", "hunter2")
	})
	if strings.Contains(got, "postgres") {
		t.Errorf("the named key was written: %q", got)
	}
	if !strings.Contains(got, "password=hunter2") {
		t.Errorf("a list of one masked something else: %q", got)
	}
}

// TestConsoleLogValuer is the other half of redaction, where the value hides
// itself rather than relying on the handler being configured.
func TestConsoleLogValuer(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.Info("charging", "card", card("4111111111111111"))
	})
	if strings.Contains(got, "4111") {
		t.Errorf("a value that masks itself was written: %q", got)
	}
	if !strings.Contains(got, "card="+Mask) {
		t.Errorf("logged %q", got)
	}
}

type card string

func (c card) LogValue() slog.Value { return slog.StringValue(Mask) }

func TestConsoleError(t *testing.T) {
	err := errs.New(errs.Unavailable, "mail.down", "the mail server is unavailable")
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.Error("job failed", "job", "SendWelcome", "err", err)
	})
	for _, want := range []string{
		`err="the mail server is unavailable"`,
		"err_kind=unavailable",
		"err_code=mail.down",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("logged %q, want it to contain %q", got, want)
		}
	}
}

// TestConsoleUnclassifiedError checks that an error from a package that has
// never heard of errs is not labelled internal, since nobody decided that.
func TestConsoleUnclassifiedError(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.Error("job failed", "err", errors.New("connection refused"))
	})
	if strings.Contains(got, "err_kind") {
		t.Errorf("an unclassified error was given a kind: %q", got)
	}
	if !strings.Contains(got, `err="connection refused"`) {
		t.Errorf("logged %q", got)
	}
}

func TestConsoleStack(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.Error("job failed", "err", boom())
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Fatalf("wrote %d lines, want 2:\n%s", len(lines), got)
	}
	if !strings.HasPrefix(lines[1], strings.Repeat(" ", 13)+"└─ ") {
		t.Errorf("the stack line is %q", lines[1])
	}
	if !strings.HasSuffix(lines[1], "github.com/go-mizu/mizu/log.boom") {
		t.Errorf("the stack line points at %q", lines[1])
	}
}

func boom() error { return errs.Internalf("the disk went away") }

// TestConsoleNoStackBelowError is the level policy. A warning about something
// retryable is not a bug and nobody is going to read a stack for it.
func TestConsoleNoStackBelowError(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.Warn("retrying", "err", boom())
	})
	if strings.Contains(got, "└─") {
		t.Errorf("a warning printed a stack: %q", got)
	}
}

func TestConsoleNoStackWithoutOne(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.Error("failed", "err", errors.New("plain"))
	})
	if strings.Contains(got, "└─") {
		t.Errorf("an error with no stack printed one: %q", got)
	}
}

func TestConsoleContextData(t *testing.T) {
	ctx := ctxdata.With(context.Background(), tenantID, "acme")
	ctx = ctxdata.With(ctx, apiKey, "sk_live_7f3a")
	ctx = ctxdata.With(ctx, attempt, 3)

	got := consoleOf(t, ConsoleOptions{}, func(log *slog.Logger) {
		log.InfoContext(ctx, "rebuilding")
	})
	want := "10:44:02.113 INF rebuilding                   tenant_id=acme api_key=[redacted]\n"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

var (
	tenantID = ctxdata.NewKey[string]("tenant_id", ctxdata.Logged())
	apiKey   = ctxdata.NewKey[string]("api_key", ctxdata.Redacted())
	attempt  = ctxdata.NewKey[int]("attempt")
)

func TestConsoleSource(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(&fixedTime{NewConsoleHandler(&buf, ConsoleOptions{AddSource: true, Color: ColorNever})})
	log.Info("here")

	if !strings.Contains(buf.String(), "source=") || !strings.Contains(buf.String(), "console_test.go:") {
		t.Errorf("logged %q", buf.String())
	}
}

func TestConsoleColor(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{Color: ColorAlways}, func(log *slog.Logger) {
		log.Error("failed", "err", errors.New("plain"))
	})
	if !strings.Contains(got, "\033[31mERR\033[0m") {
		t.Errorf("the level is not red: %q", got)
	}
	if !strings.Contains(got, "\033[90merr=\033[0m") {
		t.Errorf("the key is not dim: %q", got)
	}
}

// TestConsoleColorIndent checks that the stack line lines up under a coloured
// record, where the time is longer in bytes than it is on the screen.
func TestConsoleColorIndent(t *testing.T) {
	got := consoleOf(t, ConsoleOptions{Color: ColorAlways}, func(log *slog.Logger) {
		log.Error("failed", "err", boom())
	})
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Fatalf("wrote %d lines, want 2", len(lines))
	}
	if !strings.HasPrefix(lines[1], strings.Repeat(" ", 13)) {
		t.Errorf("the stack line is %q", lines[1])
	}
}

func TestWantColor(t *testing.T) {
	var buf bytes.Buffer
	if wantColor(&buf, ColorAuto) {
		t.Error("a buffer is not a terminal")
	}
	if !wantColor(&buf, ColorAlways) {
		t.Error("ColorAlways said no")
	}
	if wantColor(&buf, ColorNever) {
		t.Error("ColorNever said yes")
	}

	// A file is an *os.File and still not a terminal, which is what a log
	// redirected into a file looks like.
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")
	f, err := os.Create(filepath.Join(t.TempDir(), "log"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if wantColor(f, ColorAuto) {
		t.Error("a file on disk is not a terminal")
	}

	t.Setenv("NO_COLOR", "1")
	if wantColor(nil, ColorAuto) {
		t.Error("NO_COLOR was ignored")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if wantColor(nil, ColorAuto) {
		t.Error("TERM=dumb was ignored")
	}
}

// TestConsoleConcurrent is for the race detector, and for the property that
// makes a log readable: one record is one write, so two goroutines never end up
// on the same line.
func TestConsoleConcurrent(t *testing.T) {
	var buf syncBuffer
	log := slog.New(NewConsoleHandler(&buf, ConsoleOptions{Color: ColorNever}))

	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				log.Info("working", "worker", i, "detail", strings.Repeat("x", 100))
			}
		}()
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 400 {
		t.Errorf("wrote %d lines, want 400", len(lines))
	}
	for _, line := range lines {
		if strings.Count(line, "INF") != 1 {
			t.Fatalf("two records landed on one line: %q", line)
		}
	}
}

// syncBuffer is a writer that can take concurrent writes, which the handler
// does not promise its writer will not get.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
