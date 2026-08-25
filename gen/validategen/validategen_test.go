package validategen

import (
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
const goldenPath = "testdata/app/validate_gen.go"

func TestGenerate(t *testing.T) {
	files := generate(t, "testdata", "./app")
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if got := files[0].Path; got != "app/validate_gen.go" {
		t.Errorf("path = %q, want app/validate_gen.go", got)
	}

	golden.Assert(t, format(t, files[0]), golden.At(goldenPath))
}

// TestStructs checks that every marked struct came back, in declaration order,
// and that each one hands the work to a function of its own.
func TestStructs(t *testing.T) {
	p := plan(t)

	var got []string
	for _, s := range p.Structs {
		got = append(got, s.Type+" "+s.Call)
	}
	want := []string{
		"Listing validateListing",
		"Order validateOrder",
		"Webhook validateWebhook",
		"Tree validateTree",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("structs are %v, want %v in declaration order", got, want)
	}
}

// TestHelpers checks which types got a function of their own.
//
// A type gets one when a list of it was dived into, and only then. A struct
// sitting in a field is checked where it sits, because there is exactly one
// place it can be reached from and a call would be a call to somewhere else to
// read four lines.
func TestHelpers(t *testing.T) {
	p := plan(t)

	var got []string
	for _, h := range p.Helpers {
		got = append(got, h.Name)
	}
	want := []string{
		"validateListing",
		"validateLine",
		"validateOrder",
		"validateWebhook",
		"validateTree",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("helpers are %v, want %v, with a nested one written before what reached it", got, want)
	}
}

// TestATypeReachedTwiceIsWrittenOnce is what the functions are for. Two orders
// holding lists of the same line type share one function, and a type holding a
// list of itself has one to call.
func TestATypeReachedTwiceIsWrittenOnce(t *testing.T) {
	p := plan(t)

	seen := map[string]int{}
	for _, h := range p.Helpers {
		seen[h.Type]++
	}
	for typ, n := range seen {
		if n > 1 {
			t.Errorf("%s has %d functions, want 1", typ, n)
		}
	}

	tree := helperOf(t, p, "validateTree")
	if !strings.Contains(tree.Body, "validateTree(bad,") {
		t.Errorf("validateTree does not call itself:\n%s", tree.Body)
	}
}

// TestImports checks that the generated file imports what it writes and
// nothing else, because an unused import does not compile and a missing one
// does not either.
func TestImports(t *testing.T) {
	want := []importLine{
		{Path: "context"},
		{Path: "reflect"},
		{Path: "strconv"},
		{Path: "time"},
		{Path: "unicode/utf8"},
		{Path: "github.com/go-mizu/mizu/validate"},
	}
	got := plan(t).Imports()
	if len(got) != len(want) {
		t.Fatalf("imports = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("import %d = %v, want %v", i, got[i], want[i])
		}
	}
}

// TestPackageWithNothingMarkedWritesNothing covers the case every project has,
// which is a package that has never heard of this generator.
func TestPackageWithNothingMarkedWritesNothing(t *testing.T) {
	if files := generate(t, "testdata", "./broken"); len(files) != 0 {
		t.Errorf("got %d files from a package with no markers, want none", len(files))
	}
}

// TestEveryFormatCheckIsHere is the drift guard between the two modes. A format
// added to the validate package is a rule a tag can name, and a tag this
// generator does not know is a struct that stops generating, so a check that
// lands has to land here in the same change.
func TestEveryFormatCheckIsHere(t *testing.T) {
	pkgs, err := gen.Load(gen.Config{Dir: "."}, "github.com/go-mizu/mizu/validate")
	if err != nil {
		t.Fatal(err)
	}
	scope := pkgs[0].Types.Scope()

	known := map[string]bool{}
	for _, fn := range formats {
		known[fn] = true
	}

	for _, name := range scope.Names() {
		fn, ok := scope.Lookup(name).(*types.Func)
		if !ok || !fn.Exported() || !strings.HasPrefix(name, "Is") || !checksAString(fn) {
			continue
		}
		if !known[name] {
			t.Errorf("validate.%s is a format check and no tag in this generator writes it", name)
		}
		delete(known, name)
	}
	for fn := range known {
		t.Errorf("this generator writes validate.%s and the validate package has no such check", fn)
	}
}

// checksAString is whether a function is one of the format checks, which all
// take a string and answer yes or no.
func checksAString(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() != 1 || sig.Results().Len() != 1 {
		return false
	}
	return types.Identical(sig.Params().At(0).Type(), types.Typ[types.String]) &&
		types.Identical(sig.Results().At(0).Type(), types.Typ[types.Bool])
}

// TestDirOf covers the shapes a package comes in, since where a generated file
// goes depends on it and two of them only turn up outside a module.
func TestDirOf(t *testing.T) {
	cases := []struct {
		pkg  gen.Package
		want string
	}{
		{gen.Package{PkgPath: "example.com/app/web", Module: "example.com/app"}, "web"},
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

// TestNameOf checks that a field is named the way the request names it, which
// is what makes a failure line up with the thing somebody filled in.
func TestNameOf(t *testing.T) {
	cases := []struct {
		tag   string
		field string
		name  string
		ok    bool
	}{
		{"", "PerPage", "per_page", true},
		{"", "UserID", "user_id", true},
		{`json:"q"`, "Query", "q", true},
		{`json:"q,omitempty"`, "Query", "q", true},
		{`json:","`, "Query", "query", true},
		{`json:"-"`, "Query", "", false},
		{`query:"q" json:"other"`, "Query", "q", true},
		{`path:"id"`, "ID", "id", true},
		{`header:"X-Trace"`, "Trace", "X-Trace", true},
		{`cookie:"seen"`, "Seen", "seen", true},
		{`form:""`, "Query", "query", true},
		{`query:"-" json:"q"`, "Query", "", false},
	}
	for _, c := range cases {
		name, ok := nameOf(reflect.StructTag(c.tag), c.field)
		if name != c.name || ok != c.ok {
			t.Errorf("nameOf(`%s`, %q) = %q, %v, want %q, %v", c.tag, c.field, name, ok, c.name, c.ok)
		}
	}
}

// TestSnake covers the names a field gets when nothing said otherwise, which is
// the same answer web.Bind and validate.Struct give.
func TestSnake(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Q", "q"},
		{"PerPage", "per_page"},
		{"UserID", "user_id"},
		{"ID", "id"},
		{"HTTPServer", "http_server"},
		{"OAuth2Token", "o_auth2_token"},
		{"already_snake", "already_snake"},
		{"", ""},
	} {
		if got := snake(c.in); got != c.want {
			t.Errorf("snake(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestLower covers the name a local gets, which is a field's name with the
// leading capitals taken down so that it reads as a variable.
func TestLower(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Ship", "ship"},
		{"ID", "id"},
		{"IPAddr", "ipAddr"},
		{"Note", "note"},
		{"already", "already"},
		{"", "x"},
	} {
		if got := lower(c.in); got != c.want {
			t.Errorf("lower(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestIdent covers the name a function gets from the type it checks.
func TestIdent(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"Line", "Line"},
		{"other.Line", "OtherLine"},
		{"pb.User2", "PbUser2"},
		{"v2.Addr", "V2Addr"},
		{"2fa", "Fa"}, // a digit with nothing in front of it is not a name
		{"[]byte", "Byte"},
		{"struct{...}", "Struct"},
		{"", "Struct"},
	} {
		if got := ident(c.in); got != c.want {
			t.Errorf("ident(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestArticle covers the one piece of English the output writes for itself.
func TestArticle(t *testing.T) {
	for _, c := range []struct{ name, want string }{
		{"Listing", "a"}, {"Order", "an"}, {"Invoice", "an"}, {"", "a"},
	} {
		if got := article(c.name); got != c.want {
			t.Errorf("article(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func helperOf(t *testing.T, p *Plan, name string) *Helper {
	t.Helper()
	for _, h := range p.Helpers {
		if h.Name == name {
			return h
		}
	}
	t.Fatalf("the plan has no function called %s", name)
	return nil
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
// file. Those are the whole point of the generator: the same values checked by
// the generated method and by reflection, compared failure for failure.
//
// It shells out because the generated code is in a module of its own, which is
// what keeps a deliberately broken request struct in testdata from breaking the
// build of the generator that reads it.
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
