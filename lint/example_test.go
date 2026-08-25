package lint_test

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/go-mizu/mizu/gen"
	"github.com/go-mizu/mizu/lint"
)

// Reading a package and reporting what is wrong with it, which is what mizu
// lint does with a command around it.
//
// The package here keeps a *web.Ctx in a slice, in an array and in a map. All
// three outlive the handler the Ctx belongs to, and none of the three is
// something a compiler has an opinion about.
func Example() {
	pkgs, err := gen.Load(gen.Config{Dir: "testdata"}, "./deep")
	if err != nil {
		log.Fatal(err)
	}

	found, err := lint.Run(pkgs)
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range found {
		fmt.Printf("%s %s:%s %s\n", d.Code, filepath.Base(d.File), d.Range.Start, d.Message)
	}

	// Output:
	// MZ3001 holders.go:10:8 a struct field holds a *web.Ctx, which stops being valid when the handler returns
	// MZ3001 holders.go:11:8 a struct field holds a *web.Ctx, which stops being valid when the handler returns
	// MZ3001 holders.go:12:8 a struct field holds a *web.Ctx, which stops being valid when the handler returns
}

// Naming the checks a run covers.
func ExampleChecks() {
	for _, c := range lint.Checks() {
		fmt.Printf("%-4s %s\n", c.Name, c.Doc)
	}

	// Output:
	// ctx  A *web.Ctx that outlives the handler it belongs to
}
