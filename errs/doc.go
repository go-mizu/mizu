// Package errs classifies a failure once so that every transport can answer
// for it the same way.
//
// Go has no exceptions and this package does not pretend otherwise. Every
// fallible operation still returns an error, [errors.Is] and [errors.As] still
// reach the cause, and what this adds is a [Kind]: a small closed set that says
// what sort of failure it is.
//
//	return errs.NotFoundf("no post with id %d", id).WithCode("post.not_found")
//
// From that one decision come the HTTP status, the RPC code, the log level,
// whether retrying is worth anything, and whether the message may be shown to
// whoever asked. Nothing downstream has to decide any of those again, so a
// handler returns an error and one place turns it into a response.
//
// # Making one
//
// [New] and [Wrap] take a kind, a code and a message. [Newf] and [Wrapf] format
// the message, and [NotFoundf], [Invalidf], [Forbiddenf], [Conflictf] and
// [Internalf] are the kinds an application writes most often.
//
//	errs.Wrap(err, errs.Unavailable, "search.down", "search is unavailable")
//	errs.Invalidf("page must be a number, got %q", raw)
//
// The With methods each return a copy, so an error can be decorated without
// the one it came from changing underneath anybody.
//
//	e := errs.New(errs.Unprocessable, "validation.failed", "That form has errors.").
//		WithField("title", "required", "Title is required.").
//		WithMeta("form", "post.create")
//
// # Asking about one
//
// [KindOf] is the question almost everything asks. It finds this package's own
// errors, then the standard library errors that mean one definite thing, then
// asks whatever a driver registered.
//
//	switch errs.KindOf(err) {
//	case errs.NotFound:
//	case errs.Timeout, errs.Unavailable:
//	}
//
// A Kind is itself an error, so [errors.Is] works without this package
// exporting a mutable sentinel for each kind:
//
//	if errors.Is(err, errs.Conflict) {
//
// [CodeOf], [Fields], [Meta], [Retryable] and [RetryAfter] read the rest of it
// out of whichever error in the chain has it.
//
// # Errors from elsewhere
//
// [KindOf] recognises [context.Canceled], [context.DeadlineExceeded],
// [fs.ErrNotExist], [fs.ErrPermission], [fs.ErrExist], [io.ErrUnexpectedEOF],
// and anything with a Timeout method that returns true, which covers every
// network timeout.
//
// That list is short on purpose. database/sql, net/http and encoding/json all
// have errors worth mapping, and mapping them here would put those packages in
// every binary that returns an error. [RegisterMapper] is how the package that
// owns one of those says what its errors mean:
//
//	func init() {
//		errs.RegisterMapper(func(err error) (errs.Kind, string, bool) {
//			var pg *pgconn.PgError
//			if errors.As(err, &pg) && pg.Code == "23505" {
//				return errs.Exists, "db.duplicate", true
//			}
//			return 0, "", false
//		})
//	}
//
// # Stacks
//
// A stack is captured only for a kind that logs at error level, and only when
// nothing further down already captured one, so the deepest capture is the one
// that survives being wrapped. A 404 costs no stack, because nobody was ever
// going to print it.
//
// [Error.StackTrace] resolves the frames on the way out rather than at capture
// time, since most errors are logged as one line and dropped.
//
// # What is not here
//
// Rendering. Turning an error into a problem+json document, an HTML page or an
// RPC status is the job of the package that owns the transport, and each of
// them reads the same table from [Kind]. Reporting to something outside the
// process is not here either.
package errs
