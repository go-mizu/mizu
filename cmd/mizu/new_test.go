package main

import (
	"context"
	"encoding/json"
	"errors"
	"go/format"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/console/consoletest"
)

// wanted is what every preset writes.
//
// The difference between the presets is what is in the files rather than which
// ones there are, and a project missing its .gitattributes or its README is
// missing something somebody would have to write by hand.
var wanted = []string{".gitattributes", ".gitignore", "AGENTS.md", "README.md", "go.mod", "main.go", "main_test.go"}

// place is a path for a project that is not there yet.
func place(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}

func runNew(t *testing.T, argv []string, opts ...consoletest.Option) *consoletest.Result {
	t.Helper()
	return consoletest.Run(t, &New{}, append([]consoletest.Option{consoletest.Args(argv...)}, opts...)...)
}

// tree is the files directly under dir, sorted, with directories left out.
//
// A test compares against the whole list rather than checking that the files
// it cares about are there, because a file the command was not asked for is as
// much a mistake as one it forgot.
func tree(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	return names
}

// root is this checkout, which is what the generated projects are pointed at.
func root(t testing.TB) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNewWritesAnAPIProject(t *testing.T) {
	dir := place(t, "blog")

	res := runNew(t, []string{dir, "--preset=api"}).AssertSuccess()

	if got := tree(t, dir); !slices.Equal(got, wanted) {
		t.Errorf("wrote %v, want %v", got, wanted)
	}
	if got, want := read(t, filepath.Join(dir, "go.mod")), "module blog\n\ngo "+newGo+"\n"; got != want {
		t.Errorf("go.mod is %q, want %q", got, want)
	}
	if main := read(t, filepath.Join(dir, "main.go")); !strings.Contains(main, "// Command blog is an HTTP service.") {
		t.Errorf("main.go does not name the project:\n%s", main)
	}
	res.AssertOutputContains(filepath.Join(dir, "main.go"))
	res.AssertErrorContains("go mod tidy")
	res.AssertErrorContains("go run .")
}

func TestNewWritesACLIProject(t *testing.T) {
	dir := place(t, "greeter")

	res := runNew(t, []string{dir, "--preset=cli"}).AssertSuccess()

	if got := tree(t, dir); !slices.Equal(got, wanted) {
		t.Errorf("wrote %v, want %v", got, wanted)
	}
	// The two facts are checked apart from each other because the field is in
	// a struct literal, and gofmt moves the value along when a longer field
	// name lands next to it.
	main := read(t, filepath.Join(dir, "main.go"))
	if !strings.Contains(main, "console.App{") || !strings.Contains(main, `"greeter"`) {
		t.Errorf("main.go does not name the app:\n%s", main)
	}
	res.AssertErrorContains("go run . greet ada")
}

// The name is the directory, so the same command in a different place writes a
// differently named project, and everything the templates say about it follows.
func TestNewNamesTheProjectAfterTheDirectory(t *testing.T) {
	dir := place(t, "my-app")

	runNew(t, []string{dir, "--preset=api"}).AssertSuccess()

	if got := read(t, filepath.Join(dir, "go.mod")); !strings.HasPrefix(got, "module my-app\n") {
		t.Errorf("go.mod is:\n%s", got)
	}
	if got := read(t, filepath.Join(dir, ".gitignore")); !strings.Contains(got, "\n/my-app\n") {
		t.Errorf(".gitignore does not ignore the binary:\n%s", got)
	}
}

func TestNewOnAModulePathOfItsOwn(t *testing.T) {
	dir := place(t, "blog")

	runNew(t, []string{dir, "--preset=api", "--module=github.com/ada/blog"}).AssertSuccess()

	if got := read(t, filepath.Join(dir, "go.mod")); !strings.HasPrefix(got, "module github.com/ada/blog\n") {
		t.Errorf("go.mod is:\n%s", got)
	}
	// The project is still called blog, since that is its name wherever its
	// import path happens to point.
	if main := read(t, filepath.Join(dir, "main.go")); !strings.Contains(main, "Command blog is") {
		t.Errorf("main.go is:\n%s", main)
	}
}

// mizu new . in an empty directory writes a project named after that
// directory, rather than one called ".".
func TestNewIntoTheDirectoryItWasRunIn(t *testing.T) {
	dir := place(t, "blog")
	if err := os.MkdirAll(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	runNew(t, []string{".", "--preset=api"}).AssertSuccess()

	if got := read(t, filepath.Join(dir, "go.mod")); !strings.HasPrefix(got, "module blog\n") {
		t.Errorf("go.mod is:\n%s", got)
	}
}

func TestNewRefusesADirectoryWithSomethingInIt(t *testing.T) {
	dir := place(t, "blog")
	if err := os.MkdirAll(dir, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := runNew(t, []string{dir, "--preset=api"})

	res.AssertFailure()
	res.AssertErrorContains("is not empty")
	if got := tree(t, dir); !slices.Equal(got, []string{"notes.txt"}) {
		t.Errorf("the directory now holds %v", got)
	}
}

// Making the repository first and the project in it second is an ordinary way
// to start, so a directory holding nothing but a .git is one to write into.
func TestNewIntoADirectoryThatIsOnlyAGitRepository(t *testing.T) {
	dir := place(t, "blog")
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o777); err != nil {
		t.Fatal(err)
	}

	runNew(t, []string{dir, "--preset=api"}).AssertSuccess()

	if got := tree(t, dir); !slices.Equal(got, wanted) {
		t.Errorf("wrote %v, want %v", got, wanted)
	}
}

func TestNewWhereAFileIsAlreadyInTheWay(t *testing.T) {
	dir := place(t, "blog")
	if err := os.WriteFile(dir, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := runNew(t, []string{dir, "--preset=api"})

	res.AssertFailure()
	res.AssertErrorContains(dir)
}

func TestNewRefusesANameThatIsNotOne(t *testing.T) {
	dir := place(t, "my project")

	res := runNew(t, []string{dir, "--preset=api"})

	res.AssertExitCode(console.CodeUsage)
	res.AssertErrorContains("is not a name for a project")
	if _, err := os.Stat(dir); err == nil {
		t.Error("the directory was made anyway")
	}
}

func TestNewRefusesAModulePathTheGoCommandHasTakenForItself(t *testing.T) {
	dir := place(t, "tool")

	res := runNew(t, []string{dir, "--preset=api"})

	res.AssertExitCode(console.CodeUsage)
	res.AssertErrorContains("--module")
}

func TestNewRefusesAModulePathThatWouldNotParse(t *testing.T) {
	dir := place(t, "blog")

	res := runNew(t, []string{dir, "--preset=api", "--module=example.com/my blog"})

	res.AssertExitCode(console.CodeUsage)
	res.AssertErrorContains("is not a module path")
}

func TestNewRefusesAPresetThatIsNotThere(t *testing.T) {
	dir := place(t, "blog")

	res := runNew(t, []string{dir, "--preset=web"})

	res.AssertExitCode(console.CodeUsage)
	res.AssertErrorContains("no preset called web")
	res.AssertErrorContains("api, cli")
}

// With no --preset and somebody at the terminal, which one to write is a
// question rather than a guess.
func TestNewAsksWhichPreset(t *testing.T) {
	dir := place(t, "greeter")

	res := runNew(t, []string{dir}, consoletest.Choose("What are you building?", presets[1].desc+" (cli)"))

	res.AssertSuccess()
	res.AssertAsked("What are you building?")
	if main := read(t, filepath.Join(dir, "main.go")); !strings.Contains(main, "console.App") {
		t.Errorf("the cli preset was not written:\n%s", main)
	}
}

// Under --no-interaction there is nobody to ask, so it writes the first preset
// rather than stopping. A script that forgot the flag gets a project.
func TestNewWithNobodyToAsk(t *testing.T) {
	dir := place(t, "blog")

	res := runNew(t, []string{dir}, consoletest.With(console.Options{Interaction: console.InteractionNever}))

	res.AssertSuccess()
	if asked := res.Prompts(); len(asked) != 0 {
		t.Errorf("it asked %v", asked)
	}
	if main := read(t, filepath.Join(dir, "main.go")); !strings.Contains(main, "mizu.New()") {
		t.Errorf("the api preset was not written:\n%s", main)
	}
}

func TestNewJSON(t *testing.T) {
	dir := place(t, "blog")

	res := runNew(t, []string{dir, "--preset=api"}, consoletest.With(console.Options{JSON: true}))

	res.AssertSuccess()
	var got struct {
		Dir    string
		Module string
		Preset string
		Files  []string
	}
	if err := json.Unmarshal([]byte(res.Stdout()), &got); err != nil {
		t.Fatalf("%v, from:\n%s", err, res.Stdout())
	}
	if got.Dir != dir || got.Module != "blog" || got.Preset != "api" {
		t.Errorf("got %+v", got)
	}
	if len(got.Files) != len(wanted) {
		t.Errorf("%d files, want %d: %v", len(got.Files), len(wanted), got.Files)
	}
	// Nothing goes to stderr, because a script reading the JSON did not ask
	// what to do next.
	res.AssertNoErrorOutput()
}

// A run that fails partway leaves nothing behind, so long as the directory was
// not there before it started.
func TestNewTakesBackTheDirectoryItMade(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a read only directory on Windows still takes new files")
	}
	if os.Geteuid() == 0 {
		t.Skip("root writes into a read only directory")
	}
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(parent, 0o755) })
	dir := filepath.Join(parent, "blog")

	runNew(t, []string{dir, "--preset=api"}).AssertFailure()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("the directory is still there: %v", err)
	}
}

// A run that fails in a directory somebody else made leaves it where it is,
// whatever state it got to. Taking away a directory this command did not
// create is not something to do on the strength of a failed write.
func TestNewLeavesADirectoryItDidNotMake(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a read only directory on Windows still takes new files")
	}
	if os.Geteuid() == 0 {
		t.Skip("root writes into a read only directory")
	}
	dir := place(t, "blog")
	if err := os.MkdirAll(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	runNew(t, []string{dir, "--preset=api"}).AssertFailure()

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("the directory was taken away: %v", err)
	}
}

// A run interrupted while it was reading templates stops before it creates
// anything, since a directory holding half a project is worse than no
// directory at all.
func TestNewStopsOnAnInterrupt(t *testing.T) {
	dir := place(t, "blog")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res := runNew(t, []string{dir, "--preset=api"}, consoletest.Context(ctx))

	res.AssertExitCode(console.CodeInterrupted)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("the directory was made anyway")
	}
}

// A template that will not parse is a mistake in this program, and the error
// names the file rather than the line of Go that read it.
func TestATemplateThatWillNotParse(t *testing.T) {
	broken := fstest.MapFS{"template/common/go.mod.tmpl": {Data: []byte("module {{.Module")}}

	_, err := render(broken, presets[0], data{Module: "blog"})

	if err == nil || !strings.Contains(err.Error(), "template/common/go.mod.tmpl") {
		t.Fatalf("the error is %v", err)
	}
}

// A template asking for something the data does not have is a failure rather
// than the words "<no value>" in a file somebody has to notice.
func TestATemplateAskingForSomethingThatIsNotThere(t *testing.T) {
	broken := fstest.MapFS{
		"template/common/go.mod.tmpl": {Data: []byte("module {{.Nothing}}\n")},
		"template/api/main.go.tmpl":   {Data: []byte("package main\n")},
	}

	_, err := render(broken, presets[0], data{Module: "blog"})

	if err == nil || !strings.Contains(err.Error(), "go.mod.tmpl") {
		t.Fatalf("the error is %v", err)
	}
}

// A preset with no templates behind it is a registration nobody finished, and
// the error says which directory is missing.
func TestAPresetWithNoTemplatesBehindIt(t *testing.T) {
	only := fstest.MapFS{"template/common/go.mod.tmpl": {Data: []byte("module {{.Module}}\n")}}

	_, err := render(only, preset{name: "nowhere"}, data{Module: "blog"})

	if err == nil || !strings.Contains(err.Error(), "template/nowhere") {
		t.Fatalf("the error is %v", err)
	}
}

// brokenFS lists a template and then will not hand it over.
type brokenFS struct{ fs.FS }

func (b brokenFS) Open(name string) (fs.File, error) {
	if strings.HasSuffix(name, ".tmpl") {
		return nil, errBroken
	}
	return b.FS.Open(name)
}

// A template that cannot be read is reported the same as one that will not
// parse, rather than coming out as a file with nothing in it.
func TestATemplateThatCannotBeRead(t *testing.T) {
	fsys := brokenFS{fstest.MapFS{"template/common/go.mod.tmpl": {Data: []byte("module blog\n")}}}

	_, err := render(fsys, presets[0], data{Module: "blog"})

	if !errors.Is(err, errBroken) {
		t.Fatalf("the error is %v", err)
	}
}

// A question nobody answered, because the input ended, stops the command
// rather than writing a preset the person did not choose.
func TestNewWhenTheQuestionIsNotAnswered(t *testing.T) {
	c := console.New(strings.NewReader(""), io.Discard, io.Discard, console.Options{Interaction: console.InteractionAlways})

	if _, err := (&New{}).choose(c); err == nil {
		t.Fatal("an unanswered question was taken as an answer")
	}
}

// Every preset writes the same files, and render is where that is decided.
//
// AGENTS.md is not one of them. It is read off the tree once the templates are
// written rather than rendered from one, so it arrives after this step.
func TestEveryPresetWritesTheSameFiles(t *testing.T) {
	want := slices.DeleteFunc(slices.Clone(wanted), func(name string) bool { return name == agentsFile })
	for _, p := range presets {
		files, err := render(templates, p, data{Name: "blog", Module: "blog", Go: newGo})
		if err != nil {
			t.Fatalf("%s: %v", p.name, err)
		}
		names := make([]string, len(files))
		for i, f := range files {
			names[i] = f.Path
		}
		if !slices.Equal(names, want) {
			t.Errorf("%s writes %v, want %v", p.name, names, want)
		}
	}
}

// The templates are not Go files, so gofmt over this repository never reads
// them and a stray indent would travel into every project written from then
// on. Comparing what they render to with what gofmt would make of it is the
// same check, applied where it works.
func TestTheGeneratedGoIsFormatted(t *testing.T) {
	for _, p := range presets {
		files, err := render(templates, p, data{Name: "blog", Module: "blog", Go: newGo})
		if err != nil {
			t.Fatalf("%s: %v", p.name, err)
		}
		for _, f := range files {
			if filepath.Ext(f.Path) != ".go" {
				continue
			}
			want, err := format.Source(f.Data)
			if err != nil {
				t.Errorf("%s/%s does not parse: %v", p.name, f.Path, err)
				continue
			}
			if string(want) != string(f.Data) {
				t.Errorf("%s/%s is not gofmt clean, gofmt would write:\n%s", p.name, f.Path, want)
			}
		}
	}
}

// The go directive in a new project is the one mizu itself needs. Nothing
// keeps the two in step but this, and a project written against an older
// language version than the toolkit it imports does not build.
func TestTheGeneratedGoDirectiveMatchesThisModule(t *testing.T) {
	mod := read(t, filepath.Join(root(t), "go.mod"))

	if want := "\ngo " + newGo + "\n"; !strings.Contains(mod, want) {
		t.Errorf("mizu asks for a different Go than newGo says, which is %q", newGo)
	}
}

// This is the one that matters. Everything above checks what mizu new wrote,
// and this checks that what it wrote is a project: go test ./... passes in it
// with nothing done beyond resolving the module.
func TestTheProjectsPassTheirOwnTests(t *testing.T) {
	if testing.Short() {
		t.Skip("the go command builds mizu once for each preset")
	}
	for _, p := range presets {
		t.Run(p.name, func(t *testing.T) {
			t.Parallel()
			dir := place(t, "blog")
			runNew(t, []string{dir, "--preset=" + p.name}).AssertSuccess()
			resolve(t, dir)

			gocmd(t, dir, "test", "./...")
		})
	}
}

// resolve points a generated project at this checkout.
//
// github.com/go-mizu/mizu has no tag yet, so the go mod tidy the project is
// told to run has nothing to fetch. A replace and the go.sum from here give
// the same build the tag will, and give it without a network call.
func resolve(t *testing.T, dir string) {
	t.Helper()
	sum, err := os.ReadFile(filepath.Join(root(t), "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), sum, 0o644); err != nil {
		t.Fatal(err)
	}
	gocmd(t, dir, "mod", "edit",
		"-require=github.com/go-mizu/mizu@v0.0.0",
		"-replace=github.com/go-mizu/mizu="+root(t))
	// The tidy is here rather than left to the first build because a go command
	// that is only reading, go list among them, refuses to run against a go.mod
	// that is not finished.
	gocmd(t, dir, "mod", "tidy")
}

func gocmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "go", args...)
	cmd.Dir = dir
	// The generated go.mod has no requirements in it, since a project resolves
	// them the first time it is built. -mod=mod is what lets the go command
	// add them rather than report that they are missing.
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func TestPlainName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"blog", true},
		{"my-app", true},
		{"my_app", true},
		{"go1.27", true},
		{"A", true},
		{"9", true},
		{"", false},
		{"-blog", false},
		{"blog-", false},
		{".blog", false},
		{"blog.", false},
		{".", false},
		{"..", false},
		{"my project", false},
		{"blog/api", false},
		{"café", false},
		{`blog"`, false},
	}
	for _, c := range cases {
		if got := plainName(c.name); got != c.want {
			t.Errorf("plainName(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestOutPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"main.go.tmpl", "main.go"},
		{"go.mod.tmpl", "go.mod"},
		{"dot-gitignore.tmpl", ".gitignore"},
		{"dot-gitattributes.tmpl", ".gitattributes"},
		{"handlers/posts.go.tmpl", "handlers/posts.go"},
		{"handlers/dot-keep.tmpl", "handlers/.keep"},
		{"adot-b.tmpl", "adot-b"},
	}
	for _, c := range cases {
		if got := outPath(c.in); got != c.want {
			t.Errorf("outPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNewSpec(t *testing.T) {
	spec := (&New{}).Spec()

	if spec.Name != "new" {
		t.Errorf("the command is called %q", spec.Name)
	}
	if len(spec.Args) != 1 || !spec.Args[0].Required {
		t.Errorf("the directory is not one required argument: %+v", spec.Args)
	}
	for _, p := range presets {
		if !strings.Contains(spec.Long, p.name) {
			t.Errorf("the help does not name the %s preset", p.name)
		}
		if !strings.Contains(spec.Long, p.desc) {
			t.Errorf("the help does not say what the %s preset writes", p.name)
		}
	}
}

func TestNewIsInTheApp(t *testing.T) {
	out, _, code := start(t, "help", "new")

	if code != console.CodeOK {
		t.Fatalf("exited %d: %s", code, out)
	}
	for _, want := range []string{"Write a new project", "--preset", "api", "cli", "Usage:"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the help does not mention %q:\n%s", want, out)
		}
	}
}

// BenchmarkNew is the whole command, which is the templates rendered and six
// files written. It is what somebody waits through, and it is not where they
// wait: the go mod tidy it tells them to run next costs orders more.
func BenchmarkNew(b *testing.B) {
	c := console.New(nil, io.Discard, io.Discard, console.Options{Interaction: console.InteractionNever})
	// Each round writes a project of its own rather than clearing the last
	// one, since removing a tree is not part of what the command does and
	// timing it here would say the command costs twice what it does.
	under := b.TempDir()
	cmd := &New{Preset: "api"}

	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		cmd.Dir = filepath.Join(under, strconv.Itoa(i), "blog")
		if err := cmd.Run(b.Context(), c); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRender is the part with no disk in it, which is where the cost
// would go if a preset grew from six files to sixty.
func BenchmarkRender(b *testing.B) {
	d := data{Name: "blog", Module: "blog", Go: newGo}

	b.ReportAllocs()
	for b.Loop() {
		files, err := render(templates, presets[0], d)
		if err != nil {
			b.Fatal(err)
		}
		if len(files) != len(wanted) {
			b.Fatalf("rendered %d files", len(files))
		}
	}
}
