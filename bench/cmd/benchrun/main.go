// Benchrun keeps the benchmarks and the budget describing the same thing.
//
// Usage:
//
//	benchrun check    every budgeted operation has a benchmark, and every benchmark has a row
//	benchrun lint     the rules that make two runs comparable
//	benchrun table    the budget, as the markdown table the performance document carries
//
// It works from anywhere inside the benchmark module, or from a directory named
// with -C.
//
// check runs the benchmarks once each and compares the names the toolchain
// reports against the budget. That costs a few seconds, most of it the two
// password hashes, and it is the check that cannot be fooled by a benchmark
// registered under one name and run under another.
//
// lint reads the source. The rules are in the package documentation for
// github.com/go-mizu/mizu/bench, and each one is there because breaking it
// makes two runs stop being comparable without making either of them look
// wrong.
//
// table prints what belongs in section 3 of the performance document. The
// document is generated from the budget rather than kept in step with it by
// hand, so a row that moves moves in one place.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	log := func(format string, a ...any) {
		fmt.Fprintf(os.Stderr, "benchrun: "+format+"\n", a...)
	}

	dir := flag.String("C", ".", "run as if started in this directory")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}

	root, err := moduleRoot(*dir)
	if err != nil {
		log("%v", err)
		os.Exit(1)
	}

	if err := run(flag.Arg(0), root, os.Stdout); err != nil {
		log("%v", err)
		os.Exit(1)
	}
}

const usage = `usage: benchrun [-C dir] <command>

commands:
  check    every budgeted operation has a benchmark, and every benchmark has a row
  lint     the rules that make two runs comparable
  table    the budget, as the markdown table the performance document carries
`

// run does one command. It is separate from main so that the tests can call it
// with a writer they can read back.
func run(command, root string, w io.Writer) error {
	switch command {
	case "check":
		return check(root, w)
	case "lint":
		return lint(root, w)
	case "table":
		_, err := io.WriteString(w, table())
		return err
	default:
		return fmt.Errorf("unknown command %q, try one of check, lint, table", command)
	}
}

// module is the module the benchmarks live in. moduleRoot looks for it rather
// than for any go.mod, so that running from inside the toolkit next door says
// so instead of quietly linting the wrong tree.
const module = "module github.com/go-mizu/mizu/bench"

// moduleRoot returns the directory holding the benchmark module's go.mod,
// starting at dir and walking up.
func moduleRoot(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		switch {
		case err == nil && hasModuleLine(string(b)):
			return dir, nil
		case err != nil && !errors.Is(err, os.ErrNotExist):
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no benchmark module at or above %s, run this from inside bench/", dir)
		}
		dir = parent
	}
}

// plural writes a count with the right noun after it. It is here because "1
// problems" at the end of a build log reads as a tool somebody did not finish,
// and this output is read far more often than it is written.
func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

func hasModuleLine(gomod string) bool {
	for line := range strings.Lines(gomod) {
		if strings.TrimSpace(line) == module {
			return true
		}
	}
	return false
}
