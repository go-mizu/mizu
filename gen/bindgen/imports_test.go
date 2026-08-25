package bindgen

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

// The import block is worked out while the output is being written rather than
// fixed up afterwards, so the rules in it are worth checking on their own. A
// generated file that imports one package too many does not compile, and the
// corpus only reaches the shapes a request struct happens to use.

func TestImportName(t *testing.T) {
	i := newImports("example.com/app")

	cases := []struct {
		pkg  string
		want string
	}{
		{"", ""},                            // nothing to import
		{"example.com/app", ""},             // the file being written
		{"time", "time"},                    //
		{"time", "time"},                    // asked for twice, named once
		{"net/netip", "netip"},              // named by the last element
		{"example.com/thing/v2", "thing"},   // a major version is not a name
		{"example.com/other/time", "time2"}, // the second one to want a name gets a number
		{"example.com/third/time", "time3"}, //
		{"example.com/go-cache", "gocache"}, // a dash cannot be in an identifier
		{"example.com/2fa", "fa"},           // and neither can a leading digit
		{"example.com/123", "pkg"},          // nothing usable left
		{"github.com/go-mizu/mizu/web", "web"},
	}
	for _, c := range cases {
		if got := i.name(c.pkg); got != c.want {
			t.Errorf("name(%q) = %q, want %q", c.pkg, got, c.want)
		}
	}
}

// A package loaded by go/types is named by what it calls itself rather than by
// the end of its path, which is the only way to be right about a package whose
// name and directory differ.
func TestImportPkg(t *testing.T) {
	i := newImports("example.com/app")

	for _, c := range []struct {
		path, name, want string
	}{
		{"example.com/app", "app", ""},
		{"net/netip", "netip", "netip"},
		{"example.com/pb/v3", "userpb", "userpb"},
		{"example.com/other/userpb", "userpb", "userpb2"},
	} {
		if got := i.pkg(types.NewPackage(c.path, c.name)); got != c.want {
			t.Errorf("pkg(%q as %q) = %q, want %q", c.path, c.name, got, c.want)
		}
	}
}

// The block is written in the order gofmt leaves it in, so nothing has to
// reformat the file after it is rendered.
func TestImportLines(t *testing.T) {
	i := newImports("example.com/app")
	for _, pkg := range []string{"time", "net/netip", "example.com/other/time", "github.com/go-mizu/mizu/web"} {
		i.name(pkg)
	}

	want := []importLine{
		{Path: "net/netip"},
		{Path: "time"},
		{Path: "example.com/other/time", Alias: "time2"},
		{Path: "github.com/go-mizu/mizu/web"},
	}
	got := i.lines()
	if len(got) != len(want) {
		t.Fatalf("lines = %v, want %v", got, want)
	}
	for n := range got {
		if got[n] != want[n] {
			t.Errorf("line %d = %v, want %v", n, got[n], want[n])
		}
	}
}

// A struct with no times, no addresses and no files in it imports the web
// package and nothing else, which is the common case and reads better on one
// line.
func TestWriteOneImport(t *testing.T) {
	var b strings.Builder
	writeImports(&b, []importLine{{Path: "github.com/go-mizu/mizu/web"}})

	want := "import \"github.com/go-mizu/mizu/web\"\n\n"
	if b.String() != want {
		t.Errorf("got %q, want %q", b.String(), want)
	}
}

func TestWriteImports(t *testing.T) {
	var b strings.Builder
	writeImports(&b, []importLine{
		{Path: "time"},
		{Path: "example.com/other/time", Alias: "time2"},
		{Path: "github.com/go-mizu/mizu/web"},
	})

	want := `import (
	"time"

	time2 "example.com/other/time"
	"github.com/go-mizu/mizu/web"
)

`
	if b.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", b.String(), want)
	}
}

// A type in a message is written for somebody to read, so it takes the
// package's own name and asks for no import, and a type from the package being
// generated into is written bare.
func TestDocString(t *testing.T) {
	i := newImports("example.com/app")
	self := types.NewPackage("example.com/app", "app")
	other := types.NewPackage("net/netip", "netip")

	named := func(p *types.Package, name string) types.Type {
		return types.NewNamed(types.NewTypeName(token.NoPos, p, name, nil), types.Typ[types.String], nil)
	}

	for _, c := range []struct {
		typ  types.Type
		want string
	}{
		{named(self, "Address"), "Address"},
		{types.NewPointer(named(other, "Addr")), "*netip.Addr"},
		{types.NewSlice(named(other, "Addr")), "[]netip.Addr"},
	} {
		if got := i.docString(c.typ); got != c.want {
			t.Errorf("docString = %q, want %q", got, c.want)
		}
	}

	if lines := i.lines(); len(lines) != 0 {
		t.Errorf("a message asked for %v, and a message imports nothing", lines)
	}
}

func TestBase(t *testing.T) {
	cases := map[string]string{
		"time":                  "time",
		"net/netip":             "netip",
		"example.com/thing/v2":  "thing",
		"example.com/validate":  "validate", // a v is not a version without digits after it
		"example.com/v":         "v",
		"example.com/go-cache":  "gocache",
		"example.com/some_pkg":  "some_pkg",
		"example.com/oauth2":    "oauth2", // a digit is fine once there is a letter before it
		"example.com/123":       "pkg",
		"example.com/thing/v10": "thing",
	}
	for pkg, want := range cases {
		if got := base(pkg); got != want {
			t.Errorf("base(%q) = %q, want %q", pkg, got, want)
		}
	}
}

func TestIsStd(t *testing.T) {
	cases := map[string]bool{
		"time":                        true,
		"net/netip":                   true,
		"mime/multipart":              true,
		"github.com/go-mizu/mizu/web": false,
		"example.com/app":             false,
	}
	for pkg, want := range cases {
		if got := isStd(pkg); got != want {
			t.Errorf("isStd(%q) = %v, want %v", pkg, got, want)
		}
	}
}

func TestLower(t *testing.T) {
	cases := map[string]string{"": "", "Tags": "tags", "URLs": "uRLs", "x": "x"}
	for in, want := range cases {
		if got := lower(in); got != want {
			t.Errorf("lower(%q) = %q, want %q", in, got, want)
		}
	}
}
