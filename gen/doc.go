// Package gen is the code generation harness.
//
// It loads a module's packages with syntax and type information, finds the
// declarations that asked to be generated for, hands them to generators, and
// writes the result. Nothing here generates anything on its own.
//
//	pkgs, err := gen.Load(gen.Config{Dir: "."}, "./...")
//	if err != nil {
//		return err
//	}
//	targets, errs := gen.Scan(pkgs...)
//	for _, t := range targets {
//		for _, m := range t.Markers {
//			// m.Name says which generator, m.Get reads its arguments,
//			// and t.Object is the declaration to generate from
//		}
//	}
//
// # Markers
//
// A marker is a directive in a doc comment that says which generator wants a
// declaration and what it should do with it.
//
//	//mizu:model table=posts
//	//mizu:rpc method=POST path=/v1/orders ability=order.create
//	//mizu:command name="users:prune" standalone
//
// [Scan] finds them. It reads the package comment, functions and methods,
// types, constants and variables, struct fields, and interface methods, and
// returns a [Target] for each declaration that carried one, paired with the
// [go/types.Object] it declares.
//
// Two slashes, then the name, with no space. That is Go's own rule for a
// directive and it is not a detail: [go/ast.CommentGroup.Text] strips
// directives and keeps sentences, which is how it tells them apart. A marker
// written with a space is a sentence, and gets reported rather than skipped,
// because a marker that silently does nothing is a bad afternoon.
//
// The same rule is why anything reading markers off an [go/ast.CommentGroup]
// by hand has to walk Doc.List. The marker is in the parsed file. It is not in
// Text.
//
// # Why not go/packages
//
// golang.org/x/tools/go/packages does this job with a nicer API. The core
// module of mizu requires nothing outside the standard library, so the parts
// that are actually needed are built here on go list, go/parser, and
// go/types. That is what go/packages does underneath.
//
// Two things in the specification got simpler as a result rather than harder.
// An overlay is a map from filename to source text that Load checks before
// reading from disk, which is a few lines instead of a mode flag. And import
// blocks in generated files are written directly, sorted by path, because a
// generator knows its own imports; resolving them by search was always the
// wrong tool for output produced here.
//
// # The bootstrap problem
//
// Generated files are part of the package being loaded. Rename a field and
// the generated file that mentions it stops compiling, so the package fails
// to type-check, so the generator that would fix it cannot run.
//
// Load reports type errors on the package rather than refusing to return it,
// and it applies an overlay before parsing. Together those let a caller load
// once, notice the errors, and load again with the generated files replaced
// by empty stubs. Extraction only needs the hand-written declarations, so the
// second load succeeds and the generator regenerates the file that was
// broken. That is the most common edit loop there is, and a generator that
// cannot survive it is not much use.
//
// # Exemption
//
// Package gen is an implementation detail of the mizu toolchain and is exempt
// from the compatibility promise in doc 31. It is not under internal/ because
// this repository has no internal/, not because the API is stable.
package gen
