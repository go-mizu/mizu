package mw

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/web"
)

// long is a body worth compressing, which means one over the frame sized floor.
var long = strings.Repeat("mizu is water and water is mizu. ", 100)

// short is a body that is not.
const short = "mizu"

// zipped serves one GET through Compress and reports the raw response.
func zipped(tb testing.TB, accept string, h http.HandlerFunc) *httptest.ResponseRecorder {
	tb.Helper()

	r := httptest.NewRequest("GET", "/", nil)
	if accept != "" {
		r.Header.Set("Accept-Encoding", accept)
	}

	w := httptest.NewRecorder()
	Compress()(h).ServeHTTP(w, r)
	return w
}

// plain is the body the client ends up with, decoded when it was encoded.
func plain(tb testing.TB, w *httptest.ResponseRecorder) string {
	tb.Helper()

	var r io.Reader
	switch enc := w.Header().Get("Content-Encoding"); enc {
	case "":
		return w.Body.String()
	case "gzip":
		zr, err := gzip.NewReader(w.Body)
		if err != nil {
			tb.Fatalf("the gzip body did not open: %v", err)
		}
		r = zr
	case "deflate":
		r = flate.NewReader(w.Body)
	default:
		tb.Fatalf("the response is encoded as %q, which is not one of the two", enc)
	}

	b, err := io.ReadAll(r)
	if err != nil {
		tb.Fatalf("the %s body did not read: %v", w.Header().Get("Content-Encoding"), err)
	}
	return string(b)
}

// text is a handler that writes a body with a content type on it.
func text(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, body)
	}
}

func TestALongTextBodyComesBackSmaller(t *testing.T) {
	w := zipped(t, "gzip", text(long))

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Errorf("Content-Encoding is %q, want gzip", got)
	}
	if got := plain(t, w); got != long {
		t.Errorf("the body came back %d bytes long, want %d", len(got), len(long))
	}
	if w.Body.Len() >= len(long) {
		t.Errorf("the response is %d bytes and the body was %d, so nothing was saved", w.Body.Len(), len(long))
	}
}

func TestVaryPicksUpAcceptEncoding(t *testing.T) {
	w := zipped(t, "gzip", text(long))
	if got := w.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary is %q, want Accept-Encoding", got)
	}
}

// TestVaryIsThereEvenWhenNothingWasCompressed is the one that keeps a cache
// honest. The next request may be for a type that does compress, and a cache
// holding this response without the Vary would serve it to a client that asked
// for gzip and to one that cannot read it.
func TestVaryIsThereEvenWhenNothingWasCompressed(t *testing.T) {
	w := zipped(t, "gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		io.WriteString(w, long)
	})

	if got := w.Header().Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary is %q, want Accept-Encoding", got)
	}
	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("a JPEG came back as %q, and it was compressed when it was made", got)
	}
}

// TestAShortBodyIsNotWorthIt covers the floor. gzip framing is twenty three
// bytes, so a short response comes out bigger and costs both ends the CPU to
// find that out.
func TestAShortBodyIsNotWorthIt(t *testing.T) {
	w := zipped(t, "gzip", text(short))

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("a %d byte body came back as %q", len(short), got)
	}
	if w.Body.String() != short {
		t.Errorf("the body is %q, want %q", w.Body, short)
	}
}

func TestABodyWrittenInPiecesAddsUpToOneWorthCompressing(t *testing.T) {
	w := zipped(t, "gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		for range 100 {
			io.WriteString(w, "mizu is water and water is mizu. ")
		}
	})

	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Errorf("Content-Encoding is %q, want gzip", got)
	}
	if got := plain(t, w); got != long {
		t.Errorf("the body came back %d bytes long, want %d", len(got), len(long))
	}
}

func TestDeflateWhenThatIsAllTheClientTakes(t *testing.T) {
	w := zipped(t, "deflate", text(long))

	if got := w.Header().Get("Content-Encoding"); got != "deflate" {
		t.Errorf("Content-Encoding is %q, want deflate", got)
	}
	if got := plain(t, w); got != long {
		t.Errorf("the body came back %d bytes long, want %d", len(got), len(long))
	}
}

// TestACompressorFromThePoolWritesToTheResponseInFrontOfIt is the whole risk in
// pooling one. A writer that came back from the pool without a Reset would write
// this response into the connection the last one used, which is a page going to
// the wrong client.
func TestACompressorFromThePoolWritesToTheResponseInFrontOfIt(t *testing.T) {
	t.Run("gzip", func(t *testing.T) {
		var first, second strings.Builder

		gz := gzipFor(&first)
		io.WriteString(gz, "one")
		gz.Close()
		gzips.Put(gz)

		gz = gzipFor(&second)
		io.WriteString(gz, "two")
		gz.Close()

		if first.Len() == 0 || second.Len() == 0 {
			t.Fatalf("the two responses are %d and %d bytes", first.Len(), second.Len())
		}
		zr, err := gzip.NewReader(strings.NewReader(second.String()))
		if err != nil {
			t.Fatalf("the second response did not open: %v", err)
		}
		if unzip(t, zr) != "two" {
			t.Error("the second response is not what was written to it")
		}
	})

	t.Run("deflate", func(t *testing.T) {
		var first, second strings.Builder

		fl := flateFor(&first)
		io.WriteString(fl, "one")
		fl.Close()
		flates.Put(fl)

		fl = flateFor(&second)
		io.WriteString(fl, "two")
		fl.Close()

		if first.Len() == 0 || second.Len() == 0 {
			t.Fatalf("the two responses are %d and %d bytes", first.Len(), second.Len())
		}
		if unzip(t, flate.NewReader(strings.NewReader(second.String()))) != "two" {
			t.Error("the second response is not what was written to it")
		}
	})
}

// unzip reads a decompressor out.
func unzip(tb testing.TB, r io.Reader) string {
	tb.Helper()

	b, err := io.ReadAll(r)
	if err != nil {
		tb.Fatalf("the body did not read: %v", err)
	}
	return string(b)
}

func TestGzipWinsWhenBothAreOffered(t *testing.T) {
	for _, accept := range []string{"gzip, deflate", "deflate, gzip", "deflate;q=1.0, gzip;q=0.5"} {
		t.Run(accept, func(t *testing.T) {
			if got := zipped(t, accept, text(long)).Header().Get("Content-Encoding"); got != "gzip" {
				t.Errorf("Content-Encoding is %q, want gzip", got)
			}
		})
	}
}

func TestAcceptEncodingIsRead(t *testing.T) {
	cases := map[string]string{
		"":                    "",
		"identity":            "",
		"br, zstd":            "",
		"GZIP":                "gzip",
		"gzip;q=0":            "",
		"gzip;q=0.0":          "",
		"gzip;q=0, deflate":   "deflate",
		"gzip;q=0.1":          "gzip",
		" deflate , gzip ":    "gzip",
		"br;q=1.0, deflate":   "deflate",
		"*":                   "",
		"gzip;q=notanumber":   "gzip",
		"deflate;level=9":     "deflate",
		"gzip;q=0;other=1":    "",
		"identity;q=0, gzip":  "gzip",
		"deflate;q=0, gzip;q": "gzip",
	}

	for accept, want := range cases {
		t.Run(accept, func(t *testing.T) {
			if got := negotiate(accept); got != want {
				t.Errorf("negotiate(%q) is %q, want %q", accept, got, want)
			}
		})
	}
}

func TestNoAcceptEncodingMeansTheBodyGoesOutAsItIs(t *testing.T) {
	w := zipped(t, "", text(long))

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("a client that asked for nothing got %q", got)
	}
	if got := w.Header().Get("Vary"); got != "" {
		t.Errorf("Vary is %q, and nothing here varies on a header the client did not send", got)
	}
	if w.Body.String() != long {
		t.Error("the body changed")
	}
}

func TestTheTypesThatCompress(t *testing.T) {
	yes := []string{
		"text/html; charset=utf-8",
		"text/plain",
		"text/css",
		"application/json",
		"application/javascript",
		"application/wasm",
		"image/svg+xml",
		"application/vnd.example.v2+json",
		"application/atom+xml",
		"", // the handler said nothing and net/http is about to sniff one
	}
	no := []string{
		"image/jpeg",
		"image/png",
		"video/mp4",
		"application/zip",
		"application/octet-stream",
		"font/woff2",
		"audio/mpeg",
	}

	for _, ct := range yes {
		t.Run(ct, func(t *testing.T) {
			if !compressible(ct) {
				t.Errorf("%q is not compressed and it is text under the skin", ct)
			}
		})
	}
	for _, ct := range no {
		t.Run(ct, func(t *testing.T) {
			if compressible(ct) {
				t.Errorf("%q is compressed and it was compressed when it was made", ct)
			}
		})
	}
}

func TestAResponseTheHandlerAlreadyEncodedIsLeftAlone(t *testing.T) {
	w := zipped(t, "gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Encoding", "br")
		io.WriteString(w, long)
	})

	if got := w.Header().Get("Content-Encoding"); got != "br" {
		t.Errorf("Content-Encoding is %q, want the br the handler set", got)
	}
	if w.Body.String() != long {
		t.Error("a body the handler had already encoded was encoded again")
	}
}

// TestContentLengthGoesAwayWithTheBodyItDescribed is what makes the response
// legal. The handler counted the bytes it wrote and those are not the bytes
// going out.
func TestContentLengthGoesAwayWithTheBodyItDescribed(t *testing.T) {
	w := zipped(t, "gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", strconv.Itoa(len(long)))
		io.WriteString(w, long)
	})

	if got := w.Header().Get("Content-Length"); got != "" {
		t.Errorf("Content-Length is %q and the body it counted was replaced", got)
	}
}

func TestAResponseWithNoBodyIsNotCompressed(t *testing.T) {
	for _, code := range []int{http.StatusNoContent, http.StatusNotModified} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			w := zipped(t, "gzip", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(code)
			})

			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Errorf("a %d came back as %q", code, got)
			}
		})

		// The same status with a body written after it anyway, which is a
		// handler bug the server covers by dropping the body. Encoding it would
		// leave a Content-Encoding on a response that has nothing in it.
		t.Run(http.StatusText(code)+" with a body after it", func(t *testing.T) {
			w := zipped(t, "gzip", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(code)
				io.WriteString(w, long)
			})

			if got := w.Header().Get("Content-Encoding"); got != "" {
				t.Errorf("a %d came back as %q", code, got)
			}
			if w.Body.String() != long {
				t.Errorf("the body was changed on the way out of a %d", code)
			}
		})
	}
}

// TestAFlushOnAResponseThatIsNotCompressedCarriesOn. The flush is what forces the
// decision, the answer is no, and there is nothing left for the filter to do but
// let the flush through to the server.
func TestAFlushOnAResponseThatIsNotCompressedCarriesOn(t *testing.T) {
	r := httptest.NewRequest("GET", "/clip.mp4", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	Compress()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		io.WriteString(w, "the first chunk")
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("the flush failed: %v", err)
		}
		io.WriteString(w, " and the second")
	})).ServeHTTP(w, r)

	if got := w.Header().Get("Content-Encoding"); got != "" {
		t.Errorf("a video came back as %q", got)
	}
	if !w.Flushed {
		t.Error("the flush stopped at the filter and never reached the server")
	}
	if w.Body.String() != "the first chunk and the second" {
		t.Errorf("the body is %q", w.Body)
	}
}

// TestAFlushSendsWhatHasBeenWritten is a server sent event stream: the handler
// writes an event and flushes, and the client has to get it now rather than
// when the response ends.
func TestAFlushSendsWhatHasBeenWritten(t *testing.T) {
	const event = "data: something happened\n\n"

	r := httptest.NewRequest("GET", "/events", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()

	var afterFirst int
	Compress()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i := range 3 {
			io.WriteString(w, event)
			if err := http.NewResponseController(w).Flush(); err != nil {
				t.Errorf("the flush failed: %v", err)
			}
			if i == 0 {
				afterFirst = w.(*web.Recorder).Unwrap().(*httptest.ResponseRecorder).Body.Len()
			}
		}
	})).ServeHTTP(w, r)

	if afterFirst == 0 {
		t.Error("the first event was still in the compressor after it was flushed")
	}
	if got := w.Header().Get("Content-Encoding"); got != "gzip" {
		t.Errorf("Content-Encoding is %q, want gzip", got)
	}
	if got := plain(t, w); got != strings.Repeat(event, 3) {
		t.Errorf("the stream came back as %q", got)
	}
}

// TestTheCountIsTheBytesThatWentOut matters for the access log. A line saying a
// response was twenty kilobytes when three went over the wire is a line that
// makes an egress bill unexplainable.
func TestTheCountIsTheBytesThatWentOut(t *testing.T) {
	var written int64

	outer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := web.Record(w)
			next.ServeHTTP(rec, r)
			written = rec.Written()
		})
	}

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	outer(Compress()(text(long))).ServeHTTP(w, r)

	if written != int64(w.Body.Len()) {
		t.Errorf("the log would say %d bytes and %d went out", written, w.Body.Len())
	}
	if written >= int64(len(long)) {
		t.Errorf("the count is %d, which is the body the handler wrote rather than the one that went out", written)
	}
}

func TestTheStatusTheHandlerSetSurvives(t *testing.T) {
	w := zipped(t, "gzip", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, long)
	})

	if w.Code != http.StatusCreated {
		t.Errorf("the response went out as %d, want 201", w.Code)
	}
	if got := plain(t, w); got != long {
		t.Error("the body changed")
	}
}
