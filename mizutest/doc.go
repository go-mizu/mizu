// Package mizutest is the test fixture for a mizu application: a handler, a
// clock you control, a recorded log, and a request builder that reads like the
// request it sends.
//
//	func TestShowPost(t *testing.T) {
//		app := mizutest.NewApp(t)
//		app.Routes().Get("/posts/{id}", show)
//
//		res := app.Get("/posts/1").
//			WithHeader("Accept", "application/json").
//			Do()
//
//		res.AssertOK()
//		res.AssertJSONPath("$.data.title", "Hello")
//	}
//
// Nothing here starts a server or opens a socket. A request goes through
// [net/http/httptest.NewRecorder] straight into the handler, so a test costs
// about what the handler costs and there is no port to conflict over.
//
// # What a failure tells you
//
// The usual reason a test fails is that the handler returned 500 and the
// assertion says the status was not 200, which is true and useless. So the
// first failure on a response prints the whole exchange: the request line, the
// status, the response headers, the body up to 2 kB, and every error the
// handler logged while it ran.
//
//	--- FAIL: TestShowPost (0.00s)
//	    post_test.go:14: GET /posts/1: status is 500, want 200
//	        > GET /posts/1
//	        > Accept: application/json
//	        < 500 Internal Server Error
//	        < Content-Type: text/plain; charset=utf-8
//	        < Internal Server Error
//	        log ERROR handler error error=sql: no rows in result set
//
// The reason for the 500 is on the last line rather than somewhere in the
// scrollback. Later failures on the same response print the assertion alone,
// since the exchange has already been shown.
//
// # Time
//
// The fixture puts a fake clock in the context it hands to every request, so a
// handler that reads [github.com/go-mizu/mizu/clock].Now gets the time the test
// chose:
//
//	app := mizutest.NewApp(t, mizutest.At(billingPeriodEnd))
//	app.Clock().Advance(time.Hour) // and the period is over
//
// The default is 2026-01-01T00:00:00Z, which is a definite time and an
// obviously synthetic one. A test whose subject is a date should say which date
// rather than lean on the default.
//
// # Tests run in parallel
//
// [NewApp] calls Parallel on the test unless you pass [NoParallel]. Two things
// follow from that and both of them are panics rather than failures, so they
// are worth knowing before you meet them. A test that calls t.Parallel itself
// panics with "testing: t.Parallel called multiple times", and a test that
// calls t.Setenv panics with "cannot use t.Setenv in parallel tests". Pass
// [NoParallel] in either case.
//
// The first of those also turns up in a test that builds two fixtures, since
// the second one calls Parallel again. Neither panic names this package, so
// the rule to remember is that one test gets one call, and [NoParallel] on
// every fixture after the first:
//
//	app := mizutest.NewApp(t)
//	wrapped := mizutest.NewApp(t, mizutest.NoParallel(), mizutest.Serve(mw(app.Handler())))
//
// # What is not here yet
//
// The fixture is the shape it will keep, with the subsystems filled in as they
// land. ActingAs, WithSession and the session assertions wait for the session
// and auth work. Queue, Events, Mail, Storage and DB accessors wait for those
// packages. DumpQueries and the N+1 detector wait for the database. Each is
// named in the milestone that brings it rather than stubbed out here, because a
// method that exists and does nothing is worse than one that does not exist.
//
// # Cost
//
// A suite makes thousands of requests rather than millions, and the handler
// under test is nearly always the expensive part. These numbers are here so
// that a suite which has got slow can be blamed on the right thing.
//
// Building a fixture is about 3 microseconds and 33 allocations. A request
// through it, counting building the request, serving it and recording what came
// back, is about 5 microseconds and 30 allocations. Through a whole mizu
// application, with the router, the middleware and the encoding, the same
// request is about 33 microseconds. At a thousand requests that is a
// thirtieth of a second, which is not where a slow suite went.
//
// A status or body assertion is a few hundred nanoseconds and allocates
// nothing, and most of that is t.Helper rather than the comparison.
//
// The JSON assertions are the ones with a real price. A response decodes its
// body on the first one and keeps the result, so the body is paid for once
// however many things are asked of it: about 11 microseconds and 86 allocations
// for a document of a few hundred bytes, because a decoded document is a map or
// a slice per object and array in it. After that, [Response.AssertJSONPath] is
// about 1 microsecond, [Response.AssertJSONSubset] about 3.6, and
// [Response.AssertJSON] about 13, the last being higher because the expected
// value is decoded again on every call.
//
// Recording a log entry is about 800 nanoseconds and 3 allocations, and it is
// the one thing here that runs as often as the code under test does.
//
// The failure output is about 1.4 microseconds, and it only runs when a test is
// already failing.
//
// Timings came from an Apple M4 with other work running on it, so read them as
// ceilings. The allocation counts do not move.
package mizutest
