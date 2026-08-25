package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/gen"
)

// A stage is one thing verify runs.
//
// The output is what the stage printed, shown when it failed and when it
// changed something. The error is why it failed, in one line, because the
// output above it is already the detail.
//
// A stage that is quick is one mizu check runs. That is the difference between
// the two commands, and it is a field here rather than a second list so the two
// cannot drift apart.
type stage struct {
	name  string
	what  string
	quick bool
	run   func(ctx context.Context, p project, fix bool) (out string, err error)
}

// stages is what verify runs, in the order it runs them.
//
// The order is dependency order, and it is the reason verify stops at the first
// failure. Code that does not compile has nothing useful to say about its
// tests, and a stale generated file makes every stage after it a report about
// something nobody wrote.
var stages = []stage{
	{"gen", "Generated files say what the code they come from says now", false, stageGen},
	{"fmt", "Every Go file is formatted", false, stageFmt},
	{"vet", "The packages type check and vet has nothing to say", true, stageVet},
	{"build", "Every package builds", false, stageBuild},
	{"test", "The short tests pass", false, stageTest},
	{"doctor", "The project checks pass", false, stageDoctor},
}

// A stageResult is what happened in one stage.
type stageResult struct {
	Stage   string  `json:"stage"`
	Status  string  `json:"status"` // ok, fixed, failed or skipped
	Seconds float64 `json:"seconds"`
	Output  string  `json:"output,omitempty"`
	Error   string  `json:"error,omitempty"`
}

// A verification is the whole run.
//
// Every stage is in the list, including the ones that did not run, because a
// reader wants to know what was skipped as much as what failed.
type verification struct {
	OK     bool          `json:"ok"`
	Stages []stageResult `json:"stages"`
}

// Verify runs everything that has to be true before a change is finished.
//
// mizu check is this command with the quick stages only, registered from the
// same struct for the same reason mizu gen:command is: two commands that are
// one command with a different scope should not be two implementations.
type Verify struct {
	// find works out what project this is, and is a field so a test can say
	// what the environment looks like.
	find func(context.Context) (project, error)

	// quick makes this mizu check rather than mizu verify.
	quick bool

	Fix bool
}

func (c *Verify) Spec() console.Spec {
	if c.quick {
		return console.Spec{
			Name: "check",
			Desc: "Type check and vet, the fastest answer there is",
			Long: checkLong + "\n\nWhat it runs:\n\n" + stageList(c.stages()),
		}
	}
	return console.Spec{
		Name: "verify",
		Desc: "Run everything that has to pass before a change is finished",
		Long: verifyLong + "\n\nWhat it runs, in order:\n\n" + stageList(c.stages()),
		Flags: []console.Flag{
			{Name: "fix", Desc: "Write what can be written, then carry on", Value: console.Bool(&c.Fix)},
		},
	}
}

// stages is what this command runs.
func (c *Verify) stages() []stage {
	if !c.quick {
		return stages
	}
	var quick []stage
	for _, s := range stages {
		if s.quick {
			quick = append(quick, s)
		}
	}
	return quick
}

// stageList is the stages, for the help text. Somebody deciding whether a green
// run means anything wants to know what was run.
func stageList(list []stage) string {
	var b strings.Builder
	width := 0
	for _, s := range list {
		width = max(width, len(s.name))
	}
	for _, s := range list {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, s.name, s.what)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (c *Verify) Run(ctx context.Context, io *console.IO) error {
	find := c.find
	if find == nil {
		find = discover
	}
	p, err := find(ctx)
	if err != nil {
		return err
	}

	list := c.stages()
	v := verification{OK: true, Stages: make([]stageResult, 0, len(list))}
	for i, s := range list {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !v.OK {
			// Everything after a failure is a report about code that is already
			// known to be wrong, and running it wastes the seconds this command
			// exists to save.
			v.Stages = append(v.Stages, stageResult{Stage: s.name, Status: "skipped"})
			continue
		}

		start := time.Now()
		out, err := s.run(ctx, p, c.Fix)
		// The time is rounded, because nanoseconds in a report about a command
		// that takes seconds are digits nobody reads.
		r := stageResult{
			Stage:   s.name,
			Status:  "ok",
			Seconds: time.Since(start).Round(time.Millisecond).Seconds(),
			Output:  strings.TrimRight(out, "\n"),
		}
		switch {
		case err != nil:
			r.Status, r.Error = "failed", err.Error()
			v.OK = false
		case r.Output != "":
			// A stage with something to say under --fix is a stage that
			// changed the tree, which is worth its own word.
			r.Status = "fixed"
		}
		v.Stages = append(v.Stages, r)

		if !io.JSONMode() {
			c.say(io, r, i == 0)
		}
	}
	return c.report(io, v)
}

// say prints one finished stage.
//
// It prints as the stage finishes rather than at the end, because a command
// with a 45 second budget that says nothing for 45 seconds gets interrupted by
// somebody who thinks it hung.
func (c *Verify) say(io *console.IO, r stageResult, first bool) {
	if first {
		io.Line("")
	}
	io.Print("  %-7s %-7s %5.1fs\n", r.Stage, r.Status, r.Seconds)
	if r.Output != "" {
		io.Line("")
		io.Line(indent(r.Output))
		io.Line("")
	}
}

// indent puts the output of a tool under the line that named the stage, so a
// long go test failure reads as part of the report rather than as the report.
func indent(out string) string {
	var b strings.Builder
	for line := range strings.Lines(out) {
		if strings.TrimSpace(line) != "" {
			b.WriteString("    ")
		}
		b.WriteString(line)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (c *Verify) report(io *console.IO, v verification) error {
	if io.JSONMode() {
		if err := io.JSON(v); err != nil {
			return err
		}
	}

	if v.OK {
		if !io.JSONMode() {
			io.Line("")
			io.Success("%s passed.", plural(len(v.Stages), "stage"))
		}
		return nil
	}

	// The failing stage is the last one that ran, and its name is the whole
	// summary. What went wrong is already above, or in the JSON.
	failed := v.Stages[len(v.Stages)-1]
	for _, r := range v.Stages {
		if r.Status == "failed" {
			failed = r
		}
	}
	return fmt.Errorf("%s: %s", failed.Stage, failed.Error)
}

// stageGen checks that the generated files say what the code they come from
// says now, and writes them under --fix.
func stageGen(ctx context.Context, p project, fix bool) (string, error) {
	results, err := generate(ctx, p, fix)
	if err != nil {
		// Code that does not compile is the usual reason, and the compiler said
		// why, so what it said is the output and this line is the stage.
		return err.Error(), errors.New("the packages could not be read")
	}

	var changed []string
	for _, r := range results {
		if r.Changed() {
			changed = append(changed, r.Path)
		}
	}
	if len(changed) == 0 {
		return "", nil
	}
	if fix {
		return "wrote " + strings.Join(changed, ", "), nil
	}
	return strings.Join(changed, "\n"), fmt.Errorf("%s out of date, run mizu gen", plural(len(changed), "generated file"))
}

// generate runs every generator over the project, and either writes what they
// asked for or reports what it would have written.
func generate(ctx context.Context, p project, write bool) ([]gen.Result, error) {
	pkgs, err := load(p.Dir, []string{"./..."})
	if err != nil {
		return nil, err
	}

	var files []gen.File
	for _, g := range generators {
		out, err := g.run(p.Dir, pkgs...)
		if err != nil {
			return nil, err
		}
		files = append(files, out...)
	}

	// Loading is the slow part and nothing in it takes a context, so this is
	// where a run that was interrupted stops, before anything is written.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	w := &gen.Writer{Dir: p.Dir, Check: !write}
	return w.Write(files...)
}

// stageFmt checks that every Go file in the project is formatted, and formats
// them under --fix.
//
// This is gofmt, run in this process rather than found on PATH. A gofmt on PATH
// is whichever Go somebody installed first, and a formatting rule that depends
// on that is a rule that reformats the tree back and forth between two
// machines.
func stageFmt(_ context.Context, p project, fix bool) (string, error) {
	var wrong []string
	for name := range projectFiles(os.DirFS(p.Dir)) {
		if filepath.Ext(name) != ".go" {
			continue
		}
		path := filepath.Join(p.Dir, filepath.FromSlash(name))
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		want, err := format.Source(data)
		if err != nil {
			// A file that does not parse is not a formatting problem, and vet
			// is the stage that says what is wrong with it.
			continue
		}
		if bytes.Equal(data, want) {
			continue
		}
		if fix {
			if err := os.WriteFile(path, want, 0o644); err != nil {
				return "", err
			}
		}
		wrong = append(wrong, name)
	}

	if len(wrong) == 0 {
		return "", nil
	}
	if fix {
		return "formatted " + strings.Join(wrong, ", "), nil
	}
	return strings.Join(wrong, "\n"), fmt.Errorf("%s not formatted, run mizu verify --fix", plural(len(wrong), "file"))
}

func stageVet(ctx context.Context, p project, _ bool) (string, error) {
	return tool(ctx, p.Dir, "go", "vet", "./...")
}

// stageBuild builds every package.
//
// The output is thrown away by name, which the go command understands and
// treats as "compile it and keep nothing". Without that, a project whose ./...
// comes to exactly one command gets a binary written into whatever directory
// verify was run in, and a verify that leaves a binary behind is a verify
// somebody adds to .gitignore.
func stageBuild(ctx context.Context, p project, _ bool) (string, error) {
	return tool(ctx, p.Dir, "go", "build", "-o", os.DevNull, "./...")
}

func stageTest(ctx context.Context, p project, _ bool) (string, error) {
	return tool(ctx, p.Dir, "go", "test", "-short", "./...")
}

// stageDoctor runs the project checks.
//
// The generated check is left out, because verify runs it as its own stage and
// it is the slowest thing either of them does.
func stageDoctor(ctx context.Context, p project, _ bool) (string, error) {
	var found []finding
	for _, ch := range checks {
		if covered[ch.name] != "" {
			continue
		}
		out, err := ch.run(ctx, p)
		if err != nil {
			return "", fmt.Errorf("the %s check could not run: %w", ch.name, err)
		}
		found = append(found, out...)
	}

	// Only an error stops a verify. A warning is worth saying and not worth
	// failing over, which is the same rule mizu doctor --ci follows.
	var b strings.Builder
	bad := 0
	for _, f := range found {
		fmt.Fprintf(&b, "[%s] %s\n", f.Level, f.What)
		fmt.Fprintf(&b, "  Fix: %s\n", f.Fix)
		if f.Level == fail {
			bad++
		}
	}
	if bad > 0 {
		return b.String(), fmt.Errorf("%s, run mizu doctor", plural(bad, "error"))
	}
	return "", nil
}

// covered names the doctor checks that verify runs as a stage of its own, and
// the stage that covers each.
var covered = map[string]string{"generated": "gen"}

// tool runs a command and folds everything it printed into the output.
//
// Both streams, because go test says what failed on stdout and go vet says it
// on stderr, and a verify that shows one of them is a verify somebody runs the
// underlying command after.
func tool(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return "", nil
	}
	line := strings.Join(append([]string{name}, args...), " ")
	if len(bytes.TrimSpace(out)) == 0 {
		return "", fmt.Errorf("%s: %w", line, err)
	}
	return string(out), errors.New(line + " found problems")
}

const verifyLong = `A green run here means a green CI. That is the whole point of the command:
one answer, trustworthy enough that nothing else has to be run before saying a
change is finished.

The stages run in dependency order and the run stops at the first failure.
Code that does not compile has nothing useful to say about its tests, and a
stale generated file makes every stage after it a report about something
nobody wrote.

--fix writes what can be written, which is the generated files and the
formatting, and carries on rather than stopping to be run again.

--json gives a result for every stage, including the ones that did not run
after a failure, so a failure localises without reading any output.`

const checkLong = `The fastest question worth asking: is this even valid Go. It is the inner
loop, run after every edit, where mizu verify is what gets run before saying a
change is finished.

It is a strict subset of what mizu verify runs, taken from the same list, so
passing here and failing there is possible and passing there and failing here
is not.`
