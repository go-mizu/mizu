package gen

import (
	"errors"
	"go/ast"
	"go/scanner"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fixture is a small module under testdata with the four shapes the loader has
// to handle: a leaf package, a package that imports it, a package with a type
// error, and a package broken by its own generated file.
const fixture = "testdata/mod"

// listing runs the go command once for the whole test binary. Every test needs
// the same package list, and `go list -export` builds the module, so paying
// for it per test would turn a fast test file into a slow one.
var listing = sync.OnceValues(func() ([]*listed, error) {
	return golist(fixture, []string{"./..."})
})

// loadFixture builds the packages from the shared listing. The parsing and
// type-checking still happen per call, which is what most tests are looking
// at, and it means an overlay test gets a clean load without a second go list.
func loadFixture(t *testing.T, overlay map[string][]byte) map[string]*Package {
	t.Helper()
	list, err := listing()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	l, err := newLoader(Config{Dir: fixture, Overlay: overlay}, list)
	if err != nil {
		t.Fatalf("newLoader: %v", err)
	}
	byPath := map[string]*Package{}
	for _, p := range l.load() {
		byPath[p.PkgPath] = p
	}
	return byPath
}

func TestLoadReturnsOnlyTheRoots(t *testing.T) {
	pkgs := loadFixture(t, nil)
	want := []string{
		"mizu.test/gen/badmarkers",
		"mizu.test/gen/bootstrap",
		"mizu.test/gen/broken",
		"mizu.test/gen/markers",
		"mizu.test/gen/model",
		"mizu.test/gen/store",
	}
	if len(pkgs) != len(want) {
		t.Fatalf("loaded %d packages, want %d: %v", len(pkgs), len(want), keys(pkgs))
	}
	for _, path := range want {
		if pkgs[path] == nil {
			t.Errorf("%s was not loaded", path)
		}
	}
}

func TestLoadedPackageFields(t *testing.T) {
	p := loadFixture(t, nil)["mizu.test/gen/model"]
	if p.Name != "model" {
		t.Errorf("Name = %q, want model", p.Name)
	}
	if p.Module != "mizu.test/gen" {
		t.Errorf("Module = %q, want mizu.test/gen", p.Module)
	}
	if err := p.Err(); err != nil {
		t.Errorf("model should load cleanly: %v", err)
	}
	if len(p.GoFiles) != 1 || filepath.Base(p.GoFiles[0]) != "model.go" {
		t.Errorf("GoFiles = %v", p.GoFiles)
	}
	if !filepath.IsAbs(p.GoFiles[0]) {
		t.Errorf("GoFiles should be absolute, got %q", p.GoFiles[0])
	}
	if len(p.Syntax) != len(p.GoFiles) {
		t.Errorf("got %d files parsed for %d listed", len(p.Syntax), len(p.GoFiles))
	}
	if p.Types == nil || p.TypesInfo == nil {
		t.Fatal("model has no type information")
	}
	if p.Types.Scope().Lookup("User") == nil {
		t.Error("User is not in the package scope")
	}
}

// Every package from one load shares a FileSet, because a position from one
// package has to be printable against another's.
func TestPackagesShareAFileSet(t *testing.T) {
	pkgs := loadFixture(t, nil)
	first := pkgs["mizu.test/gen/model"].Fset
	for path, p := range pkgs {
		if p.Fset != first {
			t.Errorf("%s has its own FileSet", path)
		}
	}
}

// A type from a neighbouring package has to resolve to the package this load
// checked from source, not to a second copy decoded from export data. If it
// did not, a generator comparing types across packages would find that two
// identical types are not identical.
func TestTypesResolveAcrossPackages(t *testing.T) {
	pkgs := loadFixture(t, nil)
	store, model := pkgs["mizu.test/gen/store"], pkgs["mizu.test/gen/model"]
	if err := store.Err(); err != nil {
		t.Fatalf("store should load cleanly: %v", err)
	}

	find, _ := store.Types.Scope().Lookup("Find").(*types.Func)
	if find == nil {
		t.Fatal("Find is not in store's scope")
	}
	ptr, ok := find.Signature().Results().At(0).Type().(*types.Pointer)
	if !ok {
		t.Fatalf("Find returns %s, want a pointer", find.Signature().Results().At(0).Type())
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		t.Fatalf("Find returns *%s, want a named type", ptr.Elem())
	}
	if got := named.Obj().Pkg(); got != model.Types {
		t.Errorf("model.User resolved to a different *types.Package than the one loaded")
	}
	if want := model.Types.Scope().Lookup("User"); named.Obj() != want {
		t.Error("model.User resolved to a different object than the one loaded")
	}
}

// The standard library comes from export data, which is the other half of the
// importer and worth its own test.
func TestTypesResolveIntoTheStandardLibrary(t *testing.T) {
	p := loadFixture(t, nil)["mizu.test/gen/model"]
	created := p.Types.Scope().Lookup("User").Type().Underlying().(*types.Struct)
	for i := range created.NumFields() {
		f := created.Field(i)
		if f.Name() != "Created" {
			continue
		}
		if got := f.Type().String(); got != "time.Time" {
			t.Errorf("Created is %s, want time.Time", got)
		}
		return
	}
	t.Error("User has no Created field")
}

// Generators read markers out of doc comments, so the parse has to keep them.
//
// The second half of this test is the trap. A marker is a directive comment,
// and CommentGroup.Text strips directives on the way out, so the marker is in
// the parsed file and not in the text. Anything reading markers has to walk
// Doc.List. Finding that out from a test is cheaper than finding it out from a
// generator that quietly does nothing.
func TestSyntaxKeepsMarkers(t *testing.T) {
	p := loadFixture(t, nil)["mizu.test/gen/model"]

	var doc *ast.CommentGroup
	for _, f := range p.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			d, ok := n.(*ast.GenDecl)
			if !ok || d.Doc == nil || len(d.Specs) != 1 {
				return true
			}
			if s, ok := d.Specs[0].(*ast.TypeSpec); ok && s.Name.Name == "User" {
				doc = d.Doc
			}
			return true
		})
	}
	if doc == nil {
		t.Fatal("User has no doc comment")
	}

	var found bool
	for _, c := range doc.List {
		if c.Text == "//mizu:table users" {
			found = true
		}
	}
	if !found {
		t.Errorf("the marker on User was not parsed: %v", doc.List)
	}
	if strings.Contains(doc.Text(), "mizu:table") {
		t.Error("CommentGroup.Text now keeps directives, so the warning above is out of date")
	}
}

// A package that does not type-check still comes back, with its syntax and
// whatever go/types worked out. Dropping it would leave a generator with
// nothing to read at the moment it is most needed.
func TestTypeErrorsDoNotDropThePackage(t *testing.T) {
	p := loadFixture(t, nil)["mizu.test/gen/broken"]
	if len(p.Syntax) != 1 {
		t.Fatalf("got %d files, want 1", len(p.Syntax))
	}
	if p.Types == nil {
		t.Fatal("broken has no *types.Package")
	}
	if p.Types.Scope().Lookup("Count") == nil {
		t.Error("Count should still be in scope, the error is inside its body")
	}

	var found bool
	for _, e := range p.Errors {
		if e.Kind != TypeError {
			t.Errorf("unexpected %s error: %v", e.Kind, e)
			continue
		}
		if strings.Contains(e.Msg, "missing") {
			found = true
		}
		if !strings.Contains(e.Pos, "broken.go:7") {
			t.Errorf("error has no useful position: %q", e.Pos)
		}
	}
	if !found {
		t.Errorf("no error mentions the undefined function: %v", p.Errors)
	}
}

// The bootstrap problem, end to end. The first load finds the package broken
// by its own generated file. The second replaces that file with a stub, and
// the hand-written declarations a generator needs come back clean.
func TestOverlayFixesTheBootstrapProblem(t *testing.T) {
	const path = "mizu.test/gen/bootstrap"

	before := loadFixture(t, nil)[path]
	if before.Err() == nil {
		t.Fatal("the fixture is supposed to be broken by its generated file")
	}
	generated := ""
	for _, f := range before.GoFiles {
		if strings.HasSuffix(f, "_gen.go") {
			generated = f
		}
	}
	if generated == "" {
		t.Fatal("no generated file in the fixture")
	}

	after := loadFixture(t, map[string][]byte{generated: []byte("package bootstrap\n")})[path]
	if err := after.Err(); err != nil {
		t.Fatalf("stubbing the generated file should fix the package: %v", err)
	}
	if after.Types.Scope().Lookup("Config") == nil {
		t.Error("Config is gone, and it is the declaration a generator would read")
	}
	if after.Types.Scope().Lookup("String") != nil {
		t.Error("the stub should have replaced the generated declarations")
	}
}

// An overlay key can be relative to Dir, because that is how a caller holding
// a package's file list from somewhere else is likely to have written it.
func TestOverlayAcceptsRelativeKeys(t *testing.T) {
	p := loadFixture(t, map[string][]byte{
		"bootstrap/bootstrap_gen.go": []byte("package bootstrap\n"),
	})["mizu.test/gen/bootstrap"]
	if err := p.Err(); err != nil {
		t.Fatalf("a relative overlay key should have matched: %v", err)
	}
}

// An overlay naming a file the go command did not report changes nothing. The
// package's file list comes from the go command, and inventing an entry in it
// would produce a package the compiler disagrees with.
func TestOverlayIgnoresUnknownFiles(t *testing.T) {
	p := loadFixture(t, map[string][]byte{
		"model/nothere.go": []byte("package model\n\nfunc Extra() {}\n"),
	})["mizu.test/gen/model"]
	if err := p.Err(); err != nil {
		t.Fatalf("model should still load cleanly: %v", err)
	}
	if p.Types.Scope().Lookup("Extra") != nil {
		t.Error("the overlay added a file that is not in the package")
	}
}

func TestOverlayWithBadSyntaxIsAParseError(t *testing.T) {
	p := loadFixture(t, map[string][]byte{
		"model/model.go": []byte("package model\n\nfunc Broken( {}\n"),
	})["mizu.test/gen/model"]
	if len(p.Errors) == 0 {
		t.Fatal("want a parse error")
	}
	for _, e := range p.Errors {
		if e.Kind != ParseError {
			t.Errorf("got a %s error, want parse: %v", e.Kind, e)
		}
		if !strings.Contains(e.Pos, "model.go:3") {
			t.Errorf("error has no useful position: %q", e.Pos)
		}
	}
}

func TestPackagesComeBackSortedByPath(t *testing.T) {
	list, err := listing()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	l, err := newLoader(Config{Dir: fixture}, list)
	if err != nil {
		t.Fatal(err)
	}
	pkgs := l.load()
	for i := 1; i < len(pkgs); i++ {
		if pkgs[i-1].PkgPath >= pkgs[i].PkgPath {
			t.Fatalf("out of order: %s then %s", pkgs[i-1].PkgPath, pkgs[i].PkgPath)
		}
	}
}

// order has to put a package after the packages it imports, whatever order the
// go command listed them in.
func TestOrderIsTopological(t *testing.T) {
	list, err := listing()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	l, err := newLoader(Config{Dir: fixture}, list)
	if err != nil {
		t.Fatal(err)
	}
	var model, store int
	for i, p := range l.order() {
		switch p.ImportPath {
		case "mizu.test/gen/model":
			model = i
		case "mizu.test/gen/store":
			store = i
		}
	}
	if model > store {
		t.Errorf("model is checked at %d and store at %d, so store would see export data", model, store)
	}
}

// When there is nothing to parse, the go command's account of why is the only
// one available and it gets passed on.
func TestListErrorIsReportedWhenThereIsNothingToCheck(t *testing.T) {
	pkgs, err := Load(Config{Dir: fixture}, "./constrained")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("got %d packages, want 1", len(pkgs))
	}
	p := pkgs[0]
	if len(p.Syntax) != 0 {
		t.Errorf("got %d parsed files, want none", len(p.Syntax))
	}
	if len(p.Errors) != 1 || p.Errors[0].Kind != ListError {
		t.Fatalf("errors = %v, want one list error", p.Errors)
	}
	if !strings.Contains(p.Errors[0].Msg, "build constraints") {
		t.Errorf("the go command's words were not passed on: %q", p.Errors[0].Msg)
	}
}

func TestLoadWithNoPatternsUsesDir(t *testing.T) {
	pkgs, err := Load(Config{Dir: filepath.Join(fixture, "model")})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 || pkgs[0].PkgPath != "mizu.test/gen/model" {
		t.Errorf("got %v, want just model", keysOf(pkgs))
	}
}

func TestLoadReportsGoCommandFailure(t *testing.T) {
	_, err := Load(Config{Dir: fixture}, "-not-a-pattern")
	if err == nil {
		t.Fatal("want an error")
	}
	if !strings.Contains(err.Error(), "go list") {
		t.Errorf("the error should say what ran: %v", err)
	}
}

func TestLoadReportsAMissingDirectory(t *testing.T) {
	if _, err := Load(Config{Dir: "testdata/nope"}, "./..."); err == nil {
		t.Fatal("want an error for a directory that is not there")
	}
}

func TestImportCIsRejectedClearly(t *testing.T) {
	im := &stitched{loader: &loader{}, from: &listed{}}
	_, err := im.Import("C")
	if err == nil || !strings.Contains(err.Error(), "cgo") {
		t.Errorf("got %v, want an error naming cgo", err)
	}
}

func TestImportUsesTheImportMap(t *testing.T) {
	real := types.NewPackage("vendor/golang.org/x/net/idna", "idna")
	im := &stitched{
		loader: &loader{checked: map[string]*types.Package{real.Path(): real}},
		from:   &listed{ImportMap: map[string]string{"golang.org/x/net/idna": real.Path()}},
	}
	got, err := im.Import("golang.org/x/net/idna")
	if err != nil || got != real {
		t.Errorf("got %v, %v; want the vendored package", got, err)
	}
}

func TestLookupExportExplainsItself(t *testing.T) {
	l := &loader{byPath: map[string]*listed{
		"a": {ImportPath: "a"},
		"b": {ImportPath: "b", Error: &struct {
			Pos string
			Err string
		}{Err: "b/b.go:1:1: broken"}},
	}}
	for _, tc := range []struct{ path, want string }{
		{"missing", "was not in the package list"},
		{"a", "has no export data"},
		{"b", "did not build"},
	} {
		_, err := l.lookupExport(tc.path)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("lookupExport(%q) = %v, want an error saying %q", tc.path, err, tc.want)
		}
	}
}

func TestParseErrorsFlattensAList(t *testing.T) {
	var list scanner.ErrorList
	list.Add(token.Position{Filename: "a.go", Line: 1, Column: 2}, "first")
	list.Add(token.Position{Filename: "a.go", Line: 3, Column: 4}, "second")
	got := parseErrors(list)
	if len(got) != 2 || got[0].Msg != "first" || got[1].Pos != "a.go:3:4" {
		t.Errorf("got %v, want one entry per error", got)
	}

	got = parseErrors(errors.New("something else"))
	if len(got) != 1 || got[0].Msg != "something else" || got[0].Kind != ParseError {
		t.Errorf("got %v, want the error passed through", got)
	}
}

func TestNormalizeOverlay(t *testing.T) {
	if got, err := normalizeOverlay(".", nil); got != nil || err != nil {
		t.Errorf("an empty overlay should stay nil, got %v, %v", got, err)
	}
	abs, err := filepath.Abs(fixture)
	if err != nil {
		t.Fatal(err)
	}
	out, err := normalizeOverlay(fixture, map[string][]byte{
		"model/../model/model.go":      []byte("relative"),
		filepath.Join(abs, "other.go"): []byte("absolute"),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{filepath.Join(abs, "model", "model.go"), filepath.Join(abs, "other.go")} {
		if _, ok := out[want]; !ok {
			t.Errorf("%s is not in the normalised overlay %v", want, keysOfBytes(out))
		}
	}
}

func TestErrorFormatting(t *testing.T) {
	if got := (Error{Msg: "no"}).Error(); got != "no" {
		t.Errorf("got %q", got)
	}
	if got := (Error{Pos: "a.go:1:2", Msg: "no"}).Error(); got != "a.go:1:2: no" {
		t.Errorf("got %q", got)
	}
	for _, tc := range []struct {
		kind ErrorKind
		want string
	}{
		{ListError, "list"},
		{ParseError, "parse"},
		{TypeError, "type"},
		{MarkerError, "marker"},
		{ErrorKind(9), "ErrorKind(9)"},
	} {
		if got := tc.kind.String(); got != tc.want {
			t.Errorf("got %q, want %q", got, tc.want)
		}
	}
}

func TestPackageErr(t *testing.T) {
	if err := (&Package{}).Err(); err != nil {
		t.Errorf("a clean package should have no error, got %v", err)
	}
	err := (&Package{Errors: []Error{{Msg: "one"}, {Msg: "two"}}}).Err()
	if err == nil || !strings.Contains(err.Error(), "one") || !strings.Contains(err.Error(), "two") {
		t.Errorf("got %v, want both errors", err)
	}
}

func keys(m map[string]*Package) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func keysOf(pkgs []*Package) []string {
	out := make([]string, len(pkgs))
	for i, p := range pkgs {
		out[i] = p.PkgPath
	}
	return out
}

func keysOfBytes(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
