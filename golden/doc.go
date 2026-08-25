// Package golden compares output against a file checked into testdata, and
// rewrites that file when you pass -update.
//
//	func TestRender(t *testing.T) {
//		golden.AssertString(t, render(page))
//	}
//
// The file is testdata/TestRender.golden, and go test ./... -update writes it.
// A subtest goes one directory down, so TestRender/dark_mode is
// testdata/TestRender/dark_mode.golden, which keeps a table test's files
// together instead of spread through one directory with long names.
//
// # When a golden file is the right tool
//
// It is the right tool when the output is large, when it is meant to be read by
// a person, and when a change to it should be visible in a pull request. A
// generated Go file, a rendered template, a formatted error, a compiled SQL
// statement and a JSON response body are all that shape. Writing the expected
// value inline instead means either a fifty line string literal that nobody
// reads or an assertion that checks three fields and misses the rest.
//
// It is the wrong tool when the output is small enough to write down. An
// assertion that says the status was 404 says what it wants. A golden file
// containing 404 says only that the answer has not changed, and if it was wrong
// when it was written it stays wrong, because -update makes agreeing with the
// code the path of least resistance.
//
// That is the failure mode worth naming up front: -update is how a golden file
// gets fixed and it is also how a bug gets blessed. The diff in the pull
// request is the only thing standing between the two, which is why the files
// are checked in, why the output is normalised so a cosmetic change makes no
// diff, and why anything volatile has to be scrubbed rather than left to churn.
//
// # Normalising
//
// [Assert] compares bytes. The rest normalise first, so the file is stable
// against things that are not the point of the test:
//
//	golden.AssertJSON(t, v)     // re-marshalled, keys sorted, indented
//	golden.AssertSQL(t, query)  // whitespace collapsed, keywords left alone
//
// Both normalise the value under test and the file it is compared against, so a
// golden file that was written by hand or edited in an editor still matches.
//
// # Scrubbing
//
// A value that changes on every run makes a golden file useless, and the answer
// is to replace it with something fixed before comparing:
//
//	golden.AssertJSON(t, res, golden.ScrubUUIDs(), golden.ScrubTimes())
//
// That turns every UUID into <uuid> and every RFC 3339 timestamp into <time>,
// in the output and in the file. [Scrub] takes a pattern of your own for
// anything else: a request ID, a temporary directory, a port number.
//
// Scrubbing is a last resort rather than a first one. A test that injects a
// clock and a fixed ID generator has no volatile values to scrub and asserts
// something stronger, which is that the timestamp was the one it set. Reach for
// a scrubber when the value comes from somewhere you do not control.
//
// # Cost
//
// One assertion per test, so none of this is on a hot path. The numbers are
// here to say what a suite of a few thousand golden assertions adds up to.
//
// [Assert] over a 4 kilobyte file is about 21 microseconds and 11 allocations,
// nearly all of it reading the file. [AssertJSON] over the same document is
// about 260 microseconds and 1900 allocations, because both sides are decoded
// into an any and re-encoded, and a decoded JSON document is a map and a slice
// per object and array in it. That is the price of a file that does not change
// when a struct field moves, and at a thousand assertions it is a quarter of a
// second.
//
// [AssertSQL] is cheap, about 2 microseconds and two allocations for a query of
// any normal size, because it is one pass over the bytes.
//
// The diff only runs when a test is already failing, and it is about 84
// microseconds over a 2000 line file.
//
// Timings came from an Apple M4 with other work running on it, so read them as
// ceilings. The allocation counts do not move.
package golden
