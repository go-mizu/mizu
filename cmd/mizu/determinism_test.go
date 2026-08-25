package main

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/gen"
	"github.com/go-mizu/mizu/gen/bindgen"
	"github.com/go-mizu/mizu/gen/commandgen"
	"github.com/go-mizu/mizu/gen/configgen"
)

// Generated code is checked in, which is what makes determinism worth proving
// rather than assuming. A generator that depends on the machine turns the next
// contributor's pull request into a diff nobody asked for, and turns mizu gen
// --check into a check that fails for reasons that have nothing to do with the
// change in front of it.
//
// M0's eighth acceptance criterion names four things the output must not
// depend on: the operating system, the architecture, GOMAXPROCS, and the order
// the input arrives in. The last two are here. The first two come from this
// test running on every row of the Test matrix, which is Linux on amd64 and
// arm64, macOS on arm64, and Windows on amd64. darwin/amd64 is the row that
// matrix does not have, and there is a job for it in test.yml.

// repeats is how many orderings each generator is asked about at each
// GOMAXPROCS value.
//
// Map iteration order is what this is looking for, and Go randomises it per
// range statement rather than per process, so one run proves nothing and a run
// that agrees eight times over two package orderings is the cheapest way to be
// sure. Nothing here takes long enough for a larger number to be interesting.
const repeats = 8

// procs is the two GOMAXPROCS values. One is the value where a data race
// cannot happen and any disagreement is the code rather than the scheduler,
// and the other is high enough to be more than the runner has.
var procs = []int{1, 8}

func TestTheGeneratorsWriteTheSameBytesInAnyOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("the go command type checks the corpus for each ordering")
	}
	for _, c := range []struct {
		name     string
		testdata string
		generate func(...*gen.Package) ([]gen.File, error)
	}{
		{"bind", binds, bindgen.Generate},
		{"command", commands, commandgen.Generate},
		{"config", configs, configgen.Generate},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := scratch(t, c.testdata)
			patterns := spread(t, dir, "app", "b", "c", "d")

			var want map[string]string
			for _, n := range procs {
				restore(t, n)

				pkgs, err := gen.Load(gen.Config{Dir: dir}, patterns...)
				if err != nil {
					t.Fatal(err)
				}
				for _, p := range pkgs {
					if err := p.Err(); err != nil {
						t.Fatalf("%s: %v", p.PkgPath, err)
					}
				}

				for i := range repeats {
					got := written(t, c.generate, shuffle(pkgs, n, i)...)
					if want == nil {
						// One file per package, or the copies did not take and
						// the shuffle is shuffling a list of one.
						if len(got) != len(patterns) {
							t.Fatalf("%d packages produced %d files: %v", len(patterns), len(got), slices.Sorted(maps.Keys(got)))
						}
						want = got
						continue
					}
					same(t, want, got, fmt.Sprintf("GOMAXPROCS %d, ordering %d", n, i))
				}
			}
		})
	}
}

// The agents generator reads the tree rather than a package list, so there is
// no order to shuffle. What is left is the part a shuffle would not have found
// anyway, which is a map ranged over on the way to the output.
//
// This one runs against the repository itself rather than a fixture, because
// the repository is the input it has: sixty packages, four nested modules, and
// the generated files it lists in its own table.
func TestTheAgentsFileIsTheSameEveryTime(t *testing.T) {
	if testing.Short() {
		t.Skip("the generator walks the whole repository for each run")
	}
	here := root(t)
	generate := func(pkgs ...*gen.Package) ([]gen.File, error) { return agents(here, pkgs...) }

	var want map[string]string
	for _, n := range procs {
		restore(t, n)
		for i := range repeats {
			got := written(t, generate)
			if want == nil {
				want = got
				continue
			}
			same(t, want, got, fmt.Sprintf("GOMAXPROCS %d, run %d", n, i))
		}
	}
}

// restore sets GOMAXPROCS for the rest of the test and puts back what was
// there afterwards, so that the next test runs on the machine it was given.
func restore(t *testing.T, n int) {
	t.Helper()
	old := runtime.GOMAXPROCS(n)
	t.Cleanup(func() { runtime.GOMAXPROCS(old) })
}

// shuffle returns the packages in an order that depends on nothing but its
// arguments, so that a failure names an ordering somebody can get back.
func shuffle(pkgs []*gen.Package, n, i int) []*gen.Package {
	out := slices.Clone(pkgs)
	r := rand.New(rand.NewPCG(uint64(n), uint64(i)))
	r.Shuffle(len(out), func(a, b int) { out[a], out[b] = out[b], out[a] })
	return out
}

// spread copies the marked package in a testdata module into siblings and
// returns the patterns that load all of them.
//
// One package has one order, so a fixture with one package cannot show that
// the order does not matter. The copies keep everything but the tests, since
// Load does not read those, and take the directory name as their package name.
func spread(t *testing.T, dir, from string, to ...string) []string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(dir, from))
	if err != nil {
		t.Fatal(err)
	}
	patterns := []string{"./" + from}
	for _, name := range to {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, from, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			renamed := strings.Replace(string(data), "\npackage "+from+"\n", "\npackage "+name+"\n", 1)
			if renamed == string(data) {
				t.Fatalf("%s/%s does not declare package %s", from, e.Name(), from)
			}
			if err := os.WriteFile(filepath.Join(dir, name, e.Name()), []byte(renamed), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		patterns = append(patterns, "./"+name)
	}
	return patterns
}

// written is what the generator's output looks like on disk, keyed by path.
//
// It goes through the writer rather than comparing what Generate returned,
// because the writer is what formats and the bytes worth comparing are the
// ones that land in the file. The order the files come back in is not one of
// them, which is why this is a map: a shuffled input is meant to move the
// files around and to leave every one of them alone.
func written(t *testing.T, generate func(...*gen.Package) ([]gen.File, error), pkgs ...*gen.Package) map[string]string {
	t.Helper()

	files, err := generate(pkgs...)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("the generator wrote no files, so there is nothing to compare")
	}
	dir := t.TempDir()
	w := &gen.Writer{Dir: dir}
	if _, err := w.Write(files...); err != nil {
		t.Fatal(err)
	}
	out := make(map[string]string, len(files))
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f.Path)))
		if err != nil {
			t.Fatal(err)
		}
		out[f.Path] = string(data)
	}
	return out
}

// same reports every way the second run disagreed with the first.
func same(t *testing.T, want, got map[string]string, where string) {
	t.Helper()
	for _, path := range slices.Sorted(maps.Keys(want)) {
		switch data, ok := got[path]; {
		case !ok:
			t.Errorf("%s: %s was written the first time and not this time", where, path)
		case data != want[path]:
			t.Errorf("%s: %s came out differently:\n%s", where, path, difference(want[path], data))
		}
	}
	for _, path := range slices.Sorted(maps.Keys(got)) {
		if _, ok := want[path]; !ok {
			t.Errorf("%s: %s was written this time and not the first time", where, path)
		}
	}
}

// difference is the line the two runs disagree on, which is more use in a
// failure than either file in full.
func difference(want, got string) string {
	a, b := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := range min(len(a), len(b)) {
		if a[i] != b[i] {
			return fmt.Sprintf("line %d\n  first run: %s\n  this run:  %s", i+1, a[i], b[i])
		}
	}
	return fmt.Sprintf("the first run wrote %d lines and this one wrote %d", len(a), len(b))
}
