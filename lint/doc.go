// Package lint reads a project's source and reports the mistakes a compiler
// cannot.
//
// Every check here is about a rule some mizu package makes that the type system
// does not: a pointer that must not be kept, a call that must happen before
// another, an argument that has to be a constant. A rule like that is a comment
// in a doc string until something reads the code and says where it was broken,
// and this is that something.
//
//	pkgs, err := gen.Load(gen.Config{Dir: dir}, "./...")
//	if err != nil {
//		return err
//	}
//	found, err := lint.Run(pkgs)
//	if err != nil {
//		return err
//	}
//	return diag.Text(os.Stdout, found)
//
// mizu lint is this package with a command around it, and mizu verify runs it
// as a stage, so a rule broken in an editor is a rule somebody hears about
// before the tests do.
//
// # What a check may report
//
// A check reports what it is sure of. Everything here reads types rather than
// names, and nothing here follows a value through an interface, an any, or a
// closure that was passed somewhere else. That is a deliberate floor: a linter
// that is right about what it says gets fixed, and a linter that guesses gets
// turned off.
//
// So a check missing something is expected and a check inventing something is a
// bug. The other half of the rule is the guarded build, which catches at run
// time what reading the source cannot, and the two are meant to be used
// together rather than instead of each other.
//
// # Adding one
//
// A check is a name, a sentence saying what it is for, and a function over one
// loaded package. Register it in checks, take a code from the table in
// [github.com/go-mizu/mizu/errs/diag], and add a directory to testdata/_diag
// holding the source that breaks the rule and what the report looks like.
// TestEveryMessageHasACase fails until the last of those is there.
package lint
