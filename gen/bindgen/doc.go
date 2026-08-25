// Package bindgen writes the BindRequest method for a request struct. It is an
// implementation detail of the mizu command and is exempt from the
// compatibility promise in doc 31. Import it only if you are extending mizu
// itself.
//
// A struct asks for one with a //mizu:bind marker:
//
//	//mizu:bind
//	type Listing struct {
//		Q    string   `query:"q"`
//		Tags []string `query:"tags"`
//		Page int      `query:"page"`
//	}
//
// [Generate] walks it and writes bind_gen.go next to it. The method it writes
// is a [github.com/go-mizu/mizu/web.Binder], which is what web.Bind looks for
// before it reaches for reflection, so nothing at the call site changes:
//
//	in, err := web.Bind[Listing](c)
//
// # What it is for
//
// The reflective binder asks net/http for the request's values, which builds a
// map with a slice in every entry and a string in every slice before anybody
// reads the first field. A generated binder knows the names it wants, so it
// reads the pairs off the query string and the body as they come and writes
// each one straight into the field it belongs to. On the twelve field form in
// doc 29 that is most of the time and nearly all of the allocations.
//
// It is not the default and it is not meant to be. Reflection binds a struct
// nobody generated for, which is the right answer while an application is being
// written, and this is the answer for the handful of routes where the numbers
// matter.
//
// # What it writes
//
// One file per package holding one method per marked struct:
//
//	func (v *Listing) BindRequest(c *web.Ctx) error {
//		b := c.Binding()
//
//		var tags []string
//
//		for name, value := range b.Values() {
//			switch name {
//			case "q":
//				v.Q = value
//			case "tags":
//				if tags == nil {
//					tags = []string{}
//				}
//				tags = append(tags, value)
//			case "page":
//				web.Int(b, &v.Page, name, value)
//			}
//		}
//		if tags != nil {
//			v.Tags = tags
//		}
//
//		b.Body(v)
//		return b.Err()
//	}
//
// Every decision is made here and written down, so the request path has no
// reflection in it at all and a field that cannot be bound is a build failure
// rather than an error on the first request that hits the route.
//
// # Where a field comes from
//
// The same rules the reflective binder follows, because the two have to agree:
//
//	path      a route parameter
//	header    a request header
//	cookie    a cookie
//	form      the query string and a form body
//	query     the same, under another name
//
// The first of those tags a field carries wins. With none of them the name
// comes from the json tag, and then from the field's own name in snake case, so
// a struct written for a JSON body binds from a query string without a second
// set of tags on it. Both bind:"-" and json:"-" leave a field alone.
//
// A nested struct's fields are named under it, so a City inside an Address is
// address.city, and an embedded struct adds nothing to the name. The prefix is
// on the query and the form only: a header is a flat namespace the request
// shares with everything else, and address.x-country is not the name of one
// wherever the field happens to sit.
//
// A field of type *web.Upload is a file part, whatever its name says. A form is
// the only place a file arrives, so a path, header or cookie tag on one is
// reported.
//
// # Types
//
// A field is read by the helper its type matches, and a type nothing matches is
// reported when the field is tagged and skipped when it is not, since an
// untagged field of some other type is a field binding has nothing to do with
// and there is one in nearly every struct:
//
//	string and []byte, written straight in
//	bool, and the on and off a checkbox sends
//	int, uint and float of every width, and any defined type over one of them
//	time.Time and time.Duration
//	anything with an UnmarshalText method, which is an encoding.TextUnmarshaler
//	a slice of any of the above, which takes every value sent under the name
//	a pointer to any of the above, allocated when something arrives for it
//	*web.Upload and []*web.Upload
//
// Anything else is a struct that holds more fields, and the walk goes into it.
// A struct that holds one of itself stops rather than going round, and so does
// the reflective binder: a query string has no way to carry a tree, and the
// body decoder that does handles it without a plan from here.
//
// # Differences
//
// The generated binder and the reflective one answer the same request the same
// way. Three things are worth writing down anyway.
//
// Two fields reading the same name from the query or the form are reported
// here. That is the one thing the output cannot express, since it is two cases
// of one switch on the same string, and it is a mistake either way.
//
// A value behind two pointers is reported here. The reflective binder writes
// through as many as it finds, and a form has no way to mean the difference.
//
// A repeated field carrying more than one value that will not decode reports
// every one of them here, where the reflective binder stops at the first. Both
// report the field, and neither fills it in.
//
// # Limits
//
// A struct nested more than twelve deep is reported, which is further down than
// a request has any reason to go.
//
// Validation is not here. What a value has to be is a validate tag, read by the
// validate package, and this leaves it alone.
package bindgen
