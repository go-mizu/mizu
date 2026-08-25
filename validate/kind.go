package validate

import (
	"reflect"
	"time"
	"unicode/utf8"
)

// The subjects a size rule can be about, which are the five sentences min, max,
// between and size each have. [RuleError.Of] records one of these, and the
// message table is keyed by the rule and the subject together.
const (
	subjString   = "string"
	subjNumeric  = "numeric"
	subjArray    = "array"
	subjDuration = "duration"
)

// isEmpty is whether a value is one somebody did not fill in.
//
// The rule is the zero value for the type, with one adjustment: an empty slice
// or map is missing even though a non-nil empty slice is not the zero value.
// So "", 0, false, a nil pointer, an empty list and the zero time.Time are all
// missing, and everything else is present.
//
// A field that has to tell zero apart from absent is a field whose type says
// so, a pointer or an xs.Option. That is the binder's job rather than this
// one's: by the time a value is here it is a Go value, and a Go value that is 0
// cannot remember whether the request said so.
func isEmpty(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return v == ""
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Pointer:
		return rv.IsNil() || isEmpty(rv.Elem().Interface())
	case reflect.Slice, reflect.Map, reflect.Array:
		return rv.Len() == 0
	default:
		return rv.IsZero()
	}
}

// sizeOf says what a size rule counts for a value, and what it is counting.
//
// A string is counted in runes rather than bytes, so an emoji is one character
// the way somebody typing it would say. A number compares as itself, a list and
// a map count their elements, and a duration compares as a duration.
//
// It panics on a value that has no size, because .Min(3) on a struct is a
// mistake in the program and not a problem with the request. A rule that
// quietly passed would hide it until somebody read the code, and a rule that
// quietly failed would tell a user their input was wrong when it was not.
func sizeOf(value any) (subject string, n float64) {
	value = indirect(value)

	switch v := value.(type) {
	case nil:
		panic("validate: a size rule on a nil value")
	case string:
		return subjString, float64(utf8.RuneCountInString(v))
	case time.Duration:
		return subjDuration, float64(v)
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return subjNumeric, float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return subjNumeric, float64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return subjNumeric, rv.Float()
	case reflect.Slice, reflect.Map, reflect.Array:
		return subjArray, float64(rv.Len())
	case reflect.String:
		return subjString, float64(utf8.RuneCountInString(rv.String()))
	}
	panic("validate: a size rule on a " + rv.Kind().String())
}

// number turns a rule's parameter into something comparable with what sizeOf
// returned.
//
// The parameter keeps its own type in the message, so Min(time.Hour) writes an
// hour and Min(3) writes 3. This is only the comparison.
func number(bound any) float64 {
	bound = indirect(bound)

	switch v := bound.(type) {
	case nil:
		panic("validate: a size rule with no bound")
	case time.Duration:
		return float64(v)
	}

	rv := reflect.ValueOf(bound)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint())
	case reflect.Float32, reflect.Float64:
		return rv.Float()
	}
	panic("validate: a size rule bounded by a " + rv.Kind().String())
}

// text is the string a format check reads, and whether the value held one.
//
// A pointer is followed, so a *string field checks the string it points at and
// a nil one is empty rather than a panic.
func text(value any) (string, bool) {
	value = indirect(value)
	if s, ok := value.(string); ok {
		return s, true
	}
	if value == nil {
		return "", true
	}
	if rv := reflect.ValueOf(value); rv.Kind() == reflect.String {
		return rv.String(), true
	}
	return "", false
}

// indirect follows pointers down to the value inside, and returns nil rather
// than a typed nil pointer.
//
// reflect.ValueOf takes the dynamic value out of an interface on the way in, so
// there is nothing to unwrap for that case and no branch for it here.
func indirect(value any) any {
	for {
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Pointer {
			return value
		}
		if rv.IsNil() {
			return nil
		}
		value = rv.Elem().Interface()
	}
}
