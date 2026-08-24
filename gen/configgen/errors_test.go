package configgen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/gen"
)

// analyzeSrc runs the generator over one file of source, without that file
// having to be on disk, so a test of a broken configuration reads as one
// thing rather than as a directory somewhere else.
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

func TestBadStructs(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "a type that nothing reads",
			src: header + `//mizu:config
type Config struct {
	App struct {
		Ch chan int
	}
}`,
			want: []string{"App.Ch", "chan int", "no parser reads"},
		},
		{
			name: "two fields with one path",
			src: header + `//mizu:config
type Config struct {
	App struct {
		Name string
		Also string ` + "`toml:\"name\"`" + `
	}
}`,
			want: []string{"App.Name", "App.Also", "app.name"},
		},
		{
			name: "a default that picks by environment",
			src: header + `//mizu:config
type Config struct {
	Log struct {
		Format string ` + "`default:\"console|json\"`" + `
	}
}`,
			want: []string{"Log.Format", "console|json", "mizu.Base"},
		},
		{
			name: "a default that refers to another field",
			src: header + `//mizu:config
type Config struct {
	App struct {
		Name   string
		Prefix string ` + "`default:\"{App.Name}:\"`" + `
	}
}`,
			want: []string{"App.Prefix", "{App.Name}:", "mizu.Base"},
		},
		{
			name: "a marker on something that is not a struct",
			src: header + `//mizu:config
type Config int`,
			want: []string{"Config", "not a struct"},
		},
		{
			name: "a map keyed by something other than a name",
			src: header + `//mizu:config
type Config struct {
	App struct {
		Ports map[int]string
	}
}`,
			want: []string{"App.Ports", "map[int]string", "no parser reads"},
		},
		{
			name: "two structs marked as configuration",
			src: header + `//mizu:config
type Config struct {
	Name string
}

//mizu:config
type Other struct {
	Name string
}`,
			want: []string{"2 structs marked as configuration"},
		},
		{
			name: "a struct with nothing in it",
			src: header + `//mizu:config
type Config struct{}`,
			want: []string{"Config", "no settings in it"},
		},
		{
			name: "a number no file can write",
			src: header + `//mizu:config
type Config struct {
	App struct {
		Ratio complex128
	}
}`,
			want: []string{"App.Ratio", "complex128", "no parser reads"},
		},
		{
			name: "a list of something nothing reads",
			src: header + `//mizu:config
type Config struct {
	App struct {
		Ratios []complex128
	}
}`,
			want: []string{"App.Ratios", "[]complex128", "no parser reads"},
		},
		{
			name: "a table of something nothing reads",
			src: header + `//mizu:config
type Config struct {
	App struct {
		Ratios map[string]complex128
	}
}`,
			want: []string{"App.Ratios", "map[string]complex128", "no parser reads"},
		},
		{
			name: "a marker written with a space",
			src: header + `// mizu:config
type Config struct {
	Name string
}`,
			want: []string{"mizu:config", "has a space after the slashes"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			files, err := analyzeSrc(t, c.src)
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

// TestTooDeep checks that the walk stops rather than going down forever, and
// says so when it does.
func TestTooDeep(t *testing.T) {
	var src strings.Builder
	src.WriteString(header + "//mizu:config\ntype Config struct {\n")
	depth := maxDepth + 1
	for i := range depth {
		fmt.Fprintf(&src, "%sLevel%d struct {\n", strings.Repeat("\t", i+1), i)
	}
	fmt.Fprintf(&src, "%sName string\n", strings.Repeat("\t", depth+1))
	for i := depth; i > 0; i-- {
		fmt.Fprintf(&src, "%s}\n", strings.Repeat("\t", i))
	}
	src.WriteString("}\n")

	_, err := analyzeSrc(t, src.String())
	if err == nil {
		t.Fatal("a struct nested past the limit was walked without complaint")
	}
	for _, want := range []string{"nests more than 12 deep", "Level0.Level1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not mention %q:\n%v", want, err)
		}
	}
}
