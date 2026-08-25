package commandgen

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/gen"
)

// The tests here are about the shapes a command comes in and what the
// generator writes for each of them. What it says when it refuses one is in
// testdata/_diag, where the message is the thing being reviewed.

// generateSrc runs the generator over one file of source, without that file
// having to be on disk, so a test of one command reads as one thing rather than
// as a directory somewhere else.
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
