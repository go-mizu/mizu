package configgen

import (
	"go/types"
)

// configPkg is the runtime the generated code calls.
const configPkg = "github.com/go-mizu/mizu/config"

// A parser is how the output reads one field and how it writes it back out.
type parser struct {
	// expr is the parser, as config.Duration or config.Slice(config.Prefix).
	expr string

	// show is the call that turns the field into a line of text, which is
	// config.Show for a single value and config.ShowSlice or config.ShowMap
	// for the two that hold more than one.
	show string

	// generic says the expression still has a type parameter to work out. Go
	// settles the type arguments of a call before it looks at what the result
	// is used as, so a generic parser nested inside Slice or Map has to name
	// its type, while one handed straight to Get does not.
	generic bool
}

// parserFor is how the output reads a field of this type, or nil when nothing
// reads it. It records the packages the answer needs along the way.
func parserFor(t types.Type, imps *imports) *parser {
	cfg := imps.name(configPkg)
	show := cfg + ".Show"

	// A named type is asked about itself first, so time.Duration is a length
	// of time rather than the int64 it is made of.
	if p := byName(t, cfg, show); p != nil {
		return p
	}
	if p := byMethod(t, cfg, show); p != nil {
		return p
	}

	switch u := t.Underlying().(type) {
	case *types.Basic:
		return byKind(u, cfg, show)

	case *types.Slice:
		if isByte(u.Elem()) {
			return &parser{expr: cfg + ".Bytes", show: show}
		}
		elem := parserFor(u.Elem(), imps)
		if elem == nil {
			return nil
		}
		return &parser{
			expr: cfg + ".Slice(" + instantiate(elem, u.Elem(), imps) + ")",
			show: cfg + ".ShowSlice",
		}

	case *types.Map:
		if b, ok := u.Key().Underlying().(*types.Basic); !ok || b.Kind() != types.String {
			return nil // a table in a file is keyed by a name and nothing else
		}
		elem := parserFor(u.Elem(), imps)
		if elem == nil {
			return nil
		}
		return &parser{
			expr: cfg + ".Map(" + instantiate(elem, u.Elem(), imps) + ")",
			show: cfg + ".ShowMap",
		}
	}
	return nil
}

// byName covers the types the config package has a parser named after.
func byName(t types.Type, cfg, show string) *parser {
	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return nil
	}
	switch named.Obj().Pkg().Path() + "." + named.Obj().Name() {
	case "time.Duration":
		// Duration reads any ~int64, so it is one of the generic ones.
		return &parser{expr: cfg + ".Duration", show: show, generic: true}
	case "time.Time":
		return &parser{expr: cfg + ".Time", show: show}
	case "log/slog.Level":
		return &parser{expr: cfg + ".Level", show: show}
	case "net/netip.Addr":
		return &parser{expr: cfg + ".Addr", show: show}
	case "net/netip.Prefix":
		return &parser{expr: cfg + ".Prefix", show: show}
	case "net/netip.AddrPort":
		return &parser{expr: cfg + ".AddrPort", show: show}
	}
	return nil
}

// byMethod covers a type that reads itself, which is the way in for anything
// this package has never heard of.
func byMethod(t types.Type, cfg, show string) *parser {
	switch {
	case implementsParser(t):
		return &parser{expr: cfg + ".Config", show: show, generic: true}
	case implementsTextUnmarshaler(t):
		return &parser{expr: cfg + ".Text", show: show, generic: true}
	}
	return nil
}

func byKind(b *types.Basic, cfg, show string) *parser {
	switch b.Kind() {
	case types.String:
		return &parser{expr: cfg + ".String", show: show, generic: true}
	case types.Bool:
		return &parser{expr: cfg + ".Bool", show: show, generic: true}
	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64:
		return &parser{expr: cfg + ".Int", show: show, generic: true}
	case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64, types.Uintptr:
		return &parser{expr: cfg + ".Uint", show: show, generic: true}
	case types.Float32, types.Float64:
		return &parser{expr: cfg + ".Float", show: show, generic: true}
	}
	return nil
}

// instantiate names the type a nested parser is for, when it has one left to
// work out. config.Slice(config.Prefix) compiles and
// config.Slice(config.String) does not.
func instantiate(p *parser, t types.Type, imps *imports) string {
	if !p.generic {
		return p.expr
	}
	return p.expr + "[" + imps.typeString(t) + "]"
}

// implementsParser is whether a pointer to the type has ParseConfig, which is
// where such a method always is, since it has to write to the value.
func implementsParser(t types.Type) bool {
	return hasMethod(t, "ParseConfig", 1, 1)
}

// implementsTextUnmarshaler is the same question about UnmarshalText.
func implementsTextUnmarshaler(t types.Type) bool {
	return hasMethod(t, "UnmarshalText", 1, 1)
}

// hasMethod looks for a method on the pointer to a type, by name and by shape.
// Checking the shape rather than the exact signature keeps this from needing
// the config package's own types loaded, which would make the generator depend
// on the thing it generates calls into.
func hasMethod(t types.Type, name string, params, results int) bool {
	ms := types.NewMethodSet(types.NewPointer(t))
	for i := range ms.Len() {
		fn := ms.At(i).Obj()
		if fn.Name() != name {
			continue
		}
		// A method set holds functions and a function has a signature, so
		// neither of those is worth asking about.
		sig := fn.Type().(*types.Signature)

		// One argument in and one error out is the shape of both methods this
		// asks about. A method of the right name and the wrong shape is not
		// the method, and the field falls back to whatever its type would get
		// anyway.
		return sig.Params().Len() == params &&
			sig.Results().Len() == results &&
			isError(sig.Results().At(0).Type())
	}
	return false
}

func isError(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() == nil && named.Obj().Name() == "error"
}

func isByte(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.Uint8
}

// zeroOf is what Redact puts in the place of a secret.
//
// A secret string becomes the same three stars everything else prints, so a
// redacted configuration reads the way config:show does. Everything else
// becomes its zero value, since there is no sensible three stars for a number
// and a redacted key has to be a key that does not work.
func zeroOf(t types.Type, imps *imports) string {
	if b, ok := t.Underlying().(*types.Basic); ok {
		switch {
		case b.Kind() == types.String:
			redacted := imps.name(configPkg) + ".Redacted"
			if _, named := t.(*types.Named); named {
				return imps.typeString(t) + "(" + redacted + ")"
			}
			return redacted
		case b.Info()&types.IsBoolean != 0:
			return "false"
		case b.Info()&(types.IsInteger|types.IsFloat|types.IsComplex) != 0:
			return "0"
		}
	}
	switch t.Underlying().(type) {
	case *types.Slice, *types.Map, *types.Pointer, *types.Interface, *types.Chan, *types.Signature:
		return "nil"
	}
	return imps.typeString(t) + "{}"
}

// cloneOf is how Redact copies a field that a plain assignment would share.
//
// Assigning a struct copies every field of it, which is enough for everything
// except a slice or a map, where the copy and the original would point at the
// same memory and redacting one would redact the other.
func cloneOf(t types.Type, imps *imports) (string, bool) {
	switch t.Underlying().(type) {
	case *types.Slice:
		return imps.name("slices") + ".Clone", true
	case *types.Map:
		return imps.name("maps") + ".Clone", true
	}
	return "", false
}
