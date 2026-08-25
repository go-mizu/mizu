package commandgen

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/gen"
)

// generateSrc runs the generator over one file of source, without that file
// having to be on disk, so a test of a broken command reads as one thing rather
// than as a directory somewhere else.
func generateSrc(t *testing.T, src string) ([]gen.File, error) {
	t.Helper()
	pkgs, err := gen.Load(gen.Config{
		Dir:     "testdata",
		Overlay: map[string][]byte{"broken/commands.go": []byte(src)},
	}, "./broken")
	if err != nil {
		t.Fatal(err)
	}
	return Generate(pkgs...)
}

// header is the top of every source in this file: the package clause and the
// run method every case needs, so that what each case shows is the mistake it
// is about.
const header = `package broken

import (
	"context"
	"net/netip"
	"time"

	"github.com/go-mizu/mizu/console"
)

var (
	_ context.Context
	_ time.Time
	_ netip.Addr
	_ console.Spec
)

func (c *Command) Run(ctx context.Context, io *console.IO) error { return nil }

`

// tag writes a struct tag, since a raw string literal inside a raw string
// literal is not a thing Go has.
func tag(s string) string { return "`" + s + "`" }

func TestBadCommands(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "a marker with no name",
			src: header + `//mizu:command
type Command struct{}`,
			want: []string{"Command", "has no name", `name="command"`},
		},
		{
			name: "a name with a space in it",
			src: header + `//mizu:command name="users prune"
type Command struct{}`,
			want: []string{"users prune", "a command is one word"},
		},
		{
			name: "a marker on something that is not a struct",
			src: header + `//mizu:command name=go
type Command int`,
			want: []string{"Command", "not a struct"},
		},
		{
			name: "a command with no Run",
			src: `package broken

//mizu:command name=go
type Command struct{}`,
			want: []string{"Command", "no Run method"},
		},
		{
			name: "a Run of the wrong shape",
			src: `package broken

//mizu:command name=go
type Command struct{}

func (c *Command) Run() error { return nil }`,
			want: []string{"Command.Run", "func() error", "func(context.Context, *console.IO) error"},
		},
		{
			name: "a Run that reports nothing",
			src: `package broken

import (
	"context"

	"github.com/go-mizu/mizu/console"
)

//mizu:command name=go
type Command struct{}

func (c *Command) Run(ctx context.Context, io *console.IO) {}`,
			want: []string{"Command.Run", "func(context.Context, *console.IO)"},
		},
		{
			name: "a Run that reports two things",
			src: `package broken

import (
	"context"

	"github.com/go-mizu/mizu/console"
)

//mizu:command name=go
type Command struct{}

func (c *Command) Run(ctx context.Context, io *console.IO) (int, error) { return 0, nil }`,
			want: []string{"Command.Run", "(int, error)", "func(context.Context, *console.IO) error"},
		},
		{
			name: "a marker on the package rather than on a command",
			src: `//mizu:command name=go
package broken`,
			want: []string{"marker is on something with no name"},
		},
		{
			name: "a command that already has a Spec",
			src: header + `//mizu:command name=go
type Command struct{}

func (c *Command) Spec() console.Spec { return console.Spec{Name: "go"} }`,
			want: []string{"Command", "already has a Spec"},
		},
		{
			name: "a type no value reads",
			src: header + `//mizu:command name=go
type Command struct {
	Ch chan int ` + tag(`flag:""`) + `
}`,
			want: []string{"Ch", "chan int", "no console.Value reads"},
		},
		{
			name: "a number no command line writes",
			src: header + `//mizu:command name=go
type Command struct {
	Ratio complex128 ` + tag(`flag:""`) + `
}`,
			want: []string{"Ratio", "complex128", "no console.Value reads"},
		},
		{
			name: "a method of the right name and the wrong shape",
			src: header + `type NoArgs struct{}

func (n *NoArgs) UnmarshalText() error { return nil }

type NotBytes struct{}

func (n *NotBytes) UnmarshalText(s string) error { return nil }

type NotAMethod struct {
	UnmarshalText func([]byte) error
}

//mizu:command name=go
type Command struct {
	A NoArgs     ` + tag(`flag:"a"`) + `
	B NotBytes   ` + tag(`flag:"b"`) + `
	C NotAMethod ` + tag(`flag:"c"`) + `
}`,
			want: []string{"A: is a NoArgs", "B: is a NotBytes", "C: is a NotAMethod"},
		},
		{
			name: "a list of something no parser reads",
			src: header + `//mizu:command name=go
type Command struct {
	Where []netip.Addr ` + tag(`flag:""`) + `
}`,
			want: []string{"Where", "list of netip.Addr", "no console parser reads"},
		},
		{
			name: "a list of something a flag cannot be",
			src: header + `//mizu:command name=go
type Command struct {
	On []bool ` + tag(`flag:""`) + `
}`,
			want: []string{"On", "list of bool", "no console parser reads"},
		},
		{
			name: "a map that is not a map of text",
			src: header + `//mizu:command name=go
type Command struct {
	Weights map[string]int ` + tag(`flag:""`) + `
}`,
			want: []string{"Weights", "map[string]int", "map[string]string"},
		},
		{
			name: "a list type of its own",
			src: header + `type Tags []string

//mizu:command name=go
type Command struct {
	Tags Tags ` + tag(`flag:""`) + `
}`,
			want: []string{"Tags", "list type of its own", "[]string"},
		},
		{
			name: "a map type of its own",
			src: header + `type Headers map[string]string

//mizu:command name=go
type Command struct {
	Header Headers ` + tag(`flag:""`) + `
}`,
			want: []string{"Header", "map type of its own", "map[string]string"},
		},
		{
			name: "a count that is not a plain int",
			src: header + `//mizu:command name=go
type Command struct {
	Loud uint ` + tag(`flag:"" count:"true"`) + `
}`,
			want: []string{"Loud", "counts up as -vv", "plain int"},
		},
		{
			name: "an enum that is not text",
			src: header + `//mizu:command name=go
type Command struct {
	Mode int ` + tag(`flag:"" enum:"one|two"`) + `
}`,
			want: []string{"Mode", "enum tag", "only text"},
		},
		{
			name: "a field that is both a flag and an argument",
			src: header + `//mizu:command name=go
type Command struct {
	Where string ` + tag(`flag:"" arg:"0"`) + `
}`,
			want: []string{"Where", "both a flag and an argument"},
		},
		{
			name: "a tag with no flag or arg beside it",
			src: header + `//mizu:command name=go
type Command struct {
	Where string ` + tag(`default:"here"`) + `
}`,
			want: []string{"Where", "default tag", "nothing reads it"},
		},
		{
			name: "a short flag of more than one letter",
			src: header + `//mizu:command name=go
type Command struct {
	Where string ` + tag(`flag:"where,wh"`) + `
}`,
			want: []string{"Where", `"wh"`, "one letter"},
		},
		{
			name: "a flag that is required and has a default",
			src: header + `//mizu:command name=go
type Command struct {
	Where string ` + tag(`flag:"" required:"true" default:"here"`) + `
}`,
			want: []string{"Where", "required and has a default"},
		},
		{
			name: "two flags with the same name",
			src: header + `//mizu:command name=go
type Command struct {
	Where string ` + tag(`flag:"place"`) + `
	Place string ` + tag(`flag:""`) + `
}`,
			want: []string{"Command.Where", "Command.Place", "--place"},
		},
		{
			name: "two flags with the same letter",
			src: header + `//mizu:command name=go
type Command struct {
	Where string ` + tag(`flag:"where,w"`) + `
	When  string ` + tag(`flag:"when,w"`) + `
}`,
			want: []string{"Command.Where", "Command.When", "-w"},
		},
		{
			name: "an argument with no place on the line",
			src: header + `//mizu:command name=go
type Command struct {
	Where string ` + tag(`arg:"first"`) + `
}`,
			want: []string{"Where", `"first"`, `arg:"0"`},
		},
		{
			name: "an argument nothing can reach",
			src: header + `//mizu:command name=go
type Command struct {
	Where string ` + tag(`arg:"0"`) + `
	When  string ` + tag(`arg:"2"`) + `
}`,
			want: []string{"Command.When", "argument 2", "no argument 1"},
		},
		{
			name: "an argument taking the rest that is not last",
			src: header + `//mizu:command name=go
type Command struct {
	Where []string ` + tag(`arg:"0..."`) + `
	When  string   ` + tag(`arg:"1"`) + `
}`,
			want: []string{"Command.Where", "not the last argument"},
		},
		{
			name: "an argument taking the rest that is not a list",
			src: header + `//mizu:command name=go
type Command struct {
	Where string ` + tag(`arg:"0..."`) + `
}`,
			want: []string{"Command.Where", "not a list", "last word"},
		},
		{
			name: "a required argument after an optional one",
			src: header + `//mizu:command name=go
type Command struct {
	Where string ` + tag(`arg:"0" default:"here"`) + `
	When  string ` + tag(`arg:"1"`) + `
}`,
			want: []string{"Command.When", "Where", "never be reached"},
		},
		{
			name: "a marker written with a space",
			src: header + `// mizu:command name=go
type Command struct{}`,
			want: []string{"mizu:command", "has a space after the slashes"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files, err := generateSrc(t, c.src)
			if err == nil {
				t.Fatalf("generated %d files without complaint", len(files))
			}
			msg := err.Error()
			for _, want := range c.want {
				if !strings.Contains(msg, want) {
					t.Errorf("the error does not mention %q:\n%s", want, msg)
				}
			}
		})
	}
}

// TestNoMarker is a package with a command shaped struct that never asked for
// anything, which is not an error and not a file either. The second one asked a
// different generator, which is the same answer from here.
func TestNoMarker(t *testing.T) {
	for _, src := range []string{
		header + "type Command struct{}",
		header + "//mizu:model table=commands\ntype Command struct{}",
	} {
		files, err := generateSrc(t, src)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 0 {
			t.Errorf("wrote %d files for a package that asked for none", len(files))
		}
	}
}

// TestOptionalArgumentAfterOptional checks the case next to the one that is
// reported, since an optional argument after an optional one is how a command
// takes two things it can do without.
func TestOptionalArgumentAfterOptional(t *testing.T) {
	files, err := generateSrc(t, header+`//mizu:command name=go
type Command struct {
	Where string `+tag(`arg:"0" default:"here"`)+`
	When  string `+tag(`arg:"1" required:"false"`)+`
}`)
	if err != nil {
		t.Fatal(err)
	}
	src := string(files[0].Data)
	if strings.Contains(src, "Required: true") {
		t.Errorf("an argument was made required:\n%s", src)
	}
}

// TestArgumentsOutOfDeclarationOrder checks that the arg tag decides the order
// on the line, not where the field happens to sit in the struct, since grouping
// the fields by what they are about is a reasonable thing to want.
func TestArgumentsOutOfDeclarationOrder(t *testing.T) {
	files, err := generateSrc(t, header+`//mizu:command name=go
type Command struct {
	When  string `+tag(`arg:"1"`)+`
	Where string `+tag(`arg:"0"`)+`
}`)
	if err != nil {
		t.Fatal(err)
	}
	src := string(files[0].Data)
	where, when := strings.Index(src, `"where"`), strings.Index(src, `"when"`)
	if where < 0 || when < 0 {
		t.Fatalf("an argument is missing:\n%s", src)
	}
	if where > when {
		t.Errorf("the arguments came out in declaration order rather than in tag order:\n%s", src)
	}
}

// TestCommandWithNothingToParse checks that a command taking no flags and no
// arguments still gets a spec, since a command that does one thing is the
// simplest one there is and would be an odd thing to refuse.
func TestCommandWithNothingToParse(t *testing.T) {
	files, err := generateSrc(t, header+`//mizu:command name=go desc="Go"
type Command struct{}`)
	if err != nil {
		t.Fatal(err)
	}
	src := string(files[0].Data)
	for _, want := range []string{`Name: "go"`, `Desc: "Go"`, "func Commands() []console.Command"} {
		if !strings.Contains(src, want) {
			t.Errorf("the output does not have %q:\n%s", want, src)
		}
	}
}

// TestAliasedImport checks the one package in the world that is generated into
// and called console, which is the console package's own tests.
func TestAliasedImport(t *testing.T) {
	pkgs := load(t, "testdata", "./app")
	pkgs[0].Name = "console"

	files, err := Generate(pkgs...)
	if err != nil {
		t.Fatal(err)
	}
	src := string(files[0].Data)
	if !strings.Contains(src, `import mizuconsole "github.com/go-mizu/mizu/console"`) {
		t.Errorf("the import is not aliased:\n%s", src)
	}
	if !strings.Contains(src, "mizuconsole.Int(&c.Days)") {
		t.Errorf("a value does not use the alias:\n%s", src)
	}
}
