package diagtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// covering writes a package of one file and a corpus of one golden file, and
// returns the two directories in the order Cover takes them.
//
// The source is written rather than pointed at so that a test says in one place
// what the code prints and what the corpus holds, which is the whole of what
// Cover compares.
func covering(t *testing.T, src, golden string) (dir, pkg string) {
	t.Helper()

	dir, _ = corpus(t, map[string]string{"want.txt": golden})
	pkg = t.TempDir()
	if err := os.WriteFile(filepath.Join(pkg, "messages.go"), []byte(src), 0o666); err != nil {
		t.Fatal(err)
	}
	return dir, pkg
}

// A message with an entry is the case Cover is quiet about, and the three
// shapes here are the ones a generator writes: text on its own, text around a
// verb, and text between two of them.
func TestAMessageWithAnEntryPasses(t *testing.T) {
	tests := []struct {
		name   string
		call   string
		golden string
	}{
		{
			name:   "a message with nothing in it to fill in",
			call:   `errors.New("the command marker is on something with no name")`,
			golden: "error: commands.go:2:1: the command marker is on something with no name\n",
		},
		{
			name:   "a message that ends in what it is about",
			call:   `fmt.Errorf("is a %s, which no console.Value reads", t)`,
			golden: "error: commands.go:22:2: Ch is a chan int, which no console.Value reads\n",
		},
		{
			name:   "a message with text between its verbs",
			call:   `p.errf(pos, "%s and %s are both %s", a, b, c)`,
			golden: "error: config.go:7:3: App.Name and App.Also are both app.name\n",
		},
		{
			name:   "a message holding a percent sign",
			call:   `fmt.Errorf("%s is 100%% of the budget already", name)`,
			golden: "error: budget.go:9:2: http.serve is 100% of the budget already\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, pkg := covering(t, "package p\n\nfunc f() { _ = "+tt.call+" }\n", tt.golden)

			r := watch(t, func(tb testing.TB) { Cover(tb, dir, pkg) })
			if r.fatal != "" || len(r.errs) > 0 {
				t.Errorf("complained about a message that has an entry: %q %q", r.fatal, r.errs)
			}
		})
	}
}

func TestAMessageWithNoEntryFails(t *testing.T) {
	dir, pkg := covering(t,
		"package p\n\nfunc f() { _ = fmt.Errorf(\"is a %s, which no console.Value reads\", t) }\n",
		"error: commands.go:21:6: Command has no name, which is what somebody types to run it\n")

	r := watch(t, func(tb testing.TB) { Cover(tb, dir, pkg) })
	if got := r.only(t); !strings.Contains(got, "nothing under") {
		t.Errorf("says %q, want it to say nothing in the corpus prints the message", got)
	}
}

// The words around the verbs are most of a message, so changing one has to
// fail even though every other word is still there.
func TestAMessageThatWasRewordedFails(t *testing.T) {
	dir, pkg := covering(t,
		"package p\n\nfunc f() { _ = fmt.Errorf(\"%s and %s are both bound to %s\", a, b, c) }\n",
		"error: config.go:7:3: App.Name and App.Also are both app.name\n")

	r := watch(t, func(tb testing.TB) { Cover(tb, dir, pkg) })
	if got := r.only(t); !strings.Contains(got, "nothing under") {
		t.Errorf("says %q, want it to say nothing in the corpus prints the message", got)
	}
}

// A format that is nearly all verbs matches whatever the corpus holds, so
// passing it is worse than failing it.
func TestAFormatWithAlmostNoTextInItFails(t *testing.T) {
	dir, pkg := covering(t,
		"package p\n\nfunc f() { p.errf(pos, \"%s: %v\", name, err) }\n",
		"error: config.go:6:3: App.Ch is a chan int, which no parser reads\n")

	r := watch(t, func(tb testing.TB) { Cover(tb, dir, pkg) })
	if got := r.only(t); !strings.Contains(got, "mostly verbs") {
		t.Errorf("says %q, want it to say the format is mostly verbs", got)
	}
}

func TestASkippedFormatIsLeftAlone(t *testing.T) {
	dir, pkg := covering(t,
		"package p\n\nfunc f() { p.errf(pos, \"%s: %v\", name, err) }\n",
		"error: config.go:6:3: App.Ch is a chan int, which no parser reads\n")

	r := watch(t, func(tb testing.TB) { Cover(tb, dir, pkg, "%s: %v") })
	if r.fatal != "" || len(r.errs) > 0 {
		t.Errorf("complained about a format that is in the skip list: %q %q", r.fatal, r.errs)
	}
}

// A test file holds the messages a test expects rather than the ones the code
// prints, and reading it would have the corpus covering itself.
func TestMessagesInATestFileAreNotCounted(t *testing.T) {
	dir, pkg := covering(t,
		"package p\n\nfunc f() { _ = errors.New(\"the command marker is on something with no name\") }\n",
		"error: commands.go:2:1: the command marker is on something with no name\n")
	src := "package p\n\nfunc TestF(t *testing.T) { _ = errors.New(\"a message no golden file holds\") }\n"
	if err := os.WriteFile(filepath.Join(pkg, "messages_test.go"), []byte(src), 0o666); err != nil {
		t.Fatal(err)
	}

	r := watch(t, func(tb testing.TB) { Cover(tb, dir, pkg) })
	if r.fatal != "" || len(r.errs) > 0 {
		t.Errorf("read a test file: %q %q", r.fatal, r.errs)
	}
}

// Looking in the wrong place is quiet in the way that matters: there is
// nothing to check, so everything passes.
func TestLookingAtCodeWithNoMessagesInItFails(t *testing.T) {
	dir, pkg := covering(t, "package p\n\nfunc f() int { return 1 }\n", "error: nothing\n")

	r := watch(t, func(tb testing.TB) { Cover(tb, dir, pkg) })
	if got := r.only(t); !strings.Contains(got, "no messages in it") {
		t.Errorf("says %q, want it to say the code holds no messages", got)
	}
}

func TestAnEmptyCorpusFails(t *testing.T) {
	dir, pkg := covering(t, "package p\n\nfunc f() { _ = errors.New(\"no such table in this database\") }\n", "")

	r := watch(t, func(tb testing.TB) { Cover(tb, dir, pkg) })
	if got := r.only(t); !strings.Contains(got, "no golden files with anything in them") {
		t.Errorf("says %q, want it to say the corpus is empty", got)
	}
}

// between is the part of Cover that has to know how a format string is
// written, and a verb has more to it than a letter.
func TestSplittingAFormatOnItsVerbs(t *testing.T) {
	tests := []struct {
		format string
		want   []string
	}{
		{"", []string{""}},
		{"no verbs at all", []string{"no verbs at all"}},
		{"%s", []string{"", ""}},
		{"is a %s, which no parser reads", []string{"is a ", ", which no parser reads"}},
		{"is %d deep", []string{"is ", " deep"}},
		{"the default %q refers", []string{"the default ", " refers"}},
		{"%s is 100%% of it", []string{"", " is 100% of it"}},
		{"%-8s and %+.2f", []string{"", " and ", ""}},
		{"%s, which %w", []string{"", ", which ", ""}},
	}

	for _, tt := range tests {
		got := between(tt.format)
		if len(got) != len(tt.want) {
			t.Errorf("%q splits into %q, want %q", tt.format, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%q splits into %q, want %q", tt.format, got, tt.want)
				break
			}
		}
	}
}
