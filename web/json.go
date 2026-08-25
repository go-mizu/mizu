package web

// The JSON a request body speaks is the same JSON either way, but the package
// that reads it is not. Go 1.27 builds encoding/json/v2 by default and leaves
// the old one in place, so web still compiles under GOEXPERIMENT=nojsonv2 and
// json_v1.go is what it compiles then.
//
// The two files are held to the same behaviour rather than to the same code:
// one JSON value in, trailing data refused, a member the struct has no field
// for refused unless the struct embeds AllowUnknown, and a decode failure
// carrying the name of the member it was about wherever the decoder said it.
// json_test.go is that contract written down, and running the package both ways
// is what checks it.
//
// Two things the two do not promise. The error text is different and no caller
// reads further into it than the code and the field name. A member sent twice
// is refused by json_v2.go and taken as the last one by json_v1.go, which is
// the one place the newer build is stricter and cannot be talked out of it
// without a second pass over the body.
//
// Nothing outside those two files names encoding/json, so a third version, or a
// decision to stop carrying the old one, is a change to two files and no call
// sites.
