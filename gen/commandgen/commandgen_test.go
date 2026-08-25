package commandgen

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
const goldenPath = "testdata/app/commands_gen.go"

func TestGenerate(t *testing.T) {
	files := generate(t, "testdata", "./app")
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if got := files[0].Path; got != "app/commands_gen.go" {
		t.Errorf("path = %q, want app/commands_gen.go", got)
	}

	golden.Assert(t, format(t, files[0]), golden.At(goldenPath))
}

// TestCommands checks the details that are hard to see in a golden file: that
// every command is there, in declaration order, with what the marker said about
// it.
func TestCommands(t *testing.T) {
	p := plan(t)

	var names []string
	for _, cmd := range p.Commands {
		names = append(names, cmd.Name)
	}
	want := []string{"users:prune", "db:wipe", "serve", "deploy"}
	if strings.Join(names, " ") != strings.Join(want, " ") {
		t.Fatalf("commands = %v, want %v", names, want)
	}

	byName := map[string]Command{}
	for _, cmd := range p.Commands {
		byName[cmd.Name] = cmd
	}

	// The doc comment on the struct is the description when the marker has no
	// desc argument, without the type's own name in front of it, and the marker
	// wins when it has one.
	if got, want := byName["users:prune"].Desc, "Deletes users who never verified their email"; got != want {
		t.Errorf("users:prune desc = %q, want %q", got, want)
	}
	if got, want := byName["db:wipe"].Desc, "Drop every table and start again"; got != want {
		t.Errorf("db:wipe desc = %q, want %q", got, want)
	}
	if !strings.HasPrefix(byName["db:wipe"].Long, "Every table goes") {
		t.Errorf("db:wipe long = %q", byName["db:wipe"].Long)
	}
	if !byName["db:wipe"].Hidden {
		t.Error("db:wipe is not hidden")
	}
	if byName["serve"].Hidden {
		t.Error("serve is hidden")
	}
}

// TestFlags checks the tags a flag is written with, one flag per rule.
func TestFlags(t *testing.T) {
	flags := map[string]Flag{}
	for _, cmd := range plan(t).Commands {
		for _, f := range cmd.Flags {
			flags[cmd.Name+" --"+f.Name] = f
		}
	}

	cases := []struct {
		key  string
		want Flag
	}{
		{"users:prune --days", Flag{Field: "Days", Short: 'd', Default: "30", Desc: "How long unverified is too long"}},
		{"users:prune --dry-run", Flag{Field: "DryRun", Desc: "Say what would go and delete nothing"}},
		{"users:prune --wait", Flag{Field: "Wait", Env: "MIZU_PRUNE_WAIT", Default: "5s", Desc: "How long to wait between batches"}},
		{"db:wipe --url", Flag{Field: "URL", Env: "DATABASE_URL", Required: true, Desc: "Which database, since this one has no safe default"}},
		{"db:wipe --legacy", Flag{Field: "Legacy", Hidden: true, Desc: "Kept for the build that still passes it"}},
		{"serve --header", Flag{Field: "Header", Short: 'H', Desc: "Headers to add to every response"}},
	}
	for _, c := range cases {
		got, ok := flags[c.key]
		if !ok {
			t.Errorf("%s is missing", c.key)
			continue
		}
		got.Value, got.Name = "", ""
		if got != c.want {
			t.Errorf("%s = %+v, want %+v", c.key, got, c.want)
		}
	}

	// An empty flag name comes from the field, which is the tag most fields
	// carry, so it is worth naming the rule as well as testing one of them.
	if _, ok := flags["users:prune --dry-run"]; !ok {
		t.Error("DryRun did not become --dry-run")
	}
}

// TestArgs checks the order arguments end up in and what each of them requires,
// since the order is the one thing about an argument nobody can see from its
// tag alone.
func TestArgs(t *testing.T) {
	var args []Arg
	for _, cmd := range plan(t).Commands {
		if cmd.Name == "deploy" {
			args = cmd.Args
		}
	}
	want := []Arg{
		{Field: "Target", Name: "target", Required: true, Desc: "Where to send it"},
		{Field: "Ref", Name: "ref", Default: "HEAD", Desc: "Which build, or the last one"},
		{Field: "Services", Name: "services", Rest: true, Desc: "The services to send, or all of them"},
	}
	if len(args) != len(want) {
		t.Fatalf("got %d arguments, want %d", len(args), len(want))
	}
	for i := range args {
		got := args[i]
		got.Value, got.at, got.list = "", want[i].at, want[i].list
		if got != want[i] {
			t.Errorf("argument %d = %+v, want %+v", i, got, want[i])
		}
	}
}

// TestValues checks that each type picked the constructor it should have, since
// that is the one decision the generator makes that a reader cannot check by
// eye.
func TestValues(t *testing.T) {
	want := map[string]string{
		"users:prune Tenant": "console.String(&c.Tenant)",
		"users:prune Days":   "console.Int(&c.Days)",
		"users:prune DryRun": "console.Bool(&c.DryRun)",
		"users:prune Wait":   "console.Duration(&c.Wait)",
		"users:prune Format": `console.Enum(&c.Format, "text", "json")`,
		"db:wipe Loud":       "console.Count(&c.Loud)",
		"serve Bind":         "console.Text(&c.Bind)",
		"serve Port":         "console.Uint(&c.Port)",
		"serve Sample":       "console.Float(&c.Sample)",
		"serve Built":        "console.Time(&c.Built)",
		"serve Origins":      `console.Strings(&c.Origins, ",")`,
		"serve Redirect":     `console.Slice(&c.Redirect, console.ParseUint, ",")`,
		"serve Header":       "console.KeyValues(&c.Header)",
		"serve Include":      `console.Strings(&c.Include, "")`,
		"deploy Target":      `console.Enum(&c.Target, "staging", "production")`,
		"deploy Services":    `console.Strings(&c.Services, "")`,
	}

	got := map[string]string{}
	for _, cmd := range plan(t).Commands {
		for _, f := range cmd.Flags {
			got[cmd.Name+" "+f.Field] = f.Value
		}
		for _, a := range cmd.Args {
			got[cmd.Name+" "+a.Field] = a.Value
		}
	}
	for key, w := range want {
		if got[key] != w {
			t.Errorf("%s is %s, want %s", key, got[key], w)
		}
	}
}

// TestUntaggedFieldIsSkipped checks that a field with no flag and no arg tag
// stays out of the command line, since that is how a command holds a field of
// its own.
func TestUntaggedFieldIsSkipped(t *testing.T) {
	for _, cmd := range plan(t).Commands {
		for _, f := range cmd.Flags {
			if f.Field == "pruned" {
				t.Error("an untagged field became a flag")
			}
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
		{gen.Package{PkgPath: "example.com/app/commands", Module: "example.com/app"}, "commands"},
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

// TestConsoleName covers the one package in the world that has to alias the
// import, which is the console package itself.
func TestConsoleName(t *testing.T) {
	if got := consoleName(&gen.Package{Name: "commands"}); got != "console" {
		t.Errorf("consoleName = %q, want console", got)
	}
	if got := consoleName(&gen.Package{Name: "console"}); got != "mizuconsole" {
		t.Errorf("consoleName = %q, want mizuconsole", got)
	}
}

// TestSuggestName covers the spelling offered by the error about a marker with
// no name, which is the only place a command name is ever guessed.
func TestSuggestName(t *testing.T) {
	cases := map[string]string{
		"UsersPrune":    "users:prune",
		"Serve":         "serve",
		"DbWipeAll":     "db:wipeall",
		"ParseHTTPBody": "parse:httpbody",
	}
	for in, want := range cases {
		if got := suggestName(in); got != want {
			t.Errorf("suggestName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestKebab covers the names a flag gets when its tag does not say, including
// the acronym case, which is the one a split by capital letters gets wrong.
func TestKebab(t *testing.T) {
	cases := map[string]string{
		"DryRun":        "dry-run",
		"URL":           "url",
		"MaxOpenConns":  "max-open-conns",
		"ParseHTTPBody": "parse-http-body",
		"A":             "a",
		"":              "",
	}
	for in, want := range cases {
		if got := kebab(in); got != want {
			t.Errorf("kebab(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestOneLine covers what a doc comment becomes in a help listing.
func TestOneLine(t *testing.T) {
	cases := map[string]string{
		"":                            "",
		"How long is too long.":       "How long is too long",
		"One line\nwrapped onto two.": "One line wrapped onto two",
		"The first.\n\nAnd the rest.": "The first",
	}
	for in, want := range cases {
		if got := oneLine(in); got != want {
			t.Errorf("oneLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestUnprefix covers the doc comment a command borrows for its description,
// which is written the way Go writes one and read in a list that has the name
// beside it already.
func TestUnprefix(t *testing.T) {
	cases := []struct {
		doc, typeName, want string
	}{
		{"Serve runs the server", "Serve", "Runs the server"},
		{"Drop every table", "DbWipe", "Drop every table"},
		{"", "Serve", ""},
		{"Serve", "Serve", "Serve"},
		{"Serve ", "Serve", "Serve "},
	}
	for _, c := range cases {
		if got := unprefix(c.doc, c.typeName); got != c.want {
			t.Errorf("unprefix(%q, %q) = %q, want %q", c.doc, c.typeName, got, c.want)
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
	p := Analyze(pkgs[0], targets)
	for _, err := range p.Errors {
		t.Error(err)
	}
	return p
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

// TestGeneratedCode builds and runs the tests that live beside the generated
// file. Those are the round trip: a command line parsed by the generated Spec
// into the struct's own fields, and compared with what went in.
//
// It shells out because the generated code is in a module of its own, which is
// what keeps a deliberately broken command in testdata from breaking the build
// of the generator that reads it.
func TestGeneratedCode(t *testing.T) {
	if testing.Short() {
		t.Skip("compiling the generated code takes a few seconds")
	}
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = "testdata"
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test ./...: %v\n%s", err, out)
	}
}
