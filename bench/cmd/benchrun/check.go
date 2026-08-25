package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/bench/budget"
)

// prefix is what every budgeted benchmark's reported name starts with. The
// budget ID is the rest of it, which is what makes an ID out of the table work
// as a -bench argument.
const prefix = "BenchmarkBudget/"

// check compares the benchmarks that exist against the budget.
//
// It runs them rather than reading the source, because what matters is the name
// the toolchain reports. A benchmark registered under one ID and run under
// another is a number filed against the wrong row, and reading the registry
// would not catch it.
func check(root string, w io.Writer) error {
	names, err := benchmarkNames(root)
	if err != nil {
		return err
	}

	problems := compare(names)
	if len(problems) == 0 {
		fmt.Fprintf(w, "%s, all measured\n", plural(len(names), "budgeted operation", "budgeted operations"))
		return nil
	}
	for _, p := range problems {
		fmt.Fprintln(w, p)
	}
	return fmt.Errorf("%s not lined up with the budget", plural(len(problems), "operation", "operations"))
}

// compare returns what does not line up, sorted so that two runs read the same.
func compare(names []string) []string {
	var out []string

	have := map[string]bool{}
	for _, n := range names {
		have[n] = true
		row, ok := budget.Lookup(n)
		switch {
		case !ok:
			out = append(out, fmt.Sprintf("%s is measured and has no budget row, add one to bench/budget", n))
		case !row.Measured():
			out = append(out, fmt.Sprintf("%s is measured and its row says it arrives with %s, clear the milestone", n, row.Since))
		}
	}
	for _, r := range budget.Rows() {
		if r.Measured() && !have[r.ID] {
			out = append(out, fmt.Sprintf("%s is budgeted and not measured, write the benchmark or say which milestone brings it", r.ID))
		}
	}

	slices.Sort(out)
	return out
}

// benchmarkNames runs every benchmark once and returns the budget IDs it
// reported, which is the reported name with the BenchmarkBudget prefix and the
// GOMAXPROCS suffix taken off.
//
// One iteration each is enough to learn the names and is the cheapest way to
// ask the toolchain rather than guess. Most of the few seconds it takes is the
// two password hashes, which are meant to be slow.
func benchmarkNames(root string) ([]string, error) {
	cmd := exec.Command("go", "test", "-run=^$", "-bench=.", "-benchtime=1x", "./micro/")
	cmd.Dir = root

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running the benchmarks: %v\n%s%s", err, out, stderr.String())
	}
	return parseNames(string(out)), nil
}

// parseNames pulls the benchmark names out of go test -bench output. A result
// line is the name, the iteration count, and the numbers; everything else in
// the output is a header, a log line, or the pass at the end.
func parseNames(out string) []string {
	var names []string
	for line := range strings.Lines(out) {
		name, ok := resultLine(line)
		if !ok {
			continue
		}
		names = append(names, strings.TrimPrefix(name, prefix))
	}
	return names
}

// resultLine returns the reported name on a benchmark result line, with the
// GOMAXPROCS suffix removed.
func resultLine(line string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 || !strings.HasPrefix(fields[0], "Benchmark") {
		return "", false
	}
	// The second field is the iteration count. Checking it is what keeps a log
	// line that happens to start with the word Benchmark out of the results.
	if _, err := strconv.Atoi(fields[1]); err != nil {
		return "", false
	}

	name := fields[0]
	if i := strings.LastIndexByte(name, '-'); i > 0 {
		if _, err := strconv.Atoi(name[i+1:]); err == nil {
			name = name[:i]
		}
	}
	return name, true
}
