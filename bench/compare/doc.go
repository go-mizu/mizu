// Package compare measures mizu against other frameworks.
//
// This is the part of a benchmark suite that is usually dishonest, so the rules
// are written down before there is anything here to argue about.
//
// A comparison runs the same scenario on every framework, on the same machine,
// in the same run, with each one configured the way its own documentation says
// to configure it for production. Not the way that makes it lose. If a
// framework has a faster path that its own guide recommends, the comparison
// takes it, and if taking it costs something in safety or in developer
// experience, that goes in the writeup next to the number rather than being
// quietly declined.
//
// The harnesses for other frameworks live in this directory and are not Go
// modules the toolkit depends on. Anything that needs a runtime mizu does not
// have runs in a container, described by a file here.
//
// What gets published is the scenario, the configuration, the machine, and the
// histogram. A single number with no way to reproduce it is marketing, and it
// is also useless to the person deciding whether to switch, which is who the
// comparison is for.
//
// # When
//
// The first comparison is the plaintext and json pair against net/http, chi,
// echo and gin, which arrives once the macro scenarios do in M1. The
// cross-language ones, Laravel for the shape of the framework and Rails for the
// shape of the argument, follow the reference blog in M4. The package is here
// now so the rules above are in the repository before the first number is.
package compare
