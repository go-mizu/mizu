package mizutest

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

func TestDumpPrintsTheWholeExchange(t *testing.T) {
	app, r := fake(t)
	app.Routes().Post("/posts", func(c *mizu.Ctx) error {
		c.Logger().Info("creating a post", "title", "Hello")
		c.Header().Set("Location", "/posts/1")
		return c.JSON(http.StatusCreated, map[string]any{"id": 1})
	})

	app.Post("/posts").JSON(map[string]any{"title": "Hello"}).
		WithHeader("Accept", "application/json").Do().Dump()

	for _, want := range []string{
		"> POST /posts",
		"> Accept: application/json",
		"> Content-Type: application/json",
		`> {"title":"Hello"}`,
		"< 201 Created",
		"< Location: /posts/1",
		`< {"id":1}`,
	} {
		if !r.logged(want) {
			t.Errorf("the dump does not hold %q, and is:\n%s", want, strings.Join(r.logs, "\n"))
		}
	}
}

// TestDumpsReturnTheResponse is what lets one go in the middle of a chain while
// something is being worked out, and come back out again.
func TestDumpsReturnTheResponse(t *testing.T) {
	app := serve(t, document(`{"id":1}`))

	res := app.Get("/").Do()
	for name, dump := range map[string]func(*Response) *Response{
		"Dump":        (*Response).Dump,
		"DumpHeaders": (*Response).DumpHeaders,
		"DumpJSON":    (*Response).DumpJSON,
		"DumpLogs":    (*Response).DumpLogs,
	} {
		if got := dump(res); got != res {
			t.Errorf("%s returned %p, want the response %p", name, got, res)
		}
	}
}

func TestDumpHeadersLeavesTheBodiesOut(t *testing.T) {
	app, r := fake(t)
	app.Routes().Post("/", func(c *mizu.Ctx) error {
		return c.Text(http.StatusOK, "the response body")
	})

	app.Post("/").Raw([]byte("the request body")).Do().DumpHeaders()

	if !r.logged("< 200 OK") {
		t.Errorf("the dump has no status line, and is:\n%s", strings.Join(r.logs, "\n"))
	}
	for _, body := range []string{"the request body", "the response body"} {
		if r.logged(body) {
			t.Errorf("DumpHeaders printed %q", body)
		}
	}
}

func TestDumpJSONIndentsAndSorts(t *testing.T) {
	app, r := fake(t)
	app.Routes().Get("/", document(`{"title":"a title long enough that this document does not fit on one line","id":1}`))

	app.Get("/").Do().DumpJSON()

	if !r.logged("{\n  \"id\": 1,\n  \"title\":") {
		t.Errorf("DumpJSON printed:\n%s", strings.Join(r.logs, "\n"))
	}
}

func TestDumpJSONSaysWhenTheBodyIsNotJSON(t *testing.T) {
	app, r := fake(t)
	app.Routes().Get("/", body("<html>not json</html>"))

	app.Get("/").Do().DumpJSON()

	if !r.logged("the body is not JSON") || !r.logged("<html>not json</html>") {
		t.Errorf("DumpJSON printed:\n%s", strings.Join(r.logs, "\n"))
	}
	r.passed(t)
}

func TestDumpLogsPrintsEveryEntry(t *testing.T) {
	app, r := fake(t)
	app.Routes().Get("/", func(c *mizu.Ctx) error {
		c.Logger().Debug("looking it up", "id", 1)
		c.Logger().Warn("the cache is cold")
		return c.Text(http.StatusOK, "ok")
	})

	app.Get("/").Do().DumpLogs()

	for _, want := range []string{"DEBUG looking it up id=1", "WARN the cache is cold"} {
		if !r.logged(want) {
			t.Errorf("DumpLogs does not hold %q, and printed:\n%s", want, strings.Join(r.logs, "\n"))
		}
	}
}

// TestDumpLogsSaysWhenThereIsNothing serves a handler of its own, because a
// fixture over a router has the request logger in front of it and so never
// reaches an empty log.
func TestDumpLogsSaysWhenThereIsNothing(t *testing.T) {
	app, r := fake(t, Serve(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	})))

	app.Get("/").Do().DumpLogs()

	if !r.logged("the handler logged nothing") {
		t.Errorf("DumpLogs printed:\n%s", strings.Join(r.logs, "\n"))
	}
}

// TestLogsAreThisRequestsAlone is what makes an assertion about logging mean
// anything in a test that makes more than one request.
func TestLogsAreThisRequestsAlone(t *testing.T) {
	app := NewApp(t)
	app.Routes().Get("/{n}", func(c *mizu.Ctx) error {
		c.Logger().Info("handling", "n", c.Param("n"))
		return c.Text(http.StatusOK, "ok")
	})

	first := app.Get("/1").Do()
	second := app.Get("/2").Do()

	for _, tt := range []struct {
		res  *Response
		want string
	}{{first, "1"}, {second, "2"}} {
		var found int
		for _, e := range tt.res.Logs() {
			if e.Message == "handling" {
				found++
				if got := fmt.Sprint(e.Attrs["n"]); got != tt.want {
					t.Errorf("the entry says n=%s, want n=%s", got, tt.want)
				}
			}
		}
		if found != 1 {
			t.Errorf("the response carries %d handling entries, want 1", found)
		}
	}
}

func TestLogsAreACopy(t *testing.T) {
	app := serve(t, body("ok"))

	res := app.Get("/").Do()
	logs := res.Logs()
	if len(logs) == 0 {
		t.Fatal("the request logged nothing, so there is nothing to copy")
	}
	logs[0].Message = "changed"

	if got := res.Logs()[0].Message; got == "changed" {
		t.Error("Logs handed out the slice it keeps")
	}
}

// TestTheExchangeShowsErrorsAndCountsTheRest is the shape of every failure
// message this package prints. The reason for an unexpected 500 is nearly
// always an error entry, and the request log line beside it is noise.
func TestTheExchangeShowsErrorsAndCountsTheRest(t *testing.T) {
	for _, quiet := range []int{0, 1, 3} {
		t.Run(fmt.Sprintf("%d quiet entries", quiet), func(t *testing.T) {
			app, _ := fake(t)
			app.Routes().Get("/", func(c *mizu.Ctx) error {
				for range quiet {
					c.Logger().Info("something worth saying")
				}
				c.Logger().Error("the thing that went wrong", "why", "boom")
				return c.Text(http.StatusOK, "ok")
			})

			res := app.Get("/").Do()

			// The request logger adds a line of its own, so what the count
			// should be is read off the response rather than assumed.
			var others int
			for _, e := range res.Logs() {
				if e.Level < slog.LevelError {
					others++
				}
			}
			want := fmt.Sprintf("log %d more entries below error level, see DumpLogs", others)
			switch others {
			case 0:
				want = "more entr" // which should not be there at all
			case 1:
				want = "log 1 more entry below error level, see DumpLogs"
			}

			got := res.exchange()
			if !strings.Contains(got, "log ERROR the thing that went wrong why=boom") {
				t.Errorf("the exchange does not show the error:\n%s", got)
			}
			if strings.Contains(got, want) != (others > 0) {
				t.Errorf("the exchange does not say %q for %d entries:\n%s", want, others, got)
			}
		})
	}
}

func TestALongBodyIsCutShort(t *testing.T) {
	app, r := fake(t)
	long := strings.Repeat("x", bodyLimit+500)
	app.Routes().Get("/", body(long))

	app.Get("/").Do().AssertStatus(http.StatusTeapot)

	msg := r.first()
	if strings.Contains(msg, strings.Repeat("x", bodyLimit+1)) {
		t.Error("the whole body was printed")
	}
	if !strings.Contains(msg, "... and 500 more bytes") {
		t.Errorf("the message does not say what was cut:\n%s", msg)
	}
}

// TestALongBodyIsCutAtARuneBoundary keeps the output from ending in half a
// character, which a terminal renders as a replacement box.
func TestALongBodyIsCutAtARuneBoundary(t *testing.T) {
	var b strings.Builder
	// Three bytes each, so the limit lands inside one of them.
	for b.Len() < bodyLimit+16 {
		b.WriteString("日")
	}

	var out strings.Builder
	writeBody(&out, "< ", []byte(b.String()))

	shown, _, _ := strings.Cut(strings.TrimPrefix(out.String(), "< "), "\n")
	if strings.ContainsRune(shown, '\uFFFD') {
		t.Errorf("the body was cut inside a character: %q", shown[len(shown)-8:])
	}
	if !strings.Contains(out.String(), "more bytes") {
		t.Errorf("the body was not cut at all:\n%s", out.String())
	}
}

func TestWriteBodyPrefixesEveryLine(t *testing.T) {
	var b strings.Builder
	writeBody(&b, "< ", []byte("first\nsecond\n"))

	const want = "< first\n< second\n"
	if got := b.String(); got != want {
		t.Errorf("writeBody gave %q, want %q", got, want)
	}

	b.Reset()
	writeBody(&b, "< ", nil)
	if got := b.String(); got != "" {
		t.Errorf("an empty body printed as %q", got)
	}
}

func TestIndent(t *testing.T) {
	tests := map[string]string{
		"one line":          "    one line",
		"one\ntwo":          "    one\n    two",
		"trailing\n":        "    trailing",
		"":                  "",
		"\n":                "",
		"blank\n\nbetween":  "    blank\n    \n    between",
		"trailing\n\n\n":    "    trailing",
		"leading\nspace  x": "    leading\n    space  x",
	}
	for in, want := range tests {
		if got := indent(in); got != want {
			t.Errorf("indent(%q) = %q, want %q", in, got, want)
		}
	}
}
