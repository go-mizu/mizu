package mw

import (
	"crypto/rand"
	"encoding/binary"
	"net/http"
	"time"

	"github.com/go-mizu/mizu/clock"
	"github.com/go-mizu/mizu/ctxdata"
	"github.com/go-mizu/mizu/web"
)

// RequestIDHeader is the header the id goes out on, and the one [RequestIDFrom]
// is usually pointed at.
const RequestIDHeader = "X-Request-Id"

// RequestID gives every request an id of its own.
//
// The id goes in the context under [github.com/go-mizu/mizu/web.RequestIDKey],
// which means [github.com/go-mizu/mizu/web.Ctx.RequestID] returns it and every
// record [Logger] writes carries it without anybody naming it. It also goes out
// on the X-Request-Id response header, so the person reporting a problem can
// paste the id from their browser's network tab and have it match a line in the
// log.
//
// It makes one rather than reading one. A request id off the wire is a string
// the client chose, and a client that sends the same one every time turns the
// log into a pile with one id on it. [RequestIDFrom] is the version for a
// service that is behind something which has already assigned one.
//
// The id is a ULID: forty eight bits of the millisecond it was made in, then
// eighty bits from crypto/rand, written as twenty six characters of Crockford
// base32. Sorting a pile of them puts them in the order the requests arrived,
// which is most of what anybody does with a log.
func RequestID() web.Middleware { return requestID("") }

// RequestIDFrom is [RequestID] for a service behind a proxy, a gateway or
// another service that has already assigned an id.
//
//	mw.RequestIDFrom(mw.RequestIDHeader)
//
// It uses the id on that header when there is one, and makes one when there is
// not. Anything that is not a plausible id is treated as if it were not there:
// up to sixty four characters, and every one of them a letter, a digit, a dash,
// an underscore or a dot. That rule is not about correctness, it is about what a
// request id ends up inside, which is log lines, dashboards and shell pipelines.
//
// Do not put this in front of a service that takes requests from the public. The
// header is whatever the caller wrote, so anybody can decide which id their
// request shares with somebody else's.
func RequestIDFrom(header string) web.Middleware { return requestID(header) }

func requestID(from string) web.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var id string
			if from != "" {
				id = plausible(r.Header.Get(from))
			}
			if id == "" {
				id = ulid(clock.Now(r.Context()))
			}

			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(ctxdata.With(r.Context(), web.RequestIDKey, id)))
		})
	}
}

// plausible is s if it is shaped like an id, and the empty string otherwise.
func plausible(s string) string {
	if s == "" || len(s) > 64 {
		return ""
	}
	for i := range len(s) {
		switch c := s[i]; {
		case c >= '0' && c <= '9', c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z':
		case c == '-', c == '_', c == '.':
		default:
			return ""
		}
	}
	return s
}

// ulid is a ULID for the instant t.
//
// The layout is the one the ULID specification gives: six bytes of Unix
// milliseconds, big endian, then ten bytes from crypto/rand. Two ids made in the
// same millisecond are in no particular order with respect to each other, which
// is the difference between this and a monotonic ULID, and it does not matter
// for a request id: a thousandth of a second is finer than anything a log is
// read at.
//
// The clock comes from the context rather than from time.Now, so a test with
// [github.com/go-mizu/mizu/clock.Fake] in the context gets ids whose first ten
// characters it can predict.
func ulid(t time.Time) string {
	var b [16]byte

	// Shifting the milliseconds up by sixteen puts them in the first six bytes
	// and leaves the last two of the word for the random half to overwrite.
	binary.BigEndian.PutUint64(b[:8], uint64(t.UnixMilli())<<16)

	// crypto/rand.Read fills the slice or crashes the program, so there is no
	// error to handle.
	rand.Read(b[6:])

	return encode(b)
}

// crockford is base32 without the letters that can be misread as a digit or
// spell a word, which is what [github.com/go-mizu/mizu/str.IsULID] checks
// against.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// encode writes the hundred and twenty eight bits in b as twenty six
// characters.
//
// Twenty six characters carry a hundred and thirty bits and the value is a
// hundred and twenty eight, so the value is right aligned in them and the first
// character is the top three bits. That is what makes the first character of
// every ULID a digit under eight.
func encode(b [16]byte) string {
	hi := binary.BigEndian.Uint64(b[:8])
	lo := binary.BigEndian.Uint64(b[8:])

	var out [26]byte
	for i := range out {
		out[i] = crockford[digit(hi, lo, uint(5*(len(out)-1-i)))]
	}
	return string(out[:])
}

// digit is the five bits at shift, counting from the bottom of the hundred and
// twenty eight bit value that hi and lo spell.
//
// The second case covers the character that straddles the two words as well as
// the ones below it: Go defines a shift wider than the type as zero, so hi
// contributes nothing until the window reaches it.
func digit(hi, lo uint64, shift uint) uint64 {
	if shift >= 64 {
		return (hi >> (shift - 64)) & 31
	}
	return (hi<<(64-shift) | lo>>shift) & 31
}
