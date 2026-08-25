package console

import (
	"encoding"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
)

// A Value is where a flag or an argument parses into.
//
// It is the interface from the standard library's flag package, so a Value
// written against that works here unchanged. The constructors below cover the
// types a command line usually carries, and anything else is a type with a Set
// method on it.
type Value interface {
	String() string
	Set(string) error
}

// A boolean Value takes no argument on the command line. --dry-run is enough,
// --dry-run=false is the long way round, and --no-dry-run is the opposite.
//
// The method name is the one the standard library's flag package looks for, so
// a Value written for that keeps its meaning here.
type boolean interface {
	Value
	IsBoolFlag() bool
}

// A counter counts how many times its flag appeared. -vvv is three.
type counter interface {
	Value
	Count()
}

// value is the plumbing behind most of the constructors below.
type value[T any] struct {
	p     *T
	parse func(string) (T, error)
}

func (v value[T]) Set(s string) error {
	x, err := v.parse(s)
	if err != nil {
		return err
	}
	*v.p = x
	return nil
}

// String renders what is in there now, for help text and for a message about a
// value that was rejected later.
//
// This is the one place in the package that reaches for fmt.Sprint on a type it
// knows nothing about. Parsing does not, which is the half that runs in a loop.
func (v value[T]) String() string {
	if v.p == nil {
		return ""
	}
	return fmt.Sprint(*v.p)
}

// Var returns a Value that parses into p with parse.
//
// It is what the rest of these are built on, and what a command reaches for
// when it wants a type that is not here:
//
//	console.Var(&c.Level, log.ParseLevel)
func Var[T any](p *T, parse func(string) (T, error)) Value {
	return value[T]{p: p, parse: parse}
}

// String returns a Value that takes the text as it stands.
func String[T ~string](p *T) Value {
	return Var(p, func(s string) (T, error) { return T(s), nil })
}

// Int returns a Value that parses a signed number.
//
// The base comes from the text, so 0x2a and 0b101010 and 42 all work, and an
// underscore may be used as a separator. A value too large for the type it is
// going into is an error rather than a wrap around.
func Int[T ~int | ~int8 | ~int16 | ~int32 | ~int64](p *T) Value {
	return Var(p, func(s string) (T, error) {
		n, err := strconv.ParseInt(s, 0, 64)
		if err != nil {
			return 0, numError(s, err)
		}
		if v := T(n); int64(v) == n {
			return v, nil
		}
		return 0, fmt.Errorf("%s does not fit", s)
	})
}

// Uint returns a Value that parses an unsigned number. A negative one is an
// error naming the sign rather than a very large positive number.
func Uint[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr](p *T) Value {
	return Var(p, func(s string) (T, error) {
		n, err := strconv.ParseUint(s, 0, 64)
		if err != nil {
			if strings.HasPrefix(s, "-") {
				return 0, fmt.Errorf("%s is negative", s)
			}
			return 0, numError(s, err)
		}
		if v := T(n); uint64(v) == n {
			return v, nil
		}
		return 0, fmt.Errorf("%s does not fit", s)
	})
}

// Float returns a Value that parses a number with a fractional part.
func Float[T ~float32 | ~float64](p *T) Value {
	return Var(p, func(s string) (T, error) {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, numError(s, err)
		}
		if v := T(f); !isInf(float64(v)) || isInf(f) {
			return v, nil
		}
		return 0, fmt.Errorf("%s does not fit", s)
	})
}

// isInf reports whether f is an infinity, without pulling in math for it.
func isInf(f float64) bool { return f > maxFloat || f < -maxFloat }

const maxFloat = 1.7976931348623157e308

// numError turns what strconv says into something worth printing.
//
// strconv reports the function it was in, which tells a person reading a
// command line error nothing at all.
func numError(s string, err error) error {
	if errors.Is(err, strconv.ErrRange) {
		return fmt.Errorf("%s is out of range", s)
	}
	return fmt.Errorf("%q is not a number", s)
}

// Bool returns a Value for a flag that takes no argument.
func Bool[T ~bool](p *T) Value { return boolValue[T]{p} }

type boolValue[T ~bool] struct{ p *T }

func (v boolValue[T]) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return fmt.Errorf("%q is not true or false", s)
	}
	*v.p = T(b)
	return nil
}

func (v boolValue[T]) String() string {
	if v.p == nil {
		return ""
	}
	return strconv.FormatBool(bool(*v.p))
}

func (boolValue[T]) IsBoolFlag() bool { return true }

// Count returns a Value that counts how many times its flag appeared, which is
// what -v, -vv and -vvv are.
//
// It takes no argument, and an explicit one is still allowed, so --verbose=2
// says the same thing as -vv.
func Count(p *int) Value { return countValue{p} }

type countValue struct{ p *int }

func (v countValue) Count() { *v.p++ }

func (v countValue) Set(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil {
		return numError(s, err)
	}
	*v.p = n
	return nil
}

func (v countValue) String() string {
	if v.p == nil {
		return ""
	}
	return strconv.Itoa(*v.p)
}

func (countValue) IsBoolFlag() bool { return true }

// Duration returns a Value that parses 30s, 5m, 1h30m and the rest of what
// time.ParseDuration takes.
func Duration[T ~int64](p *T) Value {
	return Var(p, func(s string) (T, error) {
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, fmt.Errorf("%q is not a length of time, try 30s or 5m", s)
		}
		return T(d), nil
	})
}

// Time returns a Value that parses a time in any of the given layouts, and in
// RFC 3339 or as a plain date when none are given.
//
// The plain date is there because a person typing --since on a command line
// types 2026-01-01, and being told that a date is not a time is not an answer.
func Time(p *time.Time, layouts ...string) Value {
	if len(layouts) == 0 {
		layouts = []string{time.RFC3339, time.DateOnly}
	}
	return Var(p, func(s string) (time.Time, error) {
		for _, layout := range layouts {
			if t, err := time.Parse(layout, s); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("%q is not a time, try %s", s, layouts[0])
	})
}

// Text returns a Value for any type that knows how to read itself from text,
// which is netip.Addr, netip.Prefix, big.Int, a UUID, and most of what a
// library exports for this.
//
//	var addr netip.Addr
//	console.Text(&addr)
func Text[T encoding.TextUnmarshaler](p T) Value { return textValue[T]{p} }

type textValue[T encoding.TextUnmarshaler] struct{ p T }

func (v textValue[T]) Set(s string) error { return v.p.UnmarshalText([]byte(s)) }

func (v textValue[T]) String() string {
	if m, ok := any(v.p).(encoding.TextMarshaler); ok {
		if text, err := m.MarshalText(); err == nil {
			return string(text)
		}
	}
	return fmt.Sprint(v.p)
}

// Slice returns a Value that appends rather than replaces, so a flag holding
// one collects every time it was given.
//
// A non-empty sep also splits a single occurrence, so --tags a,b,c and --tags a
// --tags b --tags c mean the same thing. An empty sep leaves the text alone,
// which is what a flag taking file paths or SQL wants.
func Slice[T any](p *[]T, parse func(string) (T, error), sep string) Value {
	return sliceValue[T]{p: p, parse: parse, sep: sep}
}

// Strings is the slice most commands want.
func Strings(p *[]string, sep string) Value {
	return Slice(p, func(s string) (string, error) { return s, nil }, sep)
}

type sliceValue[T any] struct {
	p     *[]T
	parse func(string) (T, error)
	sep   string
}

func (v sliceValue[T]) Set(s string) error {
	parts := []string{s}
	if v.sep != "" {
		parts = strings.Split(s, v.sep)
	}
	for _, part := range parts {
		x, err := v.parse(part)
		if err != nil {
			return err
		}
		*v.p = append(*v.p, x)
	}
	return nil
}

func (v sliceValue[T]) String() string {
	if v.p == nil {
		return ""
	}
	parts := make([]string, len(*v.p))
	for i, x := range *v.p {
		parts[i] = fmt.Sprint(x)
	}
	return strings.Join(parts, ",")
}

// KeyValues returns a Value that collects key=value pairs, one per occurrence.
//
//	--label env=staging --label team=platform
//
// A repeated key takes the last one, the same as a repeated flag anywhere else.
// A pair with no equals sign is an error naming the pair, because the shape a
// person meant is not recoverable from it.
func KeyValues(p *map[string]string) Value { return mapValue{p} }

type mapValue struct{ p *map[string]string }

func (v mapValue) Set(s string) error {
	key, val, ok := strings.Cut(s, "=")
	if !ok || key == "" {
		return fmt.Errorf("%q is not a key=value pair", s)
	}
	if *v.p == nil {
		*v.p = make(map[string]string)
	}
	(*v.p)[key] = val
	return nil
}

func (v mapValue) String() string {
	if v.p == nil || *v.p == nil {
		return ""
	}
	pairs := make([]string, 0, len(*v.p))
	for _, key := range slices.Sorted(maps.Keys(*v.p)) {
		pairs = append(pairs, key+"="+(*v.p)[key])
	}
	return strings.Join(pairs, ",")
}

// Enum returns a Value that takes one of a list, and says which list when it
// does not.
func Enum[T ~string](p *T, options ...T) Value {
	return Var(p, func(s string) (T, error) {
		if v := T(s); slices.Contains(options, v) {
			return v, nil
		}
		names := make([]string, len(options))
		for i, option := range options {
			names[i] = string(option)
		}
		return "", fmt.Errorf("%q is not one of %s", s, strings.Join(names, ", "))
	})
}
