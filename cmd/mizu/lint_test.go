package main

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/console/consoletest"
)

// runLint runs one command line. The command is built fresh every time because
// it holds what its flags parsed into.
func runLint(t *testing.T, argv ...string) *consoletest.Result {
	t.Helper()
	return consoletest.Run(t, &Lint{}, consoletest.Args(argv...))
}

// held is a package that keeps a *web.Ctx in a field, which is the rule the ctx
// check is about.
const held = `package app

import "github.com/go-mizu/mizu/web"

// A job is what a handler started and something else finishes.
type job struct {
	req *web.Ctx
}

// Start begins one.
func Start(c *web.Ctx) error {
	_ = job{req: c}
	return c.NoContent()
}
`

func TestLintOnAProjectWithNothingWrong(t *testing.T) {
	scratch(t, commands)

	runLint(t).
		AssertSuccess().
		AssertNoOutput().
		AssertErrorContains("nothing to report")
}

func TestLintReportsARuleThatWasBroken(t *testing.T) {
	dir := scratch(t, commands)
	add(t, filepath.Join(dir, "held", "job.go"), held)

	r := runLint(t)
	if err := r.AssertFailure(); !strings.Contains(err.Error(), "1 problem") {
		t.Errorf("the command failed with %v, and it found one problem", err)
	}
	r.AssertOutputContains("MZ3001")
	r.AssertOutputContains("job.go")

	// The quoted line, which is the reason a report is worth more than a list
	// of positions.
	r.AssertOutputContains("req *web.Ctx")
}

func TestLintReadsOnlyThePackagesItWasGiven(t *testing.T) {
	dir := scratch(t, commands)
	add(t, filepath.Join(dir, "held", "job.go"), held)

	runLint(t, "./app/...").
		AssertSuccess().
		AssertNoOutput()
}

func TestLintRunsOnlyTheChecksItWasGiven(t *testing.T) {
	dir := scratch(t, commands)
	add(t, filepath.Join(dir, "held", "job.go"), held)

	if err := runLint(t, "--check=ctx").AssertFailure(); err == nil {
		t.Fatal("naming the check that reports this found nothing")
	}
}

// A name that is not a check is an error rather than a run of nothing. A typo
// that quietly passes is a lint nobody is running.
func TestLintRefusesACheckThatDoesNotExist(t *testing.T) {
	scratch(t, commands)

	err := runLint(t, "--check=ctxs").AssertFailure()
	if !strings.Contains(err.Error(), "ctxs") || !strings.Contains(err.Error(), "ctx") {
		t.Errorf("the message does not say what was typed and what the checks are: %v", err)
	}
}

// Packages that cannot be read are not packages with nothing wrong with them.
func TestLintOnSomethingThatIsNotAProject(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := runLint(t).AssertFailure(); err == nil {
		t.Error("a directory that is not a module came back clean")
	}
}

// Loading is the slow part and nothing in it takes a context, so a run somebody
// interrupted stops after it rather than reading a project nobody is waiting to
// hear about.
func TestLintStopsWhenTheRunIsCancelled(t *testing.T) {
	scratch(t, commands)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	r := consoletest.Run(t, &Lint{}, consoletest.Args(), consoletest.Context(ctx))
	if err := r.AssertFailure(); !errors.Is(err, context.Canceled) {
		t.Errorf("the command failed with %v, want it to stop", err)
	}
}

// The JSON is the mizu.diag/1 document, which is what a CI job annotating a
// pull request reads.
func TestLintAnswersWithDiagnostics(t *testing.T) {
	dir := scratch(t, commands)
	add(t, filepath.Join(dir, "held", "job.go"), held)

	r := consoletest.Run(t, &Lint{},
		consoletest.Args(),
		consoletest.With(console.Options{JSON: true}))
	r.AssertFailure()

	var doc struct {
		Diagnostics []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			File    string `json:"file"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(r.Stdout()), &doc); err != nil {
		t.Fatalf("stdout is not a document: %v\n%s", err, r.Stdout())
	}
	if len(doc.Diagnostics) != 1 {
		t.Fatalf("the document holds %d diagnostics, want 1", len(doc.Diagnostics))
	}
	if got := doc.Diagnostics[0].Code; got != "MZ3001" {
		t.Errorf("the diagnostic carries the code %q", got)
	}
}

// The help lists what a run covers, since somebody deciding whether to believe
// a green run wants to know what was run.
func TestTheHelpListsEveryCheck(t *testing.T) {
	long := (&Lint{}).Spec().Long
	if !strings.Contains(long, "ctx") {
		t.Errorf("the help says nothing about the ctx check:\n%s", long)
	}
}
