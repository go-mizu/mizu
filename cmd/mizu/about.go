package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/go-mizu/mizu/console"
	"github.com/go-mizu/mizu/gen"
)

// mizuPath is the toolkit's module path, which is how a project's imports are
// told apart from everything else it uses.
const mizuPath = "github.com/go-mizu/mizu"

// An inventory is what mizu can say about the project it was run in.
//
// It is the answer to "what is this", asked by somebody who has just cloned the
// repository, by a bug report, and by an agent that has one call to spend
// before it starts editing. The three want the same facts, so there is one
// command and the difference is --json.
//
// Everything here is read from the project. Nothing is read from a
// configuration file that says what the project is meant to be, because the two
// disagree eventually and the one that is wrong is the one somebody wrote by
// hand.
type inventory struct {
	Mizu      version     `json:"mizu"`      // the binary that answered
	Module    projectInfo `json:"module"`    // the project it was run in
	Uses      []use       `json:"uses"`      // the toolkit packages it imports
	Generated []generated `json:"generated"` // what a generator wrote, by generator
}

// A projectInfo is the module and the toolchain that builds it.
type projectInfo struct {
	Path      string `json:"path"`               // the module path
	Dir       string `json:"dir"`                // the module root
	Go        string `json:"go"`                 // the go directive in go.mod
	Toolchain string `json:"toolchain"`          // the go command that would build it
	Packages  int    `json:"packages"`           // packages in the module
	Mizu      string `json:"mizu,omitempty"`     // the version of the toolkit it builds against
	Replaced  string `json:"replaced,omitempty"` // where a replace points that version at
}

// A use is one toolkit package the project imports, and how much of the project
// imports it.
//
// The count is packages rather than files, since a package is the unit somebody
// removes. It counts a test import too: a dependency a test needs is one the
// project has, and finding that out at the point of trying to drop the package
// is finding out too late.
type use struct {
	Package  string `json:"package"`
	Packages int    `json:"packages"`
}

// A generated names one tool and the files in this project that carry its
// header.
//
// The tool is whatever the header says, so a project that runs sqlc or protoc
// has those listed here beside mizu's own. That is the list somebody wants,
// because the question behind it is which files not to edit by hand, and the
// answer does not depend on who wrote them.
type generated struct {
	By    string   `json:"by"`
	Files []string `json:"files"`
}

// About prints what mizu knows about the project.
type About struct {
	// find works out what project this is, and is a field for the same reason
	// it is one on [Doctor]: a test says what the environment looks like.
	find func(context.Context) (project, error)
}

func (c *About) Spec() console.Spec {
	return console.Spec{
		Name: "about",
		Desc: "Print what this project is made of",
		Long: aboutLong,
	}
}

func (c *About) Run(ctx context.Context, io *console.IO) error {
	find := c.find
	if find == nil {
		find = discover
	}
	p, err := find(ctx)
	if err != nil {
		return err
	}

	inv, err := survey(ctx, p)
	if err != nil {
		return err
	}
	if io.JSONMode() {
		return io.JSON(inv)
	}
	inv.render(io)
	return nil
}

// survey collects the whole inventory.
//
// Two calls to the go command and one walk of the tree. The go command is asked
// rather than the go.mod parsed, because a build list is not a file: a replace,
// a workspace or a requirement two modules away all change the answer, and the
// go command is the thing that knows.
func survey(ctx context.Context, p project) (inventory, error) {
	inv := inventory{
		Mizu: self(),
		Module: projectInfo{
			Path:      p.Module,
			Dir:       p.Dir,
			Go:        p.Go,
			Toolchain: p.Toolchain,
		},
	}

	pkgs, err := listPackages(ctx, p.Dir)
	if err != nil {
		return inventory{}, err
	}
	inv.Module.Packages = len(pkgs)
	inv.Uses = uses(pkgs, p.Module)

	if len(inv.Uses) > 0 {
		// A failure here is a project whose imports are ahead of its go.mod,
		// which is what a project looks like between mizu new and go mod tidy.
		// Reporting nothing at all about it would be a poor answer to a
		// question somebody asks at exactly that moment, so the version is left
		// out and the rest is printed.
		inv.Module.Mizu, inv.Module.Replaced, _ = toolkit(ctx, p.Dir)
	}

	inv.Generated, err = generatedIn(os.DirFS(p.Dir))
	if err != nil {
		return inventory{}, err
	}
	return inv, nil
}

// A listed package is the part of go list -json this command reads.
type listed struct {
	ImportPath   string
	Imports      []string
	TestImports  []string
	XTestImports []string
}

func listPackages(ctx context.Context, dir string) ([]listed, error) {
	// -e because this command reports what is in the project rather than
	// whether it builds. A package with an import nothing provides yet still
	// says what it imports, and that import is the fact wanted here. Whether it
	// builds is what mizu doctor and mizu verify are for.
	out, err := run(ctx, dir, nil, "go", "list", "-e", "-json", "./...")
	if err != nil {
		return nil, err
	}

	// go list writes one object after another rather than an array of them, so
	// this reads until the stream runs out instead of unmarshalling once.
	var pkgs []listed
	dec := json.NewDecoder(strings.NewReader(out))
	for dec.More() {
		var p listed
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("reading go list -json: %w", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// uses counts the toolkit packages the project imports.
//
// A project that is the toolkit gets an empty list rather than a list of itself
// importing itself, which falls out of the rule rather than being a case: an
// import inside the main module is the module's own business.
func uses(pkgs []listed, main string) []use {
	n := map[string]int{}
	for _, p := range pkgs {
		seen := map[string]bool{}
		for _, imp := range slices.Concat(p.Imports, p.TestImports, p.XTestImports) {
			if !under(imp, mizuPath) || under(imp, main) || seen[imp] {
				continue
			}
			seen[imp] = true
			n[imp]++
		}
	}

	out := make([]use, 0, len(n))
	for _, name := range slices.Sorted(maps.Keys(n)) {
		out = append(out, use{Package: name, Packages: n[name]})
	}
	return out
}

// under reports whether path is root or inside it. An empty root is nowhere, so
// that a project whose module path could not be read does not swallow
// everything.
func under(path, root string) bool {
	return root != "" && (path == root || strings.HasPrefix(path, root+"/"))
}

// toolkit is the version of the toolkit this project builds against, and where
// a replace points it if there is one.
//
// A replace is worth printing on its own line. It is the difference between a
// bug report about a release and a bug report about somebody's working copy,
// and it is invisible in the version number.
func toolkit(ctx context.Context, dir string) (v, replaced string, err error) {
	out, err := run(ctx, dir, nil, "go", "list", "-m", "-json", mizuPath)
	if err != nil {
		return "", "", err
	}
	var m struct {
		Version string
		Replace *module
	}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		return "", "", fmt.Errorf("reading go list -m %s: %w", mizuPath, err)
	}
	return m.Version, replacedBy(m.Replace), nil
}

// A module is what a replace directive points at, as go list reports it.
type module struct {
	Path    string
	Dir     string
	Version string
}

// replacedBy is where a replace sends the build, or the empty string when there
// is no replace.
//
// A replace by directory has no version and a replace by module path has no
// directory, so this says whichever one it is rather than making the caller
// work it out.
func replacedBy(m *module) string {
	switch {
	case m == nil:
		return ""
	case m.Dir != "":
		return m.Dir
	case m.Version != "":
		return m.Path + "@" + m.Version
	default:
		return m.Path
	}
}

// skipped are the directories the walk does not go into.
//
// A dot directory is somebody's tooling, node_modules is an ecosystem of its
// own, vendor is a copy of other people's code, and testdata is a fixture. What
// is generated in any of them is not this project's output, and walking them is
// most of the time a walk of the project would take.
var skipped = []string{"node_modules", "testdata", "vendor"}

// generatedIn finds every file in the tree that says a tool wrote it, grouped
// by the tool.
//
// The header is the go command's convention rather than mizu's, so this reads
// the output of every generator the project runs. That is the point: the
// question it answers is which files not to edit by hand.
func generatedIn(fsys fs.FS) ([]generated, error) {
	files := map[string][]string{}
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			// A directory that cannot be read is not evidence of anything, and
			// a command that reports what a project is made of has no business
			// ending over one.
			return nil
		}
		if d.IsDir() {
			if name != "." && (strings.HasPrefix(d.Name(), ".") || slices.Contains(skipped, d.Name())) {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		head, err := head(fsys, name)
		if err != nil || !gen.Generated(head) {
			return nil
		}
		by := generatorOf(head)
		files[by] = append(files[by], name)
		return nil
	})
	if err != nil {
		return nil, err
	}

	out := make([]generated, 0, len(files))
	for _, by := range slices.Sorted(maps.Keys(files)) {
		slices.Sort(files[by])
		out = append(out, generated{By: by, Files: files[by]})
	}
	return out, nil
}

// head is the start of a file, which is as much as the header rule can be
// decided from.
func head(fsys fs.FS, name string) ([]byte, error) {
	f, err := fsys.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Peek returns what it has along with the reason it could not have more, so
	// a file shorter than the buffer comes back whole and the error with it.
	b, _ := bufio.NewReaderSize(f, 4096).Peek(4096)
	return b, nil
}

// generatorOf reads the tool out of a generated header.
//
// The go command's rule only fixes the two ends of the line, so what is in the
// middle is whatever the tool decided to say about itself. This takes it as it
// is, without the "by" that most of them write, and calls it unknown when there
// is nothing between the ends.
func generatorOf(data []byte) string {
	for line := range strings.Lines(string(data)) {
		line = strings.TrimRight(line, "\r\n")
		if !strings.HasPrefix(line, "// Code generated ") || !strings.HasSuffix(line, "DO NOT EDIT.") {
			continue
		}
		by := line[len("// Code generated ") : len(line)-len("DO NOT EDIT.")]
		by = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(by), ";"))
		by = strings.TrimPrefix(by, "by ")
		if by = strings.TrimSuffix(by, "."); by == "" {
			return "unknown"
		}
		return by
	}
	return "unknown"
}

// render writes the inventory for a person.
//
// Four blocks, shortest first: what answered, what the project is, what of the
// toolkit it uses, and what in it was written by a machine. Somebody who wanted
// only the first one has it on the first line, which is most of why this is not
// a table.
func (inv inventory) render(io *console.IO) {
	io.Print("%s\n", inv.Mizu.String())

	m := inv.Module
	io.Print("%s\n", m.Path)
	io.Print("  %s\n", m.Dir)
	io.Print("  go %s, and %s would build it\n", m.Go, m.Toolchain)
	switch {
	case m.Mizu != "":
		line := "  mizu " + m.Mizu
		if m.Replaced != "" {
			line += ", replaced by " + m.Replaced
		}
		io.Print("%s\n", line)
	case len(inv.Uses) > 0:
		io.Print("  mizu is imported and not in the build list yet, run go mod tidy\n")
	}
	io.Print("  %s\n", plural(m.Packages, "package"))

	if len(inv.Uses) > 0 {
		io.Line("")
		rows := make([][]string, 0, len(inv.Uses))
		for _, u := range inv.Uses {
			rows = append(rows, []string{shortPkg(u.Package), plural(u.Packages, "package")})
		}
		io.Table([]string{"Uses", "Imported by"}, rows)
	}

	if len(inv.Generated) > 0 {
		io.Line("")
		rows := make([][]string, 0, len(inv.Generated))
		for _, g := range inv.Generated {
			rows = append(rows, []string{g.By, generatedSummary(g.Files)})
		}
		io.Table([]string{"Generated by", "Files"}, rows)
	}
}

// shortPkg is a toolkit import path as somebody says it out loud. The module
// path on its own is the package named mizu, which is what the file that
// imports it calls it too.
func shortPkg(pkg string) string {
	if pkg == mizuPath {
		return path.Base(mizuPath)
	}
	return strings.TrimPrefix(pkg, mizuPath+"/")
}

// generatedSummary is the files a tool wrote, or the directories when there are
// too many of them to read in a row.
func generatedSummary(files []string) string {
	const most = 3
	if len(files) <= most {
		return strings.Join(files, ", ")
	}

	dirs := map[string]bool{}
	for _, f := range files {
		dirs[path.Dir(f)] = true
	}
	return fmt.Sprintf("%s in %s", plural(len(files), "file"), strings.Join(slices.Sorted(maps.Keys(dirs)), ", "))
}

const aboutLong = `Everything here is read from the project rather than from a file that says
what the project is meant to be. The two disagree eventually, and the one that
is wrong is the one somebody wrote by hand.

The toolkit packages are counted by how many packages import them, tests
included, because a package a test needs is one the project has.

The generated files are whatever carries the header the go command looks for,
so a project that also runs sqlc or protoc sees those files listed here too.
The question behind that list is which files not to edit, and the answer does
not depend on which tool wrote them.

Run it with --json for the whole inventory as one object, which is the call to
make when arriving in a project that is new to you.`
