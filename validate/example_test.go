package validate_test

import (
	"context"
	"errors"
	"fmt"
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

// shouty stands in for a translation, which is a Messages like any other.
type shouty struct{}

func (shouty) Message(field string, r validate.RuleError) string {
	return strings.ToUpper(validate.English.Message(field, r))
}
