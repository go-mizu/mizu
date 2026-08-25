package main

import (
	"cmp"
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/go-mizu/mizu/console"
)

// The projects mizu new writes.
//
// They are files rather than string constants because a Go source file inside
// a Go string constant cannot hold a backquote and a README cannot hold a
// fenced code block. The .tmpl suffix keeps them out of the build, since a
// half written main.go is not something this module should try to compile, and
// a name beginning dot- is a name beginning with a dot: embed skips a file
// whose name starts with one, and a real .gitignore sitting here would apply
// to this repository rather than travel with the template.
//
//go:embed template
var templates embed.FS

// newGo is the go directive the generated go.mod carries.
//
// It is the version mizu itself needs rather than the version of the go
// command that happens to be running, so that two people on different
// toolchains get the same project out of the same command. go mod tidy raises
// it if what the project grows into needs more.
const newGo = "1.27"

// A preset is one shape of project mizu new writes.
type preset struct {
	name string
	desc string

	// try is the command that shows the new project working, and it is the
	// last line mizu new prints. A project somebody has to read before they
	// can run it is a project they read instead of running.
	try string
}

var presets = []preset{
	{"api", "An HTTP service, with routes, a health check, and a test for each route", "go run ."},
	{"cli", "A command line tool, with a command, a version, and a test for each of them", "go run . greet ada"},
}

// data is what the templates are rendered with.
type data struct {
	Name   string // the project, which is also the directory it is written in
	Module string // the module path in go.mod
	Go     string // the go directive in go.mod
}

// A file is one file about to be written, named by its place in the project.
type file struct {
	Path string
	Data []byte
}

// New writes a new project.
type New struct {
	Dir    string
	Module string
	Preset string
}

func (c *New) Spec() console.Spec {
	return console.Spec{
		Name: "new",
		Desc: "Write a new project",
		Long: newLong + "\n\nPresets:\n\n" + presetList(),
		Flags: []console.Flag{
			{Name: "module", Desc: "The module path, the directory name by default", Value: console.String(&c.Module)},
			{Name: "preset", Desc: "What to write: " + presetNames(), Value: console.String(&c.Preset)},
		},
		Args: []console.Arg{
			{Name: "directory", Required: true, Desc: "Where the project goes, and what it is called", Value: console.String(&c.Dir)},
		},
	}
}

func (c *New) Run(ctx context.Context, io *console.IO) error {
	dir := filepath.Clean(c.Dir)

	// The name comes from the absolute path so that mizu new . inside an empty
	// blog directory writes a project called blog, rather than one called ".".
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	name := filepath.Base(abs)
	if err := checkName(name); err != nil {
		return err
	}

	module := cmp.Or(c.Module, name)
	if err := checkModule(module); err != nil {
		return err
	}

	p, err := c.choose(io)
	if err != nil {
		return err
	}

	// Everything is rendered before anything is created, because a template
	// that will not parse should not leave a directory behind it.
	files, err := render(templates, p, data{Name: name, Module: module, Go: newGo})
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	fresh, err := vacant(dir)
	if err != nil {
		return err
	}
	fail := func(err error) error {
		if fresh {
			// The directory was not there a moment ago, so taking it away
			// leaves things as they were. One that was already there belongs
			// to somebody else, half written or not.
			os.RemoveAll(dir)
		}
		return err
	}
	if err := writeTree(dir, files); err != nil {
		return fail(err)
	}

	// A project says what it is from the moment it exists. AGENTS.md is read
	// off the tree rather than rendered from a template, which is why it is
	// written once the rest of the files are there, and it is the same file
	// mizu gen writes from then on.
	notes, err := agents(dir)
	if err != nil {
		return fail(err)
	}
	for _, f := range notes {
		files = append(files, file{Path: f.Path, Data: f.Data})
	}
	if err := writeTree(dir, files[len(files)-len(notes):]); err != nil {
		return fail(err)
	}
	slices.SortFunc(files, func(a, b file) int { return strings.Compare(a.Path, b.Path) })

	return c.report(io, p, dir, module, files)
}

// choose is which preset to write.
//
// With --preset it is what was asked for. Without one it is a question when
// there is somebody to ask and the first preset otherwise, so that a script
// that forgot the flag gets a project rather than a prompt it cannot answer.
func (c *New) choose(io *console.IO) (preset, error) {
	if c.Preset == "" && io.Interactive() {
		options := make([]string, len(presets))
		for i, p := range presets {
			options[i] = fmt.Sprintf("%s (%s)", p.desc, p.name)
		}
		i, err := io.Choice("What are you building?", options, 0)
		if err != nil {
			return preset{}, err
		}
		return presets[i], nil
	}

	want := cmp.Or(c.Preset, presets[0].name)
	i := slices.IndexFunc(presets, func(p preset) bool { return p.name == want })
	if i < 0 {
		return preset{}, console.Exit(console.CodeUsage, fmt.Errorf("no preset called %s, and there are %s", want, presetNames()))
	}
	return presets[i], nil
}

func (c *New) report(io *console.IO, p preset, dir, module string, files []file) error {
	names := make([]string, len(files))
	for i, f := range files {
		names[i] = filepath.Join(dir, filepath.FromSlash(f.Path))
	}

	if io.JSONMode() {
		return io.JSON(struct {
			Dir    string   `json:"dir"`
			Module string   `json:"module"`
			Preset string   `json:"preset"`
			Files  []string `json:"files"`
		}{dir, module, p.name, names})
	}

	// The files go to stdout, because they are the answer to what was asked.
	// What to do with them goes to stderr, because that is the command talking
	// about itself, and under --quiet it has nothing to say.
	for _, name := range names {
		io.Print("  %s\n", name)
	}
	io.Success("Wrote %s.", plural(len(files), "file"))
	io.Info("")
	io.Info("What to do next:")
	io.Info("")
	io.Info("  cd %s", dir)
	io.Info("  go mod tidy")
	io.Info("  go test ./...")
	io.Info("  %s", p.try)
	return nil
}

// render is the shared templates and the preset's own, rendered together.
//
// A preset that names a file the shared set also names wins, since the shared
// set is what most projects want rather than what all of them must have.
//
// The set is a parameter rather than the package variable so that a test can
// pass one that will not render. Every failure below is one the embedded set
// cannot have and one that whoever adds the next template meets the first time
// they get it wrong.
func render(fsys fs.FS, p preset, d data) ([]file, error) {
	byPath := map[string]file{}
	for _, root := range []string{"template/common", "template/" + p.name} {
		err := fs.WalkDir(fsys, root, func(name string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return err
			}
			f, err := renderOne(fsys, name, strings.TrimPrefix(name, root+"/"), d)
			if err != nil {
				return err
			}
			byPath[f.Path] = f
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	files := slices.SortedFunc(maps.Values(byPath), func(a, b file) int {
		return strings.Compare(a.Path, b.Path)
	})
	return files, nil
}

// renderOne reads one template and works out where its output belongs.
//
// The .tmpl suffix comes off, and a leading dot- on the last element becomes
// the dot it stands for.
func renderOne(fsys fs.FS, name, rel string, d data) (file, error) {
	src, err := fs.ReadFile(fsys, name)
	if err != nil {
		return file{}, err
	}
	// missingkey=error turns a template asking for something the data does not
	// have into a failure here rather than the word "<no value>" in a file
	// somebody has to notice.
	t, err := template.New(rel).Option("missingkey=error").Parse(string(src))
	if err != nil {
		return file{}, fmt.Errorf("%s: %w", name, err)
	}
	var b strings.Builder
	if err := t.Execute(&b, d); err != nil {
		return file{}, fmt.Errorf("%s: %w", name, err)
	}
	return file{Path: outPath(rel), Data: []byte(b.String())}, nil
}

func outPath(rel string) string {
	dir, base := path.Split(strings.TrimSuffix(rel, ".tmpl"))
	if rest, ok := strings.CutPrefix(base, "dot-"); ok {
		base = "." + rest
	}
	return path.Join(dir, base)
}

// vacant reports whether dir is somewhere to write a project, and whether it
// had to be created.
//
// An empty directory is fine, and so is one holding nothing but a .git, since
// making the repository first and the project in it second is an ordinary way
// to start. Anything else is somebody's work, and writing a main.go over it is
// not a thing to do on the strength of a directory name.
func vacant(dir string) (fresh bool, err error) {
	entries, err := os.ReadDir(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return true, nil
	case err != nil:
		return false, err
	}
	for _, e := range entries {
		if e.Name() != ".git" {
			return false, fmt.Errorf("%s is not empty, and mizu new writes a project rather than into one", dir)
		}
	}
	return false, nil
}

func writeTree(dir string, files []file) error {
	for _, f := range files {
		name := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(name), 0o777); err != nil {
			return err
		}
		if err := os.WriteFile(name, f.Data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// checkName is what a project can be called.
//
// The name is the directory, and it is also the module path by default, the
// package doc of the generated main.go, a string in a JSON body and the title
// of the README. Holding it to what an import path allows means none of those
// need escaping and none of them can be made to hold something that was never
// a name.
func checkName(name string) error {
	if !plainName(name) {
		return console.Exit(console.CodeUsage, fmt.Errorf("%q is not a name for a project, which is letters and digits, with - . and _ inside them", name))
	}
	return nil
}

func plainName(s string) bool {
	if s == "" {
		return false
	}
	for i := range len(s) {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case (c == '-' || c == '.' || c == '_') && i > 0 && i < len(s)-1:
		default:
			return false
		}
	}
	return true
}

// reserved are the package patterns the go command has taken for itself.
//
// A module named one of these loads no packages at all, and the error says so
// in terms of package patterns rather than of the line in go.mod that caused
// it, which is a long way from the name somebody typed.
var reserved = []string{"all", "cmd", "std", "tool"}

// checkModule is a light check on the module path.
//
// What a module path really is, is the go command's to say, and it says so
// clearly the first time the project is built. This refuses only what would
// make go.mod unparseable rather than wrong, or what the go command answers
// for in some other language than the one the question was asked in.
func checkModule(module string) error {
	for _, r := range module {
		if r <= ' ' || r == '"' || r == '`' || r == 0x7f {
			return console.Exit(console.CodeUsage, fmt.Errorf("%q is not a module path", module))
		}
	}
	if slices.Contains(reserved, module) {
		return console.Exit(console.CodeUsage, fmt.Errorf("%s is a package pattern the go command has taken, so no module can be called it, and --module gives the project a path of its own", module))
	}
	return nil
}

// presetList is the presets, for the help text.
func presetList() string {
	var b strings.Builder
	width := 0
	for _, p := range presets {
		width = max(width, len(p.name))
	}
	for _, p := range presets {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, p.name, p.desc)
	}
	return strings.TrimRight(b.String(), "\n")
}

// presetNames is the presets on one line, for a flag description and for the
// error when somebody names one that is not there.
func presetNames() string {
	names := make([]string, len(presets))
	for i, p := range presets {
		names[i] = p.name
	}
	return strings.Join(names, ", ")
}

const newLong = `The project builds, tests and runs with nothing else installed. Run go mod tidy
first, which is what fetches mizu, and then go test ./... and go run . both
work.

Nothing is required into go.mod, so a project made today starts from what is
current the first time it is built rather than from whatever was current the
moment somebody typed mizu new.

The directory has to be empty, or hold nothing but a .git, since making the
repository first and the project in it second is an ordinary way to start.

With no --preset and somebody at the terminal it asks which one to write. Under
--no-interaction, or with the input coming from somewhere that is not a
terminal, it writes the first one, so a script that forgot the flag gets a
project rather than a question.

What is here is two presets. The rest of what mizu new is meant to ask about,
the database, the frontend, the authentication and the deployment target,
arrives with the packages that answer those questions.`
