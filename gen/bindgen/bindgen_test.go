package bindgen

import (
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
const goldenPath = "testdata/app/bind_gen.go"

func TestGenerate(t *testing.T) {
	files := generate(t, "testdata", "./app")
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if got := files[0].Path; got != "app/bind_gen.go" {
		t.Errorf("path = %q, want app/bind_gen.go", got)
	}

	golden.Assert(t, format(t, files[0]), golden.At(goldenPath))
}

// TestStructs checks that every marked struct came back, in declaration order,
// and that the two flags on each one are set from what is in it.
func TestStructs(t *testing.T) {
	p := plan(t)

	var names []string
	for _, s := range p.Structs {
		names = append(names, s.Type)
	}
	want := []string{"Listing", "Order", "Profile", "Webhook", "Tree"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("structs are %v, want %v in declaration order", names, want)
	}

	for _, c := range []struct {
		typ    string
		lax    bool
		values bool
	}{
		{"Listing", false, true},
		{"Order", false, true},
		{"Profile", false, true},
		{"Webhook", true, true},
		{"Tree", false, true},
	} {
		s := structOf(t, p, c.typ)
		if s.Lax != c.lax {
			t.Errorf("%s lax = %v, want %v", c.typ, s.Lax, c.lax)
		}
		if s.Values != c.values {
			t.Errorf("%s values = %v, want %v", c.typ, s.Values, c.values)
		}
	}
}

// TestFields checks the details that are hard to see in a golden file: where
// each field is read from, what turns the string into it, and how the output
// reaches it.
func TestFields(t *testing.T) {
	p := plan(t)

	cases := []struct {
		typ  string
		want Field
	}{
		{"Listing", Field{Name: "page", Src: FromValues, Go: "v.Paging.Page", Kind: Int}},
		{"Listing", Field{Name: "q", Src: FromValues, Go: "v.Q", Kind: Assign, Type: "string"}},
		{"Listing", Field{Name: "sort", Src: FromValues, Go: "v.Sort", Kind: Assign, Type: "Sort"}},
		{"Listing", Field{Name: "tags", Src: FromValues, Go: "v.Tags", Kind: Assign, List: true, Type: "string", Slice: "[]string", Var: "tags"}},
		{"Listing", Field{Name: "cursor", Src: FromValues, Go: "v.Cursor", Kind: Assign, Type: "[]byte"}},
		{"Listing", Field{Name: "limit", Src: FromValues, Go: "v.Limit", Kind: Uint}},
		{"Listing", Field{Name: "score", Src: FromValues, Go: "v.Score", Kind: Float}},
		{"Listing", Field{Name: "draft", Src: FromValues, Go: "v.Draft", Kind: Bool, Ptr: true, Type: "bool"}},
		{"Listing", Field{Name: "since", Src: FromValues, Go: "v.Since", Kind: Time}},
		{"Order", Field{Name: "id", Src: FromPath, Go: "v.ID", Kind: Int}},
		{"Order", Field{Name: "X-Request-Id", Src: FromHeader, Go: "v.Ref", Kind: Assign, Type: "string"}},
		{"Order", Field{Name: "locale", Src: FromCookie, Go: "v.Locale", Kind: Assign, Type: "string"}},
		{"Order", Field{Name: "X-Trace", Src: FromHeader, Go: "v.Traces", Kind: Assign, List: true, Type: "string", Slice: "[]string"}},
		{"Order", Field{Name: "note", Src: FromValues, Go: "v.Note", Kind: Assign, Ptr: true, Type: "string"}},
		{"Order", Field{Name: "wait", Src: FromValues, Go: "v.Wait", Kind: Duration}},
		{"Order", Field{Name: "codes", Src: FromValues, Go: "v.Codes", Kind: Int, List: true, Type: "int", Slice: "[]int", Var: "codes"}},
		{"Order", Field{Name: "origin", Src: FromValues, Go: "v.Origin", Kind: Text}},
		{"Order", Field{Name: "address.city", Src: FromValues, Go: "v.Address.City", Kind: Assign, Type: "string"}},
		{"Order", Field{Name: "address.zone", Src: FromValues, Go: "v.Address.Zone", Kind: Text, Ptr: true, Type: "netip.Addr"}},
		{"Order", Field{Name: "coupon_code", Src: FromValues, Go: "v.CouponCode", Kind: Assign, Type: "string"}},
		{"Profile", Field{Name: "avatar", Src: FromFile, Go: "v.Avatar"}},
		{"Profile", Field{Name: "photos", Src: FromFile, Go: "v.Photos", List: true, Slice: "[]*web.Upload"}},
		{"Tree", Field{Name: "name", Src: FromValues, Go: "v.Name", Kind: Assign, Type: "string"}},
	}
	for _, c := range cases {
		got, ok := fieldOf(p, c.typ, c.want.Go)
		if !ok {
			t.Errorf("%s has no field written as %s", c.typ, c.want.Go)
			continue
		}
		got.Prep = nil
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s %s =\n\t%+v\nwant\n\t%+v", c.typ, c.want.Go, got, c.want)
		}
	}
}

// TestPointersAreAllocatedOnTheWayDown checks the statements that stand in for
// what the reflective binder does with reflect.Value.Elem, since a field behind
// a pointer is only reachable once something has put a struct there.
func TestPointersAreAllocatedOnTheWayDown(t *testing.T) {
	p := plan(t)

	f, ok := fieldOf(p, "Order", "v.Ship.City")
	if !ok {
		t.Fatal("Order has no Ship.City")
	}
	want := []string{"if v.Ship == nil {", "\tv.Ship = new(Address)", "}"}
	if strings.Join(f.Prep, "\n") != strings.Join(want, "\n") {
		t.Errorf("the preparation is %q, want %q", f.Prep, want)
	}

	// A field reached without a pointer in the way needs nothing done first.
	if f, ok := fieldOf(p, "Order", "v.Address.City"); !ok || len(f.Prep) != 0 {
		t.Errorf("Address.City prepares %q, want nothing", f.Prep)
	}
}

// TestSkippedFields checks that what the tags leave alone stayed out, since a
// field that quietly binds is worse than one that quietly does not.
func TestSkippedFields(t *testing.T) {
	p := plan(t)

	for _, c := range []struct{ typ, expr, why string }{
		{"Listing", "v.Internal", `it is tagged bind:"-"`},
		{"Listing", "v.Secret", `it is tagged json:"-"`},
		{"Order", "v.hidden", "it is unexported"},
		{"Tree", "v.Parent", "the type holds one of itself"},
		{"Tree", "v.Children", "the type holds a list of itself"},
	} {
		if _, ok := fieldOf(p, c.typ, c.expr); ok {
			t.Errorf("%s binds %s, and it should not, because %s", c.typ, c.expr, c.why)
		}
	}
}

// TestVars checks the accumulators, which are the one name in the output that
// the generator makes up rather than reads.
func TestVars(t *testing.T) {
	p := plan(t)

	for _, c := range []struct {
		typ  string
		want []Var
	}{
		{"Listing", []Var{{Name: "tags", Type: "[]string"}}},
		{"Order", []Var{
			{Name: "codes", Type: "[]int"},
			{Name: "labels", Type: "[]string"},
			{Name: "sizes", Type: "[]*int"},
			{Name: "names", Type: "[]*string"},
		}},
		{"Profile", nil},
		{"Webhook", nil},
	} {
		got := structOf(t, p, c.typ).Vars
		if len(got) != len(c.want) {
			t.Errorf("%s declares %v, want %v", c.typ, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s variable %d = %v, want %v", c.typ, i, got[i], c.want[i])
			}
		}
	}
}

// TestVarNames covers the naming rules, which only show up in a struct nobody
// would write and are worth having anyway: a nested list is named after where
// it sits, and a name the function already uses is moved out of the way.
func TestVarNames(t *testing.T) {
	w := &walker{st: &Struct{Type: "Order"}}
	for _, c := range []struct{ expr, want string }{
		{"v.Tags", "tags"},
		{"v.Address.Tags", "addressTags"},
		{"v.Value", "valueList"},
		{"v.Name", "nameList"},
	} {
		if got := w.varName(c.expr); got != c.want {
			t.Errorf("varName(%q) = %q, want %q", c.expr, got, c.want)
		}
	}

	// Two lists that would take one name get told apart by a number, which is
	// the same rule the import block uses.
	w.st.Vars = []Var{{Name: "tags"}, {Name: "tags2"}}
	if got := w.varName("v.Tags"); got != "tags3" {
		t.Errorf("varName of a third Tags = %q, want tags3", got)
	}
}

// TestNameOf covers the tag rules on their own, since they are the half of the
// generator that has to agree with web.Bind exactly and the golden file only
// shows the answers for the tags testdata happens to use.
func TestNameOf(t *testing.T) {
	cases := []struct {
		tag    string
		field  string
		name   string
		src    Source
		tagged bool
		ok     bool
	}{
		{``, "PerPage", "per_page", FromValues, false, true},
		{``, "UserID", "user_id", FromValues, false, true},
		{``, "HTTPServer", "http_server", FromValues, false, true},
		{`json:"q"`, "Query", "q", FromValues, false, true},
		{`json:",omitempty"`, "Query", "query", FromValues, false, true},
		{`query:"q"`, "Query", "q", FromValues, true, true},
		{`form:"q,omitempty"`, "Query", "q", FromValues, true, true},
		{`form:""`, "Query", "query", FromValues, true, true},
		{`path:"id"`, "ID", "id", FromPath, true, true},
		{`header:"X-Trace"`, "Trace", "X-Trace", FromHeader, true, true},
		{`cookie:"seen"`, "Seen", "seen", FromCookie, true, true},
		{`path:"id" header:"X-Id"`, "ID", "id", FromPath, true, true},
		{`bind:"-"`, "Query", "", 0, false, false},
		{`json:"-"`, "Query", "", 0, false, false},
		{`query:"-"`, "Query", "", 0, false, false},
		{`bind:"-" query:"q"`, "Query", "", 0, false, false},
	}
	for _, c := range cases {
		name, src, tagged, ok := nameOf(reflect.StructTag(c.tag), c.field)
		if name != c.name || src != c.src || tagged != c.tagged || ok != c.ok {
			t.Errorf("nameOf(`%s`, %q) = %q, %v, %v, %v, want %q, %v, %v, %v",
				c.tag, c.field, name, src, tagged, ok, c.name, c.src, c.tagged, c.ok)
		}
	}
}

// TestImports checks that the generated file imports what it writes and
// nothing else, because an unused import does not compile and a missing one
// does not either.
func TestImports(t *testing.T) {
	want := []importLine{
		{Path: "net/netip"},
		{Path: "github.com/go-mizu/mizu/web"},
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

func structOf(t *testing.T, p *Plan, typ string) *Struct {
	t.Helper()
	for i := range p.Structs {
		if p.Structs[i].Type == typ {
			return &p.Structs[i]
		}
	}
	t.Fatalf("the plan has no struct called %s", typ)
	return nil
}

// fieldOf finds a field by the expression the output writes to reach it, which
// is the one thing about a field that is unique within a struct.
func fieldOf(p *Plan, typ, expr string) (Field, bool) {
	for _, s := range p.Structs {
		if s.Type != typ {
			continue
		}
		for _, f := range s.Fields {
			if f.Go == expr {
				return f, true
			}
		}
	}
	return Field{}, false
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

// runGo runs the go command inside the testdata module, which is where the
// generated code is compiled and exercised.
func runGo(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = "testdata"
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// TestGeneratedCode builds and runs the tests that live beside the generated
// file. Those are the whole point of the generator: the same requests bound by
// the generated method and by reflection, compared field for field.
//
// It shells out because the generated code is in a module of its own, which is
// what keeps a deliberately broken request struct in testdata from breaking the
// build of the generator that reads it.
func TestGeneratedCode(t *testing.T) {
	if testing.Short() {
		t.Skip("compiling the generated code takes a few seconds")
	}
	runGo(t, "test", "./...")
}
