//go:build !goexperiment.jsonv2

package mizu

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// errTrailingJSON is how this version reports a body carrying more than one
// JSON value. The other one has a message of its own for it, with an offset in
// it, and neither is worth matching against.
var errTrailingJSON = errors.New("trailing data")

// jsonDecodeStrict reads exactly one JSON value from r into v.
//
// A member the target has no field for is an error, and so is anything after
// the value other than whitespace. A repeated member name is not: this version
// takes the last one, which is the one place the other build is stricter than
// this one and cannot be talked into agreeing.
func jsonDecodeStrict(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		return err
	}

	// A second Decode is how this version answers "is there anything left".
	// Reaching the end of the reader is the good outcome.
	//
	// It reads into a RawMessage rather than into an empty struct, because
	// DisallowUnknownFields is still on and an empty struct would report a
	// second object as an unknown field instead of as a second object.
	var rest json.RawMessage
	if err := dec.Decode(&rest); !errors.Is(err, io.EOF) {
		if err == nil {
			return errTrailingJSON
		}
		return err
	}
	return nil
}

// jsonWrite writes v to w as one JSON value and a newline.
//
// HTML metacharacters are left alone. Escaping them is a habit from writing
// JSON into a script tag, and a response with a content type of its own is not
// that.
//
// Map keys come out sorted without being asked, which is the other half of what
// json_v2.go spends two options on.
func jsonWrite(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// jsonMarshal returns v as JSON, with no trailing newline and no HTML
// escaping.
//
// The buffer is what it costs to turn escaping off in this version, since only
// the encoder has the switch. The other build has an option and no buffer.
func jsonMarshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}
