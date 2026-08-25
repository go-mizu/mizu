package mw

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/go-mizu/mizu/web"
)

// minCompress is the smallest response worth compressing.
//
// It is one ethernet frame. Below that the response goes out in one packet
// either way, so compressing it spends CPU on both ends to save nothing, and a
// short response often comes out larger once the twenty three bytes of gzip
// framing are on it.
const minCompress = 1400

// Compress compresses the response body when the client asked for it and the
// body is the kind that gets smaller.
//
//	srv := web.Chain(routes,
//		mw.Logger(log),
//		mw.Compress(),
//	)
//
// gzip and deflate, in that order of preference, chosen from Accept-Encoding.
// There is no zstd and no brotli: neither is in the standard library, and a
// compressor is not worth the first third-party dependency in the core.
//
// # When it does nothing
//
// When the client did not ask, which is what Accept-Encoding is for.
//
// When the response is already encoded. A handler that set Content-Encoding
// itself has done this on purpose, and compressing it twice makes it bigger.
//
// When the body is under 1400 bytes, which is one ethernet frame. The first
// writes are held until there are enough to be worth it, and a response that
// ends before then goes out as it was.
//
// When the content type is not one that compresses. A JPEG, a zip and an MP4 are
// compressed already and running them through gzip spends CPU to add a few
// bytes. The types that do get compressed are text of any kind, JSON, XML,
// JavaScript, SVG, and any type whose subtype ends in +json or +xml, which is
// how a versioned API type such as application/vnd.example.v2+json is caught.
//
// When the response is a 204 or a 304, neither of which has a body.
//
// # What it changes
//
// Content-Encoding is set to what was used and Vary picks up Accept-Encoding,
// so a cache in front of the service does not hand a compressed response to a
// client that cannot read it.
//
// Content-Length is deleted, because the handler set it for the body it wrote
// and that is no longer the body going out. A compressed response is chunked,
// which is what a response of unknown length is on the wire.
//
// An ETag the handler set is left alone, and this is why [ETag] goes inside
// this rather than outside: the tag has to be for the body, not for the
// compression of it.
//
// # Streaming
//
// A flush compresses and sends whatever has been written, which is what a server
// sent event stream needs. The compressor's window survives, so the next event
// still compresses against everything before it. That is a smaller response and
// it is also a side channel, so a stream carrying a secret next to something an
// attacker controls should not be compressed at all.
func Compress() web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enc := negotiate(r.Header.Get("Accept-Encoding"))
			if enc == "" {
				next.ServeHTTP(w, r)
				return
			}

			rec := web.Record(w)
			rec.Header().Add("Vary", "Accept-Encoding")

			c := &compressor{rec: rec, sink: rec.Sink(), enc: enc}
			defer rec.Through(c)()
			defer c.finish()

			next.ServeHTTP(rec, r)
		})
	}
}

// negotiate picks the encoding to use, or "" for none.
//
// gzip wins when both are on offer, whatever the q values say, because it is
// what every client and every intermediary handles best and the difference in
// size between the two is not worth a preference sort. A q of zero is read,
// since that is the client saying an encoding is unacceptable rather than
// saying it is second choice.
func negotiate(accept string) string {
	var gz, fl bool
	for part := range strings.SplitSeq(accept, ",") {
		name, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if refused(params) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "gzip":
			gz = true
		case "deflate":
			fl = true
		}
	}
	switch {
	case gz:
		return "gzip"
	case fl:
		return "deflate"
	}
	return ""
}

// refused reports whether the parameters carry a q of zero.
func refused(params string) bool {
	for p := range strings.SplitSeq(params, ";") {
		k, v, ok := strings.Cut(strings.TrimSpace(p), "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(k), "q") {
			continue
		}
		q, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return err == nil && q <= 0
	}
	return false
}

// compressor is the body filter. It holds the first writes back until it knows
// whether compressing is worth it, and once it knows it does not ask again.
type compressor struct {
	rec  *web.Recorder
	sink io.Writer
	enc  string

	held []byte    // written before the decision
	out  io.Writer // the compressor once compressing, the sink once not
	done func()    // closes and returns the compressor, when there is one
}

func (c *compressor) Write(p []byte) (int, error) {
	if c.out != nil {
		return c.out.Write(p)
	}

	// Not enough yet to know whether it is worth it, and not enough to be
	// worth it either.
	if len(c.held)+len(p) < minCompress {
		c.held = append(c.held, p...)
		return len(p), nil
	}

	c.decide()
	return c.out.Write(p)
}

// Flush is what [web.Recorder.FlushError] calls, and it means the handler wants
// what it has written to be on its way. So the decision cannot wait for more
// bytes any longer.
func (c *compressor) Flush() error {
	if c.out == nil {
		c.decide()
	}
	if f, ok := c.out.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

// finish is the end of the response. A body that never reached the threshold is
// still sitting in held, and a compressor that was started has a trailer to
// write.
func (c *compressor) finish() {
	if c.out == nil {
		c.out = c.sink
		c.drain()
		return
	}
	if c.done != nil {
		c.done()
	}
}

// decide picks between compressing and passing through, writes out whatever was
// held, and does not run again.
func (c *compressor) decide() {
	h := c.rec.Header()
	if !worth(h, c.rec.Status()) {
		c.out = c.sink
		c.drain()
		return
	}

	h.Set("Content-Encoding", c.enc)
	h.Del("Content-Length")

	if c.enc == "gzip" {
		gz := gzipFor(c.sink)
		c.out, c.done = gz, func() { gz.Close(); gzips.Put(gz) }
	} else {
		fl := flateFor(c.sink)
		c.out, c.done = fl, func() { fl.Close(); flates.Put(fl) }
	}
	c.drain()
}

// drain writes what was held while the decision was pending.
func (c *compressor) drain() {
	if len(c.held) > 0 {
		c.out.Write(c.held)
		c.held = nil
	}
}

// worth reports whether this response is one compressing helps.
func worth(h http.Header, status int) bool {
	if h.Get("Content-Encoding") != "" {
		return false
	}
	if status == http.StatusNoContent || status == http.StatusNotModified {
		return false
	}
	return compressible(h.Get("Content-Type"))
}

// compressible reports whether a content type is text under the skin.
//
// An empty type means the handler wrote bytes without saying what they are, and
// net/http is about to sniff a type for it. Sniffing lands on text/plain or
// text/html for anything that looks like text and on an image or a binary type
// otherwise, so the answer here would be the same one and a byte prefix is not
// worth reimplementing. It is treated as compressible, which is right for the
// case that produces it, a handler writing a page or a payload without a
// Content-Type call in front.
func compressible(ct string) bool {
	ct, _, _ = strings.Cut(ct, ";")
	ct = strings.ToLower(strings.TrimSpace(ct))

	switch {
	case ct == "":
		return true
	case strings.HasPrefix(ct, "text/"):
		return true
	case strings.HasSuffix(ct, "+json"), strings.HasSuffix(ct, "+xml"), strings.HasSuffix(ct, "+text"):
		return true
	}

	switch ct {
	case "application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/ecmascript",
		"application/manifest+json",
		"application/wasm",
		"application/x-ndjson",
		"image/svg+xml",
		"image/x-icon",
		"font/ttf",
		"font/otf":
		return true
	}
	return false
}

// The two compressors are pooled because a gzip.Writer carries a 32KB window and
// a flate.Writer rather more, and allocating one per response is the whole cost
// of this middleware on a small body.
var (
	gzips  sync.Pool
	flates sync.Pool
)

// The two both make one and then reset it, rather than making one around w,
// because Reset on the way in is what keeps the last response out of this one
// and there should be one line doing it rather than two.
func gzipFor(w io.Writer) *gzip.Writer {
	v, _ := gzips.Get().(*gzip.Writer)
	if v == nil {
		v = gzip.NewWriter(nil)
	}
	v.Reset(w)
	return v
}

func flateFor(w io.Writer) *flate.Writer {
	v, _ := flates.Get().(*flate.Writer)
	if v == nil {
		// The level is the standard library's default. flate.NewWriter only
		// fails on a level it does not know, and this one it knows.
		v, _ = flate.NewWriter(nil, flate.DefaultCompression)
	}
	v.Reset(w)
	return v
}
