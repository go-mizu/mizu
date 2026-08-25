package validate

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// even is a rule that passes on a string with an even number of characters and
// says how many it wanted when it does not.
type even struct{}

func (even) Name() string { return "even" }

func (even) Validate(ctx context.Context, f Field) error {
	s, _ := f.Value.(string)
	if len(s)%2 != 0 {
		f.Fail(len(s) + 1)
	}
	return nil
}

// upstream is a rule whose answer comes from somewhere else, and sometimes that
// somewhere else is not there.
type upstream struct{ err error }

func (upstream) Name() string { return "upstream" }

func (r upstream) Validate(ctx context.Context, f Field) error {
	if r.err != nil {
		return r.err
	}
	if len(f.Params) != 1 {
		f.Fail()
		return nil
	}
	if s, _ := f.Value.(string); s != f.Params[0] {
		f.Fail(f.Params[0])
	}
	return nil
}

// sees records what the interpreter handed it, so that the Field a rule reads
// can be asserted on.
type sees struct{ got *Field }

func (sees) Name() string { return "sees" }

func (r sees) Validate(ctx context.Context, f Field) error {
	*r.got = f
	return nil
}

var seen Field

func init() {
	Register(even{})
	Register(upstream{})
	Register(sees{got: &seen})
}

func TestARegisteredRuleRunsFromATag(t *testing.T) {
	type in struct {
		Code string `json:"code" validate:"required,even"`
	}

	wantFailures(t, in{Code: "abcd"})
	wantFailures(t, in{Code: "abc"}, "code:even")
	wantFailures(t, in{}, "code:required")
}

// What a rule was configured with in the tag is what it reads off the Field,
// and it is what the sentence fills in.
func TestARegisteredRuleReadsItsParameters(t *testing.T) {
	type in struct {
		Word string `json:"word" validate:"upstream=hello"`
	}

	wantFailures(t, in{Word: "hello"})
	wantFailures(t, in{Word: "goodbye"}, "word:upstream")
}

// A rule that could not reach what it needed says so by returning, and that
// error travels up as it is. A registry that was unreachable is not somebody's
// input being wrong.
func TestARuleThatCouldNotAnswerReturnsItsError(t *testing.T) {
	type in struct {
		Word string `json:"word" validate:"downstream"`
	}

	down := errors.New("the registry did not answer")
	Register(named{"downstream", upstream{err: down}})
	t.Cleanup(func() { custom.Delete("downstream") })

	err := Struct(context.Background(), in{Word: "x"})
	if !errors.Is(err, down) {
		t.Errorf("Struct returned %v, want %v", err, down)
	}
}

// The same is true partway down a list. Nothing after it runs, because the
// answer for the rest of the fields would be as unreliable as this one.
func TestARuleThatCouldNotAnswerStopsADive(t *testing.T) {
	down := errors.New("the registry did not answer")
	Register(named{"offline", upstream{err: down}})
	t.Cleanup(func() { custom.Delete("offline") })

	type leaf struct {
		Word string `json:"word" validate:"offline"`
	}
	type list struct {
		Words []string `json:"words" validate:"dive,offline"`
	}
	type deep struct {
		Leaves []leaf `json:"leaves" validate:"dive"`
	}

	for _, value := range []any{
		list{Words: []string{"x"}},
		deep{Leaves: []leaf{{Word: "x"}}},
	} {
		if err := Struct(context.Background(), value); !errors.Is(err, down) {
			t.Errorf("Struct(%T) returned %v, want %v", value, err, down)
		}
	}
}

// A rule that failed stops the chain, the same as one of this package's own.
func TestARegisteredRuleStopsTheChain(t *testing.T) {
	type in struct {
		Code string `json:"code" validate:"even,min=4"`
	}
	wantFailures(t, in{Code: "abc"}, "code:even")

	// And a rule after one that failed does not run at all.
	type also struct {
		Code string `json:"code" validate:"min=4,even"`
	}
	wantFailures(t, also{Code: "abc"}, "code:min")
}

func TestARuleReadsThroughAPointer(t *testing.T) {
	type in struct {
		Code *string `json:"code" validate:"sees"`
	}

	code := "abc"
	if err := Struct(context.Background(), in{Code: &code}); err != nil {
		t.Fatalf("Struct: %v", err)
	}
	if seen.Name != "code" || seen.Value != "abc" {
		t.Errorf("a *string arrived as %q %#v, want code and abc", seen.Name, seen.Value)
	}

	// A nil pointer is nothing at all, rather than a pointer that has to be
	// checked before it is read.
	if err := Struct(context.Background(), in{}); err != nil {
		t.Fatalf("Struct: %v", err)
	}
	if seen.Value != nil {
		t.Errorf("a nil *string arrived as %#v, want nil", seen.Value)
	}
}

// The message table has no entry for a rule somebody registered, which is a
// missing translation rather than a broken request, so the sentence names the
// field and says it is not valid.
func TestARegisteredRuleWithNoMessage(t *testing.T) {
	type in struct {
		Code string `json:"code" validate:"even"`
	}

	err := Struct(context.Background(), in{Code: "abc"})

	var bad *Errors
	if !errors.As(err, &bad) {
		t.Fatalf("not a validation error: %v", err)
	}
	if got, want := bad.First("code"), "Code is not valid."; got != want {
		t.Errorf("said %q, want %q", got, want)
	}
}

func TestRegisterRefusesANameItCannotUse(t *testing.T) {
	cases := []struct {
		name string
		says string
	}{
		{"", "cannot be written in a tag"},
		{"two words", "cannot be written in a tag"},
		{"a=b", "cannot be written in a tag"},
		{"required", "one of this package's own rules"},
		{"min", "one of this package's own rules"},
		{"between", "one of this package's own rules"},
		{"email", "one of this package's own rules"},
		{"dive", "one of this package's own rules"},
		{"even", "registered twice"},
	}

	for _, c := range cases {
		func() {
			defer func() {
				switch r := recover().(type) {
				case nil:
					t.Errorf("Register(%q) did not panic", c.name)
				case string:
					if !strings.Contains(r, c.says) {
						t.Errorf("Register(%q) said %q, want it to mention %q", c.name, r, c.says)
					}
				default:
					t.Errorf("Register(%q) panicked with %#v", c.name, r)
				}
			}()
			Register(named{c.name, even{}})
		}()
	}
}

// A Field built by hand has no check behind it to record anything on, and
// saying so is better than dropping the failure on the floor.
func TestFailOnAFieldThatCameFromNowhere(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("Fail on a bare Field did not panic")
		}
	}()
	Field{Name: "code"}.Fail()
}

// named is a rule under a name of somebody else's choosing, for the tests that
// are about the name rather than about the check.
type named struct {
	name string
	Rule
}

func (r named) Name() string { return r.name }
