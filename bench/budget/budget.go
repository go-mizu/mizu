// Package budget holds the performance budget: the operations mizu promises to
// stay under, and by how much.
//
// It is data rather than prose because more than one thing reads it. The
// benchmarks in bench/micro are named after the rows, benchrun check holds the
// two together so that a row nobody measures and a measurement nobody budgeted
// are both a failure, and benchrun table prints the whole thing back as the
// markdown table the performance document carries. The document is generated
// from this file rather than kept in step with it by hand.
//
// A number here is a target and a ceiling, not a measurement. What a machine
// actually did is in the run output, and what counts as a regression is the
// comparison against the last recorded baseline rather than against this file.
// A budget moves when the design moves and somebody argues for it, which is why
// every row names the document it comes from.
package budget

import (
	"slices"
	"time"
)

// NoBudget marks a row the table gives no number for, either because the count
// is not the interesting thing about the operation or because the operation is
// a build step rather than a call. It is negative so that a real zero, which is
// what a generated accessor costs, stays a zero.
const NoBudget = -1

// A Row is one operation the framework has a number for.
type Row struct {
	// ID names the operation and is the name of the benchmark that measures
	// it. The benchmark runs as a subtest of BenchmarkBudget, so the full name
	// the toolchain reports is BenchmarkBudget/<ID>.
	ID string

	// Group is the heading the row appears under in the performance document.
	Group string

	// Op is what is being measured, in enough detail to write the benchmark
	// from.
	Op string

	// Time is the budget for one operation, or NoBudget.
	Time time.Duration

	// Allocs is the budget for allocations per operation, or NoBudget. An
	// operation over a collection is budgeted for the whole collection, so a
	// scan of a thousand rows at two allocations each is two thousand here.
	Allocs int

	// Doc is the design document the operation is specified in, as the number
	// its filename starts with.
	Doc string

	// Since is the milestone that brings the benchmark, and is empty for an
	// operation that is measured today. Everything the framework has not built
	// yet is in the table with the milestone that will build it, so the table
	// is the whole plan rather than the part that happens to be finished.
	Since string
}

// Measured reports whether the operation has a benchmark now. benchrun check
// requires one for exactly these rows.
func (r Row) Measured() bool { return r.Since == "" }

// Rows returns the whole budget, in the order the performance document prints
// it. The result is a copy, so a caller may sort it.
func Rows() []Row { return slices.Clone(rows) }

// Lookup returns the row for an ID.
func Lookup(id string) (Row, bool) {
	i := slices.IndexFunc(rows, func(r Row) bool { return r.ID == id })
	if i < 0 {
		return Row{}, false
	}
	return rows[i], true
}

// Groups returns the group names in the order the document prints them.
func Groups() []string {
	var out []string
	for _, r := range rows {
		if !slices.Contains(out, r.Group) {
			out = append(out, r.Group)
		}
	}
	return out
}
