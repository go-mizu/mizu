package diag

import (
	"slices"
	"strings"
)

// The registry is the allocation authority for diagnostic codes.
//
// A code is permanent. MZ1042 means what it means today in every version there
// will ever be, an agent that learned it does not relearn it, and a search
// result from three years ago still points at the right page. What that costs
// is a place where every allocation is written down, because two subsystems
// that each pick the next free number without looking will pick the same one.
// That place is codes.go, one file, edited by whoever adds a diagnostic.
//
// A central file that nine subsystems have to edit produces merge conflicts.
// That is the mechanism working rather than a flaw in it: a conflict is two
// people allocating at once, which is the case the registry exists to catch,
// and the resolution is that one of them takes the next number.
//
// Code.Valid and Code.Docs check the shape of a code and not its membership
// here, so a code this registry has never heard of still renders and still
// gets a URL. That is deliberate. A project with its own generators has its own
// diagnostics, and a renderer that refused to print them would be a renderer
// nobody outside this repository could use.

// A Subsystem is the block of codes one part of the framework allocates from.
//
// The blocks are a thousand wide, so the leading digit says which part of the
// framework a code came from without a lookup: MZ1042 is configuration because
// it starts with a 1. There are nine blocks and no more, since there are nine
// leading digits, and section 3 of the file below says what happens when that
// stops being enough.
type Subsystem struct {
	// Name is what this part of the framework is called, lower case, as it
	// would read in the middle of a sentence.
	Name string

	// Doc is the design document the subsystem is specified in, as the number
	// its filename starts with. A subsystem that spans several names the first.
	Doc string

	// Low and High are the first and last code in the block, inclusive.
	Low, High int
}

// Contains reports whether the code numbered n falls in this block.
func (s Subsystem) Contains(n int) bool { return n >= s.Low && n <= s.High }

// An Entry is one allocated code.
type Entry struct {
	// Code is the code itself.
	Code Code

	// Summary is one line saying what the code means, in the general rather
	// than about any one occurrence. The message on a diagnostic names the
	// setting that was wrong; this names the kind of wrongness.
	Summary string

	// Retired is empty while the code is in use, and otherwise says what
	// replaced it or why it went away. A retired entry stays in the file. The
	// code is not handed to anything else, because somebody out there has the
	// old meaning written down.
	Retired string
}

// InUse reports whether e is a code something still reports.
func (e Entry) InUse() bool { return e.Retired == "" }

// Subsystems returns the blocks, in code order.
func Subsystems() []Subsystem { return slices.Clone(subsystems) }

// SubsystemOf returns the block c was allocated from.
//
// It works on the shape of the code rather than on whether the code has been
// allocated, so an unallocated MZ2999 still reports the database.
func SubsystemOf(c Code) (Subsystem, bool) {
	n := c.Number()
	if !c.Valid() {
		return Subsystem{}, false
	}
	for _, s := range subsystems {
		if s.Contains(n) {
			return s, true
		}
	}
	return Subsystem{}, false
}

// Entries returns every allocated code, retired ones included, in code order.
func Entries() []Entry { return slices.Clone(entries) }

// Lookup returns the entry for c.
//
// The second result is false for a code nobody has allocated, which is a
// different thing from a code that has been retired. A retired code is found
// and reports what it used to mean.
func Lookup(c Code) (Entry, bool) {
	i, ok := slices.BinarySearchFunc(entries, c, func(e Entry, c Code) int {
		return strings.Compare(string(e.Code), string(c))
	})
	if !ok {
		return Entry{}, false
	}
	return entries[i], true
}
