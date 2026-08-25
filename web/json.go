package web

// The JSON a request body speaks is the same JSON either way, but the package
// that reads it is not. Go 1.27 builds encoding/json/v2 by default and leaves
// the old one in place, so web still compiles under GOEXPERIMENT=nojsonv2 and
// json_v1.go is what it compiles then.
//
// The two files are held to the same behaviour rather than to the same code:
// one JSON value in, trailing data refused, a member the struct has no field
// for refused unless the struct embeds AllowUnknown, a decode failure carrying
// the name of the member it was about wherever the decoder said it, and one
// JSON value out with no newline after it. json_test.go is that contract
// written down, and running the package both ways is what checks it.
//
// Writing takes three settings to line the two up. The older Encoder ends every
// value with a newline, which comes off. It turns <, > and & into escapes,
// which is turned off. And it sorts a map's members by key where the newer one
// does not, so the newer one is asked to, which a response needs anyway for an
// ETag over it to mean anything.
//
// Three things the two do not promise. The error text is different and no
// caller reads further into it than the code and the field name. A member sent
// twice is refused by json_v2.go and taken as the last one by json_v1.go, which
// is the one place the newer build is stricter and cannot be talked out of it
// without a second pass over the body.
//
// The third is omitempty, and it is the one that shows up in a response body. It
// means "omit the zero value" in the older package and "omit a value that would
// be written as null, an empty string, an empty object or an empty array" in the
// newer one, so a field holding the number zero is dropped by one build and
// written by the other. That is a change encoding/json/v2 made on purpose and
// there is no option that undoes it. omitzero means the same thing in both, it
// is what doc 08 tells people to write, and json_test.go holds both builds to
// the difference rather than papering over it.
//
// Nothing outside those two files names encoding/json, so a third version, or a
// decision to stop carrying the old one, is a change to two files and no call
// sites.
