package diag_test

import (
	"fmt"
	"strings"
	"testing"
	"unicode"

	"github.com/go-mizu/mizu/errs/diag"
	"github.com/go-mizu/mizu/golden"
)

// The whole table, as a file. This is the test that makes a deleted line show
// up as a deleted line. Nothing else in this package can see the history of the
// registry, so a code that is removed and later handed to something else would
// pass every other test here, and the only defence against it is that the
// removal has to be written down and looked at.
func TestRegistryTheWholeTable(t *testing.T) {
	var b strings.Builder
	for _, s := range diag.Subsystems() {
		fmt.Fprintf(&b, "MZ%d000-MZ%d  %s (doc %s)\n", s.Low/1000, s.High, s.Name, s.Doc)
	}
	b.WriteString("\n")
	for _, e := range diag.Entries() {
		fmt.Fprintf(&b, "%s  %s", e.Code, e.Summary)
		if !e.InUse() {
			fmt.Fprintf(&b, "\n        retired: %s", e.Retired)
		}
		b.WriteString("\n")
	}
	golden.AssertString(t, b.String())
}

func TestSubsystemsCoverTheWholeSpaceWithoutOverlapping(t *testing.T) {
	got := diag.Subsystems()
	if len(got) != 9 {
		t.Fatalf("there are %d subsystems, want 9, one per leading digit", len(got))
	}
	want := 1000
	for _, s := range got {
		if s.Low != want {
			t.Errorf("%s starts at %d, want %d", s.Name, s.Low, want)
		}
		if s.High != s.Low+999 {
			t.Errorf("%s runs %d to %d, want a block of a thousand", s.Name, s.Low, s.High)
		}
		if s.Name == "" || s.Doc == "" {
			t.Errorf("the block at %d has no name or no document", s.Low)
		}
		if capitalised(s.Name) {
			t.Errorf("%q starts with a capital, and it is read in the middle of a sentence", s.Name)
		}
		want = s.High + 1
	}
}

// MZ0xxx is held back rather than allocated, so that a block that fills has
// somewhere to go that is not a renumbering.
func TestTheZeroBlockIsNotAllocated(t *testing.T) {
	for _, c := range []diag.Code{"MZ0000", "MZ0001", "MZ0999"} {
		if s, ok := diag.SubsystemOf(c); ok {
			t.Errorf("%s belongs to %s, want the block held in reserve", c, s.Name)
		}
		if _, ok := diag.Lookup(c); ok {
			t.Errorf("%s is allocated, want the block held in reserve", c)
		}
	}
}

func TestSubsystemOf(t *testing.T) {
	for _, tt := range []struct {
		c    diag.Code
		want string
	}{
		{"MZ1042", "configuration"},
		{"MZ1000", "configuration"},
		{"MZ1999", "configuration"},
		{"MZ2000", "the database, the query builder and the ORM"},
		{"MZ9999", "the toolchain, the console and the generators"},
	} {
		s, ok := diag.SubsystemOf(tt.c)
		if !ok {
			t.Errorf("SubsystemOf(%s) found nothing", tt.c)
			continue
		}
		if s.Name != tt.want {
			t.Errorf("SubsystemOf(%s) = %q, want %q", tt.c, s.Name, tt.want)
		}
	}
}

func TestSubsystemOfSomethingThatIsNotACode(t *testing.T) {
	for _, c := range []diag.Code{"", "nonsense", "MZ104", "mz1042"} {
		if _, ok := diag.SubsystemOf(c); ok {
			t.Errorf("SubsystemOf(%q) found a subsystem", c)
		}
	}
}

func TestSubsystemContains(t *testing.T) {
	s := diag.Subsystem{Low: 1000, High: 1999}
	for _, tt := range []struct {
		n    int
		want bool
	}{{999, false}, {1000, true}, {1500, true}, {1999, true}, {2000, false}} {
		if got := s.Contains(tt.n); got != tt.want {
			t.Errorf("Contains(%d) = %v, want %v", tt.n, got, tt.want)
		}
	}
}

// Every allocation is inside a block, so a code cannot be taken from the space
// held in reserve or from a leading digit nobody assigned.
func TestEveryCodeIsInABlock(t *testing.T) {
	for _, e := range diag.Entries() {
		if !e.Code.Valid() {
			t.Errorf("%q is not shaped like a code", e.Code)
			continue
		}
		if _, ok := diag.SubsystemOf(e.Code); !ok {
			t.Errorf("%s is allocated from no block", e.Code)
		}
	}
}

// In order and each one only once. Two lines with the same code is the whole
// failure the registry exists to prevent, and out of order is how the second
// one gets missed in review.
func TestTheTableIsInOrderWithNoDuplicates(t *testing.T) {
	entries := diag.Entries()
	for i := 1; i < len(entries); i++ {
		prev, cur := entries[i-1].Code, entries[i].Code
		if cur == prev {
			t.Errorf("%s is in the table twice", cur)
		} else if cur < prev {
			t.Errorf("%s comes after %s, and the table is kept in code order", cur, prev)
		}
	}
}

// A summary is read in a list next to eight others, so it is one line, it does
// not start with a capital, and it does not end with a full stop. The rules are
// worth a test because the file is edited by whoever adds a diagnostic and
// nobody rereads the file before adding a line to it.
func TestEverySummaryReadsLikeTheOthers(t *testing.T) {
	for _, e := range diag.Entries() {
		switch {
		case e.Summary == "":
			t.Errorf("%s says nothing about what it means", e.Code)
		case strings.Contains(e.Summary, "\n"):
			t.Errorf("%s has a summary of more than one line", e.Code)
		case strings.HasSuffix(e.Summary, "."):
			t.Errorf("%s ends its summary with a full stop", e.Code)
		case capitalised(e.Summary):
			t.Errorf("%s starts its summary with a capital", e.Code)
		}
	}
}

// capitalised reports whether s starts with a capitalised ordinary word, which
// is the thing these tests are against. A name that starts with an acronym does
// not, because HTTP and RPC are spelled that way wherever they appear and the
// alternative is a table that says "http routing" in one row and "sessions" in
// the next. The second rune is what tells them apart.
func capitalised(s string) bool {
	r := []rune(s)
	if len(r) == 0 || !unicode.IsUpper(r[0]) {
		return false
	}
	return len(r) < 2 || !unicode.IsUpper(r[1])
}

// A retired code keeps its summary, because the reader who arrives with the old
// code wants to know what it used to mean before being told it went away.
func TestARetiredCodeStillSaysWhatItMeant(t *testing.T) {
	for _, e := range diag.Entries() {
		if e.InUse() {
			continue
		}
		if e.Summary == "" {
			t.Errorf("%s is retired and no longer says what it used to mean", e.Code)
		}
	}
}

func TestLookup(t *testing.T) {
	e, ok := diag.Lookup("MZ1042")
	if !ok {
		t.Fatal("MZ1042 is not in the registry, and it is the published example")
	}
	if e.Code != "MZ1042" {
		t.Errorf("looked up MZ1042 and got %s", e.Code)
	}
	if !e.InUse() {
		t.Error("MZ1042 is retired, and two documents still print it")
	}
	if e.Summary == "" {
		t.Error("MZ1042 has no summary")
	}
}

// Not allocated and retired are different answers, and a caller that wants to
// tell a typo from a code that went away needs both.
func TestLookupOfSomethingUnallocated(t *testing.T) {
	for _, c := range []diag.Code{"MZ1998", "MZ7777", "", "nonsense"} {
		if e, ok := diag.Lookup(c); ok {
			t.Errorf("Lookup(%q) found %+v", c, e)
		}
	}
}

func TestEntryInUse(t *testing.T) {
	if !(diag.Entry{}).InUse() {
		t.Error("an entry with no reason for being retired is retired")
	}
	if (diag.Entry{Retired: "replaced by MZ1004"}).InUse() {
		t.Error("an entry with a reason for being retired is in use")
	}
}

// The two accessors hand out copies. A caller that sorts what it got back
// should not reorder the registry for everybody else in the process.
func TestTheTableCannotBeEditedFromOutside(t *testing.T) {
	got := diag.Entries()
	got[0] = diag.Entry{Code: "MZ9999", Summary: "not a real code"}
	if again := diag.Entries(); again[0].Code == "MZ9999" {
		t.Error("writing to the result of Entries changed the registry")
	}

	subs := diag.Subsystems()
	subs[0] = diag.Subsystem{Name: "not a real subsystem"}
	if again := diag.Subsystems(); again[0].Name == "not a real subsystem" {
		t.Error("writing to the result of Subsystems changed the registry")
	}
}
