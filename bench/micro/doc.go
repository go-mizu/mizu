// Package micro measures one operation at a time.
//
// Every benchmark here is a row in [github.com/go-mizu/mizu/bench/budget], and
// every row that is not waiting on a later milestone has a benchmark here.
// benchrun check holds the two together, so the budget table cannot quietly
// stop describing the code and an operation cannot quietly stop being watched.
//
// They all run as subtests of one BenchmarkBudget, with the budget ID as the
// name. That is what makes an ID copied out of the table work as a command line
// argument:
//
//	go test -run='^$' -bench=Budget/crypt/seal/1kb ./micro/
//
// and it is what lets a run be lined up against the table without a translation
// step in between.
//
// A micro benchmark measures a call, which is the level a change is made at and
// so the level a regression is found at. It is not the level anything is
// experienced at: a page load is the sum of a few hundred of these plus a
// database, and bench/macro is where that gets measured. A number here that
// improves while the macro numbers do not has improved nothing.
package micro
