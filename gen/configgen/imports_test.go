package configgen

import (
	"strings"
	"testing"
)

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
		{"log/slog", "slog"},                // named by the last element
		{"example.com/thing/v2", "thing"},   // a major version is not a name
		{"example.com/other/time", "time2"}, // the second one to want a name gets a number
		{"example.com/third/time", "time3"}, //
		{"example.com/go-cache", "gocache"}, // a dash cannot be in an identifier
		{"example.com/2fa", "fa"},           // and neither can a leading digit
		{"example.com/v1", "examplecom"},    // a whole module at a version
		{"example.com/123", "pkg"},          // nothing usable left
		{"github.com/go-mizu/mizu/config", "config"},
	}
	for _, c := range cases {
		if got := i.name(c.pkg); got != c.want {
			t.Errorf("name(%q) = %q, want %q", c.pkg, got, c.want)
		}
	}
}

func TestImportLines(t *testing.T) {
	i := newImports("example.com/app")
	for _, pkg := range []string{"time", "slices", "example.com/other/time", "github.com/go-mizu/mizu/config"} {
		i.add(pkg)
	}

	want := []importLine{
		{Path: "slices"},
		{Path: "time"},
		{Path: "example.com/other/time", Alias: "time2"},
		{Path: "github.com/go-mizu/mizu/config"},
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

func TestWriteImports(t *testing.T) {
	var b strings.Builder
	writeImports(&b, []importLine{
		{Path: "time"},
		{Path: "example.com/other/time", Alias: "time2"},
		{Path: "github.com/go-mizu/mizu/config"},
	})
	want := `import (
	"time"

	time2 "example.com/other/time"
	"github.com/go-mizu/mizu/config"
)

`
	if b.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", b.String(), want)
	}
}

func TestBase(t *testing.T) {
	cases := map[string]string{
		"time":                  "time",
		"log/slog":              "slog",
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
		"time":                           true,
		"log/slog":                       true,
		"net/netip":                      true,
		"github.com/go-mizu/mizu/config": false,
		"example.com/app":                false,
	}
	for pkg, want := range cases {
		if got := isStd(pkg); got != want {
			t.Errorf("isStd(%q) = %v, want %v", pkg, got, want)
		}
	}
}

func TestLower(t *testing.T) {
	cases := map[string]string{"": "", "Config": "config", "HTTPConfig": "hTTPConfig", "x": "x"}
	for in, want := range cases {
		if got := lower(in); got != want {
			t.Errorf("lower(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOneLine(t *testing.T) {
	cases := map[string]string{
		"":                          "",
		"   \n  ":                   "",
		"One sentence.\n":           "One sentence.",
		"Wrapped across\ntwo lines": "Wrapped across two lines",
		"First paragraph.\n\nSecond one, which is for the source and not for a table.": "First paragraph.",
	}
	for in, want := range cases {
		if got := oneLine(in); got != want {
			t.Errorf("oneLine(%q) = %q, want %q", in, got, want)
		}
	}
}
