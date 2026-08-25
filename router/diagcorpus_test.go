package router

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag/diagtest"
)

// TestDiagnostics runs the golden message corpus for this package.
//
// Every directory under testdata/diag holds a routes.txt, which is a route
// table that is wrong in one way, and a want.txt, which is what somebody
// starting a program with that table sees.
//
// A route table is written in Go rather than in a file, so routes.txt is a
// stand-in for one: a line per registration, in the order they were written,
// registered at app/routes.go and that line's number. The location is made up
// on purpose, since the real one is an absolute path on whoever ran the test.
//
// Run it with -update to rewrite the want.txt files, then read the diff. That
// diff is user-facing text and the five rules in doc 36 section 2.1 are the
// review checklist for it.
func TestDiagnostics(t *testing.T) {
	diagtest.Run(t, "testdata/diag", func(tb testing.TB, c diagtest.Case) error {
		return build(tb, c.Lines(tb, "routes.txt"))
	})
}

// TestEveryMessageHasACase holds the corpus to the code: a message this package
// can print and the corpus does not hold is a message nobody has read.
func TestEveryMessageHasACase(t *testing.T) {
	diagtest.Cover(t, "testdata/diag", ".")
}

// build registers a route table written as lines and returns the first thing
// that went wrong.
//
// A line is a pattern, or one of four directives:
//
//	constrain NAME      adds a constraint called NAME to the router
//	constrain-nil NAME  adds one with nothing behind it
//	name NAME           gives the route on the line above the name NAME
//	no-handler PATTERN  registers PATTERN with nothing to run
//
// A pattern that starts with a double quote is unquoted first, which is how a
// table holds an empty pattern or one with a tab in it.
func build(tb testing.TB, lines []string) error {
	var opts []Option
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "constrain "):
			opts = append(opts, Constrain(after(line), isWord))
		case strings.HasPrefix(line, "constrain-nil "):
			opts = append(opts, Constrain(after(line), nil))
		}
	}
	r, err := open(opts...)
	if err != nil {
		return err
	}

	var last *Route
	for i, line := range lines {
		loc := fmt.Sprintf("app/routes.go:%d", i+1)
		switch {
		case strings.HasPrefix(line, "constrain "), strings.HasPrefix(line, "constrain-nil "):
			continue
		case strings.HasPrefix(line, "name "):
			if last == nil {
				tb.Fatalf("%q has no route above it to name", line)
			}
			if err := last.setName(after(line)); err != nil {
				return err
			}
		case strings.HasPrefix(line, "no-handler "):
			if _, err := r.register(patternOf(tb, after(line)), nil, loc); err != nil {
				return err
			}
		default:
			rt, err := r.register(patternOf(tb, line), http200, loc)
			if err != nil {
				return err
			}
			last = rt
		}
	}
	return nil
}

// after is the argument of a directive, which is whatever follows the first
// space.
func after(line string) string {
	_, arg, _ := strings.Cut(line, " ")
	return arg
}

// patternOf is one line of a table as a pattern to register. A line written as a
// Go string literal is unquoted, since a table is a text file and an empty
// pattern or one with a tab in it has to be written somehow.
func patternOf(tb testing.TB, line string) string {
	if !strings.HasPrefix(line, `"`) {
		return line
	}
	s, err := strconv.Unquote(line)
	if err != nil {
		tb.Fatalf("%s is not a quoted pattern: %v", line, err)
	}
	return s
}
