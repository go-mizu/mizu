package main

import (
	"os"
	"path/filepath"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
)

// A profile of the version command is a profile of almost nothing, which is
// what makes it quick. What is being tested is that the file was written and
// closed, not what is in it.
func TestProfileAndTrace(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "cpu.pprof")
	traced := filepath.Join(dir, "trace.out")

	out, errOut, code := start(t, "--profile="+profile, "--trace="+traced, "version")
	if code != console.CodeOK {
		t.Fatalf("exited %d: %s", code, errOut)
	}
	if !strings.HasPrefix(out.String(), "mizu ") {
		t.Errorf("the command did not run: %q", out)
	}

	for _, f := range []struct{ flag, name, tool string }{
		{"--profile", profile, "go tool pprof"},
		{"--trace", traced, "go tool trace"},
	} {
		info, err := os.Stat(f.name)
		if err != nil {
			t.Errorf("%s wrote no file: %v", f.flag, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("%s wrote an empty file", f.flag)
		}
		// Where the file went and what reads it, or somebody has a file they
		// cannot open and no reason to think it worked.
		if !strings.Contains(errOut.String(), f.tool+" "+f.name) {
			t.Errorf("nothing said to read %s with %s:\n%s", f.name, f.tool, errOut)
		}
	}
}

func TestProfileAndTraceAreQuietWhenAsked(t *testing.T) {
	dir := t.TempDir()
	_, errOut, code := start(t, "--quiet", "--profile="+filepath.Join(dir, "cpu.pprof"), "version")
	if code != console.CodeOK {
		t.Fatalf("exited %d: %s", code, errOut)
	}
	if errOut.Len() != 0 {
		t.Errorf("--quiet still said %q", errOut)
	}
}

func TestProfileThatCannotBeWritten(t *testing.T) {
	nowhere := filepath.Join(t.TempDir(), "no-such-directory", "cpu.pprof")

	for _, flag := range []string{"--profile", "--trace"} {
		t.Run(flag, func(t *testing.T) {
			out, errOut, code := start(t, flag+"="+nowhere, "version")
			if code != console.CodeFailure {
				t.Fatalf("exited %d, want %d", code, console.CodeFailure)
			}
			if !strings.Contains(errOut.String(), flag+":") {
				t.Errorf("the error does not name the flag:\n%s", errOut)
			}
			if out.Len() != 0 {
				t.Errorf("the command ran anyway: %q", out)
			}
		})
	}
}

// A profile that started has to be stopped when the trace after it fails, or
// the process is left profiling into a file nothing ever closes.
func TestATraceThatFailsStopsTheProfile(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "cpu.pprof")
	nowhere := filepath.Join(dir, "no-such-directory", "trace.out")

	_, errOut, code := start(t, "--profile="+profile, "--trace="+nowhere, "version")
	if code != console.CodeFailure {
		t.Fatalf("exited %d, want %d", code, console.CodeFailure)
	}
	if !strings.Contains(errOut.String(), "--trace:") {
		t.Errorf("the error does not name the flag:\n%s", errOut)
	}

	// Stopped means a second profile can start. Leaving one running would fail
	// every later test in the package instead of this one.
	if err := pprof.StartCPUProfile(new(strings.Builder)); err != nil {
		t.Fatalf("the first profile is still running: %v", err)
	}
	pprof.StopCPUProfile()

	if info, err := os.Stat(profile); err != nil || info.Size() == 0 {
		t.Errorf("the profile that did start was not written: %v", err)
	}
}

// The profiler refuses a second one, and what it says is worth passing on
// rather than a run that silently did not profile.
func TestOnlyOneProfileAtATime(t *testing.T) {
	dir := t.TempDir()

	if err := pprof.StartCPUProfile(new(strings.Builder)); err != nil {
		t.Fatal(err)
	}
	defer pprof.StopCPUProfile()

	_, errOut, code := start(t, "--profile="+filepath.Join(dir, "cpu.pprof"), "version")
	if code != console.CodeFailure {
		t.Fatalf("exited %d, want %d", code, console.CodeFailure)
	}
	if !strings.Contains(errOut.String(), "--profile:") {
		t.Errorf("the error does not name the flag:\n%s", errOut)
	}
}

func TestOnlyOneTraceAtATime(t *testing.T) {
	dir := t.TempDir()

	if err := trace.Start(new(strings.Builder)); err != nil {
		t.Fatal(err)
	}
	defer trace.Stop()

	_, errOut, code := start(t, "--trace="+filepath.Join(dir, "trace.out"), "version")
	if code != console.CodeFailure {
		t.Fatalf("exited %d, want %d", code, console.CodeFailure)
	}
	if !strings.Contains(errOut.String(), "--trace:") {
		t.Errorf("the error does not name the flag:\n%s", errOut)
	}
}

// Neither flag is the usual case, and it should cost nothing and say nothing.
func TestNeitherFlag(t *testing.T) {
	var g globals
	ctx, done, err := g.before(t.Context(), console.New(nil, new(strings.Builder), new(strings.Builder), console.Options{}))
	if err != nil {
		t.Fatal(err)
	}
	if ctx != nil {
		t.Errorf("before replaced the context with %v", ctx)
	}
	if done != nil {
		t.Error("before left something to clean up")
	}
}

// A file that cannot be closed is a warning. The command has already finished,
// and taking its exit code away now would say the work did not happen when it
// did.
func TestAFileThatWillNotClose(t *testing.T) {
	f, err := os.Create(filepath.Join(t.TempDir(), "cpu.pprof"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	var errOut strings.Builder
	c := console.New(nil, new(strings.Builder), &errOut, console.Options{})
	write(c, f, "CPU profile", "go tool pprof")

	if !strings.Contains(errOut.String(), "CPU profile") {
		t.Errorf("nothing warned about the close: %q", errOut.String())
	}
	if strings.Contains(errOut.String(), "read it with") {
		t.Errorf("it said the file was written: %q", errOut.String())
	}
}
