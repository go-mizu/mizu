package consoletest

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-mizu/mizu/console"
)

// An example function cannot take a parameter, and everything here needs the
// testing.TB of the test it belongs to. So the examples read one from here, and
// in a real test it is the t the test was handed. They are compiled rather than
// run, which is why none of them carries an Output comment.
var t *testing.T

// Prune deletes users who never verified their email.
type Prune struct {
	Days   int
	DryRun bool
}

func (c *Prune) Spec() console.Spec {
	return console.Spec{
		Name: "users:prune",
		Desc: "Delete users who never verified their email",
		Flags: []console.Flag{
			{Name: "days", Short: 'd', Default: "30", Value: console.Int(&c.Days)},
			{Name: "dry-run", Value: console.Bool(&c.DryRun)},
		},
	}
}

func (c *Prune) Run(ctx context.Context, out *console.IO) error {
	found := 3
	if !c.DryRun {
		ok, err := out.Confirm(fmt.Sprintf("Delete %d users?", found), false)
		if err != nil {
			return err
		}
		if !ok {
			return console.ErrAborted
		}
	}
	out.Line(fmt.Sprintf("Pruned %d users older than %d days.", found, c.Days))
	return nil
}

func Example() {
	r := Run(t, &Prune{},
		Args("--days", "7"),
		Confirm("Delete 3 users?", true))

	r.AssertSuccess()
	r.AssertOutputContains("Pruned 3 users")
}

// A command that takes its fields from the test rather than from a command line
// needs no [Args], which is the shorter way to write a test that is not about
// the parsing.
func ExampleRun() {
	Run(t, &Prune{Days: 7, DryRun: true}).
		AssertSuccess().
		AssertOutput("Pruned 3 users older than 7 days.\n")
}

// Answering no is not a failure of the command. It exits 130, the code for
// somebody who stopped it, rather than 1.
func ExampleConfirm() {
	r := Run(t, &Prune{Days: 30}, Confirm("Delete 3 users?", false))

	r.AssertExitCode(console.CodeInterrupted)
	r.AssertNoOutput()
}

// New asks four questions, so a test for it is four scripted answers and the
// project it should have written.
type New struct{}

func (c *New) Spec() console.Spec { return console.Spec{Name: "new"} }

func (c *New) Run(ctx context.Context, out *console.IO) error { return nil }

func ExampleChoose() {
	r := Run(t, &New{},
		Answer("Project name", "blog"),
		Choose("Database", "postgres"),
		ChooseAll("What else", "queue", "cache"),
		Confirm("Run go mod tidy", true))

	r.AssertSuccess()
	r.AssertAsked("Project name", "Database", "What else", "Run go mod tidy")
}

// A command that reads its stdin gets it from [Input] instead of from a script.
func ExampleInput() {
	r := Run(t, &Validate{}, Input(`{"name": "blog"}`))

	r.AssertSuccess()
	r.AssertNoErrorOutput()
}

// Validate reads a document and says whether it is one.
type Validate struct{}

func (c *Validate) Spec() console.Spec { return console.Spec{Name: "validate"} }

func (c *Validate) Run(ctx context.Context, out *console.IO) error { return nil }
