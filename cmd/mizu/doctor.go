package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	// This package has a version type of its own, so go/version needs a name
	// that does not collide with it.
	goversion "go/version"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/gen"
)

// A level is how much a finding matters.
//
// Ranking is most of what makes the output usable. Sixty checks that all print
// the same way is a wall of text somebody scrolls past, and the one line that
// mattered is in the middle of it.
type level int

const (
	note level = iota // worth knowing about
	warn              // worth fixing before it bites
	fail              // wrong now
)

func (l level) String() string {
	switch l {
	case fail:
		return "error"
	case warn:
		return "warning"
	}
	return "note"
}

// MarshalText puts the word in the JSON rather than the number behind it,
// which is an ordering this program chose and nothing outside it should read.
func (l level) MarshalText() ([]byte, error) { return []byte(l.String()), nil }

// A finding is one thing doctor found.
//
// What, why and how to fix it are all required. A check that can say something
// is wrong but not what to do about it sends somebody to a search engine,
// which is the same as not having run the check.
type finding struct {
	Check string `json:"check"`
	Level level  `json:"level"`
	What  string `json:"what"`
	Why   string `json:"why"`
	Fix   string `json:"fix"`
}

// A check is one question doctor asks about a project.
//
// It returns what it found, or an error if it could not look. The difference
// matters: a check that could not run is not a check that passed, and doctor
// says so rather than counting silence as health.
type check struct {
	name string
	desc string
	run  func(ctx context.Context, p project) ([]finding, error)
}

var checks = []check{
	{"toolchain", "The Go version the module asks for against the one installed", checkToolchain},
	{"generated", "Generated files against what the generators would write now", checkGenerated},
	{"lineendings", "Line endings pinned for the checkout", checkLineEndings},
	{"dotenv", "A .env file tracked by git", checkDotenv},
}

// A project is what every check is given.
//
// It is worked out once, because sixty checks that each run the go command is
// sixty seconds nobody has.
type project struct {
	Dir         string // the module root
	Module      string // the module path
	Go          string // the go directive in go.mod, such as 1.27
	Toolchain   string // the version of the go command that would build it
	GOTOOLCHAIN string // what the go command is allowed to switch to
}

// Doctor checks a project and says what is wrong with it.
type Doctor struct {
	// find works out what project this is. It is a field so a test can say
	// what the environment looks like, since half of what doctor reports
	// otherwise depends on which Go happens to be installed on the machine
	// running the test.
	find func(context.Context) (project, error)

	CI bool
}

func (c *Doctor) Spec() console.Spec {
	return console.Spec{
		Name: "doctor",
		Desc: "Check the project and the environment",
		Long: doctorLong + "\n\nWhat it looks at:\n\n" + checkList(),
		Flags: []console.Flag{
			{Name: "ci", Desc: "Exit non-zero when anything is an error", Value: console.Bool(&c.CI)},
		},
	}
}

// checkList is the checks, for the help text.
//
// Somebody deciding whether a clean report means anything wants to know what
// was looked at, and the list is the only honest answer to that.
func checkList() string {
	var b strings.Builder
	width := 0
	for _, ch := range checks {
		width = max(width, len(ch.name))
	}
	for _, ch := range checks {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, ch.name, ch.desc)
	}
	return strings.TrimRight(b.String(), "\n")
}

func (c *Doctor) Run(ctx context.Context, io *console.IO) error {
	find := c.find
	if find == nil {
		find = discover
	}
	p, err := find(ctx)
	if err != nil {
		return err
	}

	var found []finding
	for _, ch := range checks {
		if err := ctx.Err(); err != nil {
			return err
		}
		out, err := ch.run(ctx, p)
		if err != nil {
			// A check that could not look is reported rather than dropped.
			// Silence here reads as health, and this is the one command whose
			// whole job is to not let that happen.
			out = []finding{{
				Check: ch.name,
				Level: warn,
				What:  "this check could not run",
				Why:   err.Error(),
				Fix:   "fix what stopped it and run mizu doctor again",
			}}
		}
		io.Debug("%s: %s", ch.name, plural(len(out), "finding"))
		found = append(found, out...)
	}

	// Worst first, and within a level the order the checks are declared in, so
	// two runs on the same project print the same thing in the same order.
	slices.SortStableFunc(found, func(a, b finding) int { return int(b.Level) - int(a.Level) })
	return c.report(io, found)
}

func (c *Doctor) report(io *console.IO, found []finding) error {
	if io.JSONMode() {
		// A clean run is an empty list rather than null, because a script
		// reading this wants something it can range over either way.
		list := found
		if list == nil {
			list = []finding{}
		}
		if err := io.JSON(list); err != nil {
			return err
		}
	} else {
		for i, f := range found {
			if i > 0 {
				io.Line("")
			}
			io.Print("[%s] %s\n", f.Level, f.What)
			io.Print("  %s\n", f.Why)
			io.Print("  Fix: %s\n", f.Fix)
		}
	}

	if len(found) == 0 {
		io.Success("%s, nothing to report.", plural(len(checks), "check"))
		return nil
	}

	// The findings are sorted, so the first one is the worst.
	//
	// Under --ci the count comes back as the error, which is what makes the
	// exit code non-zero and what puts "error:" in front of it. Without --ci
	// it is a count of what was printed above and nothing more, so it goes out
	// the same way any other summary line does.
	if found[0].Level == fail && c.CI {
		return errors.New(tally(found))
	}
	io.Info("%s.", tally(found))
	return nil
}

// tally is the counts that are not zero, worst first.
func tally(found []finding) string {
	n := map[level]int{}
	for _, f := range found {
		n[f.Level]++
	}
	var parts []string
	for _, l := range []level{fail, warn, note} {
		if n[l] > 0 {
			parts = append(parts, plural(n[l], l.String()))
		}
	}
	return strings.Join(parts, ", ")
}

// discover works out what project the command was run in.
//
// The go command is asked rather than walking up looking for a go.mod, because
// it already knows about GOWORK, GOFLAGS and the rest, and a second answer
// that disagrees with the one the build uses is worse than no answer.
func discover(ctx context.Context) (project, error) {
	var mod struct {
		Path      string
		Dir       string
		GoVersion string
	}
	out, err := run(ctx, "", nil, "go", "list", "-m", "-json")
	if err != nil {
		return project{}, err
	}
	if err := json.Unmarshal([]byte(out), &mod); err != nil {
		return project{}, fmt.Errorf("reading go list -m: %w", err)
	}
	if mod.Dir == "" {
		return project{}, errors.New("this directory is not in a module, and mizu doctor checks a project")
	}
	p := project{Dir: mod.Dir, Module: mod.Path, Go: mod.GoVersion}

	// GOVERSION under the settings in force is whatever the go command
	// switched to on the way in, which answers a different question. Asking
	// again with GOTOOLCHAIN=local is what gives the version installed here,
	// and that call cannot fail over a version the module wants.
	local, err := run(ctx, "", []string{"GOTOOLCHAIN=local"}, "go", "env", "GOVERSION")
	if err != nil {
		return project{}, err
	}
	p.Toolchain = strings.TrimSpace(local)

	// Read separately, because the call above was told what to say.
	setting, err := run(ctx, "", nil, "go", "env", "GOTOOLCHAIN")
	if err != nil {
		return project{}, err
	}
	p.GOTOOLCHAIN = strings.TrimSpace(setting)
	return p, nil
}

// run is one command, with its output, and with what it wrote to stderr as the
// error when it fails. A tool that reports "exit status 1" and throws away the
// sentence underneath it is a tool somebody has to run again by hand.
func run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("%s: %s", name, msg)
		}
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return string(out), nil
}

func checkToolchain(_ context.Context, p project) ([]finding, error) {
	return toolchainFindings(p), nil
}

// toolchainFindings is the comparison on its own, with no go command in it.
//
// The version the module asks for and the version installed are strings that
// came from somewhere else, and the interesting part is what is done with
// them, so that is the part worth testing directly.
//
// There is no error level here on purpose. A toolchain too old to build the
// module and a GOTOOLCHAIN that will not replace it stops the go command
// before doctor has read anything, and the message it stops with already names
// both versions and the setting. Repeating that guess is worth less than what
// the go command said.
func toolchainFindings(p project) []finding {
	want, have := "go"+p.Go, p.Toolchain
	if p.Go == "" || have == "" || goversion.Compare(have, want) >= 0 {
		return nil
	}
	return []finding{{
		Check: "toolchain",
		Level: note,
		What:  fmt.Sprintf("%s asks for %s and the go command installed here is %s", p.Module, want, have),
		Why:   fmt.Sprintf("GOTOOLCHAIN is %s, so every build switches toolchain before it starts, which is a download the first time and a network call in a fresh CI job", p.GOTOOLCHAIN),
		Fix:   "install " + want + " to build without the switch, or leave it and pay for the switch once",
	}}
}

func checkGenerated(_ context.Context, p project) ([]finding, error) {
	// Nothing below writes anything, since the writer is in check mode, so
	// there is no half-finished state for a cancellation to leave behind and
	// the loop in Run is where an interrupted run stops.
	pkgs, err := load(p.Dir, []string{"./..."})
	if err != nil {
		return nil, err
	}

	var files []gen.File
	for _, g := range generators {
		out, err := g.run(pkgs...)
		if err != nil {
			return nil, err
		}
		files = append(files, out...)
	}

	w := &gen.Writer{Dir: p.Dir, Check: true}
	results, err := w.Write(files...)
	if err != nil {
		return nil, err
	}

	var stale []string
	for _, r := range results {
		if r.Changed() {
			stale = append(stale, r.Path)
		}
	}
	if len(stale) == 0 {
		return nil, nil
	}
	return []finding{{
		Check: "generated",
		Level: fail,
		What:  fmt.Sprintf("%s out of date: %s", plural(len(stale), "generated file"), strings.Join(stale, ", ")),
		Why:   "the checked-in files and the code they were written from disagree, so a fresh clone builds something nobody reviewed",
		Fix:   "mizu gen",
	}}, nil
}

func checkLineEndings(_ context.Context, p project) ([]finding, error) {
	data, err := os.ReadFile(filepath.Join(p.Dir, ".gitattributes"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if strings.Contains(string(data), "eol=lf") {
		return nil, nil
	}

	what := "no .gitattributes pins line endings"
	if len(data) > 0 {
		what = ".gitattributes does not pin line endings"
	}
	return []finding{{
		Check: "lineendings",
		Level: warn,
		What:  what,
		Why:   "a checkout on Windows with core.autocrlf set rewrites Go source to CRLF, and then gofmt lists every file in the project and mizu gen rewrites every generated file on every run",
		Fix:   "put `* text=auto eol=lf` in .gitattributes at " + p.Dir,
	}}, nil
}

func checkDotenv(ctx context.Context, p project) ([]finding, error) {
	out, err := run(ctx, p.Dir, nil, "git", "ls-files", "-z", "--", ".env", ".env.*")
	if err != nil {
		// No git, no repository, or no git repository here. None of those are
		// something to tell somebody about, and none of them mean a secret is
		// checked in.
		return nil, nil
	}

	var tracked []string
	for _, name := range strings.Split(out, "\x00") {
		// .env.example is the file that is meant to be committed, and it is
		// the one every project has.
		if name != "" && !strings.HasSuffix(name, ".example") && !strings.HasSuffix(name, ".sample") {
			tracked = append(tracked, name)
		}
	}
	if len(tracked) == 0 {
		return nil, nil
	}
	return []finding{{
		Check: "dotenv",
		Level: fail,
		What:  fmt.Sprintf("%s tracked by git: %s", plural(len(tracked), "environment file"), strings.Join(tracked, ", ")),
		Why:   "a .env holds the credentials for the environment it belongs to, and history keeps them after the file is deleted",
		Fix:   "git rm --cached " + strings.Join(tracked, " ") + ", add it to .gitignore, and rotate whatever was in it",
	}}, nil
}

const doctorLong = `Every check says what is wrong, why it matters and the command or the edit
that fixes it. A check that cannot say the third one sends somebody to a search
engine, which is the same as not having run it.

Findings are printed worst first. An error is wrong now, a warning is wrong
before long, and a note is worth knowing.

The command exits 0 whatever it finds, because a report is not a failure. Pass
--ci to make an error exit non-zero, which is what a build step wants.

A check that could not run is reported as a warning rather than passed over.
Silence from a check reads as health, and this is the one command whose job is
to not let that happen.`
