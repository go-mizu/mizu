package validategen

import (
	"go/token"
	"go/types"
	"math"
	"testing"
	"time"
)

// The corpus in testdata is the shapes a request struct has. What is here is
// the rest of the type system, which the generator still has to say something
// sensible about, and the arithmetic underneath a bound, which is easier to be
// exhaustive about from a table than from a struct.

// bare is a plan with nothing loaded, which is all the helpers below need. The
// package path is one nothing else uses, so anything it decides to import is
// imported under its own name.
func bare() *Plan {
	p := &Plan{imps: newImports("example.com/app"), taken: map[string]bool{}, local: newLocals()}
	p.Val = p.imps.name(validatePkg)
	p.Ctx = p.imps.name("context")
	return p
}

// named is a defined type over the underlying one, the way a package declares
// it, so that the helpers see what they would see from a real package.
func named(name string, under types.Type) types.Type {
	pkg := types.NewPackage("example.com/other", "other")
	return types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), under, nil)
}

// alias is another name for a type, which go/types keeps as a node of its own.
func alias(name string, of types.Type) types.Type {
	pkg := types.NewPackage("example.com/other", "other")
	return types.NewAlias(types.NewTypeName(token.NoPos, pkg, name, nil), of)
}

func TestKindOf(t *testing.T) {
	p := bare()
	str := types.Typ[types.String]

	for _, c := range []struct {
		typ  types.Type
		want string
	}{
		{str, "string"},
		{types.Typ[types.Int], "int"},
		{types.Typ[types.UnsafePointer], "unsafe.Pointer"},
		{types.NewPointer(types.Typ[types.Bool]), "bool"},
		{named("Sort", str), "other.Sort"},
		{types.NewSlice(str), "slice"},
		{types.NewArray(str, 3), "array"},
		{types.NewMap(str, str), "map"},
		{types.NewStruct(nil, nil), "struct"},
		{types.NewInterfaceType(nil, nil), "interface"},
		{types.NewChan(types.SendRecv, str), "chan"},
		{types.NewSignatureType(nil, nil, nil, nil, nil, false), "func"},
		{types.Typ[types.UntypedNil], "untyped nil"},

		// An alias is not a type of its own and not a basic one either, so it
		// falls past every case above and is named the way it was written.
		{alias("Text", str), "other.Text"},
	} {
		if got := p.kindOf(c.typ); got != c.want {
			t.Errorf("kindOf(%s) = %q, want %q", c.typ, got, c.want)
		}
	}
}

// A rule reads through pointers, and what a list holds is what dive is about.
func TestElemOf(t *testing.T) {
	str := types.Typ[types.String]

	for _, c := range []struct {
		typ  types.Type
		want string
		ok   bool
	}{
		{types.NewSlice(str), "string", true},
		{types.NewArray(str, 3), "string", true},
		{types.NewPointer(types.NewSlice(str)), "string", true},
		{named("Tags", types.NewSlice(str)), "string", true},
		{types.NewMap(str, str), "", false},
		{str, "", false},
	} {
		el, ok := elemOf(c.typ)
		if ok != c.ok {
			t.Errorf("elemOf(%s) said %v, want %v", c.typ, ok, c.ok)
			continue
		}
		if ok && el.String() != c.want {
			t.Errorf("elemOf(%s) = %s, want %s", c.typ, el, c.want)
		}
	}
}

// required has to have something to compare against for every type a field can
// be, and the two things it hands back are that comparison and its opposite,
// which is what omitempty runs on.
func TestEmptiness(t *testing.T) {
	p := bare()
	str := types.Typ[types.String]

	for _, c := range []struct {
		typ         types.Type
		empty, full string
	}{
		{str, `v.X == ""`, `v.X != ""`},
		{types.Typ[types.Bool], "!v.X", "v.X"},
		{types.Typ[types.Int], "v.X == 0", "v.X != 0"},
		{types.Typ[types.Float64], "v.X == 0", "v.X != 0"},
		{types.NewSlice(str), "len(v.X) == 0", "len(v.X) != 0"},
		{types.NewMap(str, str), "len(v.X) == 0", "len(v.X) != 0"},
		{types.NewChan(types.SendRecv, str), "v.X == nil", "v.X != nil"},
		{types.NewSignatureType(nil, nil, nil, nil, nil, false), "v.X == nil", "v.X != nil"},
	} {
		empty, full, err := p.emptiness("v.X", c.typ)
		if err != nil {
			t.Errorf("emptiness(%s): %v", c.typ, err)
			continue
		}
		if empty != c.empty || full != c.full {
			t.Errorf("emptiness(%s) = %q, %q, want %q, %q", c.typ, empty, full, c.empty, c.full)
		}
	}
}

// A struct that cannot be compared is asked the same question reflect asks,
// which is what the tag interpreter does with it too.
func TestEmptinessOfAStructWithNothingToCompare(t *testing.T) {
	p := bare()
	f := types.NewField(token.NoPos, nil, "Names", types.NewSlice(types.Typ[types.String]), false)
	st := named("Filter", types.NewStruct([]*types.Var{f}, nil))

	empty, full, err := p.emptiness("v.X", st)
	if err != nil {
		t.Fatal(err)
	}
	const want = "reflect.ValueOf(v.X).IsZero()"
	if empty != want || full != "!"+want {
		t.Errorf("emptiness = %q, %q, want %q and its opposite", empty, full, want)
	}
	if lines := p.imps.lines(); len(lines) != 3 {
		t.Errorf("the imports are %v, want reflect to have been added", lines)
	}
}

// A type with no zero value to name is the one case required cannot be written
// for, and the message has to say which field and why.
func TestEmptinessOfATypeWithNoWayToSay(t *testing.T) {
	p := bare()
	_, _, err := p.emptiness("v.X", types.Typ[types.UnsafePointer])
	if err == nil {
		t.Fatal("an unsafe.Pointer was given a way to say it is empty")
	}
	const want = "required is about a value being filled in and an unsafe.Pointer has no way to say"
	if err.Error() != want {
		t.Errorf("the error is %q, want %q", err, want)
	}
}

// exact decides whether a bound can be compared as it was written or has to go
// through a float64, which is what the tag interpreter compares in. Getting it
// wrong either loses the exactness of an integer or writes code that does not
// compile, so every width is here.
func TestExact(t *testing.T) {
	i := func(n int) bnd { return bnd{kind: bInt, i: n} }
	f := bnd{kind: bFloat}
	d := bnd{kind: bDur}

	for _, c := range []struct {
		kind types.BasicKind
		b    bnd
		want bool
	}{
		{types.Int, d, true}, // a length of time is written as one
		{types.Int8, i(127), true},
		{types.Int8, i(128), false},
		{types.Int8, i(-129), false},
		{types.Int16, i(32767), true},
		{types.Int16, i(-32769), false},
		{types.Int32, i(math.MaxInt32), true},
		{types.Int, i(math.MaxInt32 + 1), false},
		{types.Int64, i(1 << 53), true},
		{types.Int64, i(1<<53 + 1), false},
		{types.Uint8, i(255), true},
		{types.Uint8, i(-1), false},
		{types.Uint16, i(65535), true},
		{types.Uint16, i(65536), false},
		{types.Uint32, i(math.MaxUint32), true},
		{types.Uint, i(math.MaxUint32 + 1), false},
		{types.Uint64, i(1 << 53), true},
		{types.Uint64, i(-1), false},
		{types.Float64, i(3), true},
		{types.Float64, f, true},
		{types.Float32, i(3), false},
		{types.Float32, f, false},
		{types.Int, f, false}, // a fraction against a whole number
		{types.String, i(3), true},
		{types.Bool, i(3), true}, // nothing to count, which the caller refused first
	} {
		if got := exact(types.Typ[c.kind], c.b); got != c.want {
			t.Errorf("exact(%s, %+v) = %v, want %v", types.Typ[c.kind], c.b, got, c.want)
		}
	}

	// A list counts its elements, which is an int however long it is.
	if !exact(types.NewSlice(types.Typ[types.String]), i(5)) {
		t.Error("a count of elements was written as a float64")
	}
}

// A bound on a duration is written the way somebody would write it, since the
// sentence next to it says the same thing and a count of nanoseconds in the
// code is a count nobody can check against the tag.
func TestDurLit(t *testing.T) {
	p := bare()
	for _, c := range []struct {
		in   time.Duration
		want string
	}{
		{time.Hour, "time.Hour"},
		{90 * time.Minute, "90 * time.Minute"},
		{time.Second, "time.Second"},
		{30 * time.Second, "30 * time.Second"},
		{time.Millisecond, "time.Millisecond"},
		{time.Microsecond, "time.Microsecond"},
		{-time.Second, "-time.Second"},
		{-2 * time.Second, "-2 * time.Second"},
		{0, "time.Duration(0)"},
		{1500 * time.Nanosecond, "1500 * time.Nanosecond"},
	} {
		if got := p.durLit(c.in); got != c.want {
			t.Errorf("durLit(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A bound that is not a whole number is written as one the compiler reads as a
// float64, because the sentence next to it prints the type.
func TestFloatLit(t *testing.T) {
	p := bare()
	for _, c := range []struct {
		in   float64
		want string
	}{
		{0.5, "0.5"},
		{-1.25, "-1.25"},
		{1e21, "1e+21"},
		{3, "float64(3)"},
		{math.Inf(1), "math.Inf(1)"},
		{math.Inf(-1), "math.Inf(-1)"},
		{math.NaN(), "math.NaN()"},
	} {
		if got := p.floatLit(c.in); got != c.want {
			t.Errorf("floatLit(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A type on the way to itself is a tree, and the function that checks it calls
// itself rather than being written again for each level.
func TestSawType(t *testing.T) {
	p := bare()
	tree := named("Tree", types.NewStruct(nil, nil))
	other := named("Leaf", types.NewStruct(nil, nil))

	if p.sawType(tree) {
		t.Error("a walk that has been nowhere has seen a type")
	}
	p.seen = []types.Type{tree}
	if !p.sawType(tree) {
		t.Error("the type the walk started at was not recognised on the way back to it")
	}
	if p.sawType(other) {
		t.Error("a different type was taken for the one being walked")
	}
}

// A generated name must not be one the package already uses, and the way out is
// a number rather than a failure, since the alternative is telling somebody to
// rename their own type.
func TestHelperName(t *testing.T) {
	p := bare()
	here := types.NewPackage("example.com/app", "app")
	line := types.NewNamed(types.NewTypeName(token.NoPos, here, "Line", nil), types.NewStruct(nil, nil), nil)

	if got := p.helperName(line); got != "validateLine" {
		t.Fatalf("helperName = %q, want validateLine", got)
	}
	if got := p.helperName(line); got != "validateLine2" {
		t.Errorf("the second one is %q, want validateLine2", got)
	}
	if got := p.helperName(line); got != "validateLine3" {
		t.Errorf("the third one is %q, want validateLine3", got)
	}

	// A struct written out in the field rather than declared has no name to
	// take, so it gets the one every anonymous struct gets.
	if got := p.helperName(types.NewStruct(nil, nil)); got != "validateStruct" {
		t.Errorf("an unnamed struct got %q, want validateStruct", got)
	}
}

// A local named after a field can collide with another field in the same
// function, and with the names the output writes for itself.
func TestNewLocal(t *testing.T) {
	p := bare()

	if got := p.newLocal("Ship"); got != "ship" {
		t.Fatalf("newLocal = %q, want ship", got)
	}
	if got := p.newLocal("Ship"); got != "ship2" {
		t.Errorf("the second one is %q, want ship2", got)
	}
	if got := p.newLocal("Ship"); got != "ship3" {
		t.Errorf("the third one is %q, want ship3", got)
	}

	// name, len and the rest are already spoken for by the code around them.
	if got := p.newLocal("Name"); got != "name2" {
		t.Errorf("a field called Name got %q, want name2", got)
	}
	if got := p.newLocal("Len"); got != "len2" {
		t.Errorf("a field called Len got %q, want len2", got)
	}
}
