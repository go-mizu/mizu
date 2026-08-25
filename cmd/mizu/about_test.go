package main

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/console/consoletest"
)

func TestAboutOnAProjectThatUsesTheToolkit(t *testing.T) {
	dir := scratch(t, commands)

	r := consoletest.Run(t, &About{find: here(dir)}, consoletest.Args()).AssertSuccess()

	for _, want := range []string{
		"commandtest",         // the module
		dir,                   // where it is
		"go 1.27",             // what it asks for
		"2 packages",          // app and broken
		"console",             // the one toolkit package it imports
		"mizu gen command v1", // what wrote a file in it
		"app/commands_gen.go", // which file
	} {
		r.AssertOutputContains(want)
	}
	r.AssertNoErrorOutput()
}

func TestAboutAsJSON(t *testing.T) {
	dir := scratch(t, commands)

	r := consoletest.Run(t, &About{find: here(dir)},
		consoletest.Args(),
		consoletest.With(console.Options{JSON: true}),
	).AssertSuccess()

	var inv inventory
	if err := json.Unmarshal([]byte(r.Stdout()), &inv); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, r.Stdout())
	}
	if inv.Mizu.Version == "" {
		t.Error("the binary did not say what version it is")
	}
	if got, want := inv.Module.Path, "commandtest"; got != want {
		t.Errorf("module path = %q, want %q", got, want)
	}
	if got, want := inv.Module.Packages, 2; got != want {
		t.Errorf("packages = %d, want %d", got, want)
	}
	if got, want := inv.Uses, []use{{Package: mizuPath + "/console", Packages: 2}}; !slices.Equal(got, want) {
		t.Errorf("uses = %+v, want %+v", got, want)
	}
	want := []generated{{By: "mizu gen command v1", Files: []string{"app/commands_gen.go"}}}
	if got := inv.Generated; len(got) != 1 || got[0].By != want[0].By || !slices.Equal(got[0].Files, want[0].Files) {
		t.Errorf("generated = %+v, want %+v", got, want)
	}

	// The replace in the fixture points at this checkout, and a version read
	// through a replace is the kind of thing a bug report needs to say.
	if inv.Module.Replaced == "" {
		t.Error("the replace directive is not reported")
	}
}

// Between mizu new and go mod tidy the imports are ahead of go.mod, which is
// one of the moments somebody runs this command. It says what it can rather
// than refusing to say anything.
func TestAboutOnAProjectThatHasNotBeenTidied(t *testing.T) {
	dir := scratch(t, commands)
	mod := filepath.Join(dir, "go.mod")
	text, err := os.ReadFile(mod)
	if err != nil {
		t.Fatal(err)
	}
	kept := regexp.MustCompile(`(?m)^(require|replace) github\.com/go-mizu/mizu.*\n`).ReplaceAllString(string(text), "")
	if err := os.WriteFile(mod, []byte(kept), 0o644); err != nil {
		t.Fatal(err)
	}

	r := consoletest.Run(t, &About{find: here(dir)}, consoletest.Args()).AssertSuccess()
	r.AssertOutputContains("not in the build list yet, run go mod tidy")
	r.AssertOutputContains("console") // the import is still there to report
}

// With no find of its own the command discovers the project for real, which is
// what it does everywhere except in a test.
func TestAboutDiscoversTheProjectItIsRunIn(t *testing.T) {
	r := consoletest.Run(t, &About{}, consoletest.Args()).AssertSuccess()
	r.AssertOutputContains(mizuPath)
	// The toolkit is not a user of itself, so there is no table of uses here.
	if strings.Contains(r.Stdout(), "Imported by") {
		t.Errorf("the toolkit is listed as using itself:\n%s", r.Stdout())
	}
}

func TestAboutOnSomethingThatIsNotAProject(t *testing.T) {
	for _, tt := range []struct {
		name string
		find func(context.Context) (project, error)
		want string
	}{
		{
			name: "nothing to look at",
			find: func(context.Context) (project, error) {
				return project{}, errors.New("this directory is not in a module")
			},
			want: "not in a module",
		},
		{
			name: "a directory that is not there",
			find: here(filepath.Join(t.TempDir(), "gone")),
			want: "go",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := consoletest.Run(t, &About{find: tt.find}, consoletest.Args()).AssertFailure()
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("the error is %q, want it to mention %q", err, tt.want)
			}
		})
	}
}

// The version lookup is the one call this command lets fail, so the failure has
// to be the go command's own words when anything reads it.
func TestToolkitOnADirectoryWithNoModule(t *testing.T) {
	if _, _, err := toolkit(t.Context(), filepath.Join(t.TempDir(), "gone")); err == nil {
		t.Fatal("no error from a directory that is not there")
	}
}

func TestReplacedBy(t *testing.T) {
	tests := []struct {
		name string
		m    *module
		want string
	}{
		{name: "no replace at all"},
		{
			name: "replaced by a checkout",
			m:    &module{Path: mizuPath, Dir: "/home/x/mizu"},
			want: "/home/x/mizu",
		},
		{
			name: "replaced by another module",
			m:    &module{Path: "example.com/fork", Version: "v1.2.3"},
			want: "example.com/fork@v1.2.3",
		},
		{
			name: "replaced by a module with no version of its own",
			m:    &module{Path: "example.com/fork"},
			want: "example.com/fork",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := replacedBy(tt.m); got != tt.want {
				t.Errorf("replacedBy() = %q, want %q", got, tt.want)
			}
		})
	}
}

// A file or a directory that cannot be read is not evidence of anything, and a
// command that reports what a project is made of has no business ending over
// one.
func TestGeneratedInWalksPastWhatItCannotRead(t *testing.T) {
	tree := fstest.MapFS{
		"app/a_gen.go": {Data: []byte("// Code generated by mizu gen config v1; DO NOT EDIT.\n")},
		"app/b.go":     {Data: []byte("package app\n")},
	}
	for _, tt := range []struct {
		name string
		deny func(string) bool
	}{
		{name: "the files", deny: func(name string) bool { return name != "." && name != "app" }},
		{name: "a whole directory", deny: func(name string) bool { return name == "app" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := generatedIn(refusing{tree, tt.deny}); len(got) != 0 {
				t.Errorf("generatedIn() = %+v, want nothing", got)
			}
		})
	}
}

// refusing is a file system where some names cannot be opened.
type refusing struct {
	fs.FS
	deny func(name string) bool
}

func (r refusing) Open(name string) (fs.File, error) {
	if r.deny(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return r.FS.Open(name)
}

// Neither call can fail on output the go command wrote, so what these cover is
// the reading rather than the answer.
func TestParsingWhatTheGoCommandDidNotWrite(t *testing.T) {
	if _, err := parsePackages(`{"ImportPath": `); err == nil {
		t.Error("no error from a truncated go list -json")
	}
	if _, _, err := parseModule("not json at all"); err == nil {
		t.Error("no error from a go list -m -json that is not JSON")
	}
}

func TestUses(t *testing.T) {
	tests := []struct {
		name string
		main string
		pkgs []listed
		want []use
	}{
		{
			name: "nothing but the standard library",
			main: "example.com/shop",
			pkgs: []listed{{ImportPath: "example.com/shop", Imports: []string{"fmt", "net/http"}}},
		},
		{
			name: "counted once per package, however many times it is imported",
			main: "example.com/shop",
			pkgs: []listed{{
				ImportPath:  "example.com/shop",
				Imports:     []string{mizuPath + "/log"},
				TestImports: []string{mizuPath + "/log"},
			}},
			want: []use{{Package: mizuPath + "/log", Packages: 1}},
		},
		{
			name: "a package a test needs is one the project has",
			main: "example.com/shop",
			pkgs: []listed{{
				ImportPath:   "example.com/shop",
				XTestImports: []string{mizuPath + "/mizutest"},
			}},
			want: []use{{Package: mizuPath + "/mizutest", Packages: 1}},
		},
		{
			name: "the module itself counts, and sorts with the rest",
			main: "example.com/shop",
			pkgs: []listed{{
				ImportPath: "example.com/shop",
				Imports:    []string{mizuPath + "/str", mizuPath},
			}},
			want: []use{
				{Package: mizuPath, Packages: 1},
				{Package: mizuPath + "/str", Packages: 1},
			},
		},
		{
			name: "two packages importing one",
			main: "example.com/shop",
			pkgs: []listed{
				{ImportPath: "example.com/shop", Imports: []string{mizuPath + "/log"}},
				{ImportPath: "example.com/shop/app", Imports: []string{mizuPath + "/log", mizuPath + "/errs"}},
			},
			want: []use{
				{Package: mizuPath + "/errs", Packages: 1},
				{Package: mizuPath + "/log", Packages: 2},
			},
		},
		{
			// The toolkit importing itself is the toolkit's own business, and
			// listing it would say every package in it uses the toolkit.
			name: "the toolkit is not a user of itself",
			main: mizuPath,
			pkgs: []listed{{ImportPath: mizuPath + "/log", Imports: []string{mizuPath + "/errs"}}},
		},
		{
			// A module whose path starts the same way is a different module.
			name: "a lookalike import path is not the toolkit",
			main: "example.com/shop",
			pkgs: []listed{{ImportPath: "example.com/shop", Imports: []string{mizuPath + "-extra/log"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uses(tt.pkgs, tt.main); !slices.Equal(got, tt.want) {
				t.Errorf("uses() = %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestGeneratorOf(t *testing.T) {
	tests := []struct {
		name string
		head string
		want string
	}{
		{
			name: "what mizu writes",
			head: "// Code generated by mizu gen command v1; DO NOT EDIT.\n// Source: app/commands.go\n\npackage app\n",
			want: "mizu gen command v1",
		},
		{
			name: "what sqlc writes",
			head: "// Code generated by sqlc. DO NOT EDIT.\n// versions:\n//   sqlc v1.27.0\n",
			want: "sqlc",
		},
		{
			name: "a header below a build constraint",
			head: "//go:build linux\n\n// Code generated by protoc-gen-go. DO NOT EDIT.\n",
			want: "protoc-gen-go",
		},
		{
			name: "a header that names nobody",
			head: "// Code generated DO NOT EDIT.\n",
			want: "unknown",
		},
		{
			name: "a header with nothing between the ends but punctuation",
			head: "// Code generated. DO NOT EDIT.\n",
			want: "unknown",
		},
		{
			name: "a file that only talks about generated code",
			head: "// Package gen writes code that says DO NOT EDIT.\n",
			want: "unknown",
		},
		{
			name: "carriage returns",
			head: "// Code generated by stringer. DO NOT EDIT.\r\n",
			want: "stringer",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generatorOf([]byte(tt.head)); got != tt.want {
				t.Errorf("generatorOf() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGeneratedIn(t *testing.T) {
	const (
		mizu  = "// Code generated by mizu gen config v1; DO NOT EDIT.\npackage p\n"
		proto = "// Code generated by protoc-gen-go. DO NOT EDIT.\npackage p\n"
	)
	fsys := fstest.MapFS{
		"app/config_gen.go":       {Data: []byte(mizu)},
		"app/config.go":           {Data: []byte("package app\n")},
		"rpc/b_gen.go":            {Data: []byte(proto)},
		"rpc/a_gen.go":            {Data: []byte(proto)},
		"web/assets.js":           {Data: []byte("// Code generated by esbuild. DO NOT EDIT.\n")},
		".git/hooks/x_gen.go":     {Data: []byte(proto)},
		"node_modules/p/i_gen.go": {Data: []byte(proto)},
		"vendor/v/x_gen.go":       {Data: []byte(proto)},
		"app/testdata/f_gen.go":   {Data: []byte(proto)},
		// A symlink is a name for a file that is somewhere else, and reading it
		// here would report the same file twice or report one outside the tree.
		"app/link_gen.go": {Data: []byte(proto), Mode: fs.ModeSymlink},
	}

	got := generatedIn(fsys)
	want := []generated{
		{By: "esbuild", Files: []string{"web/assets.js"}},
		{By: "mizu gen config v1", Files: []string{"app/config_gen.go"}},
		{By: "protoc-gen-go", Files: []string{"rpc/a_gen.go", "rpc/b_gen.go"}},
	}
	if len(got) != len(want) {
		t.Fatalf("generatedIn() = %+v\nwant %+v", got, want)
	}
	for i := range want {
		if got[i].By != want[i].By || !slices.Equal(got[i].Files, want[i].Files) {
			t.Errorf("generatedIn()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// The header rule is about the top of a file, so a mention further down is a
// sentence in somebody's code rather than a claim about who wrote it.
func TestGeneratedInReadsOnlyTheTopOfAFile(t *testing.T) {
	buried := strings.Repeat("// filler\n", 1000) + "// Code generated by nobody. DO NOT EDIT.\n"

	if got := generatedIn(fstest.MapFS{"app/a.go": {Data: []byte(buried)}}); len(got) != 0 {
		t.Errorf("generatedIn() = %+v, want nothing", got)
	}
}

func TestGeneratedSummary(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{
			name:  "a few are worth naming",
			files: []string{"app/a_gen.go", "app/b_gen.go"},
			want:  "app/a_gen.go, app/b_gen.go",
		},
		{
			name:  "more than a few are worth counting",
			files: []string{"rpc/a.go", "rpc/b.go", "rpc/c.go", "rpc/d.go"},
			want:  "4 files in rpc",
		},
		{
			name:  "spread over directories, the directories are the answer",
			files: []string{"rpc/a.go", "rpc/b.go", "web/c.go", "web/d.go"},
			want:  "4 files in rpc, web",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generatedSummary(tt.files); got != tt.want {
				t.Errorf("generatedSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShortPkg(t *testing.T) {
	tests := []struct{ pkg, want string }{
		{mizuPath, "mizu"},
		{mizuPath + "/log", "log"},
		{mizuPath + "/console/consoletest", "console/consoletest"},
	}
	for _, tt := range tests {
		if got := shortPkg(tt.pkg); got != tt.want {
			t.Errorf("shortPkg(%q) = %q, want %q", tt.pkg, got, tt.want)
		}
	}
}

func TestUnder(t *testing.T) {
	tests := []struct {
		path, root string
		want       bool
	}{
		{mizuPath, mizuPath, true},
		{mizuPath + "/log", mizuPath, true},
		{mizuPath + "-extra", mizuPath, false},
		{"example.com/shop", mizuPath, false},
		// A project whose module path could not be read owns nothing, rather
		// than owning every import there is.
		{mizuPath + "/log", "", false},
	}
	for _, tt := range tests {
		if got := under(tt.path, tt.root); got != tt.want {
			t.Errorf("under(%q, %q) = %v, want %v", tt.path, tt.root, got, tt.want)
		}
	}
}

// about is registered, takes no arguments, and --json means the same thing on
// either side of the name.
func TestAboutIsWiredUp(t *testing.T) {
	out, errOut := say(t)
	if code := newApp().Start(t.Context(), nil, out, errOut, []string{"about", "extra"}); code != console.CodeUsage {
		t.Fatalf("exited %d, want %d", code, console.CodeUsage)
	}
	if !strings.Contains(errOut.String(), "extra") {
		t.Errorf("the error does not name the argument:\n%s", errOut)
	}
}
