package mw

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"

	"github.com/go-mizu/mizu/web"
)

// maxTag is the largest response this will hold in memory to hash.
//
// A megabyte covers a page, an API payload and a stylesheet, which are the
// responses a conditional request is about. Past that the response is a file, a
// file is usually served with a fingerprinted URL and a long cache, and holding
// one per in flight request is a way to run a server out of memory from
// outside.
const maxTag = 1 << 20

// ETag hashes the response body and answers a conditional request with a 304
// when the client already has it.
//
//	srv := web.Chain(routes,
//		mw.Compress(),
//		mw.ETag(),
//	)
//
// The saving is the body. A 304 still costs the handler, the queries and the
// rendering, because the tag is not known until the response has been built.
// What it saves is the transfer, which is the expensive part for a client on a
// phone and the whole of the bill for a service that pays for egress.
//
// A handler that can tell from the request alone that nothing has changed should
// say so itself rather than build a response for this to throw away. This is for
// the responses where that is not knowable, which is most of them.
//
// # What it applies to
//
// GET and HEAD, and a 200. A conditional request is a question about a
// representation, and the other methods and statuses do not have one.
//
// A response the handler already put an ETag on keeps it, and the conditional
// check is made against that one. That is how a handler that knows a cheaper
// tag, a version column or a file modification time, gets to use it and still
// have the 304 written here.
//
// A response over a megabyte is passed through untagged, and so is one the
// handler flushed, since a flush means the bytes are wanted now rather than
// after something has finished thinking about them.
//
// # The tag is weak
//
//	ETag: W/"5e0f1c6b2a94d3f7c81b0a6e2d4f9a13"
//
// Weak is the honest answer when [Compress] is in the chain. A strong tag
// promises the bytes are identical, and the same page compressed and not
// compressed is the same representation with different bytes. Weak promises they
// mean the same thing, which is what is true and is also all If-None-Match
// compares, since RFC 9110 says that comparison ignores the marker on both
// sides.
//
// It also keeps Range requests honest. A strong tag on an If-Range says a byte
// range of the old response is still a byte range of this one, and after a
// compressor has been through it that is not so.
func ETag() web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}

			rec := web.Record(w)
			t := &tap{sink: rec.Sink()}
			send := rec.Hold()
			restore := rec.Through(t)

			next.ServeHTTP(rec, r)

			restore()
			defer send()

			// Nothing left to decide: the body outgrew the buffer or was
			// flushed on its way, so it has already gone out.
			if t.spilled {
				return
			}
			if rec.Status() != http.StatusOK {
				t.buf.WriteTo(rec.Sink())
				return
			}

			h := rec.Header()
			tag := h.Get("ETag")
			if tag == "" {
				tag = tagFor(t.buf.Bytes())
				h.Set("ETag", tag)
			}

			if !matches(r.Header.Get("If-None-Match"), tag) {
				t.buf.WriteTo(rec.Sink())
				return
			}

			// A 304 carries no body, so the two headers that describe one go
			// with it. Everything else the handler set stays, because a cache
			// updates what it is holding from this.
			h.Del("Content-Type")
			h.Del("Content-Length")
			rec.WriteHeader(http.StatusNotModified)
		})
	}
}

// tap holds the body so it can be hashed, and gives up when the body turns out
// to be bigger than that is worth.
type tap struct {
	sink    io.Writer
	buf     bytes.Buffer
	spilled bool
}

func (t *tap) Write(p []byte) (int, error) {
	if t.spilled {
		return t.sink.Write(p)
	}
	if t.buf.Len()+len(p) > maxTag {
		if err := t.spill(); err != nil {
			return 0, err
		}
		return t.sink.Write(p)
	}
	return t.buf.Write(p)
}

// Flush is what [web.Recorder.FlushError] calls. A handler that flushes wants
// the bytes gone now, which is the end of any hope of hashing them.
func (t *tap) Flush() error { return t.spill() }

// spill sends what is held and stops holding.
func (t *tap) spill() error {
	if t.spilled {
		return nil
	}
	t.spilled = true
	_, err := t.buf.WriteTo(t.sink)
	return err
}

// tagFor is the weak validator for a body.
//
// SHA-256 cut to sixteen bytes. The full digest would be sixty four characters
// on every response and every conditional request for no gain: a collision here
// costs a client one stale page, so the size that matters is the one that makes
// an accident impossible rather than the one that makes an attack impossible.
func tagFor(body []byte) string {
	sum := sha256.Sum256(body)
	return `W/"` + hex.EncodeToString(sum[:16]) + `"`
}

// matches reports whether the client already holds this representation.
//
// The comparison ignores the weakness marker on both sides, which is what RFC
// 9110 says If-None-Match does, and a "*" matches anything that exists at all.
func matches(header, tag string) bool {
	if header == "" {
		return false
	}
	tag = strings.TrimPrefix(tag, "W/")
	for candidate := range strings.SplitSeq(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || strings.TrimPrefix(candidate, "W/") == tag {
			return true
		}
	}
	return false
}
