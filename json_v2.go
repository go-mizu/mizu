//go:build goexperiment.jsonv2

package mizu

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"io"
)

// jsonDecodeStrict reads exactly one JSON value from r into v.
//
// A member the target has no field for is an error, and so is anything after
// the value other than whitespace. A repeated member name is one too, which
// this version refuses on its own.
//
// UnmarshalRead is one call because a single value and nothing after it is what
// it already means, so there is no second read here to find the end of the
// input. That is also where the speed is: building a decoder costs a read
// buffer, and this one comes from a pool.
func jsonDecodeStrict(r io.Reader, v any) error {
	return json.UnmarshalRead(r, v, json.RejectUnknownMembers(true))
}

// marshalOptions is what both writers here pass, and what makes this version
// produce the same bytes as json_v1.go.
//
// HTML metacharacters are left alone. Escaping them is a habit from writing
// JSON into a script tag, and a response with a content type of its own is not
// that. This version does not escape them unless asked, so the option says out
// loud what the other build has to say with a method call.
//
// Deterministic is the one that costs something. This version leaves a Go map
// in whatever order the runtime hands it over, which is a different order every
// run, and that would make a response body unreproducible for anything
// comparing bytes: a golden file, an ETag, a diff between two servers. Sorting
// map keys is what the old encoder always did and what the caller who reached
// for a map instead of a struct is not thinking about.
var marshalOptions = json.JoinOptions(
	jsontext.EscapeForHTML(false),
	json.Deterministic(true),
)

// jsonWrite writes v to w as one JSON value and a newline.
func jsonWrite(w io.Writer, v any) error {
	if err := json.MarshalWrite(w, v, marshalOptions); err != nil {
		return err
	}

	// MarshalWrite stops at the end of the value. The newline is the other
	// build's, and matching it keeps a response byte for byte the same however
	// mizu was compiled.
	_, err := io.WriteString(w, "\n")
	return err
}

// jsonMarshal returns v as JSON, with no trailing newline and no HTML
// escaping.
func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v, marshalOptions)
}
