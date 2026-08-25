package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/gen"
	"github.com/go-mizu/mizu/gen/commandgen"
	"github.com/go-mizu/mizu/gen/configgen"
)

// A generator is one thing mizu gen runs.
//
// The signature is the whole contract: the module root and the loaded packages
// in, the files they ask for out. Adding one to the list below is what
// registering a generator amounts to.
//
// Most generators read the packages and ignore the root, since they write
// beside the declaration they were made from. One that describes the project
// rather than a declaration in it does the opposite.
type generator struct {
	name string
	desc string
	run  func(root string, pkgs ...*gen.Package) ([]gen.File, error)
}

var generators = []generator{
	{"agents", "Write AGENTS.md, what an agent reads before it edits anything", agents},
	{"command", "Write the Spec methods for the structs marked //mizu:command", fromPackages(commandgen.Generate)},
	{"config", "Write the decoder for the struct marked //mizu:config", fromPackages(configgen.Generate)},
}

// fromPackages adapts a generator that has no use for the module root.
func fromPackages(f func(pkgs ...*gen.Package) ([]gen.File, error)) func(string, ...*gen.Package) ([]gen.File, error) {
	return func(_ string, pkgs ...*gen.Package) ([]gen.File, error) { return f(pkgs...) }
}

// Gen runs generators over a set of packages.
//
// One command does every generator and one does each, which is why the choice
// is a field set at registration rather than a flag. mizu gen is what a person
// runs and mizu gen:command is what the development loop runs when only one
// kind of file changed.
type Gen struct {
	// only names the single generator to run. The empty string runs them all.
	only string

	Packages []string
	Check    bool
}

func (c *Gen) Spec() console.Spec {
	spec := console.Spec{
		Name: "gen",
		Desc: "Run every generator over the packages",
		Long: genLong,
		Flags: []console.Flag{
			{Name: "check", Desc: "Report what would change and write nothing", Value: console.Bool(&c.Check)},
		},
		Args: []console.Arg{
			{Name: "packages", Rest: true, Desc: "Package patterns, ./... by default", Value: console.Strings(&c.Packages, "")},
		},
	}
	if c.only != "" {
		g := c.chosen()[0]
		spec.Name = "gen:" + g.name
		spec.Desc = g.desc
	}
	return spec
}

// chosen is what this command runs, which is one generator or all of them.
//
// A name that is not in the list is a bug in the registration rather than
// anything somebody typed, so it panics where the app is built.
func (c *Gen) chosen() []generator {
	if c.only == "" {
		return generators
	}
	i := slices.IndexFunc(generators, func(g generator) bool { return g.name == c.only })
	if i < 0 {
		panic("mizu: no generator called " + c.only)
	}
	return generators[i : i+1]
}

func (c *Gen) Run(ctx context.Context, io *console.IO) error {
	patterns := c.Packages
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	pkgs, err := load("", patterns)
	if err != nil {
		return err
	}
	root, err := moduleRoot(pkgs)
	if err != nil {
		return err
	}

	var files []gen.File
	var refused []error
	for _, g := range c.chosen() {
		out, err := g.run(root, pkgs...)
		files = append(files, out...)
		refused = append(refused, err)
	}
	if err := errors.Join(refused...); err != nil {
		return err
	}

	// Loading is the slow part and nothing in it takes a context, so this is
	// where a run that was interrupted stops. Writing is the only step that
	// changes anything, and a tree half written by a command somebody pressed
	// Ctrl-C on is worse than one not written at all.
	if err := ctx.Err(); err != nil {
		return err
	}

	w := &gen.Writer{Dir: root, Check: c.Check}
	results, err := w.Write(files...)
	if err != nil {
		return err
	}
	return c.report(io, results)
}

// load reads the packages, and reads them again with the generated files
// stubbed out if their own output is what stopped them compiling.
//
// That is the ordinary edit loop: rename a field, and the generated file that
// mentions it no longer builds, so the package no longer type-checks, so the
// generator that would fix it has nothing to work from. Extraction only needs
// the hand-written declarations, so replacing the generated files with an
// empty package clause is enough to get past it.
//
// Errors from the second load are real, and they are the ones reported.
//
// The directory is where the patterns are resolved from, and the empty string
// means the one the command was run in.
func load(dir string, patterns []string) ([]*gen.Package, error) {
	cfg := gen.Config{Dir: dir}
	for {
		pkgs, err := gen.Load(cfg, patterns...)
		if err != nil {
			return nil, err
		}
		bad := errs(pkgs)
		if bad == nil {
			return pkgs, nil
		}
		// Only the first pass gets to blame the generated files. Anything
		// still wrong once they are out of the way is the package's own, and
		// a third opinion would say the same thing again.
		if cfg.Overlay != nil {
			return nil, bad
		}
		if cfg.Overlay = stubs(pkgs); len(cfg.Overlay) == 0 {
			return nil, bad
		}
	}
}

// stubs are the generated files in the packages that did not compile, mapped
// to an empty package of the same name.
func stubs(pkgs []*gen.Package) map[string][]byte {
	overlay := map[string][]byte{}
	for _, p := range pkgs {
		if len(p.Errors) == 0 || p.Name == "" {
			continue
		}
		for _, name := range p.GoFiles {
			// A file that cannot be read is not one this is going to fix, and
			// the load reported it already.
			data, err := os.ReadFile(name)
			if err == nil && gen.Generated(data) {
				overlay[name] = []byte("package " + p.Name + "\n")
			}
		}
	}
	return overlay
}

func errs(pkgs []*gen.Package) error {
	var all []error
	for _, p := range pkgs {
		all = append(all, p.Err())
	}
	return errors.Join(all...)
}

// moduleRoot is the directory the generated paths are relative to.
//
// A generator names a file by where it belongs in the module, not by where the
// command was run, so mizu gen writes the same files from any directory in the
// project.
func moduleRoot(pkgs []*gen.Package) (string, error) {
	root := ""
	for _, p := range pkgs {
		if p.Module == "" {
			return "", fmt.Errorf("%s is not in a module, and a generator writes files by their place in one", p.PkgPath)
		}
		dir := p.Dir
		if rel := strings.TrimPrefix(strings.TrimPrefix(p.PkgPath, p.Module), "/"); rel != "" {
			for range strings.Count(rel, "/") + 1 {
				dir = filepath.Dir(dir)
			}
		}
		switch {
		case root == "":
			root = dir
		case root != dir:
			return "", fmt.Errorf("the packages are in two modules, %s and %s, and one run writes into one of them", root, dir)
		}
	}
	if root == "" {
		return "", errors.New("no packages matched")
	}
	return root, nil
}

// report writes what happened, and says so as a failure when --check found
// something out of date.
func (c *Gen) report(io *console.IO, results []gen.Result) error {
	if len(results) == 0 {
		// A table header over no rows reads as though something was found.
		// JSON mode still writes the empty list, because a script reading it
		// wants a list rather than nothing at all.
		if io.JSONMode() {
			return io.JSON([]any{})
		}
		io.Info("Nothing asked to be generated.")
		return nil
	}

	rows := make([][]string, 0, len(results))
	changed := 0
	for _, r := range results {
		rows = append(rows, []string{r.Path, c.status(r)})
		if r.Changed() {
			changed++
		}
	}
	io.Table([]string{"File", "Status"}, rows)

	switch {
	case c.Check && changed > 0:
		return fmt.Errorf("%s out of date, run mizu %s", plural(changed, "generated file"), c.Spec().Name)
	case c.Check:
		io.Success("%s up to date.", plural(len(results), "generated file"))
	case changed == 0:
		io.Info("%s already up to date.", plural(len(results), "generated file"))
	default:
		io.Success("Wrote %s.", plural(changed, "file"))
	}
	return nil
}

// status is the word for one result.
//
// Check mode says what is wrong with the file rather than what would be done
// about it, because somebody reading a failed build wants to know what to look
// at. It names the first line that differs for the same reason: a generated
// file is long and the interesting part of it is one declaration.
func (c *Gen) status(r gen.Result) string {
	if !c.Check {
		return r.Status.String()
	}
	switch r.Status {
	case gen.Created:
		return "missing"
	case gen.Updated:
		return "stale from line " + strconv.Itoa(r.Line)
	}
	return "up to date"
}

const genLong = `Every generator reads the markers in the packages it is given and writes what
they ask for. With no packages it runs over ./..., which is what a project
wants and what the development loop uses.

A generated file carries a header saying so, and a file on disk without one is
never overwritten, so a generator only ever replaces its own output.

A file that has not changed is not rewritten. Writing a file that did not
change moves its mtime, which wakes every watcher and rebuilds everything
downstream of it.

--check reports what would change, writes nothing, and fails when anything is
out of date. It runs the same code as a real run, which is the only way the
answer is worth anything in CI.

A package that no longer compiles because of its own stale generated file is
read again with those files stubbed out, so renaming a field and regenerating
is one step rather than two.`
