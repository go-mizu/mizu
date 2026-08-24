// Package gen is the code generation harness.
//
// It loads a module's packages with syntax and type information, hands them
// to generators, and writes the result. This file describes the loading half.
// Nothing here generates anything on its own.
//
//	pkgs, err := gen.Load(gen.Config{Dir: "."}, "./...")
//	if err != nil {
//		return err
//	}
//	for _, p := range pkgs {
//		for _, f := range p.Syntax {
//			// walk the file, read markers, look types up in p.TypesInfo
//		}
//	}
//
// # Markers
//
// Files are parsed with comments, because markers live in doc comments. One
// thing to know before writing the code that reads them: a marker like
//
//	//mizu:table users
//
// is a directive comment, and [go/ast.CommentGroup.Text] strips directives on
// the way out. The marker is in the parsed file, it is not in Text. Walk
// Doc.List instead.
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
