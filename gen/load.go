package gen

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// A Config controls a load.
type Config struct {
	// Dir is the directory the go command runs in, which decides the module.
	// The empty string means the current working directory.
	Dir string

	// Overlay replaces the contents of files on disk. A key is a path to a Go
	// file, absolute or relative to Dir, and the value is the source text to
	// parse in its place.
	//
	// It exists for the bootstrap problem described in the package comment: a
	// caller that finds a package broken by its own generated output loads it
	// again with those files replaced by empty stubs.
	//
	// An overlay replaces files, it does not add them. A key naming a file the
	// go command did not report is ignored, because the file list comes from
	// the go command and a file that is not in it is not in the package.
	Overlay map[string][]byte
}

// A Package is one loaded Go package.
//
// The field names follow golang.org/x/tools/go/packages, because most people
// reading this have met that API and there is nothing to gain from a second
// set of names for the same things.
type Package struct {
	PkgPath string // import path
	Name    string // package name, empty if the package has no buildable files
	Dir     string // absolute path to the directory
	Module  string // module path, empty outside a module

	GoFiles []string // absolute paths, in the order the go command reports them

	Fset      *token.FileSet // shared by every package from one Load
	Syntax    []*ast.File    // parsed with comments, one per file in GoFiles
	Types     *types.Package
	TypesInfo *types.Info

	// Errors is everything that went wrong, in the order it was found. A
	// package with errors is still returned, with as much of Syntax and Types
	// filled in as go/types managed. That is what makes the bootstrap retry
	// possible.
	Errors []Error
}

// Err returns the package's errors as one error, or nil if there are none.
func (p *Package) Err() error {
	if len(p.Errors) == 0 {
		return nil
	}
	errs := make([]error, len(p.Errors))
	for i, e := range p.Errors {
		errs[i] = e
	}
	return errors.Join(errs...)
}

// An ErrorKind says which stage of the load produced an error, which is worth
// knowing because the three mean different things. A ListError comes from the
// go command and means there was no package to look at, for example because
// build constraints excluded every file in the directory. A ParseError means a
// file is not Go. A TypeError is the interesting one, because a package that
// parses but does not type-check is exactly the state a stale generated file
// leaves behind.
type ErrorKind int

const (
	ListError ErrorKind = iota
	ParseError
	TypeError
	MarkerError
)

func (k ErrorKind) String() string {
	switch k {
	case ListError:
		return "list"
	case ParseError:
		return "parse"
	case TypeError:
		return "type"
	case MarkerError:
		return "marker"
	}
	return fmt.Sprintf("ErrorKind(%d)", int(k))
}

// An Error is one problem found while loading a package.
type Error struct {
	Pos  string // "file:line:col", empty when the error has no position
	Msg  string
	Kind ErrorKind
}

func (e Error) Error() string {
	if e.Pos == "" {
		return e.Msg
	}
	return e.Pos + ": " + e.Msg
}

// Load reads the packages matching patterns, with syntax and type information.
//
// The patterns are the go command's, so "./..." is the module and everything
// under it. With no patterns it loads the package in Dir, which is what the go
// command does.
//
// Test files are not included. A generator reads the declarations a package
// exports to the rest of the program, and a _test.go file is not part of that.
//
// Load returns an error only when the go command itself fails, which means
// there was nothing to load. Problems inside a package are reported on the
// package, in Errors.
func Load(cfg Config, patterns ...string) ([]*Package, error) {
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	list, err := golist(cfg.Dir, patterns)
	if err != nil {
		return nil, err
	}
	l, err := newLoader(cfg, list)
	if err != nil {
		return nil, err
	}
	return l.load(), nil
}

// listed is one package record from `go list -json`, cut down to the fields
// this package reads.
type listed struct {
	ImportPath string
	Name       string
	Dir        string
	DepOnly    bool
	Export     string            // path to the compiled export data
	GoFiles    []string          // base names, relative to Dir
	Imports    []string          // import paths as written in the source
	ImportMap  map[string]string // written path to real path, for vendoring
	Module     *struct {
		Path      string
		GoVersion string
	}
	Error *struct {
		Pos string
		Err string
	}
}

// golist runs one `go list` and decodes the stream of records it writes.
//
// The flags matter, so: -deps brings in the whole graph, -export builds it and
// records where the export data landed, and -e keeps a broken package in the
// output instead of turning it into an exit code. Without -e the bootstrap
// problem would be unrecoverable, because the load that is supposed to
// diagnose the breakage would be the thing that fails.
func golist(dir string, patterns []string) ([]*listed, error) {
	args := append([]string{"list", "-e", "-json", "-deps", "-export"}, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("go list: %s", msg)
		}
		return nil, fmt.Errorf("go list: %w", err)
	}
	var out []*listed
	dec := json.NewDecoder(&stdout)
	for {
		var p listed
		err := dec.Decode(&p)
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, fmt.Errorf("go list: reading output: %w", err)
		}
		out = append(out, &p)
	}
}

type loader struct {
	overlay map[string][]byte
	fset    *token.FileSet
	byPath  map[string]*listed
	roots   []*listed

	// checked holds the packages type-checked from source in this load, which
	// take priority over export data. They have to, because export data was
	// produced from what is on disk and the overlay may say otherwise.
	checked map[string]*types.Package

	// export reads compiled export data for everything else. It keeps its own
	// cache, so a package deep in the standard library is decoded once however
	// many roots reach it.
	export types.Importer
}

func newLoader(cfg Config, list []*listed) (*loader, error) {
	overlay, err := normalizeOverlay(cfg.Dir, cfg.Overlay)
	if err != nil {
		return nil, err
	}
	l := &loader{
		overlay: overlay,
		fset:    token.NewFileSet(),
		byPath:  make(map[string]*listed, len(list)),
		checked: map[string]*types.Package{},
	}
	for _, p := range list {
		l.byPath[p.ImportPath] = p
		if !p.DepOnly {
			l.roots = append(l.roots, p)
		}
	}
	l.export = importer.ForCompiler(l.fset, "gc", l.lookupExport)
	return l, nil
}

func (l *loader) load() []*Package {
	out := make([]*Package, 0, len(l.roots))
	for _, p := range l.order() {
		out = append(out, l.check(p))
	}
	slices.SortFunc(out, func(a, b *Package) int { return strings.Compare(a.PkgPath, b.PkgPath) })
	return out
}

// order returns the roots with every root sorted after the roots it imports,
// so that check can hand a package the real types of its neighbours instead of
// their export data. Go forbids import cycles, so a plain depth-first walk is
// enough and there is no cycle to break.
func (l *loader) order() []*listed {
	root := make(map[string]bool, len(l.roots))
	for _, p := range l.roots {
		root[p.ImportPath] = true
	}
	seen := make(map[string]bool, len(l.roots))
	out := make([]*listed, 0, len(l.roots))
	var visit func(*listed)
	visit = func(p *listed) {
		if seen[p.ImportPath] {
			return
		}
		seen[p.ImportPath] = true
		for _, path := range p.Imports {
			path = resolve(p, path)
			if dep, ok := l.byPath[path]; ok && root[path] {
				visit(dep)
			}
		}
		out = append(out, p)
	}
	for _, p := range l.roots {
		visit(p)
	}
	return out
}

func (l *loader) check(p *listed) *Package {
	out := &Package{
		PkgPath: p.ImportPath,
		Name:    p.Name,
		Dir:     p.Dir,
		Fset:    l.fset,
	}
	if p.Module != nil {
		out.Module = p.Module.Path
	}

	const mode = parser.ParseComments | parser.SkipObjectResolution
	for _, name := range p.GoFiles {
		filename := filepath.Join(p.Dir, name)
		out.GoFiles = append(out.GoFiles, filename)
		f, err := parser.ParseFile(l.fset, filename, l.source(filename), mode)
		if f != nil {
			out.Syntax = append(out.Syntax, f)
		}
		if err != nil {
			out.Errors = append(out.Errors, parseErrors(err)...)
		}
	}
	if len(out.Syntax) == 0 {
		// There is nothing to type-check, so whatever the go command said about
		// this package is the only account of it there will be. That is the one
		// case where its error is worth passing on: everywhere else go/types is
		// about to find the same problem and describe it with a position on it,
		// and reporting both would make a caller counting errors see double.
		if p.Error != nil {
			out.Errors = append(out.Errors, Error{
				Pos:  p.Error.Pos,
				Msg:  strings.TrimSpace(p.Error.Err),
				Kind: ListError,
			})
		}
		return out
	}

	info := &types.Info{
		Types:        map[ast.Expr]types.TypeAndValue{},
		Instances:    map[*ast.Ident]types.Instance{},
		Defs:         map[*ast.Ident]types.Object{},
		Uses:         map[*ast.Ident]types.Object{},
		Implicits:    map[ast.Node]types.Object{},
		Selections:   map[*ast.SelectorExpr]*types.Selection{},
		Scopes:       map[ast.Node]*types.Scope{},
		FileVersions: map[*ast.File]string{},
	}
	conf := &types.Config{
		Importer: &stitched{loader: l, from: p},
		Sizes:    types.SizesFor("gc", runtime.GOARCH),
		Error: func(err error) {
			var e types.Error
			if errors.As(err, &e) {
				out.Errors = append(out.Errors, Error{Pos: e.Fset.Position(e.Pos).String(), Msg: e.Msg, Kind: TypeError})
				return
			}
			out.Errors = append(out.Errors, Error{Msg: err.Error(), Kind: TypeError})
		},
	}
	// Language version comes from the package's own module, because a load can
	// span more than one and the two can disagree about what Go this is.
	if p.Module != nil && p.Module.GoVersion != "" {
		conf.GoVersion = "go" + strings.TrimPrefix(p.Module.GoVersion, "go")
	}

	// Check reports errors through conf.Error and keeps going, so the package
	// it returns describes as much of the source as it could make sense of.
	// That partial result is the whole point here, and it is why the returned
	// error is dropped rather than handled.
	tpkg, _ := conf.Check(p.ImportPath, l.fset, out.Syntax, info)
	out.Types = tpkg
	out.TypesInfo = info
	if tpkg != nil {
		l.checked[p.ImportPath] = tpkg
	}
	return out
}

// source returns the overlay text for a file, or nil to read it from disk.
// The nil is meaningful: parser.ParseFile opens the file itself when the
// source argument is nil, which saves reading files nobody overlaid.
func (l *loader) source(filename string) any {
	if src, ok := l.overlay[filename]; ok {
		return src
	}
	return nil
}

func (l *loader) lookupExport(path string) (io.ReadCloser, error) {
	p, ok := l.byPath[path]
	if !ok {
		return nil, fmt.Errorf("%s was not in the package list", path)
	}
	if p.Export == "" {
		if p.Error != nil {
			return nil, fmt.Errorf("%s did not build: %s", path, p.Error.Err)
		}
		return nil, fmt.Errorf("%s has no export data", path)
	}
	return os.Open(p.Export)
}

// stitched resolves imports for one package. Packages already checked from
// source win, everything else comes from export data, and the import map is
// applied first so that a vendored path resolves the way the go command says
// it does.
type stitched struct {
	loader *loader
	from   *listed
}

func (im *stitched) Import(path string) (*types.Package, error) {
	path = resolve(im.from, path)
	if path == "C" {
		// "C" is the cgo marker rather than a package, and turning it into one
		// means running the cgo preprocessor over the file first. Generators
		// read declarations, so the cost of that is not worth paying, and a
		// clear error beats a confusing one about missing export data.
		return nil, errors.New(`import "C": the loader does not run cgo`)
	}
	if p, ok := im.loader.checked[path]; ok {
		return p, nil
	}
	return im.loader.export.Import(path)
}

// resolve maps an import path as written in the source onto the path the go
// command actually resolved it to. The two differ inside vendor directories
// and inside the standard library, which vendors a few of its dependencies.
func resolve(from *listed, path string) string {
	if real, ok := from.ImportMap[path]; ok {
		return real
	}
	return path
}

// parseErrors flattens what the parser returns. It hands back a scanner.ErrorList
// for a file with several problems, and a single error for the rest, and a
// caller wanting one error per line should not have to know which.
func parseErrors(err error) []Error {
	var list scanner.ErrorList
	if errors.As(err, &list) {
		out := make([]Error, len(list))
		for i, e := range list {
			out[i] = Error{Pos: e.Pos.String(), Msg: e.Msg, Kind: ParseError}
		}
		return out
	}
	return []Error{{Msg: err.Error(), Kind: ParseError}}
}

// normalizeOverlay turns overlay keys into absolute cleaned paths, so that a
// caller can name a file the way it is convenient to name it and still match
// the absolute paths the go command reports.
func normalizeOverlay(dir string, in map[string][]byte) (map[string][]byte, error) {
	if len(in) == 0 {
		return nil, nil
	}
	base, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("overlay: resolving %q: %w", dir, err)
	}
	out := make(map[string][]byte, len(in))
	for name, src := range in {
		if !filepath.IsAbs(name) {
			name = filepath.Join(base, name)
		}
		out[filepath.Clean(name)] = src
	}
	return out, nil
}
