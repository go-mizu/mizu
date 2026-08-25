// Package bench is the root of the benchmark module and holds the fixed
// corpora the benchmarks run against.
//
// The benchmarks live in a module of their own so that load generators,
// comparison harnesses and analysis tools never enter an application's
// dependency graph. Somebody who runs go get github.com/go-mizu/mizu gets the
// toolkit and the standard library, and nothing here reaches them.
//
// # Layout
//
//	budget/          the performance budget, as data
//	micro/           one operation at a time, standard testing.B
//	macro/           full-server scenarios driven over a real socket
//	scenarios/       the reference applications
//	compare/         cross-framework and cross-language harnesses
//	cmd/benchrun/    check, lint, and the generated table
//	testdata/        fixed corpora
//
// # Running them
//
// Everything at once, which is what a laptop does before a pull request:
//
//	cd bench && go test -run='^$' -bench=. ./micro/
//
// One operation, which is what somebody working on that operation does. The
// benchmark names are the budget IDs, so an ID copied out of the table is a
// command line argument:
//
//	go test -run='^$' -bench=Budget/log/info/json -benchtime=3s ./micro/
//
// Before comparing two runs, take more than one sample of each, because a
// single number off a shared machine is not a measurement:
//
//	go test -run='^$' -bench=. -count=10 ./micro/ > new.txt
//	benchstat old.txt new.txt
//
// # Rules
//
// These make the numbers worth keeping, and benchrun lint enforces them:
//
//   - Every benchmark calls b.ReportAllocs and drives its loop with b.Loop.
//     b.Loop keeps the compiler from deleting work whose result nobody reads
//     and leaves setup out of the measurement without stopping and starting the
//     timer by hand.
//   - Nothing reads the clock or ranges over a map inside the measured region.
//     Both make the input vary between runs, and input variance is
//     indistinguishable from a regression.
//   - Nothing uses the package-level functions of math/rand, which draw from a
//     source seeded differently on every run. A benchmark that wants random
//     input builds its own source from a fixed seed.
//   - Inputs come from testdata and are listed in testdata/README.md, which
//     says what changing one costs.
//
// # What the numbers are for
//
// A benchmark here answers whether a change made something slower, which is a
// question about two runs on the same machine minutes apart. It does not answer
// how fast mizu is, which is a question about a machine nobody here is sitting
// at. The budget in bench/budget is the design target and the run output is
// what happened; a regression is the second one moving against the last
// recorded baseline, not against the first.
package bench
