package micro

import (
	"testing"

	"github.com/go-mizu/mizu/gen"
	"github.com/go-mizu/mizu/gen/bindgen"
)

// TestGeneratedBinderIsCurrent is mizu gen --check for the one generated file
// in this module.
//
// The root module's build checks its own generated files. This module is not in
// that build, so a checked in binder here could go stale against the generator
// that wrote it and the only sign would be two benchmark rows quietly measuring
// something nobody wrote. Regenerating and comparing costs a package load and
// removes that.
func TestGeneratedBinderIsCurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("loads packages with the go command")
	}

	pkgs, err := gen.Load(gen.Config{Dir: ".."}, "./micro/...")
	if err != nil {
		t.Fatal(err)
	}
	files, err := bindgen.Generate(pkgs...)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("the generator wrote nothing, so the //mizu:bind marker is gone")
	}

	w := &gen.Writer{Dir: "..", Check: true}
	results, err := w.Write(files...)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range results {
		if r.Status != gen.Unchanged {
			t.Errorf("%s is %s, run mizu gen:bind ./micro/... in bench", r.Path, r.Status)
		}
	}
}
