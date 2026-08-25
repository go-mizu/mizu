package mizutest

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/go-mizu/mizu"
)

// echoApp answers every route with what it received, so a test about the
// request can read the request back out of the response.
func echoApp(t *testing.T, opts ...Option) *App {
	t.Helper()

	app := NewApp(t, opts...)
	echo := func(c *mizu.Ctx) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]any{
			"method": c.Request().Method,
			"path":   c.Request().URL.Path,
			"query":  c.Request().URL.RawQuery,
			"header": c.Request().Header,
			"body":   string(body),
			"remote": c.Request().RemoteAddr,
		})
	}
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "PURGE"} {
		app.Routes().Handle(m, "/echo", echo)
	}
	return app
}

func TestEveryMethodHasABuilder(t *testing.T) {
	app := echoApp(t)

	tests := map[string]*Request{
		"GET":     app.Get("/echo"),
		"POST":    app.Post("/echo"),
		"PUT":     app.Put("/echo"),
		"PATCH":   app.Patch("/echo"),
		"DELETE":  app.Delete("/echo"),
		"OPTIONS": app.Options("/echo"),
		"PURGE":   app.Request("PURGE", "/echo"),
	}
	for want, req := range tests {
		t.Run(want, func(t *testing.T) {
			req.Do().AssertOK().AssertJSONPath("$.method", want)
		})
	}
}

// TestHeadHasNoBody is separate because a HEAD response has none by definition,
// so the echo above has nothing to read back.
func TestHeadHasNoBody(t *testing.T) {
	app := echoApp(t)

	res := app.Head("/echo").Do().AssertOK()
	if len(res.Body()) != 0 {
		t.Errorf("HEAD came back with %d bytes of body", len(res.Body()))
	}
}

func TestJSONSetsTheBodyAndTheContentType(t *testing.T) {
	app := echoApp(t)

	app.Post("/echo").JSON(map[string]any{"title": "Hello"}).Do().
		AssertJSONPath("$.body", `{"title":"Hello"}`).
		AssertJSONPath(`$.header["Content-Type"][0]`, "application/json")
}

// TestJSONTakesBytesAsWritten is what lets a test send exactly the document it
// means, including one a Go value could not produce.
func TestJSONTakesBytesAsWritten(t *testing.T) {
	app := echoApp(t)

	for name, body := range map[string]any{
		"a string":       `{"a":  1}`,
		"bytes":          []byte(`{"a":  1}`),
		"a raw message":  json.RawMessage(`{"a":  1}`),
		"nothing at all": nil,
	} {
		t.Run(name, func(t *testing.T) {
			want := `{"a":  1}` // the extra space says it was not re-encoded
			if body == nil {
				want = "null"
			}
			app.Post("/echo").JSON(body).Do().AssertJSONPath("$.body", want)
		})
	}
}

func TestJSONReportsAValueItCannotEncode(t *testing.T) {
	app, r := fake(t)

	app.Post("/echo").JSON(make(chan int)).Do()

	r.says(t, "encoding the JSON body")
}

func TestFormSetsTheBodyAndTheContentType(t *testing.T) {
	app := echoApp(t)

	app.Post("/echo").Form(url.Values{"title": {"Hello"}, "tag": {"go", "web"}}).Do().
		AssertJSONPath("$.body", "tag=go&tag=web&title=Hello").
		AssertJSONPath(`$.header["Content-Type"][0]`, "application/x-www-form-urlencoded")
}

func TestMultipartBuildsAForm(t *testing.T) {
	app := NewApp(t)
	app.Routes().Post("/upload", func(c *mizu.Ctx) error {
		form, cleanup, err := c.MultipartForm(1 << 20)
		if err != nil {
			return err
		}
		defer cleanup()

		f, err := form.File["avatar"][0].Open()
		if err != nil {
			return err
		}
		defer f.Close()

		data, err := io.ReadAll(f)
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]any{
			"name":  form.File["avatar"][0].Filename,
			"bytes": len(data),
			"note":  form.Value["note"][0],
		})
	})

	app.Post("/upload").Multipart(func(w *multipart.Writer) error {
		if err := w.WriteField("note", "my face"); err != nil {
			return err
		}
		f, err := w.CreateFormFile("avatar", "me.png")
		if err != nil {
			return err
		}
		_, err = f.Write(make([]byte, 128))
		return err
	}).Do().
		AssertOK().
		AssertJSONPath("$.name", "me.png").
		AssertJSONPath("$.bytes", 128).
		AssertJSONPath("$.note", "my face")
}

func TestMultipartReportsAFailureBuildingIt(t *testing.T) {
	app, r := fake(t)

	app.Post("/upload").Multipart(func(*multipart.Writer) error { return errBoom }).Do()

	r.says(t, "building the multipart body", "boom")
}

func TestBodyReadsFromAReader(t *testing.T) {
	app := echoApp(t)

	app.Post("/echo").Body(strings.NewReader("raw bytes")).Do().
		AssertJSONPath("$.body", "raw bytes")
}

func TestBodyReportsAReaderThatFails(t *testing.T) {
	app, r := fake(t)

	app.Post("/echo").Body(failingReader{}).Do()

	r.says(t, "reading the body", "boom")
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errBoom }

// TestRawSendsWhatItIsGiven is for the tests that are about what the server
// does with something malformed, where a helper that fixes it up first would
// defeat the point.
func TestRawSendsWhatItIsGiven(t *testing.T) {
	app := echoApp(t)

	app.Post("/echo").Raw([]byte("{not json")).Do().
		AssertJSONPath("$.body", "{not json").
		AssertJSONMissing(`$.header["Content-Type"]`)
}

func TestHeadersSetAddAndReplace(t *testing.T) {
	app := echoApp(t)

	app.Get("/echo").
		WithHeader("X-One", "first").
		WithHeader("X-One", "second").
		AddHeader("X-Many", "a").
		AddHeader("X-Many", "b").
		WithHeaders(http.Header{"X-From-Map": {"yes"}}).
		Do().
		AssertJSONPath(`$.header["X-One"]`, []string{"second"}).
		AssertJSONPath(`$.header["X-Many"]`, []string{"a", "b"}).
		AssertJSONPath(`$.header["X-From-Map"][0]`, "yes")
}

func TestWithQueryAddsParameters(t *testing.T) {
	app := echoApp(t)

	app.Get("/echo").WithQuery("page", "2").WithQuery("tag", "go").WithQuery("tag", "web").Do().
		AssertJSONPath("$.query", "page=2&tag=go&tag=web")
}

// TestWithQueryKeepsAQueryAlreadyOnThePath matters because both spellings turn
// up in real tests and one silently dropping the other would be a bad surprise.
func TestWithQueryKeepsAQueryAlreadyOnThePath(t *testing.T) {
	app := echoApp(t)

	app.Get("/echo?sort=desc").WithQuery("page", "2").Do().
		AssertJSONPath("$.query", "sort=desc&page=2")
}

func TestWithCookieSendsCookies(t *testing.T) {
	app := NewApp(t)
	app.Routes().Get("/cookies", func(c *mizu.Ctx) error {
		locale, err := c.Cookie("locale")
		if err != nil {
			return err
		}
		return c.Text(http.StatusOK, locale.Value)
	})

	app.Get("/cookies").WithCookie("locale", "ja").Do().AssertSee("ja")
	app.Get("/cookies").WithCookieValue(&http.Cookie{Name: "locale", Value: "fr"}).Do().AssertSee("fr")
}

func TestWithIPSetsTheRemoteAddress(t *testing.T) {
	app := echoApp(t)

	app.Get("/echo").WithIP("203.0.113.7").Do().
		AssertJSONPath("$.remote", "203.0.113.7:12345")
	app.Get("/echo").WithIP("203.0.113.7:443").Do().
		AssertJSONPath("$.remote", "203.0.113.7:443")
}

// TestABadTargetIsReportedAtDo is the rule the builder is built around: a
// mistake anywhere in the chain surfaces in one place, so no method in the
// middle of a chain has to be checked for an error.
func TestABadTargetIsReportedAtDo(t *testing.T) {
	app, r := fake(t)

	app.Get("nowhere").Do()

	r.says(t, "GET nowhere")
}
