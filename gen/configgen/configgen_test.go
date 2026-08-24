package configgen

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/gen"
	"github.com/go-mizu/mizu/golden"
)

// goldenPath is the file the generator is expected to produce, checked in so
// that a change to the output is a change in a diff rather than something that
// only happens on someone's machine. It is a real Go file that the testdata
// module compiles, so it lives beside the code rather than under testdata/golden
// with a .golden suffix, which is what [golden.At] is for.
const goldenPath = "testdata/app/config_gen.go"

func TestGenerate(t *testing.T) {
	files := generate(t, "testdata", "./app")
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if got := files[0].Path; got != "app/config_gen.go" {
		t.Errorf("path = %q, want app/config_gen.go", got)
	}

	golden.Assert(t, format(t, files[0]), golden.At(goldenPath))
}

// TestFields checks the details that are hard to see in a golden file: that
// every setting is there, that names and paths are derived the way the
// documentation says, and that the tags do what they claim.
func TestFields(t *testing.T) {
	p := plan(t)

	if len(p.Fields) < 40 {
		t.Errorf("got %d settings, want at least 40", len(p.Fields))
	}

	byName := map[string]Field{}
	for _, f := range p.Fields {
		byName[f.Name] = f
	}

	cases := []struct {
		name string
		want Field
	}{
		{"App.Name", Field{Path: "app.name", Env: "APP_NAME", Default: "blog", Type: "string"}},
		{"App.Env", Field{Path: "app.env", Env: "APP_ENV", Default: "local", Type: "Env"}},
		{"App.Key", Field{Path: "app.key", Env: "APP_KEY", Secret: true, Type: "[]byte"}},
		{"App.Locale", Field{Path: "app.lang", Env: "APP_LANG", Default: "en", Type: "string"}},
		{"App.Internal", Field{Path: "app.internal", Type: "string"}},
		{"HTTP.MaxHeaderBytes", Field{Path: "http.max_header_bytes", Env: "HTTP_MAX_HEADER_BYTES", Default: "1048576", Type: "int"}},
		{"HTTP.TrustedProxies", Field{Path: "http.trusted_proxies", Env: "HTTP_TRUSTED_PROXIES", Default: "10.0.0.0/8,127.0.0.1/32", Type: "[]netip.Prefix"}},
		{"Database.DSN", Field{Path: "database.dsn", Env: "DATABASE_URL", Default: "sqlite:app.db", Secret: true, Type: "string"}},
		{"Queue.Weights", Field{Path: "queue.weights", Env: "QUEUE_WEIGHTS", Type: "map[string]int"}},
		{"Billing.Minimum", Field{Path: "billing.minimum", Env: "BILLING_MINIMUM", Default: "0.50", Type: "Money"}},
	}
	for _, c := range cases {
		got, ok := byName[c.name]
		if !ok {
			t.Errorf("%s is missing", c.name)
			continue
		}
		if got.Path != c.want.Path {
			t.Errorf("%s path = %q, want %q", c.name, got.Path, c.want.Path)
		}
		if got.Env != c.want.Env {
			t.Errorf("%s env = %q, want %q", c.name, got.Env, c.want.Env)
		}
		if got.Default != c.want.Default {
			t.Errorf("%s default = %q, want %q", c.name, got.Default, c.want.Default)
		}
		if got.Secret != c.want.Secret {
			t.Errorf("%s secret = %v, want %v", c.name, got.Secret, c.want.Secret)
		}
		if got.Type != c.want.Type {
			t.Errorf("%s type = %q, want %q", c.name, got.Type, c.want.Type)
		}
		if got.Doc == "" {
			t.Errorf("%s has no doc", c.name)
		}
	}
}

// TestParsers checks that each type picked the parser it should have, since
// that is the one decision the generator makes that a reader cannot check by
// eye.
func TestParsers(t *testing.T) {
	want := map[string]string{
		"App.Name":              "config.String",
		"App.Env":               "config.String",
		"App.Debug":             "config.Bool",
		"App.Key":               "config.Bytes",
		"HTTP.Addr":             "config.AddrPort",
		"HTTP.BindTo":           "config.Addr",
		"HTTP.ReadTimeout":      "config.Duration",
		"HTTP.TrustedProxies":   "config.Slice(config.Prefix)",
		"HTTP.Timeouts":         "config.Map(config.Duration[time.Duration])",
		"HTTP.Origins":          "config.Slice(config.String[string])",
		"Log.Level":             "config.Level",
		"Log.Sample":            "config.Float",
		"Log.Fields":            "config.Map(config.String[string])",
		"Database.Migrated":     "config.Time",
		"Cache.MaxBytes":        "config.Config",
		"Cache.Shards":          "config.Uint",
		"Queue.Retries":         "config.Int",
		"Queue.Weights":         "config.Map(config.Int[int])",
		"Mail.Port":             "config.Uint",
		"Billing.Minimum":       "config.Text",
		"Billing.Rate":          "config.Float",
		"Database.MaxIdleConns": "config.Int",
	}
	shows := map[string]string{
		"App.Name":            "config.Show",
		"HTTP.TrustedProxies": "config.ShowSlice",
		"Log.Fields":          "config.ShowMap",
	}

	for _, f := range plan(t).Fields {
		if w, ok := want[f.Name]; ok && f.Parse != w {
			t.Errorf("%s parses with %s, want %s", f.Name, f.Parse, w)
		}
		if w, ok := shows[f.Name]; ok && f.Show != w {
			t.Errorf("%s shows with %s, want %s", f.Name, f.Show, w)
		}
	}
}

// TestSecrets checks that Redact has something to do for every secret and
// nothing to do for anything else, since a missed one is a leak.
func TestSecrets(t *testing.T) {
	zeros := map[string]string{
		"App.Key":       "nil",
		"Database.DSN":  "config.Redacted",
		"Mail.Password": "config.Redacted",
		"Billing.Key":   "nil",
	}
	seen := 0
	for _, f := range plan(t).Fields {
		if !f.Secret {
			if f.Zero != "" {
				t.Errorf("%s is not secret and has a zero of %q", f.Name, f.Zero)
			}
			continue
		}
		seen++
		w, ok := zeros[f.Name]
		if !ok {
			t.Errorf("%s is secret and the test does not know about it", f.Name)
			continue
		}
		if f.Zero != w {
			t.Errorf("%s redacts to %s, want %s", f.Name, f.Zero, w)
		}
	}
	if seen != len(zeros) {
		t.Errorf("found %d secrets, want %d", seen, len(zeros))
	}
}

// TestClones checks that every field a struct copy would share is copied
// properly, since Redact writes over some of them.
func TestClones(t *testing.T) {
	want := map[string]string{
		"App.Key":             "slices.Clone",
		"HTTP.TrustedProxies": "slices.Clone",
		"HTTP.Timeouts":       "maps.Clone",
		"HTTP.Origins":        "slices.Clone",
		"Log.Fields":          "maps.Clone",
		"Database.Replicas":   "slices.Clone",
		"Queue.Queues":        "slices.Clone",
		"Queue.Weights":       "maps.Clone",
		"Billing.Key":         "slices.Clone",
	}
	seen := 0
	for _, f := range plan(t).Fields {
		if !f.IsCopy {
			continue
		}
		seen++
		if w := want[f.Name]; f.Clone != w {
			t.Errorf("%s clones with %q, want %q", f.Name, f.Clone, w)
		}
	}
	if seen != len(want) {
		t.Errorf("found %d fields needing a clone, want %d", seen, len(want))
	}
}

// TestImports checks that the generated file imports what it writes and
// nothing else, because an unused import does not compile and a missing one
// does not either.
func TestImports(t *testing.T) {
	want := []importLine{
		{Path: "maps"},
		{Path: "slices"},
		{Path: "time"},
		{Path: "github.com/go-mizu/mizu/config"},
	}
	got := plan(t).Imports()
	if len(got) != len(want) {
		t.Fatalf("imports = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("import %d = %v, want %v", i, got[i], want[i])
		}
	}
}

// TestDirOf covers the shapes a package comes in, since where a generated file
// goes depends on it and two of them only turn up outside a module.
func TestDirOf(t *testing.T) {
	cases := []struct {
		pkg  gen.Package
		want string
	}{
		{gen.Package{PkgPath: "example.com/app/config", Module: "example.com/app"}, "config"},
		{gen.Package{PkgPath: "example.com/app", Module: "example.com/app"}, ""},
		{gen.Package{PkgPath: "example.com/app/a/b", Module: "example.com/app"}, "a/b"},
		{gen.Package{PkgPath: "command-line-arguments"}, ""}, // outside a module
	}
	for _, c := range cases {
		if got := dirOf(&c.pkg); got != c.want {
			t.Errorf("dirOf(%q in %q) = %q, want %q", c.pkg.PkgPath, c.pkg.Module, got, c.want)
		}
	}
}

// TestFieldDocsWithoutTypes covers a package that failed badly enough that
// go/types filled nothing in, where there is no way to pair a comment with the
// field it is about.
func TestFieldDocsWithoutTypes(t *testing.T) {
	if got := fieldDocs(&gen.Package{}); len(got) != 0 {
		t.Errorf("got %d docs from a package with no types, want none", len(got))
	}
}

func plan(t *testing.T) *Plan {
	t.Helper()
	pkgs := load(t, "testdata", "./app")
	targets, errs := gen.Scan(pkgs...)
	for _, e := range errs {
		t.Fatal(e)
	}
	for _, tg := range targets {
		if !marked(tg) {
			continue
		}
		p := Analyze(tg, fieldDocs(pkgs[0]))
		for _, err := range p.Errors {
			t.Error(err)
		}
		return p
	}
	t.Fatal("testdata/app has no struct marked as configuration")
	return nil
}

func load(t *testing.T, dir string, patterns ...string) []*gen.Package {
	t.Helper()
	pkgs, err := gen.Load(gen.Config{Dir: dir}, patterns...)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pkgs {
		if err := p.Err(); err != nil {
			t.Fatalf("%s: %v", p.PkgPath, err)
		}
	}
	return pkgs
}

func generate(t *testing.T, dir string, patterns ...string) []gen.File {
	t.Helper()
	files, err := Generate(load(t, dir, patterns...)...)
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// format runs the file through the same formatter the writer uses, so that a
// golden file matches what would land on disk.
func format(t *testing.T, f gen.File) []byte {
	t.Helper()
	dir := t.TempDir()
	w := &gen.Writer{Dir: dir}
	if _, err := w.Write(f); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(f.Path)))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// runGo runs the go command inside the testdata module, which is where the
// generated code is compiled and exercised.
func runGo(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = "testdata"
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// TestGeneratedCode builds and runs the tests that live beside the generated
// file. Those are the round trip: a configuration written to files and the
// environment, read back through the generated loader, and compared with what
// went in.
//
// It shells out because the generated code is in a module of its own, which is
// what keeps a deliberately broken configuration in testdata from breaking the
// build of the generator that reads it.
func TestGeneratedCode(t *testing.T) {
	if testing.Short() {
		t.Skip("compiling the generated code takes a few seconds")
	}
	runGo(t, "test", "./...")
}
