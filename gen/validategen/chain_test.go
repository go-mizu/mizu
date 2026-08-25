package validategen

import (
	"strings"
	"testing"
)

// The chain is where the shape of the output is decided, and the shapes worth
// pinning are the ones the corpus reaches by accident rather than on purpose:
// an omitempty in the middle of a run, a count that is already declared by the
// time the rest of the chain wants it, and a chain with nothing in it at all.
//
// What comes out is unindented, because gofmt lays the file out afterwards.

func TestWriteChain(t *testing.T) {
	required := step{cond: `v.Q == ""`, fail: `validate.Failed("required")`}
	optional := step{skip: true, stop: `v.Q != ""`}
	minimum := step{init: "n := utf8.RuneCountInString(v.Q)", cond: "n < 2", fail: `validate.Failed("min", 2).Of("string")`}
	maximum := step{init: "n := utf8.RuneCountInString(v.Q)", cond: "n > 64", fail: `validate.Failed("max", 64).Of("string")`}

	for _, c := range []struct {
		name  string
		steps []step
		tail  string
		want  string
	}{
		{
			name:  "one rule",
			steps: []step{required},
			want: `if v.Q == "" {
bad.Add(name, validate.Failed("required"))
}
`,
		},
		{
			// The count is in the first arm's init, whose scope reaches the
			// else behind it, so the characters are counted once.
			name:  "a bound at each end",
			steps: []step{minimum, maximum},
			want: `if n := utf8.RuneCountInString(v.Q); n < 2 {
bad.Add(name, validate.Failed("min", 2).Of("string"))
} else if n > 64 {
bad.Add(name, validate.Failed("max", 64).Of("string"))
}
`,
		},
		{
			name:  "optional with a bound behind it",
			steps: []step{optional, minimum},
			want: `if v.Q != "" {
if n := utf8.RuneCountInString(v.Q); n < 2 {
bad.Add(name, validate.Failed("min", 2).Of("string"))
}
}
`,
		},
		{
			// The arm in front of the omitempty already declared n, so the one
			// behind it uses what is there rather than declaring it again,
			// which would not compile.
			name:  "a bound on each side of an optional",
			steps: []step{minimum, optional, maximum},
			want: `if n := utf8.RuneCountInString(v.Q); n < 2 {
bad.Add(name, validate.Failed("min", 2).Of("string"))
} else {
if v.Q != "" {
if n > 64 {
bad.Add(name, validate.Failed("max", 64).Of("string"))
}
}
}
`,
		},
		{
			// Nothing behind the omitempty means nothing to guard, so the
			// if that would have guarded it is not written.
			name:  "an optional with nothing behind it",
			steps: []step{required, optional},
			want: `if v.Q == "" {
bad.Add(name, validate.Failed("required"))
}
`,
		},
		{
			name:  "nothing at all",
			steps: nil,
			want:  "",
		},
		{
			name:  "a tail with no rules in front of it",
			steps: nil,
			tail:  "for i, e := range v.Q {\n}\n",
			want:  "for i, e := range v.Q {\n}\n",
		},
		{
			name:  "a tail behind a rule",
			steps: []step{required},
			tail:  "for i, e := range v.Q {\n}\n",
			want: `if v.Q == "" {
bad.Add(name, validate.Failed("required"))
} else {
for i, e := range v.Q {
}
}
`,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			var out block
			writeChain(&out, "name", c.steps, c.tail)
			if got := out.String(); got != c.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
			}
		})
	}
}

// A block with nothing in it is dropped rather than written, since an if with
// an empty body reads as a mistake and one whose init declares a variable
// nothing uses does not compile.
func TestBlock(t *testing.T) {
	var x block
	if !x.empty() {
		t.Error("a new block is not empty")
	}
	x.line("if %s {", "ok")
	x.raw("bad.Add(name, e)\n")
	x.line("}")
	if x.empty() {
		t.Error("a block with three lines in it says it is empty")
	}
	const want = "if ok {\nbad.Add(name, e)\n}\n"
	if x.String() != want {
		t.Errorf("got %q, want %q", x.String(), want)
	}
	if strings.Contains(x.String(), "\t") {
		t.Error("a block indented itself, which gofmt is for")
	}
}
