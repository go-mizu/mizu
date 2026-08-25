//go:build goexperiment.jsonv2

package web

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/go-mizu/mizu/errs"
)

// jsonRefusesDuplicates is what this build does with a member sent twice, and
// it is the one behaviour the two builds do not share. json_test.go holds each
// of them to what it says here.
const jsonRefusesDuplicates = true

// jsonDecode reads one JSON value from r into v.
//
// Anything after the value other than whitespace is an error, and so is a
// member sent twice, both of which this version refuses on its own.
// UnmarshalRead is one call because a single value and nothing after it is what
// it already means. That is also where the speed is: building a decoder costs a
// read buffer, and this one comes from a pool.
func jsonDecode(r io.Reader, v any, lax bool) error {
	return json.UnmarshalRead(r, v, json.RejectUnknownMembers(!lax))
}

// jsonField is the one member the decoder was talking about, when it was
// talking about one.
//
// This version carries a JSON Pointer on both of its error types, so the name
// comes off the error rather than out of its text, and a member inside an
// object comes back as address.city, which is the name binding gives a nested
// struct anywhere else.
func jsonField(err error) (errs.Field, bool) {
	var sem *json.SemanticError
	if errors.As(err, &sem) {
		if name := pointerName(sem.JSONPointer); name != "" {
			if errors.Is(sem.Err, json.ErrUnknownName) {
				return errs.Field{Name: name, Code: "unknown_field", Msg: "Is not a field this endpoint takes."}, true
			}
			return errs.Field{Name: name, Code: "invalid_value", Msg: "Is not in the right format."}, true
		}
	}

	var syn *jsontext.SyntacticError
	if errors.As(err, &syn) && errors.Is(syn.Err, jsontext.ErrDuplicateName) {
		if name := pointerName(syn.JSONPointer); name != "" {
			return errs.Field{Name: name, Code: "duplicate_field", Msg: "Was sent more than once."}, true
		}
	}
	return errs.Field{}, false
}

// pointerName is a JSON Pointer as a field name.
//
// Tokens comes back unescaped, so a member whose name has a slash in it is one
// token rather than two. An array index is a token like any other, which makes
// the third element of tags read tags.2.
func pointerName(p jsontext.Pointer) string {
	return strings.Join(slices.Collect(p.Tokens()), ".")
}
