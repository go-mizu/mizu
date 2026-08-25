package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/console/consoletest"
	"github.com/go-mizu/mizu/gen"
)

// project copies one of the generators' test modules somewhere writable and
// makes it the working directory, which is where a person runs mizu gen from.
//
// The copy is no longer beside the module it replaces, so the replace
// directive is rewritten to say where that really is. Everything else about
// the module is left alone, since the point is to run against the tree the
// generator's own tests use rather than a second one that drifts from it.
func project(tb testing.TB, testdata string) string {
	tb.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		tb.Fatal(err)
	}
	dir := tb.TempDir()
	if err := os.CopyFS(dir, os.DirFS(filepath.Join(root, testdata))); err != nil {
		tb.Fatal(err)
	}

	name := filepath.Join(dir, "go.mod")
	mod, err := os.ReadFile(name)
	if err != nil {
		tb.Fatal(err)
	}
	fixed := strings.Replace(string(mod), "=> ../../..", "=> "+filepath.ToSlash(root), 1)
	if fixed == string(mod) {
		tb.Fatalf("%s has no replace directive to point at %s", name, root)
	}
	if err := os.WriteFile(name, []byte(fixed), 0o644); err != nil {
		tb.Fatal(err)
	}

	tb.Chdir(dir)
	return dir
}

const (
	commands = "gen/commandgen/testdata"
	configs  = "gen/configgen/testdata"
)

// runGen runs one command line. The command is built fresh every time because
// it holds what its flags parsed into.
func runGen(t *testing.T, argv ...string) *consoletest.Result {
	t.Helper()
	return consoletest.Run(t, &Gen{}, consoletest.Args(argv...))
}

// touch makes a file differ from what the generator would write, without
// making it stop compiling.
func touch(t *testing.T, name string) {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, append(data, "\n// added by hand\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestGenCheckOnATreeThatIsUpToDate(t *testing.T) {
	project(t, commands)

	r := runGen(t, "--check", "./...").AssertSuccess()
	r.AssertOutputContains("app/commands_gen.go")
	r.AssertOutputContains("up to date")
	r.AssertErrorContains("1 generated file up to date")
}

// The file the generator writes is named by where it belongs in the module, so
// the answer does not depend on which directory the command was run from.
func TestGenCheckFromASubdirectory(t *testing.T) {
	dir := project(t, commands)
	t.Chdir(filepath.Join(dir, "app"))

	runGen(t, "--check", "./...").AssertSuccess().AssertOutputContains("app/commands_gen.go")
}

func TestGenCheckOnAStaleFile(t *testing.T) {
	dir := project(t, commands)
	name := filepath.Join(dir, "app", "commands_gen.go")
	before := read(t, name)
	touch(t, name)

	r := runGen(t, "--check", "./...")
	err := r.AssertFailure()
	if !strings.Contains(err.Error(), "out of date") {
		t.Errorf("the error is %q, want it to say the file is out of date", err)
	}
	if !strings.Contains(err.Error(), "mizu gen") {
		t.Errorf("the error is %q, want it to name what to run", err)
	}
	r.AssertOutputContains("stale from line")

	// The whole promise of --check is that it writes nothing.
	if got := read(t, name); got != before+"\n// added by hand\n" {
		t.Error("--check wrote to the file it was reporting on")
	}
}

func TestGenCheckOnAMissingFile(t *testing.T) {
	dir := project(t, commands)
	name := filepath.Join(dir, "app", "commands_gen.go")
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}

	runGen(t, "--check", "./...").AssertFailure()
	if _, err := os.Stat(name); !os.IsNotExist(err) {
		t.Error("--check created the file it was reporting on")
	}
}

func TestGenWrites(t *testing.T) {
	dir := project(t, commands)
	name := filepath.Join(dir, "app", "commands_gen.go")
	want := read(t, name)
	touch(t, name)

	r := runGen(t, "./...").AssertSuccess()
	r.AssertOutputContains("updated")
	r.AssertErrorContains("Wrote 1 file")

	if got := read(t, name); got != want {
		t.Error("the file the generator wrote is not the one that was checked in")
	}
}

// A file that has not changed is not rewritten, and the report says so rather
// than claiming work that did not happen.
func TestGenLeavesAnUpToDateFileAlone(t *testing.T) {
	project(t, commands)

	r := runGen(t, "./...").AssertSuccess()
	r.AssertOutputContains("unchanged")
	r.AssertErrorContains("already up to date")
}

func TestGenCreatesAMissingFile(t *testing.T) {
	dir := project(t, commands)
	name := filepath.Join(dir, "app", "commands_gen.go")
	want := read(t, name)
	if err := os.Remove(name); err != nil {
		t.Fatal(err)
	}

	runGen(t, "./...").AssertSuccess().AssertOutputContains("created")
	if got := read(t, name); got != want {
		t.Error("the file the generator created is not the one that was checked in")
	}
}

// The bootstrap case. Renaming a field breaks the generated file that mentions
// it, which breaks the package, which is the package the generator has to read
// to write the fix. It has to be one step and not two.
func TestGenReadsAPackageBrokenByItsOwnOutput(t *testing.T) {
	dir := project(t, commands)
	src := filepath.Join(dir, "app", "commands.go")
	rename(t, src, "Tenant", "Owner")

	name := filepath.Join(dir, "app", "commands_gen.go")
	if strings.Contains(read(t, name), "Owner") {
		t.Fatal("the generated file already mentions the new name")
	}

	runGen(t, "./...").AssertSuccess().AssertOutputContains("updated")

	got := read(t, name)
	if !strings.Contains(got, "c.Owner") {
		t.Errorf("the generated file does not use the new field name:\n%s", got)
	}
	if strings.Contains(got, "c.Tenant") {
		t.Error("the generated file still uses the old field name")
	}
}

// A package that does not compile for a reason of its own is reported. The
// stubbing is for stale generated output and nothing else, so an error the
// second load still finds is the one the person needs to see.
func TestGenReportsARealCompileError(t *testing.T) {
	dir := project(t, commands)
	src := filepath.Join(dir, "app", "commands.go")
	rename(t, src, "net/netip", "net/nope")

	err := runGen(t, "./...").AssertFailure()
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("the error is %q, want it to name what is missing", err)
	}
}

func rename(t *testing.T, name, from, to string) {
	t.Helper()
	src := read(t, name)
	if !strings.Contains(src, from) {
		t.Fatalf("%s does not contain %q", name, from)
	}
	if err := os.WriteFile(name, []byte(strings.ReplaceAll(src, from, to)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGenConfig(t *testing.T) {
	dir := project(t, configs)
	name := filepath.Join(dir, "app", "config_gen.go")
	want := read(t, name)
	touch(t, name)

	runGen(t, "./...").AssertSuccess().AssertErrorContains("Wrote 1 file")
	if got := read(t, name); got != want {
		t.Error("the file the generator wrote is not the one that was checked in")
	}
}

// Each generator is also a command of its own, so a project that only has one
// kind of marker does not pay for the other.
func TestGenOneGeneratorAtATime(t *testing.T) {
	dir := project(t, commands)
	name := filepath.Join(dir, "app", "commands_gen.go")
	touch(t, name)

	consoletest.Run(t, &Gen{only: "config"}, consoletest.Args("./...")).
		AssertSuccess().
		AssertErrorContains("Nothing asked to be generated")

	consoletest.Run(t, &Gen{only: "command"}, consoletest.Args("./...")).
		AssertSuccess().
		AssertOutputContains("app/commands_gen.go")
}

// With no packages it runs over everything under the current directory, which
// is what a project runs and what the development loop repeats.
func TestGenDefaultsToEverything(t *testing.T) {
	project(t, commands)
	runGen(t, "--check").AssertSuccess().AssertOutputContains("app/commands_gen.go")
}

func TestGenNamesOnePackage(t *testing.T) {
	project(t, commands)
	runGen(t, "--check", "./app").AssertSuccess().AssertOutputContains("app/commands_gen.go")
}

func TestGenOnAPackageWithNoMarkers(t *testing.T) {
	project(t, commands)
	r := runGen(t, "--check", "./broken").AssertSuccess()
	r.AssertErrorContains("Nothing asked to be generated")
	r.AssertNoOutput()
}

// A script reading the output wants a list either way, so JSON mode says the
// empty one rather than saying nothing.
func TestGenJSONWithNothingToGenerate(t *testing.T) {
	project(t, commands)

	r := consoletest.Run(t, &Gen{},
		consoletest.Args("--check", "./broken"),
		consoletest.With(console.Options{JSON: true}),
	).AssertSuccess()

	if got := strings.TrimSpace(r.Stdout()); got != "[]" {
		t.Errorf("the output is %q, want an empty list", got)
	}
	r.AssertNoErrorOutput()
}

func TestGenOnAPatternThatMatchesNothing(t *testing.T) {
	project(t, commands)
	err := runGen(t, "--check", "./nowhere/...").AssertFailure()
	if !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("the error is %q, want it to name the pattern", err)
	}
}

// An empty module is not an error and not a table either. Saying nothing at
// all would read as though the run had worked and written something.
func TestGenOnAModuleWithNoPackages(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module empty\n\ngo 1.27\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	err := runGen(t, "--check", "./...").AssertFailure()
	if !strings.Contains(err.Error(), "no packages") {
		t.Errorf("the error is %q, want it to say nothing matched", err)
	}
}

// The go command failing is different from a package failing to compile, and
// it is the one case where there is nothing at all to work from.
func TestGenWhenTheGoCommandFails(t *testing.T) {
	dir := project(t, commands)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("this is not a go.mod\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := runGen(t, "--check", "./...").AssertFailure()
	if !strings.Contains(err.Error(), "go list") {
		t.Errorf("the error is %q, want it to say what was run", err)
	}
}

// What a generator refuses reaches the person who ran the command, and no
// file is written for the packages that were fine either. Half a generated
// tree is a build that fails somewhere else for a reason nobody can see.
func TestGenReportsWhatAGeneratorRefused(t *testing.T) {
	dir := project(t, configs)
	src := filepath.Join(dir, "app", "second.go")
	const second = `package app

//mizu:config
type Second struct {
	Port int
}
`
	if err := os.WriteFile(src, []byte(second), 0o644); err != nil {
		t.Fatal(err)
	}

	want := read(t, filepath.Join(dir, "app", "config_gen.go"))
	err := runGen(t, "./...").AssertFailure()
	if !strings.Contains(err.Error(), "configtest/app") {
		t.Errorf("the error is %q, want it to name the package", err)
	}
	if got := read(t, filepath.Join(dir, "app", "config_gen.go")); got != want {
		t.Error("a run that failed still wrote a file")
	}
}

// The writer refuses to overwrite a file that does not carry a generated
// header, because the next run would take it for hand-written work.
func TestGenWillNotOverwriteAHandWrittenFile(t *testing.T) {
	dir := project(t, commands)
	name := filepath.Join(dir, "app", "commands_gen.go")
	if err := os.WriteFile(name, []byte("package app\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := runGen(t, "./...").AssertFailure()
	if !strings.Contains(err.Error(), "hand-written") {
		t.Errorf("the error is %q, want it to say why it stopped", err)
	}
	if got := read(t, name); got != "package app\n" {
		t.Error("the file was overwritten anyway")
	}
}

// A run that was interrupted stops before it writes anything, since a tree
// half generated is worse than one not generated at all.
func TestGenWritesNothingAfterAnInterrupt(t *testing.T) {
	dir := project(t, commands)
	name := filepath.Join(dir, "app", "commands_gen.go")
	touch(t, name)
	before := read(t, name)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := consoletest.Run(t, &Gen{},
		consoletest.Args("./..."),
		consoletest.Context(ctx),
	).AssertFailure()
	if !errors.Is(err, context.Canceled) {
		t.Errorf("the error is %v, want it to be a cancellation", err)
	}
	if got := read(t, name); got != before {
		t.Error("a cancelled run still wrote the file")
	}
}

func TestGenJSON(t *testing.T) {
	project(t, commands)

	r := consoletest.Run(t, &Gen{},
		consoletest.Args("--check", "./..."),
		consoletest.With(console.Options{JSON: true}),
	).AssertSuccess()

	var rows []struct {
		File   string `json:"file"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(r.Stdout()), &rows); err != nil {
		t.Fatalf("the output is not JSON: %v\n%s", err, r.Stdout())
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1:\n%s", len(rows), r.Stdout())
	}
	if rows[0].File != "app/commands_gen.go" || rows[0].Status != "up to date" {
		t.Errorf("row is %+v", rows[0])
	}
	r.AssertNoErrorOutput()
}

// A run that fails still writes the table, because the list of what is out of
// date is the answer somebody wants from a failed build.
func TestGenJSONWhenSomethingIsStale(t *testing.T) {
	dir := project(t, commands)
	touch(t, filepath.Join(dir, "app", "commands_gen.go"))

	r := consoletest.Run(t, &Gen{},
		consoletest.Args("--check", "./..."),
		consoletest.With(console.Options{JSON: true}),
	)
	r.AssertFailure()

	var rows []map[string]string
	if err := json.Unmarshal([]byte(r.Stdout()), &rows); err != nil {
		t.Fatalf("the output is not JSON: %v\n%s", err, r.Stdout())
	}
	if len(rows) != 1 || !strings.HasPrefix(rows[0]["status"], "stale") {
		t.Errorf("rows are %v", rows)
	}
}

func TestGenSpecs(t *testing.T) {
	tests := []struct {
		cmd  *Gen
		name string
		desc string
	}{
		{&Gen{}, "gen", "Run every generator over the packages"},
		{&Gen{only: "command"}, "gen:command", generators[0].desc},
		{&Gen{only: "config"}, "gen:config", generators[1].desc},
	}
	for _, tt := range tests {
		spec := tt.cmd.Spec()
		if spec.Name != tt.name {
			t.Errorf("the command is called %q, want %q", spec.Name, tt.name)
		}
		if spec.Desc != tt.desc {
			t.Errorf("%s is described as %q, want %q", spec.Name, spec.Desc, tt.desc)
		}
		if spec.Long == "" {
			t.Errorf("%s has no long help", spec.Name)
		}
	}
}

// A name that is not in the table is a mistake in the program rather than
// something anybody typed, so it stops the program where it is made.
func TestGenPanicsOnAGeneratorThatDoesNotExist(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("an unknown generator did not panic")
		}
	}()
	(&Gen{only: "nothing"}).Spec()
}

func TestGenIsInTheApp(t *testing.T) {
	out, _, code := start(t, "--help")
	if code != console.CodeOK {
		t.Fatalf("exited %d", code)
	}
	for _, want := range []string{"gen", "gen:command", "gen:config"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the help does not list %q:\n%s", want, out)
		}
	}
}

func TestModuleRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "src", "proj")

	tests := []struct {
		name string
		pkgs []*gen.Package
		want string
		err  string
	}{
		{
			name: "the module's own package",
			pkgs: []*gen.Package{{PkgPath: "proj", Module: "proj", Dir: root}},
			want: root,
		},
		{
			name: "a package one level down",
			pkgs: []*gen.Package{{PkgPath: "proj/app", Module: "proj", Dir: filepath.Join(root, "app")}},
			want: root,
		},
		{
			name: "a package three levels down",
			pkgs: []*gen.Package{{PkgPath: "proj/a/b/c", Module: "proj", Dir: filepath.Join(root, "a", "b", "c")}},
			want: root,
		},
		{
			name: "several packages agreeing",
			pkgs: []*gen.Package{
				{PkgPath: "proj/app", Module: "proj", Dir: filepath.Join(root, "app")},
				{PkgPath: "proj", Module: "proj", Dir: root},
				{PkgPath: "proj/a/b", Module: "proj", Dir: filepath.Join(root, "a", "b")},
			},
			want: root,
		},
		{
			name: "nothing matched",
			err:  "no packages",
		},
		{
			name: "outside a module",
			pkgs: []*gen.Package{{PkgPath: "lonely", Dir: root}},
			err:  "not in a module",
		},
		{
			name: "two modules at once",
			pkgs: []*gen.Package{
				{PkgPath: "proj", Module: "proj", Dir: root},
				{PkgPath: "other", Module: "other", Dir: filepath.Join(root, "..", "other")},
			},
			err: "two modules",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := moduleRoot(tt.pkgs)
			switch {
			case tt.err != "":
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("error is %v, want it to say %q", err, tt.err)
				}
			case err != nil:
				t.Fatal(err)
			case got != tt.want:
				t.Errorf("moduleRoot() = %q, want %q", got, tt.want)
			}
		})
	}
}

// BenchmarkGenCheck is what a project pays to find out whether its generated
// files are in step, which is a thing a pre-commit hook and a CI job both run.
//
// Most of it is the go command and the type checker rather than anything here,
// and that is the point: it says what the whole step costs. The parts of it
// this package owns are measured separately below.
func BenchmarkGenCheck(b *testing.B) {
	project(b, commands)
	c := console.New(nil, io.Discard, io.Discard, console.Options{})

	b.ReportAllocs()
	for b.Loop() {
		if err := (&Gen{Check: true, Packages: []string{"./..."}}).Run(b.Context(), c); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenReport(b *testing.B) {
	c := console.New(nil, io.Discard, io.Discard, console.Options{})
	results := []gen.Result{
		{Path: "app/commands_gen.go", Status: gen.Unchanged},
		{Path: "app/config_gen.go", Status: gen.Updated, Line: 42},
		{Path: "store/columns_gen.go", Status: gen.Created},
	}
	cmd := &Gen{Check: true}

	b.ReportAllocs()
	for b.Loop() {
		if err := cmd.report(c, results); err == nil {
			b.Fatal("a stale file was reported as a success")
		}
	}
}

func BenchmarkModuleRoot(b *testing.B) {
	pkgs := []*gen.Package{
		{PkgPath: "proj", Module: "proj", Dir: filepath.Join("src", "proj")},
		{PkgPath: "proj/app", Module: "proj", Dir: filepath.Join("src", "proj", "app")},
		{PkgPath: "proj/a/b/c", Module: "proj", Dir: filepath.Join("src", "proj", "a", "b", "c")},
	}

	b.ReportAllocs()
	for b.Loop() {
		if _, err := moduleRoot(pkgs); err != nil {
			b.Fatal(err)
		}
	}
}
