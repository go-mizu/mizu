package golden

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// recorder stands in for *testing.T so this package can check what it says when
// an assertion fails, which is most of what it is for.
//
// testing.TB cannot be implemented from outside the testing package, so it is
// embedded and the methods that matter are shadowed. The embedded one is nil,
// so a method this does not shadow panics rather than reporting a pass, which
// is the right way round for a fake in a test.
type recorder struct {
	testing.TB

	name   string
	failed bool
	msg    string
	logs   []string
}

func (r *recorder) Helper()                   {}
func (r *recorder) Name() string              { return r.name }
func (r *recorder) Logf(f string, a ...any)   { r.logs = append(r.logs, fmt.Sprintf(f, a...)) }
func (r *recorder) Errorf(f string, a ...any) { r.fail(fmt.Sprintf(f, a...)) }
func (r *recorder) Fatalf(f string, a ...any) { r.fail(fmt.Sprintf(f, a...)) }
func (r *recorder) Fatal(a ...any)            { r.fail(fmt.Sprint(a...)) }
func (r *recorder) Error(a ...any)            { r.fail(fmt.Sprint(a...)) }

// fail keeps the first message rather than the last. A real Fatalf stops the
// goroutine and this cannot, so an assertion that gave up part way through
// carries on here and fails a second time on the consequences of the first.
// The first one is the one that says what went wrong.
func (r *recorder) fail(msg string) {
	if !r.failed {
		r.msg = msg
	}
	r.failed = true
}

// says checks that the assertion failed and that its message mentions each of
// want, which is how every test here that is about wording is written.
func (r *recorder) says(t *testing.T, want ...string) {
	t.Helper()

	if !r.failed {
		t.Fatalf("the assertion passed, want it to fail saying %q", want)
	}
	for _, w := range want {
		if !strings.Contains(r.msg, w) {
			t.Errorf("the failure does not mention %q. it said:\n%s", w, r.msg)
		}
	}
}

// newRecorder gives a recorder writing into a directory of its own, so tests
// here never touch a checked-in golden file.
func newRecorder(t *testing.T, name string) (*recorder, Option) {
	t.Helper()
	return &recorder{name: name}, Dir(t.TempDir())
}

// updating runs fn with -update on and puts the flag back before returning, so
// what follows is comparing rather than writing again. No test here may be
// parallel, since the flag is one variable for the whole binary.
func updating(t *testing.T, fn func()) {
	t.Helper()

	before := *update
	*update = true
	defer func() { *update = before }()

	fn()
}

func TestAssertWritesThenMatches(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")

	updating(t, func() { Assert(r, []byte("hello\n"), dir) })
	if r.failed {
		t.Fatalf("writing failed: %s", r.msg)
	}
	if len(r.logs) != 1 || !strings.Contains(r.logs[0], "TestThing.golden") {
		t.Errorf("writing logged %v, want the path it wrote", r.logs)
	}

	Assert(r, []byte("hello\n"), dir)
	if r.failed {
		t.Errorf("comparing against what was just written failed: %s", r.msg)
	}
}

func TestAssertReportsADifference(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")

	updating(t, func() { Assert(r, []byte("one\ntwo\nthree\n"), dir) })
	Assert(r, []byte("one\nTWO\nthree\n"), dir)

	r.says(t, "TestThing", "- two", "+ TWO", "-update")
}

// TestAssertNamesTheMissingFile is the first thing anybody sees when they write
// a golden test, so the message has to say what to run.
func TestAssertNamesTheMissingFile(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")

	Assert(r, []byte("hello"), dir)
	r.says(t, "does not exist yet", "-update", "TestThing.golden")
}

// TestAssertNamesCRLF is worth a message of its own because the diff for it
// shows two identical looking blocks and nobody guesses line endings from that.
func TestAssertNamesCRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "TestThing.golden")
	if err := os.WriteFile(path, []byte("one\r\ntwo\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := &recorder{name: "TestThing"}
	Assert(r, []byte("one\ntwo\n"), Dir(dir))

	r.says(t, "line endings", "CRLF", "autocrlf")
}

func TestSubtestsGoInADirectory(t *testing.T) {
	if got, want := pathFor("testdata", "TestThing"), filepath.Join("testdata", "TestThing.golden"); got != want {
		t.Errorf("pathFor gave %q, want %q", got, want)
	}
	if got, want := pathFor("testdata", "TestThing/dark_mode"), filepath.Join("testdata", "TestThing", "dark_mode.golden"); got != want {
		t.Errorf("pathFor gave %q, want %q", got, want)
	}
	if got, want := pathFor("testdata", "TestA/b/c"), filepath.Join("testdata", "TestA", "b", "c.golden"); got != want {
		t.Errorf("pathFor gave %q, want %q", got, want)
	}
}

// TestSafeFileNames matters on Windows, where a name the testing package is
// happy with is a file name the filesystem refuses.
func TestSafeFileNames(t *testing.T) {
	tests := map[string]string{
		"plain":         "plain",
		"with:colon":    "with_colon",
		`with"quote`:    "with_quote",
		"with*star":     "with_star",
		"with?question": "with_question",
		`back\slash`:    "back_slash",
		"pipe|pipe":     "pipe_pipe",
		"":              "_",
		"unicode_日本語":   "unicode_日本語",
	}
	for in, want := range tests {
		if got := safe(in); got != want {
			t.Errorf("safe(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNameOverridesTheTestName(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")

	updating(t, func() { Assert(r, []byte("x"), dir, Name("something/else")) })
	if len(r.logs) != 1 || !strings.Contains(r.logs[0], filepath.Join("something", "else.golden")) {
		t.Errorf("wrote %v, want something/else.golden", r.logs)
	}
}

func TestAtNamesTheFileOutright(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.sql")

	r := &recorder{name: "TestThing"}
	updating(t, func() { Assert(r, []byte("create table t ();\n"), At(path)) })

	if _, err := os.Stat(path); err != nil {
		t.Errorf("At did not write %s: %v", path, err)
	}
}

func TestAssertStringIsAssert(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")

	updating(t, func() { AssertString(r, "hello\n", dir) })
	AssertString(r, "hello\n", dir)

	if r.failed {
		t.Errorf("AssertString failed against what it just wrote: %s", r.msg)
	}
}

// TestScrubbing checks the four built-in patterns and that a scrubber runs over
// the golden file as well as over the output. Without the second half, a file
// written before a scrubber was added would stop matching the moment one was.
func TestScrubbing(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")

	first := "id=f81d4fae-7dec-11d0-a765-00a0c91e6bf6 at=2026-08-23T10:00:00Z took=1.5ms"
	second := "id=550e8400-e29b-41d4-a716-446655440000 at=2027-01-02T03:04:05+09:00 took=920µs"

	opts := []Option{dir, ScrubUUIDs(), ScrubTimes(), ScrubDurations()}

	updating(t, func() { AssertString(r, first, opts...) })
	AssertString(r, second, opts...)

	if r.failed {
		t.Errorf("two runs with different volatile values did not match: %s", r.msg)
	}
}

func TestScrubPatterns(t *testing.T) {
	tests := []struct {
		name string
		opt  Option
		in   string
		want string
	}{
		{"uuid", ScrubUUIDs(), "f81d4fae-7dec-11d0-a765-00a0c91e6bf6", "<uuid>"},
		{"uuid v4", ScrubUUIDs(), "550e8400-e29b-41d4-a716-446655440000", "<uuid>"},
		{"not a uuid", ScrubUUIDs(), "f81d4fae-7dec-91d0-c765-00a0c91e6bf6", "f81d4fae-7dec-91d0-c765-00a0c91e6bf6"},
		{"ulid", ScrubULIDs(), "01ARZ3NDEKTSV4RRFFQ69G5FAV", "<ulid>"},
		{"time", ScrubTimes(), "2026-08-23T10:00:00Z", "<time>"},
		{"time with offset", ScrubTimes(), "2026-08-23T10:00:00.5+09:00", "<time>"},
		{"bare date", ScrubTimes(), "1987-04-12", "1987-04-12"},
		{"duration", ScrubDurations(), "1.5ms", "<duration>"},
		{"micros", ScrubDurations(), "920µs", "<duration>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var o options
			tt.opt(&o)
			if got := string(o.scrub([]byte(tt.in))); got != tt.want {
				t.Errorf("scrubbing %q gave %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestScrubTakesAPatternOfYourOwn(t *testing.T) {
	r, dir := newRecorder(t, "TestThing")
	port := regexp.MustCompile(`:\d{4,5}\b`)

	updating(t, func() { AssertString(r, "listening on 127.0.0.1:53142", dir, Scrub(port, ":<port>")) })
	AssertString(r, "listening on 127.0.0.1:8080", dir, Scrub(port, ":<port>"))

	if r.failed {
		t.Errorf("the port was not scrubbed: %s", r.msg)
	}
}

func TestPackageArgNamesThisPackage(t *testing.T) {
	if got := packageArg(&recorder{}); got != "./golden" {
		t.Errorf("packageArg gave %q, want ./golden", got)
	}
}

// TestAssertReportsAFileItCannotRead is the read failure that is not a missing
// file, where the advice to run -update would be wrong and the error itself is
// the whole story.
func TestAssertReportsAFileItCannotRead(t *testing.T) {
	dir := t.TempDir()
	r := &recorder{name: "TestThing"}

	// A directory where the golden file should be. Reading it fails with
	// something that is not os.ErrNotExist, which is the branch under test.
	if err := os.Mkdir(filepath.Join(dir, "TestThing.golden"), 0o777); err != nil {
		t.Fatal(err)
	}

	Assert(r, []byte("x"), Dir(dir))
	r.says(t, "TestThing.golden")
	if strings.Contains(r.msg, "-update") {
		t.Errorf("the failure suggests -update for an error -update will not fix:\n%s", r.msg)
	}
}

// TestAssertReportsAFileItCannotWrite covers the other half: -update is on and
// the file cannot be written, which is a real thing on a read-only checkout and
// a confusing one if the test simply passes.
func TestAssertReportsAFileItCannotWrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "wall"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	r := &recorder{name: "TestThing"}

	// The parent of the golden file is a regular file, so making the directory
	// fails before anything is written.
	updating(t, func() { Assert(r, []byte("x"), Dir(filepath.Join(dir, "wall", "under"))) })

	if !r.failed {
		t.Fatal("writing into a path that is not a directory passed, want it to fail")
	}
}

// TestAssertReportsAFileItCannotReplace is the third way writing goes wrong:
// the directory is fine and the path itself is not a file. It happens for real
// when a golden file and a golden directory have been given the same name.
func TestAssertReportsAFileItCannotReplace(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "TestThing.golden"), 0o777); err != nil {
		t.Fatal(err)
	}
	r := &recorder{name: "TestThing"}

	updating(t, func() { Assert(r, []byte("x"), Dir(dir)) })

	if !r.failed {
		t.Fatal("writing over a directory passed, want it to fail")
	}
}
