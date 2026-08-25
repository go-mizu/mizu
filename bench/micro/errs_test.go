package micro

import (
	"errors"
	"testing"

	"github.com/go-mizu/mizu/errs"
)

func init() {
	register("errs/wrap", benchErrsWrap)
	register("rpc/errmap", benchErrsMap)
}

// cause is what a driver hands back, and is built once because a benchmark that
// allocates its own input is measuring the input.
var cause = errors.New("dial tcp 10.0.0.7:5432: connect: connection refused")

// benchErrsWrap is the call every layer that catches an error makes on the way
// out. It is on the failure path, which is not the hot path, right up until
// something downstream is down and every request takes it.
//
// The cost is a stack capture and the metadata map, and it is here so that a
// change to either shows up rather than being noticed the day a dependency
// fails.
func benchErrsWrap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		err := errs.Wrap(cause, errs.Unavailable, "db.unreachable", "The service is busy, try again shortly.").
			WithMeta("host", "10.0.0.7").
			WithMeta("attempt", 3)
		_ = err
	}
}

// benchErrsMap is the translation every failed request ends with: a kind
// becomes a status for HTTP and a code for RPC. It is a table lookup and it is
// budgeted at nothing, so this exists to notice the day it stops being one.
func benchErrsMap(b *testing.B) {
	err := errs.Wrap(cause, errs.Unavailable, "db.unreachable", "The service is busy, try again shortly.")

	b.ReportAllocs()
	for b.Loop() {
		kind := errs.KindOf(err)
		_ = kind.Status()
		_ = kind.RPCCode()
	}
}
