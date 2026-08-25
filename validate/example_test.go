package validate_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/go-mizu/mizu/errs"
	"github.com/go-mizu/mizu/validate"
)

// CreatePost is what a form sends, and it checks itself.
//
// The method below is what mizu gen:validate writes from struct tags. Written
// by hand it is the same method, which is the point: the generator is a way of
// not typing it, and not a mode the package runs in.
type CreatePost struct {
	Title     string
	Body      string
	PublishAt string
}

func (p CreatePost) Validate(ctx context.Context) error {
	var bad validate.Errors

	if p.Title == "" {
		bad.Add("title", validate.Failed("required"))
	} else if n := utf8.RuneCountInString(p.Title); n < 3 {
		bad.Add("title", validate.Failed("min", 3).Of("string"))
	}
	if p.Body == "" {
		bad.Add("body", validate.Failed("required"))
	}
	if p.PublishAt == "" {
		bad.Add("publish_at", validate.Failed("required"))
	}

	return bad.OrNil()
}

// A value that checks itself is checked by calling it, and what comes back is
// an ordinary error that the rest of the toolkit already knows how to answer
// for.
func Example() {
	err := CreatePost{Title: "Hi", Body: "Some words."}.Validate(context.Background())

	fmt.Println(errs.KindOf(err), errs.CodeOf(err), errs.KindOf(err).Status())
	for _, f := range errs.Fields(err) {
		fmt.Printf("%s: %s (%s)\n", f.Name, f.Msg, f.Code)
	}

	// Output:
	// unprocessable validation.failed 422
	// title: Title must be at least 3 characters. (min)
	// publish_at: Publish at is required. (required)
}

// A caller that renders its own messages wants the rules rather than the
// sentences, and they are still on the error.
func ExampleErrors_Fields() {
	err := CreatePost{}.Validate(context.Background())

	var bad *validate.Errors
	if !errors.As(err, &bad) {
		return
	}
	for field, rules := range bad.Fields() {
		for _, r := range rules {
			fmt.Println(field, r.Rule, r.Params)
		}
	}

	// Output:
	// title required []
	// body required []
	// publish_at required []
}

// A locale sets Msgs once and the checks after it do not change.
func ExampleErrors_msgs() {
	var bad validate.Errors
	bad.Msgs = shouty{}
	bad.Add("publish_at", validate.Failed("required"))

	fmt.Println(bad.First("publish_at"))
	// Output: PUBLISH AT IS REQUIRED.
}

// A format check answers one question, so a field that is allowed to be blank
// pairs it with a check for that and a field that is not pairs it with
// required.
func ExampleIsEmail() {
	var bad validate.Errors

	for _, in := range []string{"", "user@localhost", "user@example.com"} {
		if in != "" && !validate.IsEmail(in) {
			bad.Add("email", validate.Failed("email"))
		}
	}

	fmt.Println(bad.Len(), bad.First("email"))
	// Output: 1 Email must be an email address.
}

// The builder is for the rules a struct tag cannot say: one that depends on
// another field, or on an answer from somewhere else.
func ExampleV() {
	in := struct {
		Type    string
		Email   string
		Website string
		VAT     string
	}{Type: "company", Email: "someone@localhost", Website: ""}

	taken := false // what a lookup would have said

	v := validate.New()
	v.Field("email", in.Email).Required().Email().That(!taken, "unique")
	v.Field("website", in.Website).Optional().URL()
	v.When(in.Type == "company", func(v *validate.V) {
		v.Field("vat", in.VAT).Required().Min(4)
	})

	for field, msgs := range v.Errors().All() {
		fmt.Println(field, msgs)
	}

	// Unordered output:
	// email [Email must be an email address.]
	// vat [Vat is required.]
}

// A chain stops at its first failure, so a blank field says the one thing that
// is wrong with it.
func ExampleCheck() {
	v := validate.New()
	v.Field("email", "").Required().Email().Min(5)

	fmt.Println(v.Errors().Len(), v.Errors().First("email"))
	// Output: 1 Email is required.
}

// Struct reads the rules off the struct tags, which is what most code writes.
// The generator turns the same tags into a Validate method, so a program that
// runs it gets the same failures without the reflection.
func ExampleStruct() {
	type signup struct {
		Email string   `json:"email" validate:"required,email"`
		Site  string   `json:"site" validate:"omitempty,url"`
		Tags  []string `json:"tags" validate:"max=3,dive,required"`
	}

	err := validate.Struct(context.Background(), signup{
		Email: "someone@localhost",
		Tags:  []string{"go", ""},
	})

	for _, f := range errs.Fields(err) {
		fmt.Printf("%s: %s (%s)\n", f.Name, f.Msg, f.Code)
	}

	// Output:
	// email: Email must be an email address. (email)
	// tags.1: Tags 1 is required. (required)
}

// A rule a tag cannot say and this package has never heard of is a Rule of
// somebody's own, registered under the name a tag calls it.
func ExampleRegister() {
	type order struct {
		Code string `json:"code" validate:"omitempty,coupon"`
	}

	// In a real program this is an init function, and the rule holds whatever
	// it needs to answer with.
	validate.Register(coupon{live: []string{"SPRING"}})

	err := validate.Struct(context.Background(), order{Code: "AUTUMN"})
	for _, f := range errs.Fields(err) {
		fmt.Printf("%s: %s (%s)\n", f.Name, f.Msg, f.Code)
	}

	// Output: code: Code is not valid. (coupon)
}

// coupon rejects a code that is not one of the ones still running.
type coupon struct{ live []string }

func (coupon) Name() string { return "coupon" }

func (r coupon) Validate(ctx context.Context, f validate.Field) error {
	code, _ := f.Value.(string)
	if !slices.Contains(r.live, code) {
		f.Fail()
	}
	return nil
}

// shouty stands in for a translation, which is a Messages like any other.
type shouty struct{}

func (shouty) Message(field string, r validate.RuleError) string {
	return strings.ToUpper(validate.English.Message(field, r))
}
