package web

import (
	"encoding"
	"errors"
	"reflect"
	"strconv"
	"time"
)

// A setter puts one string from the request into one field.
//
// It reports what is wrong with the string rather than what went wrong reading
// it, because the two are the same thing here and the caller is a form that has
// to say something next to the field.
type setter func(v reflect.Value, s string) *badValue

// A badValue is one value that is not what the field it arrived for holds.
//
// The code is the machine name a client switches on and a test asserts, and the
// message is what goes next to the field. Neither says what arrived: a message
// that quotes the input is a message that renders whatever somebody sent.
type badValue struct {
	code string
	msg  string
}

var (
	timeType     = reflect.TypeFor[time.Time]()
	durationType = reflect.TypeFor[time.Duration]()
	textType     = reflect.TypeFor[encoding.TextUnmarshaler]()
)

// setterFor is how a string becomes a value of type t, and whether an empty
// string is one of them.
//
// A nil setter means no string becomes one of these, which is true of a map, a
// channel, an interface and a struct that is not a time. The caller decides
// what to do about it, since a struct is worth going into and the rest are only
// worth complaining about when somebody tagged them.
//
// The second result is why an empty value is not an error. A blank text input
// posts an empty string, which is a value, and a blank number input posts one
// too, which is not: it is a field somebody left alone. Binding treats it as
// unset and validation is what says it had to be filled in.
func setterFor(t reflect.Type) (setter, bool) {
	switch t {
	case timeType:
		return setTime, false
	case durationType:
		return setDuration, false
	}

	// A type that reads itself out of text says so, and that covers netip.Addr,
	// a uuid, and every id type an application writes for itself.
	if reflect.PointerTo(t).Implements(textType) {
		return setText, false
	}

	switch t.Kind() {
	case reflect.String:
		return func(v reflect.Value, s string) *badValue {
			v.SetString(s)
			return nil
		}, true

	case reflect.Bool:
		return setBool, false

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		bits := t.Bits()
		return func(v reflect.Value, s string) *badValue {
			n, err := strconv.ParseInt(s, 10, bits)
			if err != nil {
				return numberError(err, "Must be a whole number.")
			}
			v.SetInt(n)
			return nil
		}, false

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		bits := t.Bits()
		return func(v reflect.Value, s string) *badValue {
			n, err := strconv.ParseUint(s, 10, bits)
			if err != nil {
				return numberError(err, "Must be a whole number and not negative.")
			}
			v.SetUint(n)
			return nil
		}, false

	case reflect.Float32, reflect.Float64:
		bits := t.Bits()
		return func(v reflect.Value, s string) *badValue {
			f, err := strconv.ParseFloat(s, bits)
			if err != nil {
				return numberError(err, "Must be a number.")
			}
			v.SetFloat(f)
			return nil
		}, false

	case reflect.Slice:
		if !isBytes(t) {
			return nil, false
		}
		return func(v reflect.Value, s string) *badValue {
			v.SetBytes([]byte(s))
			return nil
		}, true

	case reflect.Pointer:
		set, text := setterFor(t.Elem())
		if set == nil {
			return nil, false
		}
		elem := t.Elem()
		return func(v reflect.Value, s string) *badValue {
			if v.IsNil() {
				v.Set(reflect.New(elem))
			}
			return set(v.Elem(), s)
		}, text
	}
	return nil, false
}

// isBytes reports whether t is a []byte rather than a slice of something that
// is one byte wide and means something else.
func isBytes(t reflect.Type) bool {
	return t.Elem().Kind() == reflect.Uint8 && t.Elem().PkgPath() == ""
}

// setText hands the string to the type, which knows what it wants.
//
// What comes back when it refuses says nothing about why. The type's own error
// was written for whoever called it and not for whoever filled in the form, and
// putting it in a response is how the name of an internal type ends up on a
// page.
func setText(v reflect.Value, s string) *badValue {
	err := v.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(s))
	if err != nil {
		return &badValue{"invalid_value", "Is not in the right format."}
	}
	return nil
}

// setBool also takes what a checkbox sends.
//
// A ticked checkbox posts on and an unticked one posts nothing at all, which is
// why a bool that was not sent stays false and why there is no way to tell a
// cleared checkbox from a field that was never on the form. A hidden input with
// off in it next to the checkbox is the usual answer, and it is why off is here
// as well.
func setBool(v reflect.Value, s string) *badValue {
	switch s {
	case "on", "yes":
		v.SetBool(true)
		return nil
	case "off", "no":
		v.SetBool(false)
		return nil
	}

	b, err := strconv.ParseBool(s)
	if err != nil {
		return &badValue{"invalid_boolean", "Must be true or false."}
	}
	v.SetBool(b)
	return nil
}

// timeLayouts are the shapes a date arrives in.
//
// RFC 3339 is what a program sends and the other three are what the date,
// datetime-local and seconds-enabled datetime-local inputs send, none of which
// carry a zone. Those three are read as UTC, which is what time.Parse does with
// a layout that has no zone in it, and an application that wants them in the
// user's zone reads the zone from somewhere it can trust rather than from the
// input.
var timeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02",
}

func setTime(v reflect.Value, s string) *badValue {
	for _, layout := range timeLayouts {
		t, err := time.Parse(layout, s)
		if err != nil {
			continue
		}
		v.Set(reflect.ValueOf(t))
		return nil
	}
	return &badValue{"invalid_time", "Must be a date, such as 2006-01-02, or a date and a time."}
}

func setDuration(v reflect.Value, s string) *badValue {
	d, err := time.ParseDuration(s)
	if err != nil {
		return &badValue{"invalid_duration", "Must be a length of time, such as 30s or 5m."}
	}
	v.SetInt(int64(d))
	return nil
}

// numberError tells a number that will not parse from one that will not fit.
//
// They are the same call and different mistakes. A client sending letters where
// a number goes has a bug, and a client sending a number too big for an int32
// has a number that was fine until somebody chose the field's width.
func numberError(err error, msg string) *badValue {
	if errors.Is(err, strconv.ErrRange) {
		return &badValue{"out_of_range", "Is too large for this field."}
	}
	return &badValue{"invalid_number", msg}
}
