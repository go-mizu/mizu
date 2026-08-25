package validategen

import (
	"fmt"
	"strings"
)

// A block is a run of statements on their way into the output.
//
// It is built without indentation, because the file goes through gofmt before
// anybody sees it and counting tabs here would be counting them twice. Building
// a group in one of these rather than straight into the file is what lets a
// group with nothing in it be dropped, and an empty one has to be dropped: an
// if statement with no body reads as a mistake, and one whose init declares a
// variable nothing uses does not compile.
type block struct{ b strings.Builder }

func (x *block) line(format string, args ...any) {
	fmt.Fprintf(&x.b, format, args...)
	x.b.WriteByte('\n')
}

func (x *block) raw(s string)   { x.b.WriteString(s) }
func (x *block) empty() bool    { return x.b.Len() == 0 }
func (x *block) String() string { return x.b.String() }

// writeChain writes one field's rules as a chain of if and else if arms, with
// tail in the last else.
//
// The chain is what makes a field report one thing at a time. A title that is
// missing is missing, and saying it is also shorter than three characters is
// saying the same thing twice, so a rule that failed is the last one that runs.
// The tail is the loop over the field's elements, which is in the else for the
// same reason: a field that is not there has no elements worth looking at.
func writeChain(out *block, name string, steps []step, tail string) {
	declared := false
	chain(out, name, steps, tail, &declared)
}

// chain splits at the first omitempty and writes what is in front of it as
// arms, with the rest inside a test that the value was filled in.
//
// omitempty is not a check, so it records nothing and has no arm of its own.
// What it does is stop the rules behind it from running on a value that is not
// there, which is how min=3 and an optional field are written together.
func chain(out *block, name string, steps []step, tail string, declared *bool) {
	for i, s := range steps {
		if !s.skip {
			continue
		}
		// Whether the arms in front of the omitempty count something decides
		// whether the ones behind it have to count it again, and the arms in
		// front are written first, so the answer is worked out before the rest
		// of the chain is.
		inner := *declared
		for _, b := range steps[:i] {
			inner = inner || b.init != ""
		}

		var rest block
		chain(&rest, name, steps[i+1:], tail, &inner)

		wrapped := ""
		if !rest.empty() {
			wrapped = "if " + s.stop + " {\n" + rest.String() + "}\n"
		}
		arms(out, name, steps[:i], wrapped, declared)
		return
	}
	arms(out, name, steps, tail, declared)
}

// arms writes a run of rules, none of which is an omitempty.
func arms(out *block, name string, steps []step, tail string, declared *bool) {
	if len(steps) == 0 {
		out.raw(tail)
		return
	}
	for i, s := range steps {
		head := "if "
		if i > 0 {
			head = "} else if "
		}
		cond := s.cond
		if s.init != "" && !*declared {
			// The count goes in the if statement's init, whose scope is the
			// whole of the else chain behind it, so a field with a min and a
			// max on it counts its characters once.
			cond = s.init + "; " + s.cond
			*declared = true
		}
		out.line("%s%s {", head, cond)
		out.line("bad.Add(%s, %s)", name, s.fail)
	}
	if tail != "" {
		out.line("} else {")
		out.raw(tail)
	}
	out.line("}")
}
