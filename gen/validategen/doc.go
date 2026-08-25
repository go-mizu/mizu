// Package validategen writes the Validate method for a request struct. It is
// an implementation detail of the mizu command and is exempt from the
// compatibility promise in doc 31. Import it only if you are extending mizu
// itself.
//
// A struct asks for one with a //mizu:validate marker:
//
//	//mizu:validate
//	type CreatePost struct {
//		Title string   `json:"title" validate:"required,min=3,max=200"`
//		Slug  string   `json:"slug" validate:"required,slug"`
//		Tags  []string `json:"tags" validate:"max=5,dive,required"`
//	}
//
// [Generate] walks it and writes validate_gen.go next to it. The method it
// writes is a [github.com/go-mizu/mizu/validate.Validator], which is what the
// rest of the toolkit looks for, so nothing at the call site changes.
//
// # What it is for
//
// The rules are the same rules either way. validate.Struct reads the tags with
// reflection, works out a plan for the type the first time it sees one, and
// walks the value against that plan on every request. This reads the same tags
// once, at build time, and writes the comparisons out as Go.
//
// That makes three things better and one thing worse. A tag with a typo in it
// is a build failure instead of an error on the first request that hits the
// route. The checking is a run of comparisons with nothing between them, so it
// allocates only when something failed. It is code somebody can read and step
// through in a debugger. And a rule registered with validate.Register is not
// available, because a name looked up while the program runs is not a name this
// can resolve while it is being compiled.
//
// It is not the default and it is not meant to be. Reflection checks a struct
// nobody generated for, which is the right answer while an application is being
// written, and this is the answer for the routes where the numbers matter.
//
// # What it writes
//
// One file per package, holding one method per marked struct and one function
// per struct type the rules reach:
//
//	func (v CreatePost) Validate(ctx context.Context) error {
//		var bad validate.Errors
//		validateCreatePost(&bad, "", v)
//		return bad.OrNil()
//	}
//
//	func validateCreatePost(bad *validate.Errors, at string, v CreatePost) {
//		if v.Title == "" {
//			bad.Add(at+"title", validate.Failed("required"))
//		} else if n := utf8.RuneCountInString(v.Title); n < 3 {
//			bad.Add(at+"title", validate.Failed("min", 3).Of("string"))
//		} else if n > 200 {
//			bad.Add(at+"title", validate.Failed("max", 200).Of("string"))
//		}
//		...
//	}
//
// The chain is what makes a field report one thing at a time. A title that is
// missing is missing, and saying it is also shorter than three characters is
// saying the same thing twice, so a rule that failed is the last one that runs
// on that field. Every other field is still checked, because somebody filling
// in a form wants the whole list.
//
// The method is the whole of what a caller sees and the checking is in a
// function beside it. That is what lets a type holding a list of itself have
// something to call, and what lets a type reached from two marked structs be
// written down once. at is what the fields are named under, empty at the top
// and lines.1. inside a list, so one function checks a type wherever it sits.
//
// # Where a field's name comes from
//
// The same rules web.Bind and validate.Struct follow, because all three have to
// agree. A path, header, cookie, form or query tag names the field, then the
// json tag, and then the field's own name in snake case. A request that failed
// to bind and a request that failed to validate come back in one document, and
// a field named two ways in it is a field a form cannot mark.
//
// A nested struct's fields are named under it, so a City inside an Address is
// address.city, and an embedded struct adds nothing to the name. An element of
// a list is named by its position, so tags.1, and a struct element's own fields
// are named under that, lines.1.sku.
//
// Both validate:"-" and json:"-" leave a field alone.
//
// # Rules
//
// Every rule the validate package's tag interpreter knows:
//
//	required     the value is filled in, which is the zero value being absent
//	omitempty    the rules after this one do not run on a value that is not
//	min max      a bound, on characters, elements, a number or a length of time
//	size         exactly that many
//	between      two bounds, as in between=3 10
//	dive         the rules after this one are about each element
//
// And the format checks, each of which is one of the exported Is functions in
// the validate package: cidr, e164, email, hostname, ip, ipv4, ipv6, mac, port,
// ulid, uri, url and uuid.
//
// What a size rule counts is what the field is. Characters in a string, counted
// as runes so an emoji is one. Elements in a slice, a map or an array. The
// number itself. The length of time itself, for a field that is exactly a
// time.Duration, whose bounds are then written as 1h30m rather than as a count
// of nanoseconds.
//
// A rule reads through pointers, so a *string is checked as a string. A nil
// pointer is checked as the zero value of what it points at, which is the same
// thing an absent value of that type would be, and telling absent from empty is
// what required is for.
//
// # Limits
//
// A rule added with validate.Register is refused. The name is looked up while
// the program runs and this runs before there is one, so a struct that needs
// one is a struct to leave to validate.Struct.
//
// A field that holds an interface is refused, for the same reason: what is in
// it is not known until there is a value, and reflection is the mode that can
// look.
//
// A struct nested more than twelve deep is refused, which is further down than
// a request has any reason to go.
//
// A whole number bound on an integer field is compared as it was written, and
// the tag interpreter compares the two as float64. The two answers differ only
// for a value above 2^53 on a 64 bit field, where the generated comparison is
// the exact one and the interpreted one has rounded. Every other bound that
// would not compare the same is written through a float64 conversion so that
// both modes round together.
//
// A map is not something dive goes into. Failures come back in the order they
// were found and a map has no order, so a map dived into would report the same
// problems in a different order on every run.
package validategen
