package validate

import (
	"context"
	"strconv"
	"sync"
)

// A Rule is a check with a name, so that a struct tag can ask for it.
//
// The rules this package ships with are functions and methods, because a rule
// that is part of the package does not need to be looked up by name at run
// time. A Rule is for the ones that are not: a VAT number checked against a
// registry, a coupon code checked against a table, anything that belongs to one
// program rather than to every program.
//
//	type vat struct{ client *api.Client }
//
//	func (vat) Name() string { return "vat" }
//
//	func (r vat) Validate(ctx context.Context, f validate.Field) error {
//		ok, err := r.client.CheckVAT(ctx, f.Value.(string))
//		if err != nil {
//			return err
//		}
//		if !ok {
//			f.Fail()
//		}
//		return nil
//	}
//
// The two ways of finishing are the point. A value the rule rejected is
// recorded with [Field.Fail] and the request comes back a 422 with a sentence
// next to the field. An error returned is something that went wrong outside the
// request, and it travels up as it is, because a registry that was unreachable
// is not somebody's VAT number being wrong.
//
// A rule is used by name in a tag once [Register] has been given it.
type Rule interface {
	// Name is what a tag calls the rule, and what ends up on the failure. It is
	// on the rule rather than said again at registration so that there is one
	// place the name is written.
	Name() string

	// Validate looks at one field. Nil means the value passed, or that it
	// failed and [Field.Fail] has already said so.
	Validate(ctx context.Context, f Field) error
}

// A Field is the value a [Rule] is looking at.
//
// It is a struct rather than a set of arguments because a rule that only wants
// the value should not have to write down the two it does not, and because
// anything a later rule needs is added here without changing what is already
// written.
type Field struct {
	// Name is what the request called the field, publish_at or
	// items.0.quantity, and it is the name the failure comes back under.
	Name string

	// Value is what the field holds, with pointers followed, so a rule that
	// wants a string asserts for a string whether the field was a string or a
	// pointer to one. A nil pointer arrives as the zero value of what it points
	// at, which is what makes that assertion safe to write without a branch for
	// the pointer case. Telling an absent field from an empty one is what
	// required is for, and a rule that should not run on an empty field is
	// written after omitempty.
	Value any

	// Params are what the tag configured the rule with, so vat=DE arrives as
	// one element. They are the tag's text, unescaped and not otherwise looked
	// at, since what they mean is the rule's business.
	Params []string

	bad  *Errors
	rule string
}

// Fail records that the value did not satisfy the rule.
//
// The parameters are what the sentence for the rule fills in and what a client
// reads off the error, which is the rule's configuration and not the value. A
// message built from what arrived is a message that renders whatever somebody
// sent, and that is [Failed]'s call as much as it is this one's.
//
// A rule may call it more than once. The chain stops after the rule returns
// either way, so a later rule on the same field does not run.
func (f Field) Fail(params ...any) {
	if f.bad == nil {
		panic("validate: Fail on a Field that did not come from a check")
	}
	f.bad.Add(f.Name, Failed(f.rule, params...))
}

// custom is every [Rule] that has been registered, by name.
var custom sync.Map // string -> Rule

// Register makes a rule usable by name in a struct tag.
//
// It is called from an init function, or from main before anything is checked,
// which is the same shape http.Handle has and for the same reason: a tag naming
// a rule that is not there yet is a tag that fails, and the plan for a struct
// is worked out once and kept.
//
//	func init() { validate.Register(vat{client: registry}) }
//
// Registering a name twice, or a name this package already uses, or a name that
// cannot be written in a tag, panics. All three are a mistake in the program
// and none of them has a sensible way to carry on.
//
// The generator resolves tag names when it runs, so a struct whose validator is
// generated says a typo in a tag at build time instead.
func Register(r Rule) {
	name := r.Name()
	switch {
	case !tagSafe(name):
		panic("validate: " + strconv.Quote(name) + " cannot be written in a tag")
	case builtin(name):
		panic("validate: " + name + " is one of this package's own rules")
	}
	if _, taken := custom.LoadOrStore(name, r); taken {
		panic("validate: " + name + " is registered twice")
	}
}

// ruleFor is the registered rule of that name, if there is one.
func ruleFor(name string) (Rule, bool) {
	r, ok := custom.Load(name)
	if !ok {
		return nil, false
	}
	return r.(Rule), true
}

// builtin is whether the interpreter already knows the name.
func builtin(name string) bool {
	if name == dive {
		return true
	}
	_, plain := plainRules[name]
	_, size := sizeRules[name]
	_, span := spanRules[name]
	return plain || size || span
}
