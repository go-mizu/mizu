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
