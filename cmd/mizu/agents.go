package main

import (
	"cmp"
	"errors"
	"fmt"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/go-mizu/mizu/gen"
)

// What the agents generator is called, and what it writes.
//
// The version is the shape of the output rather than the version of the binary.
// A file whose content moved with every release would be out of date on every
// release, and mizu gen --check would fail in a repository nobody had touched.
const (
	agentsName    = "agents"
	agentsVersion = "1"
	agentsFile    = "AGENTS.md"
	agentsSource  = "go.mod and the packages of this module"
)

// agents writes AGENTS.md, which is what an agent reads before it edits
// anything.
//
// The facts in it are the ones somebody arriving at a project has to have and
// cannot guess: what to run, what is where, and which files are written by a
// machine. They are read off the project every time, so the file cannot drift
// from what it describes. What cannot be read off the project goes in the keep
// block at the end, which survives a regeneration untouched.
//
// The loaded packages are not used. mizu gen takes patterns, and a file
// describing the whole project must not change depending on which part of it
// somebody happened to generate. The tree under root is the same either way.
func agents(root string, _ ...*gen.Package) ([]gen.File, error) {
	fsys := os.DirFS(root)
	mod, err := readModule(fsys)
	if err != nil {
		return nil, err
	}
	notes, err := keptNotes(fsys)
	if err != nil {
		return nil, err
	}
	nested := nestedModules(fsys)
	facts := projectFacts{
		Module:    mod,
		Packages:  packagesIn(fsys, nested),
		Nested:    nested,
		Generated: generatedIn(fsys),
		Notes:     notes,
	}
	return []gen.File{{Path: agentsFile, Data: renderAgents(facts)}}, nil
}

// A projectFacts is everything the file is written from.
type projectFacts struct {
	Module    moduleFacts
	Packages  []pkgInfo
	Nested    []string    // directories in this checkout that are modules of their own
	Generated []generated // what carries a generated header, by the tool that wrote it
	Notes     string      // what was in the keep block
}

// A moduleFacts is what go.mod says that this file repeats.
type moduleFacts struct {
	Path string
	Go   string
}

// readModule reads the module path and the go directive out of go.mod.
//
// Both are single line directives at the left margin, and neither can appear in
// a block, so this is the whole of the grammar that matters here. The go
// command is not asked because a generator runs inside one already, and
// spawning another to read two words of a file that is right there is time
// taken off the budget mizu verify has to fit in.
func readModule(fsys fs.FS) (moduleFacts, error) {
	data, err := fs.ReadFile(fsys, "go.mod")
	if err != nil {
		return moduleFacts{}, err
	}

	var mod moduleFacts
	for line := range strings.Lines(string(data)) {
		switch fields := strings.Fields(line); {
		case len(fields) < 2 || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t"):
			// A continuation line inside a block says nothing about either.
		case fields[0] == "module" && mod.Path == "":
			mod.Path = fields[1]
		case fields[0] == "go" && mod.Go == "":
			mod.Go = fields[1]
		}
	}
	if mod.Path == "" {
		return moduleFacts{}, errors.New("go.mod does not name a module, and this file is written about one")
	}
	return mod, nil
}

// nestedModules is the directories in the tree that have a go.mod of their own.
//
// They are in the same checkout and are not part of this module: go build ./...
// does not reach them, go test ./... does not run their tests, and mizu gen
// does not write into them. Saying so is worth a line, because a directory that
// looks like part of the project and is not is a mistake everybody makes once.
func nestedModules(fsys fs.FS) []string {
	var dirs []string
	for name := range projectFiles(fsys) {
		if path.Base(name) == "go.mod" && name != "go.mod" {
			dirs = append(dirs, path.Dir(name))
		}
	}
	slices.Sort(dirs)
	return dirs
}

// A pkgInfo is one package, as a row of the layout table.
type pkgInfo struct {
	Dir  string // from the module root, "." for the root package
	Name string
	Doc  string // the first sentence of the package comment, if it has one
}

// packagesIn finds every package in the tree and what it says about itself.
//
// The files are parsed for their package clause and doc comment only, which is
// fast enough to run on every generate and works on a package that does not
// compile. A project describes itself in its package comments, so the summary
// is already written and this only has to collect it.
//
// Test files are left out. The package they are in is the one being described,
// and an external test package is a second name for the same directory.
//
// The nested modules are left out with everything under them, since a package
// in one of them is not a package of this module.
func packagesIn(fsys fs.FS, nested []string) []pkgInfo {
	byDir := map[string][]string{}
	for name := range projectFiles(fsys) {
		switch {
		case !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go"):
		case slices.ContainsFunc(nested, func(dir string) bool { return strings.HasPrefix(name, dir+"/") }):
		default:
			dir := path.Dir(name)
			byDir[dir] = append(byDir[dir], name)
		}
	}

	pkgs := make([]pkgInfo, 0, len(byDir))
	for _, dir := range slices.Sorted(maps.Keys(byDir)) {
		if p, ok := describe(fsys, dir, byDir[dir]); ok {
			pkgs = append(pkgs, p)
		}
	}
	return pkgs
}

// describe reads one directory's package name and synopsis.
//
// The name comes from the first file that parses, and the synopsis from the
// first that carries a package comment, which is doc.go when there is one. A
// directory of Go files that all fail to parse is not reported, because there
// is nothing to say about it that a build would not say better.
func describe(fsys fs.FS, dir string, files []string) (pkgInfo, bool) {
	rank := func(name string) int {
		if path.Base(name) == "doc.go" {
			return 0
		}
		return 1
	}
	slices.SortFunc(files, func(a, b string) int {
		return cmp.Or(cmp.Compare(rank(a), rank(b)), strings.Compare(a, b))
	})

	p := pkgInfo{Dir: dir}
	for _, name := range files {
		src, err := fs.ReadFile(fsys, name)
		if err != nil {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.PackageClauseOnly|parser.ParseComments)
		if err != nil {
			continue
		}
		if p.Name == "" {
			p.Name = f.Name.Name
		}
		if f.Doc != nil {
			if p.Doc = (&doc.Package{}).Synopsis(f.Doc.Text()); p.Doc != "" {
				break
			}
		}
	}
	return p, p.Name != ""
}

// The markers around the part of the file a person writes.
const (
	keepStart = "<!-- mizu:keep start -->"
	keepEnd   = "<!-- mizu:keep end -->"
)

// keptNotes is what the file on disk holds between the keep markers.
//
// Everything else in the file is read off the project, so this is the only
// place somebody can write in. Losing it would teach people not to trust the
// generator, so a start marker with no end is an error rather than a guess at
// where the notes stopped.
func keptNotes(fsys fs.FS) (string, error) {
	data, err := fs.ReadFile(fsys, agentsFile)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return firstNotes, nil
	case err != nil:
		return "", err
	case !gen.Generated(data):
		return "", fmt.Errorf("%[1]s here was written by hand, and this generator would replace it. Run mv %[1]s %[1]s.old, generate again, and put what is worth keeping between the %[2]s and %[3]s markers", agentsFile, keepStart, keepEnd)
	}

	text := string(data)
	i := strings.Index(text, keepStart)
	if i < 0 {
		// A generated file with the markers taken out gets them back, since
		// there is nothing in it that was not written here in the first place.
		return firstNotes, nil
	}
	rest := text[i+len(keepStart):]
	j := strings.Index(rest, keepEnd)
	if j < 0 {
		return "", fmt.Errorf("%s has a %s with no %s after it, so there is no telling where the notes end", agentsFile, keepStart, keepEnd)
	}
	return strings.Trim(rest[:j], "\n"), nil
}

// firstNotes is what the keep block says before anybody writes in it.
const firstNotes = `Anything written between the two markers is kept as it is when this file is generated again.
It is the place for what cannot be read off the project: the mistake everybody makes here once, the test that has to be run a particular way, the directory that looks unused and is not.`

// renderAgents writes the file.
func renderAgents(f projectFacts) []byte {
	var b strings.Builder
	b.WriteString(agentsHeader())
	fmt.Fprintf(&b, "\n# %s\n\n", f.Module.Path)
	b.WriteString("This file is generated by `mizu gen agents` and read by the agents working on this project.\n")
	b.WriteString("Run `mizu gen:agents` when the project changes, or `mizu verify --fix`, which runs it along with everything else.\n")
	b.WriteString("Notes written between the markers at the end are kept.\n\n")

	mains := mainPackages(f.Packages)
	fmt.Fprintf(&b, "Go %s, %s, %s.\n", f.Module.Go, plural(len(f.Packages), "package"), plural(len(mains), "command"))

	b.WriteString("\n## Commands\n\n")
	b.WriteString(commandBlock(mains, f.Nested))

	b.WriteString("\n## Layout\n\n")
	b.WriteString(layoutTable(f.Packages))

	b.WriteString("\n## Generated files\n\n")
	b.WriteString("These carry a header saying a tool wrote them, and an edit by hand is gone at the next run.\n")
	b.WriteString("What mizu wrote comes back with `mizu gen`.\n\n")
	b.WriteString(mdTable([]string{"Written by", "Files"}, generatedRows(f.Generated)))

	b.WriteString("\n## Notes\n\n")
	b.WriteString(keepStart + "\n\n")
	b.WriteString(f.Notes)
	b.WriteString("\n\n" + keepEnd + "\n")
	return []byte(b.String())
}

// agentsHeader is the comment that marks the file as generated.
//
// Markdown has no comment of its own and takes HTML's, so the header goes in
// one of those. A reader sees nothing and every tool that skips generated files
// still skips it.
func agentsHeader() string {
	return gen.HTMLHeader(agentsName, agentsVersion, agentsSource)
}

// mainPackages is the commands the project builds, which are the paths a person
// or an agent can run.
func mainPackages(pkgs []pkgInfo) []pkgInfo {
	var mains []pkgInfo
	for _, p := range pkgs {
		if p.Name == "main" {
			mains = append(mains, p)
		}
	}
	return mains
}

// commandBlock is the section that earns the file.
//
// One fenced block of lines to copy, in the order somebody reaches for them:
// the fast check first, the whole thing next, then the go commands underneath
// them, then whatever this project builds.
func commandBlock(mains []pkgInfo, nested []string) string {
	lines := [][2]string{
		{"mizu check", "type check and vet, the fastest answer there is"},
		{"mizu verify", "everything that has to pass before a change is done"},
		{"mizu verify --fix", "write what can be written, then carry on"},
		{"mizu gen", "write what the markers in the code ask for"},
		{"mizu doctor", "check the project and the tools it needs"},
		{"go build ./...", ""},
		{"go test ./...", ""},
		{"go test -race ./...", ""},
	}
	for _, p := range mains {
		dir := "."
		if p.Dir != "." {
			dir = "./" + p.Dir
		}
		lines = append(lines, [2]string{"go run " + dir, ""})
	}
	for _, dir := range nested {
		lines = append(lines, [2]string{"go -C " + dir + " test ./...", "a module of its own"})
	}

	width := 0
	for _, l := range lines {
		if l[1] != "" {
			width = max(width, len(l[0]))
		}
	}

	var b strings.Builder
	b.WriteString("```sh\n")
	for _, l := range lines {
		b.WriteString(l[0])
		if l[1] != "" {
			b.WriteString(strings.Repeat(" ", width-len(l[0])))
			b.WriteString("  # " + l[1])
		}
		b.WriteString("\n")
	}
	b.WriteString("```\n")
	return b.String()
}

// mostRows is where the layout table stops.
//
// The file is read at the start of a session, so its length is a cost paid over
// and over. A project with more directories than this has an answer to where
// things are that a table was never going to give.
const mostRows = 60

func layoutTable(pkgs []pkgInfo) string {
	rows := make([][]string, 0, len(pkgs))
	for _, p := range pkgs[:min(len(pkgs), mostRows)] {
		rows = append(rows, []string{p.Dir, p.Name, p.Doc})
	}
	out := mdTable([]string{"Directory", "Package", "What it is"}, rows)
	if len(pkgs) > mostRows {
		out += fmt.Sprintf("\nAnd %s, left out to keep this readable.\n", plural(len(pkgs)-mostRows, "more package"))
	}
	return out
}

// generatedRows is the do not edit table.
//
// AGENTS.md is taken out of what the walk found and put back under this
// generator's name, whether or not it is on disk yet. Otherwise the first run
// writes a file that does not mention itself and the second writes one that
// does, and a project generated twice is never up to date. The name it goes
// back under is read out of the header this generator writes, so the two agree
// by construction.
func generatedRows(found []generated) [][]string {
	files := map[string][]string{}
	for _, g := range found {
		if rest := slices.DeleteFunc(slices.Clone(g.Files), func(f string) bool { return f == agentsFile }); len(rest) > 0 {
			files[g.By] = rest
		}
	}
	self := gen.GeneratedBy([]byte(agentsHeader()))
	files[self] = append(files[self], agentsFile)

	rows := make([][]string, 0, len(files))
	for _, by := range slices.Sorted(maps.Keys(files)) {
		slices.Sort(files[by])
		rows = append(rows, []string{by, generatedSummary(files[by])})
	}
	return rows
}

// mdTable writes a GitHub table, padded so the source is readable as it is.
//
// An empty cell gets a dash. A row of nothing between two pipes reads as though
// something failed to print.
func mdTable(headers []string, rows [][]string) string {
	width := make([]int, len(headers))
	for i, h := range headers {
		width[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			width[i] = max(width[i], len(cell))
		}
	}

	var b strings.Builder
	line := func(cells []string) {
		for i, cell := range cells {
			if cell == "" {
				cell = "-"
			}
			fmt.Fprintf(&b, "| %-*s ", width[i], cell)
		}
		b.WriteString("|\n")
	}
	line(headers)
	for i := range headers {
		fmt.Fprintf(&b, "| %s ", strings.Repeat("-", width[i]))
	}
	b.WriteString("|\n")
	for _, row := range rows {
		line(row)
	}
	return b.String()
}
