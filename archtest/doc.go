// Package archtest asserts things about a module's import graph and its
// exported API.
//
// It answers three questions that are easy to state and hard to keep true by
// hand. What may this package depend on, what must it never reach, and what
// does somebody have to import before they can call it? All three are
// properties a repository loses gradually, one convenient import at a time, so
// they belong in a test rather than in a review checklist.
//
// The graph comes from the go command, so what is asserted here is what the
// compiler actually links rather than a model of it.
//
//	g, err := archtest.Load(".", "./...")
//	if err != nil {
//		t.Fatal(err)
//	}
//	for _, v := range g.AllowOnly("std") {
//		t.Error(v)
//	}
//
// That is the rule the toolkit holds itself to. The standard library and
// nothing else, so that go get github.com/go-mizu/mizu adds one line to a
// go.mod and pulls in no transitive graph at all. Anything that genuinely
// needs a third-party library lives in a separate module, the way
// tools/milestonebot does.
//
// A violation carries the chain that produced it, because the useful half of
// "cache reaches gopkg.in/yaml.v3" is which four imports got it there.
//
// The third question is about signatures rather than imports, and it needs the
// type checker.
//
//	a, err := archtest.LoadAPI(".", "./log")
//	if err != nil {
//		t.Fatal(err)
//	}
//	for _, r := range a.AllowOnly("github.com/go-mizu/mizu/log", "std") {
//		t.Error(r)
//	}
//
// That is the standalone rule from doc 35. A package whose constructor takes a
// type from three packages away is four packages to adopt rather than one, and
// the import graph says nothing about it, because the imports run the other
// way: log imports config, and it is the caller who ends up holding both.
//
// Package archtest is an implementation detail of mizu's own tests and is
// exempt from the compatibility promise in doc 31. Import it only if you are
// extending mizu itself.
package archtest
