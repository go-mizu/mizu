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

// The API of a fixture module with a package that cannot be picked up on its
// own. Its constructor takes a type from the package next door, so a caller
// imports two packages to use one.
const api = "testdata/api"

// A standalone rule is the same call over signatures rather than imports.
// Passing "std" and nothing else says that everything a caller has to name to
// use this package is either the package itself or the standard library.
func ExampleAPI_AllowOnly() {
	a, err := archtest.LoadAPI(api, "./...")
	if err != nil {
		panic(err)
	}
	fmt.Println(len(a.AllowOnly("mizu.test/api/store", "std")), "requirements")
	for _, r := range a.AllowOnly("mizu.test/api/web", "std") {
		fmt.Println(r.Error())
	}
	// Output:
	// 0 requirements
	// web.Handler cannot be called without mizu.test/api/store
	// web.Index cannot be called without mizu.test/api/store
	// web.Server.Add cannot be called without mizu.test/api/store
	// web.Server.Filter cannot be called without mizu.test/api/store
}

// Funcs is the whole exported surface, for a rule that wants to say something
// else about it. Needs holds the packages the parameters name, and a result
// costs a caller nothing, which is why Handler needs store and not net/http.
func ExampleAPI_Funcs() {
	a, err := archtest.LoadAPI(api, "./web")
	if err != nil {
		panic(err)
	}
	for _, f := range a.Funcs("mizu.test/api/web") {
		fmt.Println(f, f.Needs)
	}
	// Output:
	// web.Handler [mizu.test/api/store]
	// web.Index [mizu.test/api/store]
	// web.New []
	// web.Server.Add [mizu.test/api/store]
	// web.Server.Filter [mizu.test/api/store]
	// web.Wrap [net/http]
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
