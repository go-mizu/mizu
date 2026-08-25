// Package consoletest is the test fixture for a command: buffers for the
// streams, scripted answers to the questions it asks, and assertions about what
// it wrote and what it exited with.
//
//	func TestUsersPrune(t *testing.T) {
//		r := consoletest.Run(t, &UsersPrune{},
//			consoletest.Args("--days", "7", "acme"),
//			consoletest.Confirm("Delete 3 users?", true))
//
//		r.AssertSuccess()
//		r.AssertOutputContains("Pruned 3 users")
//	}
//
// A command's Run is an ordinary method taking a [console.IO], so a test costs
// what the command costs. Nothing starts a process, opens a terminal or reaches
// for os.Stdout, and a test that fails says why rather than leaving an exit code
// for somebody to work out.
//
// # Answering questions
//
// [Answer], [Confirm], [Choose] and [ChooseAll] are the four things a person
// does at a prompt, and they are matched against the question that was asked
// rather than against the order they were written in:
//
//	consoletest.Run(t, &New{},
//		consoletest.Answer("Project name", "blog"),
//		consoletest.Choose("Database", "postgres"),
//		consoletest.ChooseAll("What else", "queue", "cache"),
//		consoletest.Confirm("Run go mod tidy", true))
//
// [Choose] and [ChooseAll] name the option rather than the number next to it,
// so a test does not have to be rewritten when a choice is added to the list.
//
// A question the script has no answer for fails the test, and so does an answer
// nothing asked for. Both of those are the sort of thing that otherwise shows
// up as a command hanging or as a value arriving in the wrong field, and both
// are reported with the whole of what the command wrote.
//
// # What a failure tells you
//
// The first failure on a result prints both streams and the error, because a
// command that exited 1 when a test wanted 0 is not explained by the exit code:
//
//	--- FAIL: TestUsersPrune (0.00s)
//	    prune_test.go:15: users:prune: failed with no such tenant "acme", want it to succeed
//	        exit 1, error no such tenant "acme"
//	        out (nothing)
//	        err Looking for users to prune.
//	        err error: no such tenant "acme"
//
// Later failures on the same result print the assertion alone.
//
// # What the command sees
//
// Prompts ask, colour is off, and the terminal is 80 columns wide. Those are
// what a test wants rather than what the streams say, since a buffer is not a
// terminal and would otherwise answer no to all three. A test about one of them
// says so with [With]:
//
//	consoletest.Run(t, &Report{}, consoletest.With(console.Options{JSON: true}))
//
// Anything left at auto in those options keeps the answer above.
//
// # Parsing
//
// Without [Args] the command runs on the fields it was built with, which is the
// shorter way to write a test that is about what the command does. With it the
// command line goes through [console.Parse] first, which is how to test the
// flags themselves, the defaults and the environment variables:
//
//	consoletest.Run(t, &UsersPrune{}, consoletest.Args())
//
// That last one is an empty command line rather than no command line, so the
// defaults in the spec are applied.
//
// # What is not here
//
// There is nothing for running a whole [console.App] with scripted answers. An
// App builds its own IO from the global flags, and whether prompts ask is
// decided there by looking at the streams, so a test would have to lie to it
// about being a terminal. Testing dispatch and help needs none of this and can
// call [console.App.Start] with buffers directly.
package consoletest
