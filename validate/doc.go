// Package validate says what is wrong with a value, in one shape that a
// response, a form redisplay and a log line can all read.
//
// A check that has failed produces two things. There is the machine half, a
// rule name and whatever the rule was configured with, which is what a client
// switches on and what a test asserts. There is the human half, a sentence to
// put next to the input. Both come out of an [Errors], which collects failures
// as they are found rather than stopping at the first one, because a form that
// reports one problem per submission is a form somebody fills in four times.
//
//	var bad validate.Errors
//	if in.Title == "" {
//		bad.Add("title", validate.Failed("required"))
//	}
//	if n := utf8.RuneCountInString(in.Body); n < 10 {
//		bad.Add("body", validate.Failed("min", 10).Of("string"))
//	}
//	return bad.OrNil()
//
// # Checks that are not a line of Go
//
// A rule like min is written out where it is used, because
// utf8.RuneCountInString(in.Title) < 3 is something to read in a debugger and a
// call into this package is something to step into. The rules that take a page
// to get right are functions instead, and the generated code, the reflective
// interpreter and a hand-written method all call the same one.
//
//	if in.Email != "" && !validate.IsEmail(in.Email) {
//		bad.Add("email", validate.Failed("email"))
//	}
//
// [IsEmail], [IsURL], [IsURI], [IsHostname], [IsIP], [IsIPv4], [IsIPv6],
// [IsCIDR], [IsMAC], [IsPort], [IsUUID], [IsULID] and [IsE164] are the ones
// that have landed. Each takes a string, because a string is what a form sends,
// and each has a sentence in [English] under the same name a struct tag would
// spell it.
//
// None of them treat the empty string as a pass. Whether a field is allowed to
// be missing is what required says, and a check that answered both questions
// would make a field that is optional and a field that is optional-but-valid
// impossible to tell apart.
//
// # Rules on the struct
//
// Most rules are the same three or four on every field, and writing them out
// is writing the same lines again. A struct says them in a tag instead, and
// [Struct] runs them:
//
//	type CreatePost struct {
//		Title string   `json:"title" validate:"required,min=3,max=200"`
//		Body  string   `json:"body" validate:"required"`
//		Site  string   `json:"site" validate:"omitempty,url"`
//		Tags  []string `json:"tags" validate:"max=5,dive,required"`
//	}
//
// Rules are separated by commas and a rule's parameters follow an equals sign.
// required, omitempty, min, max, between, size and the thirteen format checks
// are the ones that are there, and each is the method of the same name on a
// [Check]: the tag interpreter runs the chain above rather than a second copy
// of it, so the two cannot answer differently.
//
// dive says that the rules after it are about the elements rather than about
// the field, so max=5 counts the tags and required is asked of each one. A
// failure inside a list is named for where it was, tags.1, and a struct inside
// one is named the way a nested struct is anywhere else, lines.1.sku.
//
// The name a failure comes back under is the name the request used. It comes
// from the same tags web.Bind reads, which is a path, header, cookie, form or
// query tag, then the json tag, then the field's own name in snake case, so a
// field that would not bind and a field that would not validate come back
// under one name and a form marks one input.
//
// A tag this cannot run is a mistake in the program rather than in the request:
// an unknown rule, a bound that is not a number, a format check on an int. It
// comes back as an [errs.Internal] naming the field and saying what is wrong
// with it, on every check of that type rather than only on the requests that
// fill the field in.
//
// [Register] adds a rule of somebody's own to the ones a tag can name. It takes
// a [Rule], which is a name and a method, and the method is handed a [Field] to
// look at. A value the rule rejected is recorded with [Field.Fail]. An error
// returned is something that went wrong outside the request, a lookup that
// could not be made, and it travels up rather than becoming a 422.
//
// # Rules written out in order
//
// [New] starts a list of checks for the rules a struct tag cannot say: a rule
// that depends on another field, on a row in a database, or on which branch of
// a form was filled in.
//
//	v := validate.New()
//	v.Field("email", in.Email).Required().Email().That(!taken, "unique")
//	v.Field("website", in.Website).Optional().URL()
//	v.When(in.Type == "company", func(v *validate.V) {
//		v.Field("vat", in.VAT).Required().Min(4)
//	})
//	return v.Err()
//
// A chain stops at its first failure, so [Check.Required] guards everything
// after it and a blank field says the one thing that is wrong with it rather
// than a list of consequences. A field that may be left blank writes
// [Check.Optional] instead, which stops the chain quietly when there is nothing
// there.
//
// [Check.Min], [Check.Max], [Check.Between] and [Check.Size] read the value's
// type to decide what they are counting: characters for a string, the number
// itself for a number, elements for a list or a map, and length for a duration.
// The bound keeps the type it was written with, so Min(3) says 3 in the
// sentence and Min(time.Hour) says an hour.
//
// [Check.That] is the way in for anything with no method, including a check
// that had to ask somebody else. [V.When] takes an ordinary Go expression, so a
// condition that depends on another field is written where the condition is
// rather than spelled out in a string on the field it is about.
//
// A struct that can be described in tags does not need any of this. The
// generator writes a [Validator] method for it, which costs nothing at runtime
// and reads in a debugger.
//
// # What comes out
//
// [Errors.OrNil] is nil when nothing failed and an [errs.Error] when something
// did: kind [errs.Unprocessable], code validation.failed, and one
// [errs.Field] per rule that failed, in the order they were added.
//
// That is the same shape binding produces, per D-092, so a 422 document and a
// form redisplay are written once and neither has to ask which half of the
// request rejected it. The [Errors] is left on the error as the cause, so a
// caller that wants the rule names and their parameters back reaches them with
// [errors.As] rather than parsing a sentence.
//
// # Where the sentences come from
//
// [English] writes them, and it is what an [Errors] uses when nobody has said
// otherwise. It turns a field name into a label the way a form would, so
// publish_at is written Publish at, and fills the rule's parameters into a
// template.
//
//	bad.First("body")   // "Body must be at least 10 characters."
//
// A rule with more than one sentence says which it wants with [RuleError.Of].
// min on a string counts characters, on a number compares, on a list counts
// elements and on an upload counts bytes, and those are four sentences for one
// rule name. The name a client sees does not change.
//
// [Errors.Msgs] is the seam for translation. It is an interface with one
// method, so a package that knows the request's locale sets it and the checks
// above it do not change. The zero [Errors] writes English, which is what makes
// a generated validator a plain function with no setup in front of it.
package validate
