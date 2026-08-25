package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/bench/budget"
)

// measuredIDs is what the benchmarks should report, taken from the budget so
// that the tests below do not go stale the day a row is added.
func measuredIDs() []string {
	var out []string
	for _, r := range budget.Rows() {
		if r.Measured() {
			out = append(out, r.ID)
		}
	}
	return out
}

func TestCompareIsHappyWithTheRealSet(t *testing.T) {
	if got := compare(measuredIDs()); len(got) != 0 {
		t.Errorf("the budget does not line up with itself:\n%s", strings.Join(got, "\n"))
	}
}

func TestCompareFindsABenchmarkNobodyBudgeted(t *testing.T) {
	got := compare(append(measuredIDs(), "made/up"))

	if len(got) != 1 || !strings.Contains(got[0], "made/up is measured and has no budget row") {
		t.Errorf("compare said %v", got)
	}
}

func TestCompareFindsAnOperationNobodyMeasures(t *testing.T) {
	ids := measuredIDs()
	dropped := ids[0]

	got := compare(ids[1:])

	if len(got) != 1 || !strings.Contains(got[0], dropped+" is budgeted and not measured") {
		t.Errorf("compare said %v after dropping %s", got, dropped)
	}
}

// TestCompareFindsARowThatArrivedEarly is the case that would otherwise pass
// quietly. The benchmark is there, the row is there, and the row still says the
// work has not started, so the table tells a reader to expect nothing.
//
// The row it uses is whichever one is waiting today, for the same reason
// measuredIDs reads the budget: naming one here means rewriting this test the
// day that row is measured.
func TestCompareFindsARowThatArrivedEarly(t *testing.T) {
	early := waiting(t)

	got := compare(append(measuredIDs(), early.ID))

	if len(got) != 1 || !strings.Contains(got[0], "its row says it arrives with "+early.Since) {
		t.Errorf("compare said %v about %s", got, early.ID)
	}
}

// waiting is the first row in the budget that names a milestone it is waiting
// for, and it skips the test once every row is measured.
func waiting(t *testing.T) budget.Row {
	t.Helper()
	for _, r := range budget.Rows() {
		if !r.Measured() {
			return r
		}
	}
	t.Skip("every row is measured, so there is no early arrival to find")
	return budget.Row{}
}

func TestCompareSortsWhatItFinds(t *testing.T) {
	got := compare([]string{"zzz/made/up", "aaa/made/up"})

	if len(got) < 2 || !strings.HasPrefix(got[0], "aaa/") {
		t.Errorf("compare said %v, want it sorted", got)
	}
}

func TestResultLine(t *testing.T) {
	tests := map[string]struct {
		in   string
		want string
		ok   bool
	}{
		"a result": {
			"BenchmarkBudget/log/info/json-10   \t  867788\t      1501 ns/op\t      48 B/op\t       1 allocs/op",
			"BenchmarkBudget/log/info/json", true,
		},
		"one processor": {
			"BenchmarkBudget/errs/wrap-1 \t 100 \t 1046 ns/op", "BenchmarkBudget/errs/wrap", true,
		},
		"no suffix at all": {
			"BenchmarkThing 100 5 ns/op", "BenchmarkThing", true,
		},
		"an ID ending in a number": {
			"BenchmarkBudget/xs/map/1000-10 \t 214918 \t 5314 ns/op",
			"BenchmarkBudget/xs/map/1000", true,
		},
		"the header":    {"goos: darwin", "", false},
		"the processor": {"cpu: Apple M4", "", false},
		"the end":       {"ok  \tgithub.com/go-mizu/mizu/bench/micro\t13.212s", "", false},
		"pass":          {"PASS", "", false},
		"nothing":       {"", "", false},
		"a log line": {
			"BenchmarkBudget/thing-10: bench_test.go:12: something happened", "", false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := resultLine(tt.in)
			if got != tt.want || ok != tt.ok {
				t.Errorf("resultLine = %q, %v, want %q, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestParseNamesTakesThePrefixOff(t *testing.T) {
	const out = `goos: darwin
goarch: arm64
pkg: github.com/go-mizu/mizu/bench/micro
cpu: Apple M4
BenchmarkBudget/crypt/open/1kb-10  1000000  1629 ns/op  628.76 MB/s  2496 B/op  9 allocs/op
BenchmarkBudget/errs/wrap-10       1000000  1046 ns/op  1184 B/op  7 allocs/op
PASS
ok  	github.com/go-mizu/mizu/bench/micro	13.212s
`

	got := parseNames(out)

	want := []string{"crypt/open/1kb", "errs/wrap"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("parseNames gave %v, want %v", got, want)
	}
}

// TestCheckRunsTheBenchmarks is the whole command against the real module. It
// is the one test here that catches a benchmark registered under one name and
// reported under another, because it reads what the toolchain printed rather
// than what the registry holds.
//
// It costs a few seconds, most of it the two password hashes, so it sits behind
// -short.
func TestCheckRunsTheBenchmarks(t *testing.T) {
	if testing.Short() {
		t.Skip("running the benchmarks takes a few seconds")
	}

	root, err := moduleRoot(".")
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := check(root, &out); err != nil {
		t.Fatalf("check: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "all measured") {
		t.Errorf("check said %q", out.String())
	}
}
