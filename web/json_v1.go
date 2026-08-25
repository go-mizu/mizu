//go:build !goexperiment.jsonv2

package web

import (
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/go-mizu/mizu/errs"
)

// errTrailing is how this version reports a body carrying more than one JSON
// value. The other one has a message of its own for it, with an offset in it,
// and neither is worth matching against.
var errTrailing = errors.New("more than one JSON value in the body")

// jsonRefusesDuplicates is what this build does with a member sent twice, and
// it is the one behaviour the two builds do not share. json_test.go holds each
// of them to what it says here.
const jsonRefusesDuplicates = false

// jsonOmitemptyMeansZero is how this build reads the omitempty tag when it
// writes a response, and it is the other behaviour the two builds do not share.
// Here omitempty drops the zero value, so a field holding the number zero is
// left out. reply_test.go holds each build to what it says here.
const jsonOmitemptyMeansZero = true

// jsonDecode reads one JSON value from r into v.
//
// A member sent twice is taken as the last one, which is what this version
// does and what it cannot be talked out of without reading the body a second
// time. json_v2.go refuses it.
func jsonDecode(r io.Reader, v any, lax bool) error {
	dec := json.NewDecoder(r)
	if !lax {
		dec.DisallowUnknownFields()
	}

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
			return errTrailing
		}
		return err
	}
	return nil
}

// jsonEncode writes v into buf as JSON, with no newline after it.
//
// The Encoder is what builds a value straight into a writer here, and it ends
// every value with a newline that it cannot be told not to write. json_v2.go
// writes no newline, so this one comes off again and the two builds put the
// same bytes on the wire.
//
// A value that fails to marshal writes nothing, because this version builds the
// whole thing before it writes any of it. The caller truncates the buffer
// anyway, so the two builds do not have to agree about that.
func jsonEncode(buf *jsonBuf, v any) error {
	enc := json.NewEncoder(buf)

	// This version turns <, > and & into escapes so that a body can be dropped
	// into a script tag without closing it. json_v2.go dropped that, on the
	// grounds that a response served as application/json is not in a script
	// tag and one that is belongs to the template package, which escapes what
	// it is given. Off here, so both builds send the character that was there.
	enc.SetEscapeHTML(false)

	if err := enc.Encode(v); err != nil {
		return err
	}
	if n := len(buf.b); n > 0 && buf.b[n-1] == '\n' {
		buf.b = buf.b[:n-1]
	}
	return nil
}

// jsonField is the one member the decoder was talking about, when it was
// talking about one.
//
// A value of the wrong type comes back as a type this version exports, with the
// member's name already on it and already joined with a dot for a member inside
// an object. An unknown member does not: it is an unexported error with nothing
// on it but the text, so the text is what there is to read.
func jsonField(err error) (errs.Field, bool) {
	var mismatch *json.UnmarshalTypeError
	if errors.As(err, &mismatch) && mismatch.Field != "" {
		return errs.Field{Name: mismatch.Field, Code: "invalid_value", Msg: "Is not in the right format."}, true
	}
	if name, ok := unknownName(err.Error()); ok {
		return errs.Field{Name: name, Code: "unknown_field", Msg: "Is not a field this endpoint takes."}, true
	}
	return errs.Field{}, false
}

// unknownName reads the member's name back out of the sentence this version
// writes about it, which is the whole of what it says.
func unknownName(msg string) (string, bool) {
	rest, ok := strings.CutPrefix(msg, `json: unknown field "`)
	if !ok {
		return "", false
	}
	return strings.CutSuffix(rest, `"`)
}
