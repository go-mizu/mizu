package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
)

// say returns the two streams a command line writes to. Buffers rather than
// the process's own, which is the whole reason [console.App.Start] exists next
// to [console.App.Main].
func say(tb testing.TB) (out, errOut *bytes.Buffer) {
	tb.Helper()
	return new(bytes.Buffer), new(bytes.Buffer)
}

// start runs a command line and returns what it printed and the code a process
// would have exited with.
func start(tb testing.TB, argv ...string) (out, errOut *bytes.Buffer, code int) {
	tb.Helper()
	out, errOut = say(tb)
	return out, errOut, newApp().Start(tb.Context(), nil, out, errOut, argv)
}

// Registration is where a command with no name or a name already taken is
// found, and it panics. Building the app in a test is what makes that a test
// failure rather than something the first person to run the binary discovers.
func TestTheAppBuilds(t *testing.T) {
	a := newApp()
	if a.Name != "mizu" {
		t.Errorf("the app is called %q", a.Name)
	}
	if a.Version == "" {
		t.Error("the app has no version, so --version says nothing")
	}
}

func TestNoArgumentsPrintsHelp(t *testing.T) {
	out, errOut, code := start(t)
	if code != console.CodeOK {
		t.Fatalf("exited %d", code)
	}
	for _, want := range []string{"mizu", "hash:tune", "version"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the help does not mention %q:\n%s", want, out)
		}
	}
	if errOut.Len() != 0 {
		t.Errorf("help went to stderr: %q", errOut)
	}
}

func TestHelpForms(t *testing.T) {
	for _, argv := range [][]string{{"help"}, {"-h"}, {"--help"}, {"help", "version"}} {
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			out, _, code := start(t, argv...)
			if code != console.CodeOK {
				t.Fatalf("exited %d", code)
			}
			if !strings.Contains(out.String(), "version") {
				t.Errorf("printed no help:\n%s", out)
			}
		})
	}
}

func TestUnknownCommand(t *testing.T) {
	_, errOut, code := start(t, "verison")
	if code != console.CodeUsage {
		t.Fatalf("exited %d, want %d", code, console.CodeUsage)
	}
	// The suggestion is the point. A typo one letter away from a real command
	// should not send somebody to the help.
	if !strings.Contains(errOut.String(), "version") {
		t.Errorf("nothing suggested the command they meant:\n%s", errOut)
	}
}

// The global flags are listed in the help a program prints, so somebody who has
// only ever run mizu can find out that --json exists.
func TestHelpListsTheGlobalFlags(t *testing.T) {
	out, _, _ := start(t, "--help")
	for _, want := range []string{"--verbose", "--json", "--profile", "--trace"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the help does not mention %q:\n%s", want, out)
		}
	}
}

func TestVersionFlag(t *testing.T) {
	out, _, code := start(t, "--version")
	if code != console.CodeOK {
		t.Fatalf("exited %d", code)
	}
	if !strings.HasPrefix(out.String(), "mizu ") {
		t.Errorf("--version printed %q", out)
	}
}

// Every command's help is read by somebody who has not run it yet, so the two
// lines it is described by are worth holding to a shape.
func TestEveryCommandDescribesItself(t *testing.T) {
	for _, argv := range [][]string{{"help", "hash:tune"}, {"help", "version"}} {
		out, _, code := start(t, argv...)
		if code != console.CodeOK {
			t.Errorf("%v exited %d", argv, code)
			continue
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("%v printed no usage line:\n%s", argv, out)
		}
	}
}
