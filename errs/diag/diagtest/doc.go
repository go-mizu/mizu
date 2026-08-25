// Package diagtest runs the golden message corpus.
//
// The corpus is what makes the five rules in doc 36 section 2.1 enforceable
// rather than aspirational. Each entry is a directory holding an input that is
// deliberately broken and the report mizu produces for it, and the report is
// checked in beside the input. Changing a message changes a file, so it appears
// in a pull request as user-facing text and gets reviewed as user-facing text,
// which is the only mechanism that has ever kept error messages good.
//
// A corpus lives beside the package whose messages it holds, under
// testdata/diag, and each package runs it with one test:
//
//	func TestDiagnostics(t *testing.T) {
//		diagtest.Run(t, "testdata/diag", func(tb testing.TB, c diagtest.Case) error {
//			return load(c.Path("app.toml"))
//		})
//	}
//
// Beside the code rather than in one directory at the root, so that a new
// diagnostic and its corpus entry are the same change to the same package,
// which is the rule doc 36 asks for. The entries are still countable across the
// repository, since they are all under the same path.
//
// # What an entry looks like
//
//	console/testdata/diag/unknown-flag/
//	    args        the command line, one token per line
//	    want.txt    the report, written by -update and read by a person
//
// The input files are whatever the case needs and the producer knows what to
// ask for. It asks the case for them, with [Case.Path] or [Case.Read], and the
// path to the case's own directory is taken back out of the report before the
// golden file is compared. So an entry whose message quotes a file says
// app.toml on every platform rather than a path with the separator of whoever
// ran it, and moving the corpus does not rewrite every file in it.
//
// # What [Run] checks
//
// The golden file is the part a person reviews. Around it are the parts a
// machine can hold, which is [Check]: a message is one line, it starts the way
// a Go error message starts, it does not end in a full stop, it does not reach
// for the words that say nothing, and a code on it is a code the registry knows
// about. A report is also rendered twice and required to come out the same,
// because a list built from a map is a report that changes between runs and a
// golden file is the only place that shows up.
//
// An entry whose input produced no error at all fails. A corpus of deliberately
// broken inputs where nothing broke is a corpus that has stopped testing
// anything, and the usual cause is that the input drifted rather than that the
// message got better.
package diagtest
