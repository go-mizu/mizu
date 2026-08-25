package web

import (
	"net/http"
	"strconv"

	"github.com/go-mizu/mizu/conc"
	"github.com/go-mizu/mizu/errs"
)

// JSON sends v as an application/json body.
//
//	func show(c *web.Ctx) error {
//		p, err := posts.Find(c.Context(), c.Param("id"))
//		if err != nil {
//			return err
//		}
//		return web.JSON(c, p)
//	}
//
// The status is whatever [Ctx.Status] was given, or 200. [JSONStatus] is the
// same thing with the status in front of it, for the handful of places where
// naming it reads better than setting it.
//
// It is a function rather than a method because a method cannot have a type
// parameter. What T buys is that the value is checked where it is written: a
// handler that returns the wrong resource type hears about it from the
// compiler, and a generated encoder can be resolved per type later without any
// of these call sites changing.
//
// A MarshalJSON declared on *T is used even when v is a T, which is not what
// [encoding/json.Marshal] of the same value would do. See [writeJSON].
func JSON[T any](c *Ctx, v T) error {
	c.live("JSON")
	return writeJSON(c, &v)
}

// JSONStatus sends v as JSON under a status of the caller's choosing.
//
//	return web.JSONStatus(c, http.StatusConflict, report)
func JSONStatus[T any](c *Ctx, code int, v T) error {
	c.live("JSONStatus")
	c.status = code
	return writeJSON(c, &v)
}

// Created sends a 201 with a Location header and the thing that was made.
//
//	return web.Created(c, "/posts/"+p.ID, p)
//
// location is put in the header as given and an empty one is left off. A
// relative reference is what most handlers have and is what RFC 9110 says to
// resolve against the request, so nothing here makes it absolute.
//
// The body is the resource rather than a wrapper around it, because a client
// that just posted a form wants the record back with whatever the server filled
// in, and a client that does not want it has a Location to fetch instead.
func Created[T any](c *Ctx, location string, v T) error {
	c.live("Created")
	if location != "" {
		c.res.Header().Set("Location", location)
	}
	c.status = http.StatusCreated
	return writeJSON(c, &v)
}

// Accepted sends a 202 with no body, and a Location pointing at where the
// answer will be.
//
//	return web.Accepted(c, "/jobs/"+id)
//
// It is the answer to a request that started something rather than finished
// it. The Location is the resource to poll, and an empty one is left off for
// the case where there is nothing to poll.
func Accepted(c *Ctx, location string) error {
	c.live("Accepted")
	if location != "" {
		c.res.Header().Set("Location", location)
	}
	c.status = http.StatusAccepted
	return c.NoContent()
}

// writeJSON builds the body, then sends it.
//
// The order is the point. Encoding into a buffer first means a value that will
// not marshal is an error the handler returns, with nothing written and the
// status still to play for, rather than half an object under a 200 that the
// client will fail to parse. That costs holding the body, which is the trade
// [Ctx.Stream] makes the other way for a response too big to hold.
//
// Buffering is also what makes Content-Length knowable. Without it net/http
// sends anything over its own buffer as chunked, and the middleware that
// replaces a body, compression and ETag, deletes the header rather than
// trusting it. Setting it costs two allocations, the number as a string and the
// slot it goes in, and it is set on every response rather than on the big ones
// so that an endpoint does not change framing with its payload.
//
// The callers pass &v rather than v, for two reasons. The encoder that reads a
// value out of an interface has to copy it somewhere it can take the address of,
// and handing it an address already skips that copy. The other reason shows: a
// MarshalJSON declared on *T is found, where marshaling a T would have walked
// past it without a word. That is the answer people mean when they write the
// method, and the shapes that were already a pointer are unaffected, because
// both encoders follow a chain of them.
func writeJSON(c *Ctx, v any) error {
	buf := bufs.Get()
	defer putBuf(buf)

	buf.b = buf.b[:0]
	if err := jsonEncode(buf, v); err != nil {
		return errs.Wrap(err, errs.Internal, "respond.json", "web: cannot write the response as JSON")
	}

	c.res.Header().Set("Content-Length", strconv.Itoa(len(buf.b)))
	c.head("application/json")
	_, err := c.res.Write(buf.b)
	return err
}

// A jsonBuf is a response being built, and it is an io.Writer so that both
// JSON encoders can write into one.
type jsonBuf struct{ b []byte }

func (j *jsonBuf) Write(p []byte) (int, error) {
	j.b = append(j.b, p...)
	return len(p), nil
}

// bufs is where a jsonBuf comes from between responses.
//
// It is a pool of its own rather than a field on the pooled Ctx, because a Ctx
// is reset field by field between requests and a buffer that survives that on
// purpose would be the one exception to a rule worth keeping whole.
var bufs = conc.Pool(func() *jsonBuf { return new(jsonBuf) })

// keepBuf is the largest buffer worth holding on to.
//
// A response is usually a few hundred bytes and the pool exists so that those
// stop allocating. One export that ran to ten megabytes should not leave ten
// megabytes parked in the pool for the rest of the process, so a buffer that
// grew past this is dropped and the next caller gets a fresh one.
const keepBuf = 64 << 10

func putBuf(j *jsonBuf) {
	if cap(j.b) > keepBuf {
		return
	}
	bufs.Put(j)
}
