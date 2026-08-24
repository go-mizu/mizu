package mizu

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/go-mizu/mizu/archtest"
)

// The toolkit depends on the standard library and on nothing else.
//
// That is a promise to whoever runs go get github.com/go-mizu/mizu: one line
// in a go.mod, no transitive graph, no upgrade to schedule, no advisory to
// read that is not ours. It is worth a lot and it is easy to lose, because
// every third-party library arrives as a reasonable pull request that solves
// a real problem.
//
// So the rule is enforced here rather than remembered. Two tests, because a
// dependency can arrive two ways. The import graph catches what the toolkit
// links. The go.mod catches what it requires, including a library that only
// the tests use, which does not reach an importer but does end up in the
// module graph.
//
// Anything that genuinely needs a third-party library goes in a module of its
// own, the way tools/milestonebot does with its YAML parser. A nested module
// keeps its dependencies to itself.
//
// To add an exception, put it in allowedModules below with a comment saying
// what it buys and why the standard library cannot, and write it up in the
// decision register. The list being empty is the point.
var allowedModules = []string{}

// allowedPatterns is the same rule stated against the import graph.
var allowedPatterns = []archtest.Pattern{"std"}

func TestModuleGraphIsStandardLibraryOnly(t *testing.T) {
	g, err := archtest.Load(".", "./...")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range g.AllowOnly(allowedPatterns...) {
		t.Errorf("outside dependency\n%s", v.Error())
	}
}

func TestGoModRequiresNothing(t *testing.T) {
	out, err := exec.Command("go", "mod", "edit", "-json").Output()
	if err != nil {
		t.Fatalf("go mod edit -json: %v", err)
	}
	var mod struct {
		Require []struct {
			Path     string
			Version  string
			Indirect bool
		}
		Replace []struct{ Old struct{ Path string } }
		Exclude []struct{ Path string }
	}
	if err := json.Unmarshal(out, &mod); err != nil {
		t.Fatalf("decode go.mod: %v", err)
	}

	allowed := map[string]bool{}
	for _, m := range allowedModules {
		allowed[m] = true
	}
	for _, r := range mod.Require {
		if allowed[r.Path] {
			continue
		}
		kind := "requires"
		if r.Indirect {
			kind = "indirectly requires"
		}
		t.Errorf("go.mod %s %s %s, and the toolkit is meant to require nothing", kind, r.Path, r.Version)
	}

	// A replace or an exclude in the toolkit's own go.mod would mean there is
	// a dependency to redirect, and there is not. Both are also ignored by
	// anybody importing the module, so a build that needs one is a build that
	// only works here.
	for _, r := range mod.Replace {
		t.Errorf("go.mod replaces %s", r.Old.Path)
	}
	for _, e := range mod.Exclude {
		t.Errorf("go.mod excludes %s", e.Path)
	}
}

// Nothing imports the composition root.
//
// github.com/go-mizu/mizu is the level that wires the pieces together. A
// package that reaches back up to it turns the toolkit into a framework you
// cannot take apart, and it is the one import that makes eject impossible.
// The dependency runs one way, always.
func TestNothingImportsTheCompositionRoot(t *testing.T) {
	g, err := archtest.Load(".", "./...")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range g.Forbid("github.com/go-mizu/mizu/...", "github.com/go-mizu/mizu") {
		t.Errorf("package imports the composition root\n%s", v.Error())
	}
}

// The tools are their own modules.
//
// tools/milestonebot parses YAML, which means it has a dependency, which is
// fine as long as it stays over there. This test says out loud what keeps it
// there, so that moving a tool into the main module is a decision somebody
// makes rather than one that happens.
func TestToolsAreNotInTheMainModule(t *testing.T) {
	g, err := archtest.Load(".", "./...")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range g.Packages() {
		if strings.HasPrefix(p, "github.com/go-mizu/mizu/tools/") {
			t.Errorf("%s is in the main module's build graph, and tools belong in their own module", p)
		}
	}
}
