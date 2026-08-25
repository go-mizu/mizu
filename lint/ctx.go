package lint

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/go-mizu/mizu/errs/diag"
	"github.com/go-mizu/mizu/gen"
)

// The web package the check is about. Naming it by import path rather than by
// the identifier in the file is what makes an alias, a dot import and a local
// type called web all read correctly.
const (
	webPath = "github.com/go-mizu/mizu/web"
	ctxName = "Ctx"
)

// checkCtx reports a *web.Ctx that outlives the handler it belongs to.
//
// A Ctx comes from a pool and goes back when the handler returns, so a pointer
// kept past that point is either a use after free or, worse, a live pointer
// into somebody else's request. The web package comment is the rule and this is
// the reading of it.
//
// Four shapes are reported, and each of them is a declaration rather than a
// flow. A field of that type, a result of that type, a channel of that type and
// a package level variable of that type are all wrong wherever they appear, so
// none of this has to work out where a value came from or where it went. What
// that misses is a Ctx put in an any, and what catches that is the guarded
// build.
//
// The web package is skipped. It is where the pool is, so the shapes this check
// is against are the shapes it is made of.
func checkCtx(pkg *gen.Package) diag.List {
	if pkg.PkgPath == webPath {
		return nil
	}

	c := ctxChecker{pkg: pkg}
	for _, f := range pkg.Syntax {
		ast.Inspect(f, c.node)
	}
	return c.found
}

// A ctxChecker is one run of the ctx check over one package.
type ctxChecker struct {
	pkg   *gen.Package
	found diag.List
}

// node looks at one piece of syntax.
func (c *ctxChecker) node(n ast.Node) bool {
	switch n := n.(type) {
	case *ast.ChanType:
		if c.holds(n.Value) {
			c.report(n, "MZ3003",
				"a channel carries a *web.Ctx, and whatever reads it is not the handler that owns it",
				"send what the reader needs instead",
				"take the values out of the Ctx and send those, or send the context from Ctx.Detach")
		}
	case *ast.StructType:
		c.fields(n.Fields, "MZ3001",
			"a struct field holds a *web.Ctx, which stops being valid when the handler returns",
			"this field outlives the request",
			"keep what the field is for instead: the request id, the user, or the context from Ctx.Context")
	case *ast.FuncType:
		c.fields(n.Results, "MZ3002",
			"a function returns a *web.Ctx, which stops being valid when the handler that made it returns",
			"this pointer escapes the handler",
			"return what the caller needs, or take a *web.Ctx as an argument so the handler still owns it")
	case *ast.GenDecl:
		c.globals(n)
	case *ast.GoStmt:
		c.goroutine(n)
	}
	return true
}

// fields reports the fields of a struct or the results of a signature that hold
// a Ctx.
//
// A field written as a channel is left to the channel case, which has more to
// say about it and has already seen the same piece of syntax.
func (c *ctxChecker) fields(list *ast.FieldList, code diag.Code, message, detail, fix string) {
	if list == nil {
		return
	}
	for _, f := range list.List {
		if _, isChan := f.Type.(*ast.ChanType); isChan {
			continue
		}
		if c.holds(f.Type) {
			c.report(f.Type, code, message, detail, fix)
		}
	}
}

// globals reports a package level variable that holds a Ctx.
//
// A var inside a function is not one of these: it belongs to the call, which is
// the handler, and it goes away with it. What tells them apart is the scope the
// name was declared in.
func (c *ctxChecker) globals(d *ast.GenDecl) {
	if d.Tok != token.VAR {
		return
	}
	for _, spec := range d.Specs {
		v, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range v.Names {
			obj, ok := c.pkg.TypesInfo.Defs[name].(*types.Var)
			if !ok || obj.Parent() != c.pkg.Types.Scope() {
				continue
			}
			if holds(obj.Type()) {
				c.report(name, "MZ3004",
					"a package level variable holds a *web.Ctx, and the next request will find the last one there",
					"this outlives every request",
					"pass the Ctx to what needs it, or use web.FromContext where a context is all there is")
			}
		}
	}
}

// goroutine reports a go statement that uses a Ctx from the handler that
// started it.
//
// The handler does not wait for it, so the Ctx is back in the pool while the
// goroutine is still reading it. A name declared inside the statement is fine,
// which is what tells this apart from a goroutine that makes its own.
//
// One report per variable rather than per mention. A goroutine that reads the
// Ctx four times has one thing wrong with it, and four carets under four
// halves of the same line is a report people skim.
func (c *ctxChecker) goroutine(g *ast.GoStmt) {
	seen := make(map[*types.Var]bool)
	ast.Inspect(g.Call, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		obj, ok := c.pkg.TypesInfo.Uses[id].(*types.Var)
		if !ok || !holds(obj.Type()) {
			return true
		}
		if g.Pos() <= obj.Pos() && obj.Pos() < g.End() {
			return true
		}
		if seen[obj] {
			return true
		}
		seen[obj] = true
		c.report(id, "MZ3005",
			"a go statement uses a *web.Ctx from the handler that started it, and the handler does not wait",
			"the handler may return before this runs",
			"take what the goroutine needs out of the Ctx first, and pass Ctx.Detach for the context")
		return true
	})
}

// holds reports whether an expression's type is, or directly contains, a
// *web.Ctx.
func (c *ctxChecker) holds(e ast.Expr) bool {
	return holds(c.pkg.TypesInfo.TypeOf(e))
}

// holds reports whether a value of t is, or directly contains, a *web.Ctx.
//
// Directly is the whole of it: through a pointer, a slice, an array, a map or a
// channel, and no further. A named type that holds one is not one, because the
// place that named type declares the field is where the report belongs and the
// check has already been there.
func holds(t types.Type) bool {
	switch t := types.Unalias(t).(type) {
	case *types.Pointer:
		return isCtx(t.Elem())
	case *types.Slice:
		return holds(t.Elem())
	case *types.Array:
		return holds(t.Elem())
	case *types.Map:
		return holds(t.Key()) || holds(t.Elem())
	case *types.Chan:
		return holds(t.Elem())
	}
	return false
}

// isCtx reports whether t is web.Ctx itself.
func isCtx(t types.Type) bool {
	n, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := n.Obj()
	return obj != nil && obj.Name() == ctxName && obj.Pkg() != nil && obj.Pkg().Path() == webPath
}

// report adds one finding.
func (c *ctxChecker) report(n ast.Node, code diag.Code, message, detail, fix string) {
	file, span := at(c.pkg.Fset, n)
	c.found = append(c.found, diag.Diagnostic{
		Code:     code,
		Severity: diag.Error,
		Message:  message,
		File:     file,
		Range:    span,
		Detail:   detail,
		Fix:      fix,
	})
}
