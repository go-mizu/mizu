package archtest_test

import (
	"strings"
	"testing"

	"github.com/go-mizu/mizu/archtest"
)

// fixture is the API of the module under testdata/api.
func fixture(t *testing.T) *archtest.API {
	t.Helper()

	a, err := archtest.LoadAPI(api, "./...")
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// The roots are what a rule runs over, and a command is one of them even though
// there is nothing to type check.
func TestLoadAPIFindsTheRoots(t *testing.T) {
	got := fixture(t).Roots()

	want := []string{"mizu.test/api/cmd/tool", "mizu.test/api/deep", "mizu.test/api/store", "mizu.test/api/web"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("loaded %v, want %v", got, want)
	}
}

func TestLoadAPIDefaultsToEverything(t *testing.T) {
	a, err := archtest.LoadAPI(api)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(a.Roots()); got != 4 {
		t.Errorf("loaded %d packages, want 4: %v", got, a.Roots())
	}
}

func TestLoadAPIOnAPackageThatIsNotThere(t *testing.T) {
	if _, err := archtest.LoadAPI(api, "./nowhere/..."); err == nil {
		t.Fatal("loaded a package that does not exist")
	}
}

// A pattern that matches nothing is a warning to the go command and a rule that
// reads no code, which is the way a test passes without testing anything.
func TestLoadAPIOnAPatternThatMatchesNothing(t *testing.T) {
	_, err := archtest.LoadAPI(api, "./nothing/...")
	if err == nil {
		t.Fatal("loaded a pattern that matches no packages")
	}
	if !strings.Contains(err.Error(), "matched no packages") {
		t.Errorf("says %q, want it to say the pattern matched nothing", err)
	}
}

// The walk over a signature is where a rule is right or wrong, so every way a
// parameter can name a package gets an entry.
func TestFuncsRecordsWhatACallerHasToName(t *testing.T) {
	tests := []struct {
		pkg   string
		name  string
		needs string // packages, comma separated, or "" for none
	}{
		{"mizu.test/api/store", "Encode", ""},
		{"mizu.test/api/store", "Decode", ""},
		{"mizu.test/api/store", "Batch.Add", ""},
		{"mizu.test/api/web", "New", ""},
		{"mizu.test/api/web", "Wrap", "net/http"},
		{"mizu.test/api/web", "Handler", "mizu.test/api/store"},
		{"mizu.test/api/web", "Index", "mizu.test/api/store"},
		{"mizu.test/api/web", "Server.Add", "mizu.test/api/store"},
		{"mizu.test/api/web", "Server.Filter", "mizu.test/api/store"},
		{"mizu.test/api/deep", "Chan", "mizu.test/api/store"},
		{"mizu.test/api/deep", "Array", "mizu.test/api/store"},
		{"mizu.test/api/deep", "Variadic", "mizu.test/api/store"},
		{"mizu.test/api/deep", "Args", "mizu.test/api/store"},
		{"mizu.test/api/deep", "Anon", "mizu.test/api/store"},
		{"mizu.test/api/deep", "Iface", "mizu.test/api/store"},
		{"mizu.test/api/deep", "Aliased", "mizu.test/api/store"},
		{"mizu.test/api/deep", "Result", ""},
		{"mizu.test/api/deep", "Constrained", ""},
		{"mizu.test/api/deep", "Nothing", ""},
	}

	a := fixture(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, ok := find(a, tt.pkg, tt.name)
			if !ok {
				t.Fatalf("%s is not in the API of %s", tt.name, tt.pkg)
			}
			if got := strings.Join(f.Needs, ","); got != tt.needs {
				t.Errorf("needs %q, want %q", got, tt.needs)
			}
		})
	}
}

// What is out of the API matters as much as what is in it, since a rule that
// reads an unexported method is a rule about code nobody can call.
func TestFuncsLeavesOutWhatNobodyCanCall(t *testing.T) {
	a := fixture(t)
	for _, tt := range []struct{ pkg, name string }{
		{"mizu.test/api/store", "Batch.count"},
		{"mizu.test/api/web", "Server.drop"},
		{"mizu.test/api/deep", "hidden"},
		{"mizu.test/api/deep", "Alias.Add"},
	} {
		if _, ok := find(a, tt.pkg, tt.name); ok {
			t.Errorf("%s is in the API of %s", tt.name, tt.pkg)
		}
	}
}

// Export data is not only the exported surface, which is worth a test of its
// own because it is the assumption a reader would make. An unexported function
// small enough to inline is in there with its body, so that the packages
// importing this one can inline it, and a rule that read the package scope as
// it comes would report a requirement nobody can hit.
func TestAnInlinedFunctionIsNotInTheAPI(t *testing.T) {
	a := fixture(t)

	pkg, ok := a.Package("mizu.test/api/deep")
	if !ok {
		t.Fatal("deep is not loaded")
	}
	if pkg.Scope().Lookup("hidden") == nil {
		t.Skip("this toolchain kept hidden out of the export data, so there is nothing to leave out")
	}
	if _, ok := find(a, "mizu.test/api/deep", "hidden"); ok {
		t.Error("hidden is in the API of deep")
	}
}

func TestFuncsOfAPackageThatIsNotLoaded(t *testing.T) {
	if got := fixture(t).Funcs("mizu.test/api/nowhere"); got != nil {
		t.Errorf("found %v in a package that is not there", got)
	}
}

func TestPackageReturnsTheTypeCheckedPackage(t *testing.T) {
	a := fixture(t)

	pkg, ok := a.Package("mizu.test/api/store")
	if !ok {
		t.Fatal("store is not loaded")
	}
	if got := pkg.Scope().Lookup("Record"); got == nil {
		t.Error("store.Record is not in the package scope")
	}
	if _, ok := a.Package("mizu.test/api/nowhere"); ok {
		t.Error("found a package that is not there")
	}
}

func TestAllowOnlyPassesAPackageThatNeedsNothing(t *testing.T) {
	if got := fixture(t).AllowOnly("mizu.test/api/store", "std"); len(got) > 0 {
		t.Errorf("store cannot be used on its own: %v", got)
	}
}

func TestAllowOnlyReportsEveryFunctionThatNeedsMore(t *testing.T) {
	got := fixture(t).AllowOnly("mizu.test/api/web", "std")

	want := []string{
		"web.Handler cannot be called without mizu.test/api/store",
		"web.Index cannot be called without mizu.test/api/store",
		"web.Server.Add cannot be called without mizu.test/api/store",
		"web.Server.Filter cannot be called without mizu.test/api/store",
	}
	var lines []string
	for _, r := range got {
		lines = append(lines, r.Error())
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("reported\n%s\nwant\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

// A rule is a list of what a package is allowed to need, and naming store is
// how the toolkit's own table records that log needs config.
func TestAllowOnlyTakesAPackageThatIsAllowed(t *testing.T) {
	got := fixture(t).AllowOnly("mizu.test/api/web", "std", "mizu.test/api/store")
	if len(got) > 0 {
		t.Errorf("complained about a package that is allowed: %v", got)
	}
}

// A command has nothing anybody can call, and a rule over a whole module runs
// into one sooner or later.
func TestAPackageWithNoAPIHasNoRequirements(t *testing.T) {
	a := fixture(t)
	for _, path := range []string{"mizu.test/api/cmd/tool", "mizu.test/api/nowhere"} {
		if got := a.AllowOnly(path, "std"); len(got) > 0 {
			t.Errorf("reported %v for %s", got, path)
		}
	}
}

func find(a *archtest.API, pkg, name string) (archtest.Func, bool) {
	for _, f := range a.Funcs(pkg) {
		if f.Name == name {
			return f, true
		}
	}
	return archtest.Func{}, false
}
