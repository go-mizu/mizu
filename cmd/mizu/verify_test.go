package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/console/consoletest"
)

// runVerify runs one command line against the directory scratch left behind.
func runVerify(t *testing.T, dir string, argv ...string) *consoletest.Result {
	t.Helper()
	return consoletest.Run(t, &Verify{find: here(dir)}, consoletest.Args(argv...))
}

// verifyJSON runs verify and reads the result back.
func verifyJSON(t *testing.T, dir string, argv ...string) verification {
	t.Helper()
	r := consoletest.Run(t, &Verify{find: here(dir)},
		consoletest.Args(argv...),
		consoletest.With(console.Options{JSON: true}),
	)

	var v verification
	if err := json.Unmarshal([]byte(r.Stdout()), &v); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, r.Stdout())
	}
	return v
}

// statuses is what each stage did, in order, as "name=status".
func statuses(v verification) []string {
	out := make([]string, 0, len(v.Stages))
	for _, r := range v.Stages {
		out = append(out, r.Stage+"="+r.Status)
	}
	return out
}

func TestVerifyOnAProjectWithNothingWrong(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)

	r := runVerify(t, dir).AssertSuccess()
	for _, s := range stages {
		r.AssertOutputContains(s.name)
	}
	r.AssertErrorContains("6 stages passed")
}

// Everything after a failure is a report about code already known to be wrong,
// and the point of the command is the seconds it saves.
func TestVerifyStopsAtTheFirstFailure(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	add(t, filepath.Join(dir, "app", "sloppy.go"), "package app\n\nfunc  Sloppy( ) {}\n")

	v := verifyJSON(t, dir)
	if v.OK {
		t.Fatal("a file that is not formatted passed verify")
	}
	want := []string{"gen=ok", "fmt=failed", "vet=skipped", "build=skipped", "test=skipped", "doctor=skipped"}
	if got := statuses(v); !slices.Equal(got, want) {
		t.Errorf("stages = %v\nwant %v", got, want)
	}
	if got := v.Stages[1].Output; got != "app/sloppy.go" {
		t.Errorf("the failing stage named %q, want the file that is not formatted", got)
	}
}

func TestVerifyFixWritesAndCarriesOn(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	name := filepath.Join(dir, "app", "sloppy.go")
	add(t, name, "package app\n\nfunc  Sloppy( ) {}\n")
	touch(t, filepath.Join(dir, "app", "commands_gen.go"))

	v := verifyJSON(t, dir, "--fix")
	if !v.OK {
		t.Fatalf("verify --fix failed: %+v", v.Stages)
	}
	want := []string{"gen=fixed", "fmt=fixed", "vet=ok", "build=ok", "test=ok", "doctor=ok"}
	if got := statuses(v); !slices.Equal(got, want) {
		t.Errorf("stages = %v\nwant %v", got, want)
	}

	if got := read(t, name); got != "package app\n\nfunc Sloppy() {}\n" {
		t.Errorf("the file was not formatted:\n%s", got)
	}
}

// A run with nothing to fix is not a run that says it fixed something.
func TestVerifyFixOnAProjectWithNothingWrong(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)

	v := verifyJSON(t, dir, "--fix")
	for _, r := range v.Stages {
		if r.Status != "ok" {
			t.Errorf("%s = %s, want ok", r.Stage, r.Status)
		}
	}
}

func TestCheckRunsTheQuickStagesOnly(t *testing.T) {
	dir := scratch(t, commands)

	r := consoletest.Run(t, &Verify{find: here(dir), quick: true}, consoletest.Args()).AssertSuccess()
	r.AssertOutputContains("vet")
	if strings.Contains(r.Stdout(), "test") {
		t.Errorf("check ran the tests:\n%s", r.Stdout())
	}
	r.AssertErrorContains("1 stage passed")
}

// Passing check and failing verify is the point of having both. Passing verify
// and failing check would mean check is asking something verify does not.
func TestCheckIsASubsetOfVerify(t *testing.T) {
	quick := (&Verify{quick: true}).stages()
	if len(quick) == 0 {
		t.Fatal("check runs nothing")
	}
	for _, s := range quick {
		if !slices.ContainsFunc(stages, func(v stage) bool { return v.name == s.name }) {
			t.Errorf("check runs %s, which verify does not", s.name)
		}
	}
}

// A stage that cannot run is a failure like any other, and the message says
// which one it was.
func TestVerifyOnSomethingThatIsNotAProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "gone")

	err := runVerify(t, dir).AssertFailure()
	if !strings.HasPrefix(err.Error(), "gen:") {
		t.Errorf("the error is %q, want it to name the stage", err)
	}
}

// Every test above says what the project is. This one is the command as
// somebody runs it, with the project worked out from where it was run.
func TestVerifyFindsTheProjectItIsRunIn(t *testing.T) {
	scratch(t, commands)

	consoletest.Run(t, &Verify{quick: true}, consoletest.Args()).AssertSuccess()
}

func TestVerifyOutsideAProject(t *testing.T) {
	t.Chdir(t.TempDir())

	err := consoletest.Run(t, &Verify{quick: true}, consoletest.Args()).AssertFailure()
	if !strings.Contains(err.Error(), "not in a module") {
		t.Errorf("the error is %q, want it to say this is not a module", err)
	}
}

// Between two stages is where an interrupted run stops, rather than after the
// stage that was going to take another thirty seconds.
func TestVerifyStopsWhenTheRunIsCancelled(t *testing.T) {
	dir := scratch(t, commands)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	c := &Verify{find: here(dir)}
	err := c.Run(ctx, console.New(nil, os.Stderr, os.Stderr, console.Options{}))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("the run came back with %v, want it cancelled", err)
	}
}

// Loading is the slow part, and it is where an interrupted mizu gen or verify
// --fix would otherwise be caught halfway through writing the tree.
func TestGenerateStopsBeforeWritingAnything(t *testing.T) {
	dir := scratch(t, commands)
	name := filepath.Join(dir, "app", "commands_gen.go")
	before := read(t, name)
	touch(t, name)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := generate(ctx, project{Dir: dir}, true); !errors.Is(err, context.Canceled) {
		t.Errorf("generate() = %v, want it cancelled", err)
	}
	if got := read(t, name); got == before {
		t.Error("the generated file was written by a run that was cancelled")
	}
}

// The result is the answer, so a run that cannot write it has not answered,
// whatever it found.
func TestVerifyWhenTheOutputCannotBeWritten(t *testing.T) {
	c := &Verify{}
	io := console.New(nil, brokenWriter{}, brokenWriter{}, console.Options{JSON: true})
	if err := c.report(io, verification{OK: true}); err == nil {
		t.Error("writing the result to a broken stream came back clean")
	}
}

func TestVerifyOnAStaleGeneratedFile(t *testing.T) {
	dir := scratch(t, commands)
	pinEndings(t, dir)
	touch(t, filepath.Join(dir, "app", "commands_gen.go"))

	err := runVerify(t, dir).AssertFailure()
	if want := "gen: 1 generated file out of date, run mizu gen"; err.Error() != want {
		t.Errorf("the error is %q, want %q", err, want)
	}
}

func TestStageFmt(t *testing.T) {
	dir := t.TempDir()
	add(t, filepath.Join(dir, "fine.go"), "package p\n")
	// A file that does not parse is not a formatting problem, and vet is the
	// stage that says what is wrong with it.
	add(t, filepath.Join(dir, "broken.go"), "package p\n\nfunc (\n")
	// Neither is a file somebody else wrote, or a fixture that is meant to
	// look the way it looks.
	add(t, filepath.Join(dir, "vendor", "x", "vendored.go"), "package  x\n")
	add(t, filepath.Join(dir, "testdata", "golden.go"), "package  g\n")
	// Nor is a file that is not Go at all.
	add(t, filepath.Join(dir, "notes.md"), "#  Notes\n")

	p := project{Dir: dir}
	if out, err := stageFmt(t.Context(), p, false); err != nil {
		t.Fatalf("stageFmt() = %q, %v, want no complaint", out, err)
	}
}

// A file the stage cannot read is not a file it can say anything about, so it
// says that rather than reporting the project as formatted.
func TestStageFmtOnAFileItCannotRead(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "secret.go")
	add(t, name, "package  p\n")
	shut(t, name, 0o000)

	if _, err := stageFmt(t.Context(), project{Dir: dir}, false); err == nil {
		t.Error("a file that cannot be read came back formatted")
	}
}

func TestStageFmtOnAFileItCannotWrite(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "readonly.go")
	add(t, name, "package  p\n")
	shut(t, name, 0o400)

	if _, err := stageFmt(t.Context(), project{Dir: dir}, true); err == nil {
		t.Error("a file that cannot be written came back formatted")
	}
}

// shut takes the permissions off a file or a directory for the rest of the
// test, and puts them back so the temporary directory can be removed.
func shut(t *testing.T, name string, mode fs.FileMode) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root reads and writes whatever it likes, so there is nothing to test here")
	}
	if runtime.GOOS == "windows" && mode&0o400 == 0 {
		t.Skip("chmod on Windows carries the write bit and nothing else, so a file cannot be made unreadable this way")
	}
	if err := os.Chmod(name, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(name, 0o755) })
}

// The generated check is the one stage verify runs itself, and running it twice
// is most of a verify.
func TestStageDoctorLeavesTheChecksVerifyAlreadyRan(t *testing.T) {
	for name, by := range covered {
		if !slices.ContainsFunc(checks, func(c check) bool { return c.name == name }) {
			t.Errorf("%s is not a doctor check", name)
		}
		if !slices.ContainsFunc(stages, func(s stage) bool { return s.name == by }) {
			t.Errorf("the %s check is said to be covered by %s, which is not a stage", name, by)
		}
	}
}

func TestStageDoctorOnAProjectWithAnError(t *testing.T) {
	dir := t.TempDir()
	pinEndings(t, dir)
	add(t, filepath.Join(dir, ".env"), "SECRET=hunter2\n")
	repo(t, dir, ".env")

	out, err := stageDoctor(t.Context(), project{Dir: dir}, false)
	if err == nil {
		t.Fatal("a project with a secret checked in passed the doctor stage")
	}
	if want := "1 error, run mizu doctor"; err.Error() != want {
		t.Errorf("the error is %q, want %q", err, want)
	}
	if !strings.Contains(out, "[error] 1 environment file tracked by git") {
		t.Errorf("the output is %q, want it to name what was found", out)
	}
}

// A check that could not look is not a check that passed, and this stage is
// what CI runs, so it stops rather than reporting the project as healthy.
func TestStageDoctorWhenACheckCannotRun(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".gitattributes"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := stageDoctor(t.Context(), project{Dir: dir}, false)
	if err == nil || !strings.Contains(err.Error(), "the lineendings check could not run") {
		t.Errorf("stageDoctor() = %v, want it to name the check that could not run", err)
	}
}

// What a tool printed is the answer, and a stage that throws it away is a
// stage somebody runs the underlying command after.
func TestToolReportsWhatTheCommandPrinted(t *testing.T) {
	out, err := tool(t.Context(), t.TempDir(), "go", "vet", "-mizu-not-a-flag")
	if err == nil {
		t.Fatal("go vet with a flag it does not have came back clean")
	}
	if want := "go vet -mizu-not-a-flag found problems"; err.Error() != want {
		t.Errorf("the error is %q, want %q", err, want)
	}
	if strings.TrimSpace(out) == "" {
		t.Error("the output is empty, want what the command printed")
	}
}

// A command that never started printed nothing, so the reason it did not start
// is the only thing there is to say.
func TestToolOnACommandThatCannotRun(t *testing.T) {
	out, err := tool(t.Context(), t.TempDir(), "mizu-no-such-command")
	if err == nil {
		t.Fatal("running a command that does not exist came back clean")
	}
	if !strings.HasPrefix(err.Error(), "mizu-no-such-command: ") {
		t.Errorf("the error is %q, want it to name the command", err)
	}
	if out != "" {
		t.Errorf("the output is %q, want nothing", out)
	}
}

func TestIndent(t *testing.T) {
	got := indent("--- FAIL: TestX\n    x_test.go:5: nope\n\nFAIL\n")
	want := "    --- FAIL: TestX\n        x_test.go:5: nope\n\n    FAIL"
	if got != want {
		t.Errorf("indent() = %q\nwant %q", got, want)
	}
}

func TestProjectFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"app/a.go":             {},
		"app/testdata/skip.go": {},
		"README.md":            {},
		".git/config":          {},
		"node_modules/p/i.js":  {},
		"vendor/v/x.go":        {},
		"web/link.go":          {Mode: fs.ModeSymlink},
	}

	want := []string{"README.md", "app/a.go"}
	if got := slices.Collect(projectFiles(fsys)); !slices.Equal(got, want) {
		t.Errorf("projectFiles() = %v, want %v", got, want)
	}
}

// The walk stops when the caller stops, rather than reading the whole tree and
// throwing most of it away.
func TestProjectFilesStopsEarly(t *testing.T) {
	fsys := fstest.MapFS{"a.go": {}, "b.go": {}, "c.go": {}}

	var seen []string
	for name := range projectFiles(fsys) {
		seen = append(seen, name)
		break
	}
	if want := []string{"a.go"}; !slices.Equal(seen, want) {
		t.Errorf("the walk yielded %v, want %v", seen, want)
	}
}

func TestVerifyIsWiredUp(t *testing.T) {
	for _, argv := range [][]string{{"verify", "extra"}, {"check", "extra"}} {
		out, errOut := say(t)
		if code := newApp().Start(t.Context(), nil, out, errOut, argv); code != console.CodeUsage {
			t.Errorf("%v exited %d, want %d", argv, code, console.CodeUsage)
		}
	}
}

// add writes a file a test needs to be there, with its directory.
func add(t *testing.T, name, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
