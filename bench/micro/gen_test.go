package micro

import (
	"testing"

	"github.com/go-mizu/mizu/gen"
	"github.com/go-mizu/mizu/gen/bindgen"
	"github.com/go-mizu/mizu/gen/validategen"
)

// TestTheGeneratedFilesAreCurrent is mizu gen --check for the generated files
// in this module.
//
// The root module's build checks its own generated files. This module is not in
// that build, so a checked in binder or validator here could go stale against
// the generator that wrote it and the only sign would be two benchmark rows
// quietly measuring something nobody wrote. Regenerating and comparing costs
// one package load and removes that.
func TestTheGeneratedFilesAreCurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("loads packages with the go command")
	}

	pkgs, err := gen.Load(gen.Config{Dir: ".."}, "./micro/...")
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name     string
		marker   string
		generate func(...*gen.Package) ([]gen.File, error)
	}{
		{"bind", "//mizu:bind", bindgen.Generate},
		{"validate", "//mizu:validate", validategen.Generate},
	} {
		t.Run(c.name, func(t *testing.T) {
			files, err := c.generate(pkgs...)
			if err != nil {
				t.Fatal(err)
			}
			if len(files) == 0 {
				t.Fatalf("the generator wrote nothing, so the %s marker is gone", c.marker)
			}

			w := &gen.Writer{Dir: "..", Check: true}
			results, err := w.Write(files...)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range results {
				if r.Status != gen.Unchanged {
					t.Errorf("%s is %s, run mizu gen:%s ./micro/... in bench", r.Path, r.Status, c.name)
				}
			}
		})
	}
}
