package web

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTextSendsPlainText(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return c.Text("hello")
	})
	if w.Code != http.StatusOK {
		t.Errorf("the status is %d, want 200", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("the body is %q, want hello", w.Body.String())
	}
	if want := "text/plain; charset=utf-8"; w.Header().Get("Content-Type") != want {
		t.Errorf("the content type is %q, want %q", w.Header().Get("Content-Type"), want)
	}
}

func TestStatusAppliesToTheNextWrite(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return c.Status(http.StatusCreated).Text("made")
	})
	if w.Code != http.StatusCreated {
		t.Errorf("the status is %d, want 201", w.Code)
	}
	if w.Body.String() != "made" {
		t.Errorf("the body is %q, want made", w.Body.String())
	}
}

func TestStatusWithNoWriteIsStillA200(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		c.Status(http.StatusTeapot)
		return nil
	})
	if w.Code != http.StatusOK {
		t.Errorf("the status is %d, want 200, since nothing wrote", w.Code)
	}
}

func TestStatusCodeReportsWhatWentOut(t *testing.T) {
	serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		if c.StatusCode() != 0 {
			t.Errorf("the status is %d before anything wrote", c.StatusCode())
		}
		if err := c.Status(http.StatusAccepted).Text("ok"); err != nil {
			return err
		}
		if c.StatusCode() != http.StatusAccepted {
			t.Errorf("the status is %d, want 202", c.StatusCode())
		}
		return nil
	})
}

func TestASecondStatusDoesNotChangeTheFirst(t *testing.T) {
	serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		w := c.Writer()
		w.WriteHeader(http.StatusCreated)
		w.WriteHeader(http.StatusTeapot)
		if c.StatusCode() != http.StatusCreated {
			t.Errorf("the status is %d, want 201", c.StatusCode())
		}
		return nil
	})
}

func TestBytesTakesTheContentTypeItIsGiven(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return c.Bytes("application/octet-stream", []byte{1, 2, 3})
	})
	if want := "application/octet-stream"; w.Header().Get("Content-Type") != want {
		t.Errorf("the content type is %q, want %q", w.Header().Get("Content-Type"), want)
	}
	if w.Body.Len() != 3 {
		t.Errorf("the body is %d bytes, want 3", w.Body.Len())
	}
}

func TestAContentTypeAlreadySetIsLeftAlone(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return c.SetHeader("Content-Type", "text/markdown").Text("# hello")
	})
	if want := "text/markdown"; w.Header().Get("Content-Type") != want {
		t.Errorf("the content type is %q, want %q", w.Header().Get("Content-Type"), want)
	}
}

func TestStreamCopiesAReader(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return c.Stream("text/csv", strings.NewReader("a,b\n1,2\n"))
	})
	if w.Body.String() != "a,b\n1,2\n" {
		t.Errorf("the body is %q", w.Body.String())
	}
	if want := "text/csv"; w.Header().Get("Content-Type") != want {
		t.Errorf("the content type is %q, want %q", w.Header().Get("Content-Type"), want)
	}
}

func TestWriteMakesACtxAnIOWriter(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		_, err := io.Copy(c, strings.NewReader("streamed"))
		return err
	})
	if w.Body.String() != "streamed" {
		t.Errorf("the body is %q, want streamed", w.Body.String())
	}
}

func TestNoContent(t *testing.T) {
	w := serve(t, httptest.NewRequest("DELETE", "/things/7", nil), func(c *Ctx) error {
		return c.NoContent()
	})
	if w.Code != http.StatusNoContent {
		t.Errorf("the status is %d, want 204", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("the body is %q, want nothing", w.Body.String())
	}

	w = serve(t, httptest.NewRequest("PUT", "/things/7", nil), func(c *Ctx) error {
		return c.Status(http.StatusResetContent).NoContent()
	})
	if w.Code != http.StatusResetContent {
		t.Errorf("the status is %d, want 205", w.Code)
	}
}

func TestRedirect(t *testing.T) {
	w := serve(t, httptest.NewRequest("POST", "/things", nil), func(c *Ctx) error {
		return c.Redirect("/things/7")
	})
	if w.Code != http.StatusSeeOther {
		t.Errorf("the status is %d, want 303", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/things/7" {
		t.Errorf("Location is %q, want /things/7", got)
	}

	w = serve(t, httptest.NewRequest("GET", "/old", nil), func(c *Ctx) error {
		return c.Status(http.StatusMovedPermanently).Redirect("/new")
	})
	if w.Code != http.StatusMovedPermanently {
		t.Errorf("the status is %d, want 301", w.Code)
	}
}

func TestSetCookie(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return c.SetCookie(&http.Cookie{Name: "session", Value: "abc"}).NoContent()
	})
	if got := w.Header().Get("Set-Cookie"); !strings.HasPrefix(got, "session=abc") {
		t.Errorf("Set-Cookie is %q", got)
	}
}

func TestTheResponseControllerReachesTheServersWriter(t *testing.T) {
	serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		if err := c.Text("first"); err != nil {
			return err
		}
		// httptest.ResponseRecorder is a Flusher, so this works through
		// Unwrap or not at all.
		if err := http.NewResponseController(c.Writer()).Flush(); err != nil {
			t.Errorf("Flush through the wrapper failed: %v", err)
		}
		return nil
	})
}

func TestReadFromKeepsTheFastPath(t *testing.T) {
	var w readerFromRecorder
	c := direct(t, &w)

	// A strings.Reader is an io.WriterTo, which io.Copy prefers over the
	// destination's ReadFrom, so the source here is a plain reader.
	src := struct{ io.Reader }{strings.NewReader("bytes")}
	if _, err := io.Copy(c.res, src); err != nil {
		t.Fatal(err)
	}
	if !w.used {
		t.Error("io.Copy did not go through ReadFrom, so a file response loses sendfile")
	}
	if c.res.status != http.StatusOK {
		t.Errorf("ReadFrom recorded the status %d, want 200", c.res.status)
	}
}

// readerFromRecorder is a ResponseWriter that can say whether ReadFrom was the
// path taken.
type readerFromRecorder struct {
	httptest.ResponseRecorder
	used bool
}

func (w *readerFromRecorder) ReadFrom(r io.Reader) (int64, error) {
	w.used = true
	return io.Copy(io.Discard, r)
}

func TestReadFromFallsBackWhenTheServersWriterCannotDoIt(t *testing.T) {
	c := direct(t, plainWriter{httptest.NewRecorder()})
	n, err := c.res.ReadFrom(strings.NewReader("bytes"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("ReadFrom copied %d bytes, want 5", n)
	}
}

// plainWriter is a ResponseWriter and nothing else.
type plainWriter struct{ w http.ResponseWriter }

func (p plainWriter) Header() http.Header         { return p.w.Header() }
func (p plainWriter) Write(b []byte) (int, error) { return p.w.Write(b) }
func (p plainWriter) WriteHeader(code int)        { p.w.WriteHeader(code) }

func TestWriteReportsAFailedConnection(t *testing.T) {
	c := direct(t, failWriter{httptest.NewRecorder()})
	if err := c.Text("hello"); !errors.Is(err, errBroken) {
		t.Errorf("Text reported %v, want the write error", err)
	}
	if err := c.Bytes("text/plain", []byte("hello")); !errors.Is(err, errBroken) {
		t.Errorf("Bytes reported %v, want the write error", err)
	}
	if err := c.Stream("text/plain", strings.NewReader("hello")); !errors.Is(err, errBroken) {
		t.Errorf("Stream reported %v, want the write error", err)
	}
}

// failWriter is a ResponseWriter whose client has gone away.
type failWriter struct{ *httptest.ResponseRecorder }

func (failWriter) Write([]byte) (int, error) { return 0, errBroken }
