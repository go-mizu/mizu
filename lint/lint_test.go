package lint

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
	"github.com/go-mizu/mizu/gen"
)

func TestChecksHandsBackACopy(t *testing.T) {
	got := Checks()
	if len(got) != len(checks) {
		t.Fatalf("Checks returned %d checks and there are %d", len(got), len(checks))
	}

	got[0].Name = "something else"
	if checks[0].Name == "something else" {
		t.Fatal("Checks handed back the package's own slice, so a caller can rename a check for everybody")
	}
}

func TestEveryCheckSaysWhatItIsFor(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range checks {
		switch {
		case c.Name == "":
			t.Error("a check has no name, so there is nothing to type after mizu lint")
		case seen[c.Name]:
			t.Errorf("%s is the name of two checks, and naming it runs one of them", c.Name)
		case c.Doc == "":
			t.Errorf("%s has no doc, so the help says nothing about it", c.Name)
		case c.Run == nil:
			t.Errorf("%s has nothing to run", c.Name)
		}
		seen[c.Name] = true
	}
}

func TestNamingNoCheckRunsAllOfThem(t *testing.T) {
	got, err := pick(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(checks) {
		t.Fatalf("naming no check picked %d of %d", len(got), len(checks))
	}
}

func TestNamingACheckRunsThatOne(t *testing.T) {
	got, err := pick([]string{"ctx"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "ctx" {
		t.Fatalf("naming ctx picked %v", picked(got))
	}
}

func TestATypoIsAnError(t *testing.T) {
	_, err := pick([]string{"ctxs"})
	if err == nil {
		t.Fatal("a check name nothing matches was accepted, so a typo runs nothing and passes")
	}
	if !strings.Contains(err.Error(), "ctx") {
		t.Errorf("the message does not say what the checks are: %v", err)
	}
}

// TestAPackageThatDidNotTypeCheckIsSkipped is about the run somebody has in
// front of them rather than a run in a test: an editor loads a package while
// it is being typed, and half a type graph is what a check would be reading.
func TestAPackageThatDidNotTypeCheckIsSkipped(t *testing.T) {
	for _, p := range []*gen.Package{
		{PkgPath: "app"},
		{PkgPath: "app", Types: mustLoad(t, "clean").Types},
	} {
		found, err := Run([]*gen.Package{p})
		if err != nil {
			t.Fatal(err)
		}
		if len(found) != 0 {
			t.Errorf("a package with nothing in it produced %d diagnostics", len(found))
		}
	}
}

// TestTheReportIsSorted is what makes two runs over the same source read the
// same way, whatever order the go command listed the packages in.
func TestTheReportIsSorted(t *testing.T) {
	found, err := Run([]*gen.Package{mustLoad(t, "deep")})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) < 2 {
		t.Fatalf("testdata/deep produced %d diagnostics, so this is not testing an order", len(found))
	}
	for i := 1; i < len(found); i++ {
		if before(found[i], found[i-1]) {
			t.Errorf("diagnostic %d is at %s and comes after one at %s", i,
				found[i].Range.Start, found[i-1].Range.Start)
		}
	}
}

func before(a, b diag.Diagnostic) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Range.Start.Line != b.Range.Start.Line {
		return a.Range.Start.Line < b.Range.Start.Line
	}
	return a.Range.Start.Col < b.Range.Start.Col
}

// TestARangeStaysOnOneLine is what keeps a report readable when the thing it
// points at is a struct that runs down the page.
func TestARangeStaysOnOneLine(t *testing.T) {
	fset := token.NewFileSet()
	f := fset.AddFile("app.go", 1, 100)
	f.SetLines([]int{0, 10, 20})

	one := &ast.Ident{NamePos: token.Pos(3), Name: "abc"}
	if _, r := at(fset, one); !r.IsValid() || r.End.Col != 6 {
		t.Errorf("a name on one line got the range %v, and it spans the name", r)
	}

	two := &ast.StructType{Struct: token.Pos(3), Fields: &ast.FieldList{Closing: token.Pos(25)}}
	if _, r := at(fset, two); !r.IsValid() || r.End.IsValid() {
		t.Errorf("something that runs across lines got the range %v, and it points at the start", r)
	}
}

// picked is what a failed pick prints, so a failure names the checks it got
// rather than a slice of structs.
func picked(list []Check) []string {
	var out []string
	for _, c := range list {
		out = append(out, c.Name)
	}
	return out
}

// mustLoad loads one ordinary package out of the test module.
func mustLoad(t *testing.T, name string) *gen.Package {
	t.Helper()

	dir, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	pkgs, err := gen.Load(gen.Config{Dir: dir}, "./"+name)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("./%s loaded as %d packages", name, len(pkgs))
	}
	if err := pkgs[0].Err(); err != nil {
		t.Fatalf("testdata/%s did not load: %v", name, err)
	}
	return pkgs[0]
}
