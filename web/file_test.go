package web

import (
	"errors"
	"html/template"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-mizu/mizu/errs"
)

// onDisk writes a file into a directory the test owns and answers with its path.
func onDisk(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("writing the fixture: %v", err)
	}
	return p
}

func TestHTMLSendsMarkup(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/", nil), func(c *Ctx) error {
		return c.HTML(template.HTML("<h1>water</h1>"))
	})

	if want := "<h1>water</h1>"; w.Body.String() != want {
		t.Errorf("the body is %s, want %s", w.Body.String(), want)
	}
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("the content type is %q, want text/html; charset=utf-8", got)
	}
}

func TestHTMLTakesTheStatusItWasGiven(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/missing", nil), func(c *Ctx) error {
		return c.Status(http.StatusNotFound).HTML(template.HTML("<p>gone</p>"))
	})
	if w.Code != http.StatusNotFound {
		t.Errorf("the status is %d, want 404", w.Code)
	}
}

func TestFileSendsWhatIsOnDisk(t *testing.T) {
	p := onDisk(t, "notes.txt", "water is wet")

	w := serve(t, httptest.NewRequest("GET", "/notes", nil), func(c *Ctx) error {
		return c.File(p)
	})

	if w.Code != http.StatusOK {
		t.Errorf("the status is %d, want 200", w.Code)
	}
	if want := "water is wet"; w.Body.String() != want {
		t.Errorf("the body is %q, want %q", w.Body.String(), want)
	}
	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Errorf("the content type is %q, want something text/plain", got)
	}
	if w.Header().Get("Last-Modified") == "" {
		t.Error("there is no Last-Modified, so a browser cannot ask whether it changed")
	}
}

// A range is the reason these go through ServeContent rather than io.Copy: a
// download that dropped at 90 percent should not start again from nothing.
func TestFileAnswersARange(t *testing.T) {
	p := onDisk(t, "notes.txt", "0123456789")

	r := httptest.NewRequest("GET", "/notes", nil)
	r.Header.Set("Range", "bytes=2-5")
	w := serve(t, r, func(c *Ctx) error { return c.File(p) })

	if w.Code != http.StatusPartialContent {
		t.Errorf("the status is %d, want 206", w.Code)
	}
	if want := "2345"; w.Body.String() != want {
		t.Errorf("the body is %q, want %q", w.Body.String(), want)
	}
	if got := w.Header().Get("Content-Range"); got != "bytes 2-5/10" {
		t.Errorf("the content range is %q, want bytes 2-5/10", got)
	}
}

func TestFileAnswersAConditionalRequest(t *testing.T) {
	p := onDisk(t, "notes.txt", "water is wet")

	first := serve(t, httptest.NewRequest("GET", "/notes", nil), func(c *Ctx) error {
		return c.File(p)
	})
	modified := first.Header().Get("Last-Modified")
	if modified == "" {
		t.Fatal("there is no Last-Modified to send back")
	}

	r := httptest.NewRequest("GET", "/notes", nil)
	r.Header.Set("If-Modified-Since", modified)
	w := serve(t, r, func(c *Ctx) error { return c.File(p) })

	if w.Code != http.StatusNotModified {
		t.Errorf("the status is %d, want 304", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Errorf("a 304 carried %d bytes of body", w.Body.Len())
	}
}

func TestAMissingFileIsNotFound(t *testing.T) {
	var err error
	serve(t, httptest.NewRequest("GET", "/notes", nil), func(c *Ctx) error {
		err = c.File(filepath.Join(t.TempDir(), "nothing.txt"))
		return nil
	})

	if !errors.Is(err, errs.NotFound) {
		t.Fatalf("the kind is %v, want NotFound", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != "file.not_found" {
		t.Errorf("the code is %q, want file.not_found", got)
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Error("the original error is gone, so the log has lost the path")
	}
	// Error() is the developer's string and has the whole chain in it. Msg is
	// the part a renderer shows, and a filesystem path does not belong there.
	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("the error is a %T, want an *errs.Error", err)
	}
	if strings.Contains(e.Msg, "nothing.txt") {
		t.Errorf("the message is %q and names the file", e.Msg)
	}
	if !strings.Contains(err.Error(), "nothing.txt") {
		t.Error("the path is nowhere in the error, so the log cannot say which file")
	}
}

// There is no directory listing here, and the client asked for a file.
func TestADirectoryIsNotAFile(t *testing.T) {
	var err error
	serve(t, httptest.NewRequest("GET", "/notes", nil), func(c *Ctx) error {
		err = c.File(t.TempDir())
		return nil
	})

	if !errors.Is(err, errs.NotFound) {
		t.Fatalf("the kind is %v, want NotFound", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != "file.is_directory" {
		t.Errorf("the code is %q, want file.is_directory", got)
	}
}

// Cleaning the path first would turn /srv/../etc into /etc, which has no ..
// left in it and is still not the directory the caller meant.
func TestAPathThatClimbsIsRefused(t *testing.T) {
	paths := []string{
		"../../etc/passwd",
		"/var/www/../../etc/passwd",
		`..\..\windows\win.ini`,
		`/var/www\..\..\etc\passwd`,
	}

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			var err error
			serve(t, httptest.NewRequest("GET", "/f", nil), func(c *Ctx) error {
				err = c.File(p)
				return nil
			})
			if !errors.Is(err, errs.Invalid) {
				t.Fatalf("the kind is %v, want Invalid", errs.KindOf(err))
			}
			if got := errs.CodeOf(err); got != "file.path" {
				t.Errorf("the code is %q, want file.path", got)
			}
		})
	}
}

// A file called ..notes.txt has no .. segment in it and is a file somebody can
// legitimately have.
func TestATwoDotPrefixIsNotAClimb(t *testing.T) {
	p := onDisk(t, "..notes.txt", "still a file")

	w := serve(t, httptest.NewRequest("GET", "/notes", nil), func(c *Ctx) error {
		return c.File(p)
	})
	if want := "still a file"; w.Body.String() != want {
		t.Errorf("the body is %q, want %q", w.Body.String(), want)
	}
}

func TestFileFSSendsWhatIsInTheFS(t *testing.T) {
	fsys := fstest.MapFS{"assets/app.css": {Data: []byte("body{}")}}

	w := serve(t, httptest.NewRequest("GET", "/assets/app.css", nil), func(c *Ctx) error {
		return c.FileFS(fsys, "assets/app.css")
	})

	if want := "body{}"; w.Body.String() != want {
		t.Errorf("the body is %q, want %q", w.Body.String(), want)
	}
	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Errorf("the content type is %q, want something text/css", got)
	}
}

// A client sends the leading slash and an fs.FS path is always relative to its
// own root, so taking it off is the only thing it can mean.
func TestFileFSTakesTheLeadingSlashOff(t *testing.T) {
	fsys := fstest.MapFS{"app.css": {Data: []byte("body{}")}}

	w := serve(t, httptest.NewRequest("GET", "/app.css", nil), func(c *Ctx) error {
		return c.FileFS(fsys, "/app.css")
	})
	if want := "body{}"; w.Body.String() != want {
		t.Errorf("the body is %q, want %q", w.Body.String(), want)
	}
}

func TestAnFSPathThatClimbsIsRefused(t *testing.T) {
	fsys := fstest.MapFS{"app.css": {Data: []byte("body{}")}}

	for _, p := range []string{"../secret", "assets/../../secret", "./app.css", ""} {
		t.Run(p, func(t *testing.T) {
			var err error
			serve(t, httptest.NewRequest("GET", "/x", nil), func(c *Ctx) error {
				err = c.FileFS(fsys, p)
				return nil
			})
			if !errors.Is(err, errs.Invalid) {
				t.Fatalf("the kind is %v, want Invalid", errs.KindOf(err))
			}
			if got := errs.CodeOf(err); got != "file.path" {
				t.Errorf("the code is %q, want file.path", got)
			}
		})
	}
}

func TestAMissingFSFileIsNotFound(t *testing.T) {
	var err error
	serve(t, httptest.NewRequest("GET", "/x", nil), func(c *Ctx) error {
		err = c.FileFS(fstest.MapFS{}, "nothing.css")
		return nil
	})

	if !errors.Is(err, errs.NotFound) {
		t.Fatalf("the kind is %v, want NotFound", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != "file.not_found" {
		t.Errorf("the code is %q, want file.not_found", got)
	}
}

func TestAnFSDirectoryIsNotAFile(t *testing.T) {
	fsys := fstest.MapFS{"assets/app.css": {Data: []byte("body{}")}}

	var err error
	serve(t, httptest.NewRequest("GET", "/assets", nil), func(c *Ctx) error {
		err = c.FileFS(fsys, "assets")
		return nil
	})

	if got := errs.CodeOf(err); got != "file.is_directory" {
		t.Errorf("the code is %q, want file.is_directory", got)
	}
}

// A path the operating system will not take at all is the server's problem
// rather than the client's, so it is a 500 and not a 404. A NUL byte is the one
// way to get there that means the same thing on every platform in CI.
func TestAPathThatWillNotOpenIsTheServersProblem(t *testing.T) {
	var err error
	serve(t, httptest.NewRequest("GET", "/f", nil), func(c *Ctx) error {
		err = c.File("notes\x00.txt")
		return nil
	})

	if errors.Is(err, fs.ErrNotExist) {
		t.Fatal("the operating system said the name is not usable, not that the file is missing")
	}
	if !errors.Is(err, errs.Internal) {
		t.Fatalf("the kind is %v, want Internal", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != "file.unreadable" {
		t.Errorf("the code is %q, want file.unreadable", got)
	}
}

// stuck is a file that reads but does not seek, which is what a range needs.
type stuck struct{ fs.File }

func (stuck) Read([]byte) (int, error) { return 0, nil }

type stuckFS struct{ fs.FS }

func (s stuckFS) Open(name string) (fs.File, error) {
	f, err := s.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return stuck{f}, nil
}

// errStat is a file that opened and then will not say anything about itself.
type errStat struct{ fs.File }

func (errStat) Stat() (fs.FileInfo, error) { return nil, errors.New("the disk went away") }

type errStatFS struct{ fs.FS }

func (s errStatFS) Open(name string) (fs.File, error) {
	f, err := s.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return errStat{f}, nil
}

func TestAFileThatWillNotStatIsTheServersProblem(t *testing.T) {
	fsys := errStatFS{fstest.MapFS{"app.css": {Data: []byte("body{}")}}}

	var err error
	serve(t, httptest.NewRequest("GET", "/app.css", nil), func(c *Ctx) error {
		err = c.FileFS(fsys, "app.css")
		return nil
	})

	if !errors.Is(err, errs.Internal) {
		t.Fatalf("the kind is %v, want Internal", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != "file.unreadable" {
		t.Errorf("the code is %q, want file.unreadable", got)
	}
}

func TestAFileThatDoesNotSeekSaysSo(t *testing.T) {
	fsys := stuckFS{fstest.MapFS{"app.css": {Data: []byte("body{}")}}}

	var err error
	serve(t, httptest.NewRequest("GET", "/app.css", nil), func(c *Ctx) error {
		err = c.FileFS(fsys, "app.css")
		return nil
	})

	if !errors.Is(err, errs.Internal) {
		t.Fatalf("the kind is %v, want Internal", errs.KindOf(err))
	}
	if got := errs.CodeOf(err); got != "file.unseekable" {
		t.Errorf("the code is %q, want file.unseekable", got)
	}
}

func TestDownloadSaysToSaveIt(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/export", nil), func(c *Ctx) error {
		return c.Download("rows.csv", strings.NewReader("a,b\n1,2\n"))
	})

	if want := "a,b\n1,2\n"; w.Body.String() != want {
		t.Errorf("the body is %q, want %q", w.Body.String(), want)
	}
	if got := w.Header().Get("Content-Disposition"); got != `attachment; filename=rows.csv` {
		t.Errorf("the disposition is %q, want attachment; filename=rows.csv", got)
	}
}

func TestDownloadOfSomethingUnrecognisedIsOctetStream(t *testing.T) {
	w := serve(t, httptest.NewRequest("GET", "/export", nil), func(c *Ctx) error {
		return c.Download("blob.mizunotatype", strings.NewReader("..."))
	})
	if got := w.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Errorf("the content type is %q, want application/octet-stream", got)
	}
}

// The name can be something a person typed, and a header is one line.
func TestADownloadNameCannotBreakTheHeader(t *testing.T) {
	cases := []struct{ name, want string }{
		{"../../etc/passwd", "attachment; filename=passwd"},
		{`c:\windows\win.ini`, "attachment; filename=win.ini"},
		{"a\r\nX-Injected: yes", "attachment; filename=\"aX-Injected: yes\""},
		{"", "attachment; filename=download"},
		{"..", "attachment; filename=download"},
		{"quarterly report.pdf", `attachment; filename="quarterly report.pdf"`},
		{"みず.txt", `attachment; filename*=utf-8''%E3%81%BF%E3%81%9A.txt`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := serve(t, httptest.NewRequest("GET", "/export", nil), func(c *Ctx) error {
				return c.Download(tc.name, strings.NewReader("x"))
			})

			got := w.Header().Get("Content-Disposition")
			if got != tc.want {
				t.Errorf("the disposition is %q, want %q", got, tc.want)
			}
			if len(w.Header()["Content-Disposition"]) != 1 {
				t.Errorf("there are %d Content-Disposition headers", len(w.Header()["Content-Disposition"]))
			}
			if strings.ContainsAny(got, "\r\n") {
				t.Error("the header has a line break in it")
			}
		})
	}
}

func TestAttachmentSendsAFileToSave(t *testing.T) {
	p := onDisk(t, "q1.dat", "a,b\n1,2\n")

	w := serve(t, httptest.NewRequest("GET", "/report", nil), func(c *Ctx) error {
		return c.Attachment(p, "Q1 report.csv")
	})

	if want := "a,b\n1,2\n"; w.Body.String() != want {
		t.Errorf("the body is %q, want %q", w.Body.String(), want)
	}
	if want := `attachment; filename="Q1 report.csv"`; w.Header().Get("Content-Disposition") != want {
		t.Errorf("the disposition is %q, want %q", w.Header().Get("Content-Disposition"), want)
	}
	// The name decides the type, not the extension the file has on disk.
	if got := w.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/csv") {
		t.Errorf("the content type is %q, want something text/csv", got)
	}
}

func TestAttachmentWithNoNameKeepsTheOneOnDisk(t *testing.T) {
	p := onDisk(t, "q1.csv", "a,b\n")

	w := serve(t, httptest.NewRequest("GET", "/report", nil), func(c *Ctx) error {
		return c.Attachment(p, "")
	})
	if want := "attachment; filename=q1.csv"; w.Header().Get("Content-Disposition") != want {
		t.Errorf("the disposition is %q, want %q", w.Header().Get("Content-Disposition"), want)
	}
}

func TestAttachmentAnswersARange(t *testing.T) {
	p := onDisk(t, "big.bin", "0123456789")

	r := httptest.NewRequest("GET", "/report", nil)
	r.Header.Set("Range", "bytes=4-6")
	w := serve(t, r, func(c *Ctx) error { return c.Attachment(p, "big.bin") })

	if w.Code != http.StatusPartialContent {
		t.Errorf("the status is %d, want 206", w.Code)
	}
	if want := "456"; w.Body.String() != want {
		t.Errorf("the body is %q, want %q", w.Body.String(), want)
	}
}

// disposition has no fallback for FormatMediaType answering with nothing, on
// the grounds that a cleaned name is always something it can format. This is
// where that is held to be true, since the alternative is a branch nothing can
// reach and nothing can test.
func TestEveryNameStillFormats(t *testing.T) {
	names := []string{
		"", ".", "..", "...", "/", `\`, "   ", "\r\n", "\x00",
		"a.txt", "a b.txt", `a"b.txt`, "a;b.txt", "a,b.txt", "a=b.txt",
		"みず.txt", "😀", "ünïcødé.tar.gz", "%2e%2e%2f", "*?<>|.txt",
		strings.Repeat("a", 5000) + ".txt", strings.Repeat("é", 300),
	}

	for _, name := range names {
		got := disposition(name)
		if got == "" {
			t.Errorf("disposition(%q) is empty, so the response would carry a header with nothing in it", name)
		}
		if strings.ContainsAny(got, "\r\n") {
			t.Errorf("disposition(%q) is %q and has a line break in it", name, got)
		}
		if !strings.HasPrefix(got, "attachment") {
			t.Errorf("disposition(%q) is %q and does not say to save it", name, got)
		}
	}
}

func TestAMissingAttachmentIsNotFound(t *testing.T) {
	var err error
	serve(t, httptest.NewRequest("GET", "/report", nil), func(c *Ctx) error {
		err = c.Attachment(filepath.Join(t.TempDir(), "gone.csv"), "gone.csv")
		return nil
	})
	if got := errs.CodeOf(err); got != "file.not_found" {
		t.Errorf("the code is %q, want file.not_found", got)
	}
}

func TestAttachingADirectoryIsNotFound(t *testing.T) {
	var err error
	w := serve(t, httptest.NewRequest("GET", "/report", nil), func(c *Ctx) error {
		err = c.Attachment(t.TempDir(), "everything.zip")
		return nil
	})

	if got := errs.CodeOf(err); got != "file.is_directory" {
		t.Errorf("the code is %q, want file.is_directory", got)
	}
	// The disposition is set after the file is known to be servable, so a
	// failure does not leave a header telling the browser to save the 500.
	if got := w.Header().Get("Content-Disposition"); got != "" {
		t.Errorf("the disposition is %q on a response that failed", got)
	}
}
