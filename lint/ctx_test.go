package lint

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/go-mizu/mizu/errs/diag"
	"github.com/go-mizu/mizu/gen"
)

// TestTheRightUseOfACtxIsQuiet is the half of a linter that decides whether
// anybody keeps it turned on.
//
// testdata/clean holds the shapes somebody reaches for when what they wanted
// is the shape the check reports: a local name for the Ctx, a middleware that
// takes one and gives one back, a package level map of what came out of
// requests, a channel of that, and a goroutine handed the context from
// Ctx.Detach. A check that reports any of them is wrong.
func TestTheRightUseOfACtxIsQuiet(t *testing.T) {
	found, err := Run([]*gen.Package{mustLoad(t, "clean")})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range found {
		t.Errorf("%s: %s", d.Range.Start, d.Message)
	}
}

// TestTheWebPackageIsSkipped is not a formality. The web package is where the
// pool is, so a function returning a *web.Ctx and a struct holding one are
// what it is made of, and running the check over it without the skip reports
// most of the file.
func TestTheWebPackageIsSkipped(t *testing.T) {
	web := mustLoadFrom(t, "..", "./web")

	if found := checkCtx(web); len(found) != 0 {
		t.Fatalf("the web package produced %d diagnostics about itself", len(found))
	}

	// And the skip is the reason, rather than the check having nothing to say
	// about that source. Renaming the package is enough to bring it back.
	web.PkgPath = "example.com/app"
	if found := checkCtx(web); len(found) == 0 {
		t.Fatal("the same source under another import path produced nothing, so the skip is not what made it quiet")
	}
}

// TestACtxInsideSomethingElseIsFound is the part of holds that the corpus does
// not reach: a slice, an array and a map each hold the pointer as surely as a
// field does.
func TestACtxInsideSomethingElseIsFound(t *testing.T) {
	found, err := Run([]*gen.Package{mustLoad(t, "deep")})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 3 {
		t.Fatalf("testdata/deep has three fields holding a Ctx and produced %d diagnostics:\n%s", len(found), lines(found))
	}
	for _, d := range found {
		if d.Code != "MZ3001" {
			t.Errorf("a field holding a Ctx was reported as %s", d.Code)
		}
	}
}

// TestOneMistakeInMoreThanOneShape covers what the corpus cannot say in one
// case each: a channel written as a field is a channel and not a field, a
// package level channel is both and is reported twice, and a pointer to
// something that is not a Ctx is nobody's business.
func TestOneMistakeInMoreThanOneShape(t *testing.T) {
	found, err := Run([]*gen.Package{mustLoad(t, "awkward")})
	if err != nil {
		t.Fatal(err)
	}

	found.Sort()
	var codes []diag.Code
	for _, d := range found {
		codes = append(codes, d.Code)
	}
	// In the order a person reads them: the field, then the variable and the
	// channel it is, which are two reports about the same line.
	want := []diag.Code{"MZ3003", "MZ3004", "MZ3003"}
	if !slices.Equal(codes, want) {
		t.Errorf("testdata/awkward produced %v, want %v:\n%s", codes, want, lines(found))
	}
}

// TestEveryDiagnosticIsWorthPrinting holds the fields a person reads to the
// shape doc 36 asks for. The corpus checks the wording; this checks that the
// fields are filled in at all, on every diagnostic the checks can produce
// rather than on the ones a case happens to reach.
func TestEveryDiagnosticIsWorthPrinting(t *testing.T) {
	var found diag.List
	for _, name := range []string{"deep", "clean"} {
		list, err := Run([]*gen.Package{mustLoad(t, name)})
		if err != nil {
			t.Fatal(err)
		}
		found = append(found, list...)
	}
	_, pkgs := loadCorpus(t)
	for _, p := range pkgs {
		list, err := Run([]*gen.Package{p})
		if err != nil {
			t.Fatal(err)
		}
		found = append(found, list...)
	}

	if len(found) == 0 {
		t.Fatal("nothing was reported, so this checked nothing")
	}
	for i, d := range found {
		switch {
		case d.Severity != diag.Error:
			t.Errorf("diagnostic %d is at severity %s, and a rule that is broken is broken", i, d.Severity)
		case d.Code == "":
			t.Errorf("diagnostic %d carries no code, so mizu explain has nothing to look up", i)
		case d.File == "":
			t.Errorf("diagnostic %d names no file", i)
		case !d.Range.IsValid():
			t.Errorf("diagnostic %d has no place in the file", i)
		case d.Detail == "":
			t.Errorf("diagnostic %d has no label for under the carets", i)
		case d.Fix == "":
			t.Errorf("diagnostic %d says what is wrong and not what to do instead", i)
		}
	}
}

// lines is a list of diagnostics as something to put in a failure.
func lines(l diag.List) string {
	var out string
	for _, d := range l {
		out += "  " + d.File + ":" + d.Range.Start.String() + ": " + d.Message + "\n"
	}
	return out
}

// mustLoadFrom loads one package out of the module rooted at dir.
func mustLoadFrom(t *testing.T, dir, pattern string) *gen.Package {
	t.Helper()

	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	pkgs, err := gen.Load(gen.Config{Dir: abs}, pattern)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("%s loaded as %d packages", pattern, len(pkgs))
	}
	if err := pkgs[0].Err(); err != nil {
		t.Fatalf("%s did not load: %v", pattern, err)
	}
	return pkgs[0]
}
