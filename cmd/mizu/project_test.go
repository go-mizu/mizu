package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
)

// The tests here are M0's first two acceptance criteria, which are about the
// project mizu new writes rather than about mizu new itself. Everything in
// new_test.go checks what was written. These check that what was written is a
// project: it builds, the binary answers, and the generators are clean in it.
//
// Linux and macOS, amd64 and arm64 are what the first criterion asks for, and
// they come from CI running go test ./... on ubuntu-latest, ubuntu-24.04-arm
// and macos-latest. There is nothing here that has to know which one it is on.

// binary is the executable go build leaves in a project directory.
func binary(dir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, name)
}

// runBinary runs a built project and returns what it wrote to both streams.
func runBinary(t *testing.T, dir, name string, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), binary(dir, name), args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}

// The criterion is one command line:
//
//	mizu new hello && cd hello && go build ./... && ./hello version
//
// The go mod tidy that resolve stands in for is the one thing missing from it,
// and it is not optional for any module with a dependency: there is no go.sum
// until something writes one, and go build does not. What resolve adds on top
// of that is a replace, because github.com/go-mizu/mizu has no tag carrying
// these packages yet and a tidy in a fresh project has nothing to fetch.
func TestTheProjectBuildsAndSaysWhatVersionItIs(t *testing.T) {
	if testing.Short() {
		t.Skip("the go command builds mizu once for each preset")
	}
	for _, p := range presets {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()

			dir := place(t, "hello")
			runNew(t, []string{dir, "--preset=" + p.name}).AssertSuccess()
			resolve(t, dir)

			gocmd(t, dir, "build", "./...")

			// The version is what a bug report is worth nothing without, and
			// what it says depends on how the binary was built, so the name is
			// the part worth asserting on.
			if out := runBinary(t, dir, "hello", "version"); !strings.HasPrefix(out, "hello ") {
				t.Errorf("hello version printed %q", out)
			}
			if out := runBinary(t, dir, "hello", "--version"); !strings.HasPrefix(out, "hello ") {
				t.Errorf("hello --version printed %q", out)
			}
		})
	}
}

// The second criterion is that mizu gen --check is clean on a fresh project,
// and that it fails with a file and a line when a source file is edited
// without regenerating.
//
// A fresh project has one generated file, the AGENTS.md that mizu gen agents
// writes from go.mod and the package list. Adding a package is what makes it
// stale, which is also the way somebody meets this for real: the file went out
// of date because the project changed, not because anybody touched the file.
func TestGenIsCleanOnAFreshProjectAndSaysWhereItIsNot(t *testing.T) {
	if testing.Short() {
		t.Skip("the go command loads the packages of the generated project")
	}
	dir := place(t, "hello")
	runNew(t, []string{dir, "--preset=cli"}).AssertSuccess()
	resolve(t, dir)
	t.Chdir(dir)

	out, errOut, code := start(t, "gen", "--check")
	if code != console.CodeOK {
		t.Fatalf("gen --check on a fresh project exited %d\n%s%s", code, out, errOut)
	}
	if !strings.Contains(out.String(), "up to date") {
		t.Errorf("gen --check said nothing about being up to date:\n%s", out)
	}

	add(t, filepath.Join(dir, "store", "store.go"), "package store\n\n// Get is a placeholder.\nfunc Get() string { return \"\" }\n")

	out, errOut, code = start(t, "gen", "--check")
	if code == console.CodeOK {
		t.Fatalf("gen --check passed with a package AGENTS.md does not know about\n%s", out)
	}
	// The file is what to open and the line is where to look, and a check that
	// reports neither leaves somebody diffing the whole thing.
	for _, want := range []string{"AGENTS.md", "stale", "line "} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("gen --check does not say %q:\n%s%s", want, out, errOut)
		}
	}
	if !strings.Contains(errOut.String(), "run mizu gen") {
		t.Errorf("gen --check does not say what to run:\n%s", errOut)
	}
}
