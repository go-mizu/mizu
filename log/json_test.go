package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"strings"
	"sync"
	"testing"
	"testing/slogtest"
	"time"

	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/errs"
)

// TestJSONSlogtest is the compliance suite in the standard library. It is the
// reason this handler can be handed to anything that takes a slog.Handler.
func TestJSONSlogtest(t *testing.T) {
	var buf bytes.Buffer
	h := NewJSONHandler(&buf, JSONOptions{Redact: []string{}})

	results := func() []map[string]any {
		var out []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				t.Fatalf("the handler wrote something that is not JSON: %v\n%s", err, line)
			}
			out = append(out, m)
		}
		return out
	}
	if err := slogtest.TestHandler(h, results); err != nil {
		t.Error(err)
	}
}

// jsonOf logs through a JSON handler and returns the one record it wrote.
func jsonOf(t *testing.T, o JSONOptions, f func(*slog.Logger)) string {
	t.Helper()
	var buf bytes.Buffer
	f(slog.New(&fixedTime{NewJSONHandler(&buf, o)}))
	return strings.TrimSuffix(buf.String(), "\n")
}

func TestJSON(t *testing.T) {
	got := jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
		log.Info("request", "method", "GET", "status", 200, "dur", 4200*time.Microsecond)
	})
	want := `{"time":"2026-08-24T10:44:02.113Z","level":"INFO","msg":"request","method":"GET","status":200,"dur":4200000}`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestJSONOrder(t *testing.T) {
	got := jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
		log.With("service", "api").Error("failed", "z", 1, "a", 2)
	})
	want := `{"time":"2026-08-24T10:44:02.113Z","level":"ERROR","msg":"failed","service":"api","z":1,"a":2}`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestJSONGroups(t *testing.T) {
	cases := []struct {
		name string
		log  func(*slog.Logger)
		want string
	}{
		{
			"a group in the record",
			func(log *slog.Logger) { log.Info("m", slog.Group("db", "rows", 12)) },
			`"db":{"rows":12}`,
		},
		{
			"a group on the logger",
			func(log *slog.Logger) { log.WithGroup("db").Info("m", "rows", 12) },
			`"db":{"rows":12}`,
		},
		{
			"two groups on the logger",
			func(log *slog.Logger) { log.WithGroup("a").WithGroup("b").Info("m", "n", 1) },
			`"a":{"b":{"n":1}}`,
		},
		{
			"attributes before and inside a group",
			func(log *slog.Logger) { log.With("x", 1).WithGroup("g").With("y", 2).Info("m", "z", 3) },
			`"x":1,"g":{"y":2,"z":3}`,
		},
		{
			"a group nobody put anything in",
			func(log *slog.Logger) { log.WithGroup("empty").Info("m") },
			`"msg":"m"}`,
		},
		{
			"a group with an empty attribute in it",
			func(log *slog.Logger) { log.WithGroup("empty").Info("m", slog.Attr{}) },
			`"msg":"m"}`,
		},
		{
			"a group with no name",
			func(log *slog.Logger) { log.Info("m", slog.Group("", "n", 1)) },
			`"msg":"m","n":1}`,
		},
	}
	for _, c := range cases {
		got := jsonOf(t, JSONOptions{}, c.log)
		if !strings.Contains(got, c.want) {
			t.Errorf("%s:\ngot  %s\nwant it to contain %s", c.name, got, c.want)
		}
		if !json.Valid([]byte(got)) {
			t.Errorf("%s: wrote something that is not JSON: %s", c.name, got)
		}
	}
}

func TestJSONValues(t *testing.T) {
	cases := []struct {
		name string
		attr slog.Attr
		want string
	}{
		{"a string", slog.String("v", "x"), `"v":"x"`},
		{"a string with a quote", slog.String("v", `a"b`), `"v":"a\"b"`},
		{"a string with a backslash", slog.String("v", `a\b`), `"v":"a\\b"`},
		{"a string with a newline", slog.String("v", "a\nb"), `"v":"a\nb"`},
		{"a string with a carriage return", slog.String("v", "a\rb"), `"v":"a\rb"`},
		{"a string with a tab", slog.String("v", "a\tb"), `"v":"a\tb"`},
		{"a string with a control character", slog.String("v", "a\x01b"), `"v":"a\u0001b"`},
		{"a string with a rune in it", slog.String("v", "café"), `"v":"café"`},
		{"a string that is not UTF-8", slog.String("v", "a\xffb"), "\"v\":\"a\ufffdb\""},
		{"an angle bracket, unescaped", slog.String("v", "<a>"), `"v":"<a>"`},
		{"an integer", slog.Int("v", -3), `"v":-3`},
		{"an unsigned integer", slog.Uint64("v", 3), `"v":3`},
		{"a float", slog.Float64("v", 1.5), `"v":1.5`},
		{"a float that is not a number", slog.Float64("v", math.NaN()), `"v":"NaN"`},
		{"a float that is too big", slog.Float64("v", math.Inf(1)), `"v":"+Inf"`},
		{"a float that is too small", slog.Float64("v", math.Inf(-1)), `"v":"-Inf"`},
		{"a boolean", slog.Bool("v", true), `"v":true`},
		{"a duration", slog.Duration("v", time.Second), `"v":1000000000`},
		{"a time", slog.Time("v", at), `"v":"2026-08-24T10:44:02.113Z"`},
		{"a struct", slog.Any("v", struct{ N int }{3}), `"v":{"N":3}`},
		{"a slice", slog.Any("v", []int{1, 2}), `"v":[1,2]`},
		{"nothing", slog.Any("v", nil), `"v":null`},
		{"something that will not marshal", slog.Any("v", func() {}), `"v":"`},
	}
	for _, c := range cases {
		got := jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
			log.LogAttrs(context.Background(), slog.LevelInfo, "m", c.attr)
		})
		if !strings.Contains(got, c.want) {
			t.Errorf("%s:\ngot  %s\nwant it to contain %s", c.name, got, c.want)
		}
		if !json.Valid([]byte(got)) {
			t.Errorf("%s: wrote something that is not JSON: %s", c.name, got)
		}
	}
}

// TestJSONNonUTF8Key checks that a key nobody sanitised cannot break the line,
// since a key comes from a program and a program can pass anything.
func TestJSONNonUTF8Key(t *testing.T) {
	got := jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
		log.Info("m", "a\xffb\"c", 1)
	})
	if !json.Valid([]byte(got)) {
		t.Errorf("wrote something that is not JSON: %s", got)
	}
}

func TestJSONRedaction(t *testing.T) {
	got := jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
		log.Info("login", "user", "kim", "Password", "hunter2")
	})
	if strings.Contains(got, "hunter2") {
		t.Errorf("a secret was written: %s", got)
	}
	if !strings.Contains(got, `"Password":"[redacted]"`) {
		t.Errorf("got %s", got)
	}
}

func TestJSONError(t *testing.T) {
	err := errs.New(errs.Unavailable, "mail.down", "the mail server is unavailable")
	got := jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
		log.Error("job failed", "err", err)
	})
	want := `"err":"the mail server is unavailable","err_kind":"unavailable","err_code":"mail.down"`
	if !strings.Contains(got, want) {
		t.Errorf("got  %s\nwant it to contain %s", got, want)
	}
}

func TestJSONUnclassifiedError(t *testing.T) {
	got := jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
		log.Error("failed", "err", errors.New("connection refused"))
	})
	if strings.Contains(got, "err_kind") {
		t.Errorf("an unclassified error was given a kind: %s", got)
	}
}

// TestJSONErrorKeyIsWhateverItWasCalled checks that the expansion follows the
// key rather than assuming everything is called err.
func TestJSONErrorKeyIsWhateverItWasCalled(t *testing.T) {
	got := jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
		log.Error("failed", "cause", errs.NotFoundf("no such post"))
	})
	if !strings.Contains(got, `"cause":"no such post","cause_kind":"not_found"`) {
		t.Errorf("got %s", got)
	}
}

func TestJSONStack(t *testing.T) {
	got := jsonOf(t, JSONOptions{Stack: true}, func(log *slog.Logger) {
		log.Error("job failed", "err", boom())
	})
	var record map[string]any
	if err := json.Unmarshal([]byte(got), &record); err != nil {
		t.Fatalf("%v: %s", err, got)
	}
	stack, ok := record["stack"].(string)
	if !ok {
		t.Fatalf("there is no stack in %s", got)
	}
	first := strings.Split(stack, "\n")[0]
	if !strings.HasSuffix(first, "github.com/go-mizu/mizu/log.boom") {
		t.Errorf("the stack starts at %q", first)
	}
}

func TestJSONNoStack(t *testing.T) {
	for _, c := range []struct {
		name string
		o    JSONOptions
		log  func(*slog.Logger)
	}{
		{"not asked for", JSONOptions{}, func(log *slog.Logger) { log.Error("m", "err", boom()) }},
		{"below error level", JSONOptions{Stack: true}, func(log *slog.Logger) { log.Warn("m", "err", boom()) }},
		{"no stack in the error", JSONOptions{Stack: true}, func(log *slog.Logger) {
			log.Error("m", "err", errors.New("plain"))
		}},
	} {
		if got := jsonOf(t, c.o, c.log); strings.Contains(got, `"stack"`) {
			t.Errorf("%s: wrote a stack: %s", c.name, got)
		}
	}
}

func TestJSONSource(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(&fixedTime{NewJSONHandler(&buf, JSONOptions{AddSource: true})})
	log.Info("here")

	if !strings.Contains(buf.String(), `"source":"`) || !strings.Contains(buf.String(), "json_test.go:") {
		t.Errorf("got %s", buf.String())
	}
}

func TestJSONContextData(t *testing.T) {
	ctx := ctxdata.With(context.Background(), tenantID, "acme")
	ctx = ctxdata.With(ctx, apiKey, "sk_live_7f3a")
	ctx = ctxdata.With(ctx, attempt, 3)

	got := jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
		log.InfoContext(ctx, "rebuilding")
	})
	want := `"msg":"rebuilding","tenant_id":"acme","api_key":"[redacted]"}`
	if !strings.Contains(got, want) {
		t.Errorf("got  %s\nwant it to contain %s", got, want)
	}
}

// TestJSONContextDataInsideAGroup records what happens to context data on a
// logger that is inside a group. The data is the record's, not the group's, so
// it goes in the object at the top.
func TestJSONContextDataInsideAGroup(t *testing.T) {
	ctx := ctxdata.With(context.Background(), tenantID, "acme")
	got := jsonOf(t, JSONOptions{}, func(log *slog.Logger) {
		log.WithGroup("db").InfoContext(ctx, "query", "rows", 12)
	})
	want := `"msg":"query","tenant_id":"acme","db":{"rows":12}}`
	if !strings.Contains(got, want) {
		t.Errorf("got  %s\nwant it to contain %s", got, want)
	}
}

func TestJSONLevel(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(NewJSONHandler(&buf, JSONOptions{Level: slog.LevelWarn}))
	log.Info("quiet")
	log.Warn("loud")

	if strings.Contains(buf.String(), "quiet") || !strings.Contains(buf.String(), "loud") {
		t.Errorf("got %s", buf.String())
	}
}

func TestJSONConcurrent(t *testing.T) {
	var buf syncBuffer
	log := slog.New(NewJSONHandler(&buf, JSONOptions{}))

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
		t.Fatalf("wrote %d lines, want 400", len(lines))
	}
	for _, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Fatalf("two records landed on one line: %s", line)
		}
	}
}
