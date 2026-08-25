package commandgen

import (
	"fmt"
	"go/types"
	"reflect"
	"strconv"
	"strings"
)

// value is the expression that builds the console.Value for a field: the
// constructor that reads the field's type, pointed at the field.
//
// This is the one decision in the generator a reader cannot check by eye, so
// every type it knows about is listed in one place and anything else is an
// error naming the field rather than a guess.
func (p *Plan) value(t types.Type, addr string, tag reflect.StructTag) (string, error) {
	c := p.Console

	if tag.Get("count") == "true" {
		if !types.Identical(t, types.Typ[types.Int]) {
			// No article in front of the type. It is written by the reader
			// rather than by this, and "a int8" is the sort of thing that
			// makes somebody wonder what else the tool got wrong.
			return "", fmt.Errorf("counts up as -vv and is %s, which only a plain int does", p.short(t))
		}
		return fmt.Sprintf("%s.Count(%s)", c, addr), nil
	}

	if options, ok := tag.Lookup("enum"); ok {
		if !isString(t) {
			return "", fmt.Errorf("has an enum tag and is %s, which only text can be", p.short(t))
		}
		return fmt.Sprintf("%s.Enum(%s%s)", c, addr, quoteAll(options)), nil
	}

	// A slice collects rather than replaces, so its element decides the parser
	// and the tag decides whether one occurrence may hold several values.
	if slice, ok := t.Underlying().(*types.Slice); ok {
		if err := p.plain(t, "list"); err != nil {
			return "", err
		}
		sep, ok := tag.Lookup("sep")
		if !ok {
			sep = ","
		}
		if types.Identical(slice.Elem(), types.Typ[types.String]) {
			return fmt.Sprintf("%s.Strings(%s, %q)", c, addr, sep), nil
		}
		parse, err := p.parser(slice.Elem())
		if err != nil {
			return "", fmt.Errorf("is a list of %s, which %w", p.short(slice.Elem()), err)
		}
		return fmt.Sprintf("%s.Slice(%s, %s, %q)", c, addr, parse, sep), nil
	}

	if m, ok := t.Underlying().(*types.Map); ok {
		if !types.Identical(m.Key(), types.Typ[types.String]) || !types.Identical(m.Elem(), types.Typ[types.String]) {
			return "", fmt.Errorf("is a %s, and a flag holding pairs is a map[string]string", p.short(t))
		}
		if err := p.plain(t, "map"); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.KeyValues(%s)", c, addr), nil
	}

	if isBool(t) {
		return fmt.Sprintf("%s.Bool(%s)", c, addr), nil
	}

	if name := p.constructor(t); name != "" {
		return fmt.Sprintf("%s.%s(%s)", c, name, addr), nil
	}

	// Anything that reads itself from text is its own answer, which is netip,
	// big.Int, a UUID, and most of what a library exports for this.
	if implementsTextUnmarshaler(t) {
		return fmt.Sprintf("%s.Text(%s)", c, addr), nil
	}
	return "", fmt.Errorf("is a %s, which no console.Value reads", p.short(t))
}

// plain reports a named list or map type.
//
// console.Strings, console.Slice and console.KeyValues take a pointer to the
// type they name, and a pointer to a defined type is not that pointer however
// alike the two look. The constructors that take an approximate type parameter,
// such as console.Int, have no such trouble, which is why this is only asked
// about the two that collect.
func (p *Plan) plain(t types.Type, kind string) error {
	if _, ok := t.(*types.Named); !ok {
		return nil
	}
	return fmt.Errorf("is %s, a %s type of its own, and the field has to be declared as %s for the value to point at it",
		p.short(t), kind, p.short(t.Underlying()))
}

// constructor is the console constructor for a scalar type, or nothing when
// there is none.
func (p *Plan) constructor(t types.Type) string {
	switch named(t) {
	case "time.Duration":
		return "Duration"
	case "time.Time":
		return "Time"
	}
	basic, ok := t.Underlying().(*types.Basic)
	if !ok {
		return ""
	}
	switch info := basic.Info(); {
	case info&types.IsString != 0:
		return "String"
	case info&types.IsUnsigned != 0:
		return "Uint"
	case info&types.IsInteger != 0:
		return "Int"
	case info&types.IsFloat != 0:
		return "Float"
	}
	return ""
}

// parser is the console parse function for the element of a list, which is the
// same parsing and the same messages a scalar flag of that type would get.
func (p *Plan) parser(t types.Type) (string, error) {
	name := p.constructor(t)
	if name == "" {
		return "", fmt.Errorf("no console parser reads")
	}
	return p.Console + ".Parse" + name, nil
}

// named is a defined type's full name, as in time.Duration, and nothing for a
// type that has none.
func named(t types.Type) string {
	n, ok := t.(*types.Named)
	if !ok || n.Obj().Pkg() == nil {
		return ""
	}
	return n.Obj().Pkg().Path() + "." + n.Obj().Name()
}

func isString(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsString != 0
}

func isBool(t types.Type) bool {
	basic, ok := t.Underlying().(*types.Basic)
	return ok && basic.Info()&types.IsBoolean != 0
}

// implementsTextUnmarshaler reports whether a pointer to t has the method, in
// the shape encoding.TextUnmarshaler asks for.
//
// The method set of the pointer is the one to ask, since UnmarshalText writes
// to the value it is on and is therefore always on the pointer.
func implementsTextUnmarshaler(t types.Type) bool {
	m, _, _ := types.LookupFieldOrMethod(types.NewPointer(t), true, nil, "UnmarshalText")
	fn, ok := m.(*types.Func)
	if !ok {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Params().Len() != 1 || sig.Results().Len() != 1 {
		return false
	}
	bytes, ok := sig.Params().At(0).Type().(*types.Slice)
	if !ok {
		return false
	}
	return types.Identical(bytes.Elem(), types.Typ[types.Byte]) &&
		types.TypeString(sig.Results().At(0).Type(), nil) == "error"
}

// short is a type as a person would write it, for a message about a field: the
// package name for a type from somewhere else, and nothing in front of a type
// declared where the command is, which is where the reader already is.
func (p *Plan) short(t types.Type) string {
	return types.TypeString(t, func(pkg *types.Package) string {
		if pkg == p.types {
			return ""
		}
		return pkg.Name()
	})
}

// quoteAll writes an enum's options as arguments, so a|b|c becomes , "a", "b",
// "c". They are untyped constants, which is what lets console.Enum take them
// for a named string type without a conversion per option.
func quoteAll(options string) string {
	var b strings.Builder
	for _, option := range strings.Split(options, "|") {
		b.WriteString(", ")
		b.WriteString(strconv.Quote(option))
	}
	return b.String()
}
