package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/console/consoletest"
)

// here is the project a test says it is in.
//
// Discovering it for real would make the answers depend on which Go is
// installed on the machine running the test, and the toolchain check exists to
// notice exactly that difference. The directory is still the real one, because
// the checks that read files have to find them.
func here(dir string) func(context.Context) (project, error) {
	return func(context.Context) (project, error) {
		return project{
			Dir:         dir,
			Module:      "commandtest",
			Go:          "1.27",
			Toolchain:   "go1.27.0",
			GOTOOLCHAIN: "auto",
		}, nil
	}
}

// runDoctor runs one command line against the directory scratch left behind.
func runDoctor(t *testing.T, dir string, argv ...string) *consoletest.Result {
	t.Helper()
	return consoletest.Run(t, &Doctor{find: here(dir)}, consoletest.Args(argv...))
}

// real is a path with the symlinks taken out of it.
func real(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// pinEndings writes the .gitattributes the line endings check asks for, which
// takes that check out of the way of the one under test.
func pinEndings(tb testing.TB, dir string) {
	tb.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("* text=auto eol=lf\n"), 0o644); err != nil {
		tb.Fatal(err)
	}
}

// repo makes dir a git repository with the named files committed to the index.
// Nothing is committed, so there is no author to configure.
func repo(t *testing.T, dir string, files ...string) {
	t.Helper()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	git("init", "-q")
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("SECRET=1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git(append([]string{"add", "--"}, files...)...)
}

func TestDoctorOnAProjectWithNothingWrong(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)

	r := runDoctor(t, dir).AssertSuccess()
	r.AssertNoOutput()
	r.AssertErrorContains("4 checks, nothing to report")
}

func TestDoctorReportsAStaleGeneratedFile(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	touch(t, filepath.Join(dir, "app", "commands_gen.go"))

	r := runDoctor(t, dir).AssertSuccess()
	r.AssertOutputContains("[error] 1 generated file out of date: app/commands_gen.go")
	r.AssertOutputContains("Fix: mizu gen")
	r.AssertErrorContains("1 error.")
}

// The exit code is what a build step reads, and doctor only changes it when it
// was asked to.
func TestDoctorCIFailsOnAnError(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	touch(t, filepath.Join(dir, "app", "commands_gen.go"))

	err := runDoctor(t, dir, "--ci").AssertFailure()
	if got, want := err.Error(), "1 error"; got != want {
		t.Errorf("the error is %q, want %q", got, want)
	}
}

// A warning is worth saying and not worth failing over, so --ci makes no
// difference to a run that found one.
func TestDoctorCIPassesOnAWarning(t *testing.T) {
	dir := scratch(t, commands)

	r := runDoctor(t, dir, "--ci").AssertSuccess()
	r.AssertOutputContains("[warning] no .gitattributes pins line endings")
	r.AssertErrorContains("1 warning.")
}

func TestDoctorOnAGitattributesThatSaysNothingAboutLineEndings(t *testing.T) {
	dir := scratch(t, commands)
	if err := os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte("*.png binary\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := runDoctor(t, dir).AssertSuccess()
	r.AssertOutputContains("[warning] .gitattributes does not pin line endings")
	r.AssertOutputContains("Fix: put `* text=auto eol=lf` in .gitattributes at " + dir)
}

func TestDoctorReportsATrackedEnvFile(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	repo(t, dir, ".env")

	r := runDoctor(t, dir).AssertSuccess()
	r.AssertOutputContains("[error] 1 environment file tracked by git: .env")
	r.AssertOutputContains("Fix: git rm --cached .env")
}

// .env.example is the file a project is meant to commit, and .env.sample is
// what the same idea is called elsewhere.
func TestDoctorLeavesTheExampleEnvFileAlone(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	repo(t, dir, ".env.example", ".env.sample")

	runDoctor(t, dir).AssertSuccess().AssertNoOutput()
}

// Not every project is in git, and a project that is not has no secret checked
// into one.
func TestDoctorOnAProjectWithoutGit(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	runDoctor(t, dir).AssertSuccess().AssertNoOutput()
}

// Worst first, and the counts underneath name each level once.
func TestDoctorRanksFindings(t *testing.T) {
	dir := scratch(t, commands)
	touch(t, filepath.Join(dir, "app", "commands_gen.go"))
	repo(t, dir, ".env")

	r := runDoctor(t, dir).AssertSuccess()
	levels := []string{}
	for _, line := range strings.Split(r.Stdout(), "\n") {
		if strings.HasPrefix(line, "[") {
			levels = append(levels, line[1:strings.Index(line, "]")])
		}
	}
	want := []string{"error", "error", "warning"}
	if strings.Join(levels, ",") != strings.Join(want, ",") {
		t.Errorf("the findings came out %v, want %v", levels, want)
	}
	r.AssertErrorContains("2 errors, 1 warning.")
}

// A check that could not look is not a check that passed, so it says so rather
// than leaving a gap the reader reads as health.
func TestDoctorReportsACheckThatCouldNotRun(t *testing.T) {
	dir := scratch(t, configs)
	pinEndings(t, dir)
	second := filepath.Join(dir, "app", "second.go")
	if err := os.WriteFile(second, []byte("package app\n\n//mizu:config\ntype Second struct {\n\tPort int\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := runDoctor(t, dir).AssertSuccess()
	r.AssertOutputContains("[warning] this check could not run")
	r.AssertOutputContains("Second is marked as configuration and so is Config, and an application has one")
	r.AssertOutputContains("Fix: fix what stopped it and run mizu doctor again")
}

// A package that will not load is the generated check failing rather than the
// command failing, so the rest of the report still arrives.
func TestDoctorOnAProjectTheGoCommandWillNotRead(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("this is not a go.mod\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := runDoctor(t, dir).AssertSuccess()
	r.AssertOutputContains("[warning] this check could not run")
	r.AssertOutputContains("go.mod")
	r.AssertErrorContains("1 warning.")
}

// A generator will not overwrite a file it did not write, and doctor reports
// that as the check having failed rather than as the file being up to date.
func TestDoctorOnAHandWrittenFileWhereAGeneratedOneBelongs(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	name := filepath.Join(dir, "app", "commands_gen.go")
	if err := os.WriteFile(name, []byte("package app\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := runDoctor(t, dir).AssertSuccess()
	r.AssertOutputContains("[warning] this check could not run")
	r.AssertOutputContains("refusing to overwrite hand-written file")
}

// A .gitattributes that cannot be read is not a .gitattributes that says the
// right thing, and the two are told apart.
func TestDoctorOnAGitattributesItCannotRead(t *testing.T) {
	dir := scratch(t, commands)
	if err := os.Mkdir(filepath.Join(dir, ".gitattributes"), 0o755); err != nil {
		t.Fatal(err)
	}

	r := runDoctor(t, dir).AssertSuccess()
	r.AssertOutputContains("[warning] this check could not run")
}

func TestDoctorJSON(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	repo(t, dir, ".env")

	r := consoletest.Run(t, &Doctor{find: here(dir)}, consoletest.With(console.Options{JSON: true})).AssertSuccess()

	// The level goes out as the word for it rather than the number it is
	// stored as, since the number means nothing to whatever reads this.
	var found []struct {
		Check, Level, What, Why, Fix string
	}
	if err := json.Unmarshal([]byte(r.Stdout()), &found); err != nil {
		t.Fatalf("%v in %s", err, r.Stdout())
	}
	if len(found) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(found), found)
	}
	if found[0].Check != "dotenv" || found[0].Level != "error" {
		t.Errorf("the finding is %+v", found[0])
	}
	for _, part := range []string{found[0].What, found[0].Why, found[0].Fix} {
		if part == "" {
			t.Errorf("the finding left something out: %+v", found[0])
		}
	}
}

// A script reading this wants a list either way, and null is not one.
func TestDoctorJSONWithNothingToReport(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)

	r := consoletest.Run(t, &Doctor{find: here(dir)}, consoletest.With(console.Options{JSON: true})).AssertSuccess()
	if got := strings.TrimSpace(r.Stdout()); got != "[]" {
		t.Errorf("the output is %q, want []", got)
	}
	r.AssertNoErrorOutput()
}

func TestDoctorStopsOnAnInterrupt(t *testing.T) {
	dir := scratch(t, commands)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	r := consoletest.Run(t, &Doctor{find: here(dir)}, consoletest.Context(ctx))
	if err := r.AssertFailure(); err != context.Canceled {
		t.Errorf("the error is %v, want %v", err, context.Canceled)
	}
	r.AssertNoOutput()
}

func TestDoctorSaysWhereItIsNotAProject(t *testing.T) {
	t.Chdir(t.TempDir())

	r := consoletest.Run(t, &Doctor{})
	if err := r.AssertFailure(); !strings.Contains(err.Error(), "not in a module") {
		t.Errorf("the error is %q, want it to say this is not a module", err)
	}
}

// Without a go command there is no project to check and no point guessing at
// one, so the whole command stops rather than any single check failing.
func TestDoctorWithoutAGoCommand(t *testing.T) {
	scratch(t, commands)
	t.Setenv("PATH", "")

	err := consoletest.Run(t, &Doctor{}).AssertFailure()
	if !strings.HasPrefix(err.Error(), "go: ") {
		t.Errorf("the error is %q, want it to name the command that would not run", err)
	}
}

// The findings are the answer, so a run that cannot write them has not
// answered, whatever it found.
func TestDoctorWhenTheOutputCannotBeWritten(t *testing.T) {
	c := &Doctor{}
	io := console.New(nil, brokenWriter{}, brokenWriter{}, console.Options{JSON: true})
	if err := c.report(io, nil); err == nil {
		t.Error("writing the report to a broken stream came back clean")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errBroken }

var errBroken = errors.New("the stream went away")

// discover asks the go command rather than guessing, so this checks the
// answers come back in the shape the checks expect.
func TestDiscover(t *testing.T) {
	dir := scratch(t, commands)

	p, err := discover(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// A temporary directory is reached through a symlink on some platforms, and
	// which of the two paths comes back is not what this is testing.
	if got, want := real(t, p.Dir), real(t, dir); got != want {
		t.Errorf("the module is at %q, want %q", got, want)
	}
	if p.Module != "commandtest" {
		t.Errorf("the module is %q, want commandtest", p.Module)
	}
	if p.Go != "1.27" {
		t.Errorf("the go directive is %q, want 1.27", p.Go)
	}
	if !strings.HasPrefix(p.Toolchain, "go1.") {
		t.Errorf("the toolchain is %q, want a go version", p.Toolchain)
	}
	if p.GOTOOLCHAIN == "" {
		t.Error("GOTOOLCHAIN came back empty")
	}
}

func TestToolchainFindings(t *testing.T) {
	tests := []struct {
		name string
		p    project
		want string // the level, or "" for nothing to say
	}{
		{"the same version", project{Go: "1.27", Toolchain: "go1.27"}, ""},
		{"a patch release of it", project{Go: "1.27", Toolchain: "go1.27.3"}, ""},
		{"a later version", project{Go: "1.28", Toolchain: "go1.29.0"}, ""},
		{"an older one", project{Go: "1.27", Toolchain: "go1.26.7", GOTOOLCHAIN: "auto"}, "note"},
		{"an older one down to the patch", project{Go: "1.27.4", Toolchain: "go1.27.1", GOTOOLCHAIN: "auto"}, "note"},
		{"no go directive", project{Toolchain: "go1.27"}, ""},
		{"no toolchain", project{Go: "1.27"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := toolchainFindings(tt.p)
			if tt.want == "" {
				if len(found) != 0 {
					t.Fatalf("got %+v, want nothing", found)
				}
				return
			}
			if len(found) != 1 {
				t.Fatalf("got %d findings, want 1", len(found))
			}
			if got := found[0].Level.String(); got != tt.want {
				t.Errorf("the level is %q, want %q", got, tt.want)
			}
			if found[0].What == "" || found[0].Why == "" || found[0].Fix == "" {
				t.Errorf("a finding left something out: %+v", found[0])
			}
		})
	}
}

func TestTally(t *testing.T) {
	tests := []struct {
		found []finding
		want  string
	}{
		{nil, ""},
		{[]finding{{Level: fail}}, "1 error"},
		{[]finding{{Level: warn}, {Level: warn}}, "2 warnings"},
		{[]finding{{Level: note}, {Level: fail}, {Level: warn}}, "1 error, 1 warning, 1 note"},
		{[]finding{{Level: fail}, {Level: fail}, {Level: note}}, "2 errors, 1 note"},
	}
	for _, tt := range tests {
		if got := tally(tt.found); got != tt.want {
			t.Errorf("tally(%+v) = %q, want %q", tt.found, got, tt.want)
		}
	}
}

func TestLevelString(t *testing.T) {
	tests := map[level]string{note: "note", warn: "warning", fail: "error", level(9): "note"}
	for l, want := range tests {
		if got := l.String(); got != want {
			t.Errorf("level(%d) is %q, want %q", int(l), got, want)
		}
		text, err := l.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		if string(text) != want {
			t.Errorf("level(%d) marshals to %q, want %q", int(l), text, want)
		}
	}
}

// Every check has to be able to say what it looks at, because a clean report
// only means something next to the list of what was asked.
func TestDoctorSpec(t *testing.T) {
	spec := (&Doctor{}).Spec()
	if spec.Name != "doctor" {
		t.Errorf("the command is called %q, want doctor", spec.Name)
	}
	for _, ch := range checks {
		if ch.desc == "" {
			t.Errorf("the %s check has no description", ch.name)
		}
		if !strings.Contains(spec.Long, ch.name) || !strings.Contains(spec.Long, ch.desc) {
			t.Errorf("the help does not list the %s check", ch.name)
		}
	}
}

func TestDoctorIsInTheApp(t *testing.T) {
	out, _, code := start(t, "help", "doctor")
	if code != console.CodeOK {
		t.Fatalf("exited %d: %s", code, out)
	}
	for _, want := range []string{"Check the project and the environment", "--ci", "Usage:"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the help does not mention %q:\n%s", want, out)
		}
	}
}

// BenchmarkDoctor is what a person waits through when they run the command,
// which is four checks over a project with one package in it.
//
// Nearly all of it is the go command, the type checker and git rather than
// anything here, and that is what it is for: it says what the step costs. The
// parts this package owns are measured separately below.
func BenchmarkDoctor(b *testing.B) {
	dir := scratch(b, commands)
	pinEndings(b, dir)
	c := console.New(nil, io.Discard, io.Discard, console.Options{})
	cmd := &Doctor{find: here(dir)}

	b.ReportAllocs()
	for b.Loop() {
		if err := cmd.Run(b.Context(), c); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDoctorReport(b *testing.B) {
	c := console.New(nil, io.Discard, io.Discard, console.Options{})
	found := []finding{
		{Check: "generated", Level: fail, What: "1 generated file out of date", Why: "the checked-in files and the code disagree", Fix: "mizu gen"},
		{Check: "lineendings", Level: warn, What: "no .gitattributes pins line endings", Why: "a checkout on Windows rewrites Go source to CRLF", Fix: "put `* text=auto eol=lf` in .gitattributes"},
		{Check: "toolchain", Level: note, What: "the go command installed here is go1.26.7", Why: "every build switches toolchain first", Fix: "install go1.27"},
	}
	cmd := &Doctor{CI: true}

	b.ReportAllocs()
	for b.Loop() {
		if err := cmd.report(c, found); err == nil {
			b.Fatal("an error level finding was reported as a success")
		}
	}
}

func BenchmarkToolchainFindings(b *testing.B) {
	p := project{Module: "proj", Go: "1.27", Toolchain: "go1.26.7", GOTOOLCHAIN: "auto"}

	b.ReportAllocs()
	for b.Loop() {
		if len(toolchainFindings(p)) != 1 {
			b.Fatal("an old toolchain went unmentioned")
		}
	}
}
