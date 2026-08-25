package configgen

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/gen"
)

// The tests here are about the shapes a configuration struct comes in and what
// the generator writes for each of them. What it says when it refuses one is in
// testdata/_diag, where the message is the thing being reviewed.

// analyzeSrc runs the generator over one file of source, without that file
// having to be on disk, so a test of one struct reads as one thing rather than
// as a directory somewhere else.
func analyzeSrc(t *testing.T, src string) ([]gen.File, error) {
	t.Helper()
	pkgs, err := gen.Load(gen.Config{
		Dir:     "testdata",
		Overlay: map[string][]byte{"broken/config.go": []byte(src)},
	}, "./broken")
	if err != nil {
		t.Fatal(err)
	}
	return Generate(pkgs...)
}

const header = "package broken\n\n"

// TestNoMarker is a package with a configuration shaped struct that never
// asked for anything, which is not an error and not a file either. The second
// one asked a different generator, which is the same answer from here.
func TestNoMarker(t *testing.T) {
	for _, src := range []string{
		header + "type Config struct {\n\tName string\n}",
		header + "//mizu:model table=configs\ntype Config struct {\n\tName string\n}",
	} {
		files, err := analyzeSrc(t, src)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 0 {
			t.Errorf("wrote %d files for a package that asked for none", len(files))
		}
	}
}

// TestZeroValues checks what Redact writes over a secret with, for one secret
// of every shape a secret can be.
func TestZeroValues(t *testing.T) {
	files, err := analyzeSrc(t, header+`import (
	"net/netip"
	"time"
)

type Token string

//mizu:config
type Config struct {
	Text    string            `+"`secret:\"true\"`"+`
	Named   Token             `+"`secret:\"true\"`"+`
	On      bool              `+"`secret:\"true\"`"+`
	Count   int               `+"`secret:\"true\"`"+`
	Share   float64           `+"`secret:\"true\"`"+`
	Key     []byte            `+"`secret:\"true\"`"+`
	Table   map[string]string `+"`secret:\"true\"`"+`
	At      time.Time         `+"`secret:\"true\"`"+`
	For     time.Duration     `+"`secret:\"true\"`"+`
	Address netip.Addr        `+"`secret:\"true\"`"+`
}`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"out.Text = config.Redacted",
		"out.Named = Token(config.Redacted)",
		"out.On = false",
		"out.Count = 0",
		"out.Share = 0",
		"out.Key = nil",
		"out.Table = nil",
		"out.At = time.Time{}",
		"out.For = 0",
		"out.Address = netip.Addr{}",
	}
	src := string(files[0].Data)
	for _, w := range want {
		if !strings.Contains(src, w) {
			t.Errorf("the output does not have %q", w)
		}
	}
}

// TestNoSecrets checks that a configuration with nothing secret in it still
// gets a Redact, and that the copy says why it has nothing to do.
func TestNoSecrets(t *testing.T) {
	files, err := analyzeSrc(t, header+`//mizu:config
type Config struct {
	Name string
}`)
	if err != nil {
		t.Fatal(err)
	}
	src := string(files[0].Data)
	if !strings.Contains(src, "Nothing in this configuration is marked secret.") {
		t.Error("Redact does not say it has nothing to do")
	}
}

// TestNearMissMethods checks that a method of the right name and the wrong
// shape is not mistaken for the real one, since a field that reads itself
// wrongly is worse than one that does not read itself at all.
func TestNearMissMethods(t *testing.T) {
	files, err := analyzeSrc(t, header+`import "github.com/go-mizu/mizu/config"

// NoArgs has ParseConfig with nothing to parse.
type NoArgs string

func (n *NoArgs) ParseConfig() error { return nil }

// NoError has ParseConfig that says nothing about going wrong.
type NoError string

func (n *NoError) ParseConfig(v config.Value) {}

// WrongResult has ParseConfig that reports something other than an error.
type WrongResult string

func (w *WrongResult) ParseConfig(v config.Value) string { return "" }

//mizu:config
type Config struct {
	A NoArgs
	B NoError
	C WrongResult
}`)
	if err != nil {
		t.Fatal(err)
	}
	src := string(files[0].Data)
	if strings.Contains(src, "config.Config") {
		t.Error("a method of the wrong shape was taken for a parser")
	}
	for _, want := range []string{"&c.A, configFields[0].Field, config.String", "&c.B, configFields[1].Field, config.String", "&c.C, configFields[2].Field, config.String"} {
		if !strings.Contains(src, want) {
			t.Errorf("the output does not have %q", want)
		}
	}
}

// TestPrivateFieldsSkipped checks that a field generated code cannot reach is
// left out rather than reported, since a configuration struct with a private
// field in it usually means that field is not configuration.
func TestPrivateFieldsSkipped(t *testing.T) {
	files, err := analyzeSrc(t, header+`import "sync"

//mizu:config
type Config struct {
	App struct {
		Name string
		mu   sync.Mutex
	}
}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	src := string(files[0].Data)
	if strings.Contains(src, "sync") {
		t.Error("the private field made it into the output")
	}
	if !strings.Contains(src, "app.name") {
		t.Error("the field beside it did not")
	}
}

// TestEmbedded checks that an embedded struct adds no segment to a path, which
// is what lets an embedded mizu.Base sit at the top level.
func TestEmbedded(t *testing.T) {
	files, err := analyzeSrc(t, header+`type Base struct {
	Env   string
	Debug bool
}

//mizu:config
type Config struct {
	Base
	App struct {
		Name string
	}
}`)
	if err != nil {
		t.Fatal(err)
	}
	src := string(files[0].Data)
	for _, want := range []string{`Name: "Env", Path: "env"`, `Name: "Debug", Path: "debug"`, `Name: "App.Name", Path: "app.name"`} {
		if !strings.Contains(src, want) {
			t.Errorf("the output does not have %s", want)
		}
	}
	if strings.Contains(src, "base.env") {
		t.Error("the embedded struct added a segment to the path")
	}
}
