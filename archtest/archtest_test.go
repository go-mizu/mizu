package archtest

import (
	"slices"
	"strings"
	"sync"
	"testing"
)

// fixture is the module under testdata/graph. Its shape is
//
//	wire -> app -> web   -> store -> encoding/json
//	            -> store         -> net/http
//
// so every question the package answers has an obvious right answer here.
const fixture = "testdata/graph"

// loaded memoises graphs by pattern. Every Load runs the go command twice, so
// without this the suite spends most of its time reloading the same four
// packages. The tests only read the graph, so sharing one is safe.
var loaded sync.Map

func load(t *testing.T, patterns ...string) *Graph {
	t.Helper()
	key := strings.Join(patterns, " ")
	if g, ok := loaded.Load(key); ok {
		return g.(*Graph)
	}
	g, err := Load(fixture, patterns...)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	loaded.Store(key, g)
	return g
}

func TestPatternMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern Pattern
		pkg     Package
		want    bool
	}{
		{"exact hit", "net/http", Package{ImportPath: "net/http"}, true},
		{"exact miss", "net/http", Package{ImportPath: "net/http/httptest"}, false},
		{"prefix covers itself", "a/b/...", Package{ImportPath: "a/b"}, true},
		{"prefix covers below", "a/b/...", Package{ImportPath: "a/b/c/d"}, true},
		{"prefix stops at a segment", "a/b/...", Package{ImportPath: "a/bc"}, false},
		{"prefix does not climb", "a/b/...", Package{ImportPath: "a"}, false},
		{"everything", "...", Package{ImportPath: "anything/at/all"}, true},
		{"std by flag", "std", Package{ImportPath: "net/http", Standard: true}, true},
		{"std is not a prefix", "std", Package{ImportPath: "std/thing"}, false},
		{"vendored std counts", "std", Package{ImportPath: "vendor/golang.org/x/net/idna", Standard: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pattern.match(&tt.pkg); got != tt.want {
				t.Errorf("Pattern(%q).match(%q) = %v, want %v", tt.pattern, tt.pkg.ImportPath, got, tt.want)
			}
		})
	}
}

func TestLoadRoots(t *testing.T) {
	g := load(t, "./...")
	want := []string{
		"mizu.test/graph/app",
		"mizu.test/graph/store",
		"mizu.test/graph/web",
		"mizu.test/graph/wire",
	}
	if got := g.Roots(); !slices.Equal(got, want) {
		t.Errorf("Roots() = %v, want %v", got, want)
	}

	// The roots are a small part of the graph, because -deps drags in
	// everything net/http reaches.
	if len(g.Packages()) <= len(want) {
		t.Errorf("Packages() returned %d entries, want more than the %d roots", len(g.Packages()), len(want))
	}
}

func TestLoadRecordsModuleAndStandard(t *testing.T) {
	g := load(t, "./...")

	own, ok := g.Lookup("mizu.test/graph/store")
	if !ok {
		t.Fatal("store is missing from the graph")
	}
	if own.Standard {
		t.Error("store is marked standard")
	}
	if own.Module != "mizu.test/graph" {
		t.Errorf("store module = %q, want mizu.test/graph", own.Module)
	}

	std, ok := g.Lookup("encoding/json")
	if !ok {
		t.Fatal("encoding/json is missing from the graph")
	}
	if !std.Standard {
		t.Error("encoding/json is not marked standard")
	}
	if std.Module != "" {
		t.Errorf("encoding/json module = %q, want empty", std.Module)
	}
}

func TestLoadDefaultsToAllPackages(t *testing.T) {
	withPatterns := load(t, "./...")
	withNone := load(t)
	if !slices.Equal(withPatterns.Roots(), withNone.Roots()) {
		t.Errorf("Load with no patterns gave %v, want the same as ./... which gave %v", withNone.Roots(), withPatterns.Roots())
	}
}

func TestLoadReportsGoCommandErrors(t *testing.T) {
	if _, err := Load(fixture, "./nope/..."); err == nil {
		t.Fatal("Load of a pattern that matches nothing returned no error")
	} else if !strings.Contains(err.Error(), "archtest:") {
		t.Errorf("error %q does not name the package", err)
	}
}

func TestDepsOf(t *testing.T) {
	g := load(t, "./...")

	deps := g.DepsOf("mizu.test/graph/store")
	if !slices.Contains(deps, "encoding/json") {
		t.Errorf("store does not reach encoding/json, got %v", deps)
	}
	if slices.Contains(deps, "net/http") {
		t.Error("store reaches net/http, which only web imports")
	}

	// Transitive, not direct: app imports web, and web imports net/http.
	if !slices.Contains(g.DepsOf("mizu.test/graph/app"), "net/http") {
		t.Error("app does not reach net/http through web")
	}

	if got := g.DepsOf("mizu.test/graph/nothing"); got != nil {
		t.Errorf("DepsOf on an unknown package = %v, want nil", got)
	}
}

func TestChain(t *testing.T) {
	g := load(t, "./...")

	tests := []struct {
		name     string
		from, to string
		want     []string
	}{
		{
			name: "direct",
			from: "mizu.test/graph/web", to: "mizu.test/graph/store",
			want: []string{"mizu.test/graph/web", "mizu.test/graph/store"},
		},
		{
			name: "through one hop",
			from: "mizu.test/graph/wire", to: "mizu.test/graph/web",
			want: []string{"mizu.test/graph/wire", "mizu.test/graph/app", "mizu.test/graph/web"},
		},
		{
			name: "same package",
			from: "mizu.test/graph/app", to: "mizu.test/graph/app",
			want: []string{"mizu.test/graph/app"},
		},
		{
			name: "unreachable, since nothing imports wire",
			from: "mizu.test/graph/app", to: "mizu.test/graph/wire",
			want: nil,
		},
		{
			name: "unknown start",
			from: "mizu.test/graph/nothing", to: "mizu.test/graph/app",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := g.Chain(tt.from, tt.to); !slices.Equal(got, tt.want) {
				t.Errorf("Chain(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestChainIsShortest(t *testing.T) {
	g := load(t, "./...")
	// app imports store directly and also reaches it through web. The direct
	// edge is the one worth reporting.
	want := []string{"mizu.test/graph/app", "mizu.test/graph/store"}
	if got := g.Chain("mizu.test/graph/app", "mizu.test/graph/store"); !slices.Equal(got, want) {
		t.Errorf("Chain took the long way: %v", got)
	}
}

func TestAllowOnlyPassesForTheFixture(t *testing.T) {
	// Every non-standard package the fixture reaches is one of its own, so
	// "std" alone is enough. This is the same shape as the toolkit's rule.
	g := load(t, "./...")
	if v := g.AllowOnly("std"); len(v) != 0 {
		t.Errorf("AllowOnly(std) found %d violations, want none:\n%s", len(v), join(v))
	}
}

func TestAllowOnlyTreatsNonRootsAsOutside(t *testing.T) {
	// Loading one package makes its siblings outsiders, which is how a rule
	// about one layer of a repository gets written.
	g := load(t, "./app")

	got := map[string]bool{}
	for _, v := range g.AllowOnly("std") {
		if v.Package != "mizu.test/graph/app" {
			t.Errorf("violation blamed %q, want app", v.Package)
		}
		got[v.Dep] = true
	}
	for _, want := range []string{"mizu.test/graph/store", "mizu.test/graph/web"} {
		if !got[want] {
			t.Errorf("AllowOnly did not report %q", want)
		}
	}
	if len(got) != 2 {
		t.Errorf("AllowOnly reported %d deps, want 2: %v", len(got), got)
	}
}

func TestAllowOnlyWithNoPatterns(t *testing.T) {
	// No patterns means no dependencies allowed at all. store reaches the
	// standard library, so it has to fail.
	g := load(t, "./store")
	if v := g.AllowOnly(); len(v) == 0 {
		t.Error("AllowOnly with no patterns allowed the standard library")
	}
}

func TestAllowOnlyCarriesTheChain(t *testing.T) {
	g := load(t, "./app")
	for _, v := range g.AllowOnly("std") {
		if v.Dep != "mizu.test/graph/store" {
			continue
		}
		want := []string{"mizu.test/graph/app", "mizu.test/graph/store"}
		if !slices.Equal(v.Chain, want) {
			t.Errorf("Chain = %v, want %v", v.Chain, want)
		}
		if !strings.Contains(v.Error(), "mizu.test/graph/store") {
			t.Errorf("Error() = %q, does not name the dependency", v.Error())
		}
		return
	}
	t.Fatal("no violation reported for store")
}

func TestForbid(t *testing.T) {
	g := load(t, "./...")

	// The rule the toolkit cares about: nothing imports the composition root.
	if v := g.Forbid("mizu.test/graph/...", "mizu.test/graph/wire"); len(v) != 0 {
		t.Errorf("something reaches wire:\n%s", join(v))
	}

	// A layering rule that the fixture breaks on purpose.
	v := g.Forbid("mizu.test/graph/store", "net/http")
	if len(v) != 0 {
		t.Errorf("store reaches net/http:\n%s", join(v))
	}

	v = g.Forbid("mizu.test/graph/web", "net/http")
	if len(v) != 1 {
		t.Fatalf("Forbid found %d violations, want 1:\n%s", len(v), join(v))
	}
	if v[0].Package != "mizu.test/graph/web" || v[0].Dep != "net/http" {
		t.Errorf("violation = %s, want web depends on net/http", v[0].Error())
	}
	if len(v[0].Chain) < 2 {
		t.Errorf("chain = %v, want at least two entries", v[0].Chain)
	}
}

func TestForbidMatchesEveryPairing(t *testing.T) {
	g := load(t, "./...")
	// Three packages reach net/http: web directly, app and wire through it.
	v := g.Forbid("mizu.test/graph/...", "net/http")
	if len(v) != 3 {
		t.Fatalf("Forbid found %d violations, want 3:\n%s", len(v), join(v))
	}
	for _, got := range v {
		if got.Dep != "net/http" {
			t.Errorf("violation names %q, want net/http", got.Dep)
		}
	}
}

func TestViolationErrorWithoutAChain(t *testing.T) {
	v := Violation{Package: "a", Dep: "b"}
	if got, want := v.Error(), "a depends on b"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func join(vs []Violation) string {
	var b strings.Builder
	for _, v := range vs {
		b.WriteString(v.Error())
		b.WriteByte('\n')
	}
	return b.String()
}
