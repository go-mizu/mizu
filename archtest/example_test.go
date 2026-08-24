package archtest_test

import (
	"fmt"

	"github.com/go-mizu/mizu/archtest"
)

// The graph loaded here is a fixture with four packages. A real test would
// load "." and "./...", which is the module the test itself lives in.
const module = "testdata/graph"

// A dependency rule is one call. Passing "std" and nothing else says that the
// module depends on the standard library and on itself, which is the rule the
// toolkit holds itself to.
func ExampleGraph_AllowOnly() {
	g, err := archtest.Load(module, "./...")
	if err != nil {
		panic(err)
	}
	fmt.Println(len(g.AllowOnly("std")), "violations")
	// Output: 0 violations
}

// Forbid is the other half. It answers "does anything reach this", which is
// how a composition root stays a leaf that nobody imports.
func ExampleGraph_Forbid() {
	g, err := archtest.Load(module, "./...")
	if err != nil {
		panic(err)
	}
	for _, v := range g.Forbid("mizu.test/graph/...", "mizu.test/graph/store") {
		fmt.Println(v.Package, "->", v.Dep)
	}
	// Output:
	// mizu.test/graph/app -> mizu.test/graph/store
	// mizu.test/graph/web -> mizu.test/graph/store
	// mizu.test/graph/wire -> mizu.test/graph/store
}

// A violation prints with the chain that produced it, which is the part
// somebody can act on.
func ExampleViolation_Error() {
	g, err := archtest.Load(module, "./wire")
	if err != nil {
		panic(err)
	}
	for _, v := range g.AllowOnly("std") {
		if v.Dep == "mizu.test/graph/store" {
			fmt.Println(v.Error())
		}
	}
	// Output:
	// mizu.test/graph/wire depends on mizu.test/graph/store
	//	mizu.test/graph/wire
	//	  -> mizu.test/graph/app
	//	  -> mizu.test/graph/store
}
