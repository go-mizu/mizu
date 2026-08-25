package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/errs/diag"
	"github.com/go-mizu/mizu/lint"
)

// Lint reports the mistakes a compiler cannot.
//
// The work is in the lint package. What is here is the command around it: which
// packages to read, which checks to run, and what the answer looks like on a
// terminal and under --json.
type Lint struct {
	Packages []string
	Checks   []string
}

func (c *Lint) Spec() console.Spec {
	return console.Spec{
		Name: "lint",
		Desc: "Report the mistakes the compiler cannot",
		Long: lintLong + "\n\nThe checks:\n\n" + lintChecks(),
		Flags: []console.Flag{
			{Name: "check", Desc: "Run only these checks, comma separated", Value: console.Strings(&c.Checks, ",")},
		},
		Args: []console.Arg{
			{Name: "packages", Rest: true, Desc: "Package patterns, ./... by default", Value: console.Strings(&c.Packages, "")},
		},
	}
}

// lintChecks is the checks, for the help text. Somebody deciding whether to
// believe a green run wants to know what was run.
func lintChecks() string {
	var b strings.Builder
	width := 0
	for _, ch := range lint.Checks() {
		width = max(width, len(ch.Name))
	}
	for _, ch := range lint.Checks() {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, ch.Name, ch.Doc)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (c *Lint) Run(ctx context.Context, io *console.IO) error {
	patterns := c.Packages
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	pkgs, err := load("", patterns)
	if err != nil {
		return err
	}

	// Loading is the slow part and nothing in it takes a context, so this is
	// where a run somebody interrupted stops.
	if err := ctx.Err(); err != nil {
		return err
	}

	found, err := lint.Run(pkgs, c.Checks...)
	if err != nil {
		return err
	}

	// The findings are the answer, so they go to stdout in both modes, and the
	// line below is why the exit code is what it is.
	if err := io.Diag(found.Err()); err != nil {
		return err
	}
	if n := found.Count(diag.Error); n > 0 {
		return fmt.Errorf("%s", plural(n, "problem"))
	}
	io.Success("%s, nothing to report.", plural(len(pkgs), "package"))
	return nil
}

const lintLong = `Every check here is about a rule some mizu package makes that the type system
does not, such as a *web.Ctx that stops being valid when the handler returns
and is kept anyway. A rule like that is a comment in a doc string until
something reads the code and says where it was broken.

With no packages it runs over ./..., which is what a project wants and what
mizu verify runs.

A check reports what it is sure of. Nothing here follows a value through an
interface or an any, so a check missing something is expected and a check
inventing something is a bug. The other half of the rule is the guarded build,
which catches at run time what reading the source cannot.

--check runs the named checks and nothing else. A name that is not a check is
an error rather than a run of nothing.

--json gives the mizu.diag/1 document, which is every finding with its code,
its place and what to do about it.`
