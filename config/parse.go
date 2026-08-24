package config

import (
	"encoding"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/toml"
)

// A Parse reads one value into dst.
//
// Every parser in this package has this shape, so a field of a type the
// package has never heard of is one function away, and [Slice] and [Map] build
// new parsers out of the ones that are here.
//
//	config.Get(l, &c.HTTP.Addr, addr, config.String)
//	config.Get(l, &c.HTTP.Proxies, proxies, config.Slice(config.Prefix))
type Parse[T any] func(dst *T, v Value) error

// A Parser is a type that reads itself from configuration.
//
// Use it for a type that cannot be an [encoding.TextUnmarshaler], usually
// because it wants the whole of a table rather than a line of text.
type Parser interface {
	ParseConfig(v Value) error
}

// String reads a string, or any type whose underlying type is a string.
func String[T ~string](dst *T, v Value) error {
	s, err := text(v)
	if err != nil {
		return err
	}
	*dst = T(s)
	return nil
}

// Bool reads a boolean, or any type whose underlying type is a boolean. From
// a file it has to be written as one, and from anywhere else it is what
// [strconv.ParseBool] accepts, which includes 1, t, T, TRUE, true and True.
func Bool[T ~bool](dst *T, v Value) error {
	if v.TOML != nil {
		if v.TOML.Kind != toml.KindBool {
			return wantErr("a boolean", v)
		}
		*dst = T(v.TOML.Bool)
		return nil
	}
	b, err := strconv.ParseBool(v.Text)
	if err != nil {
		return fmt.Errorf("want a boolean, got %s", strconv.Quote(v.Text))
	}
	*dst = T(b)
	return nil
}

// Int reads a signed integer of any width, or any type whose underlying type
// is one. A number too large for the type is an error rather than a
// wraparound.
func Int[T ~int | ~int8 | ~int16 | ~int32 | ~int64](dst *T, v Value) error {
	size := bits(dst)
	if v.TOML != nil {
		if v.TOML.Kind != toml.KindInt {
			return wantErr("an integer", v)
		}
		if !fits(v.TOML.Int, size) {
			return tooBig(strconv.FormatInt(v.TOML.Int, 10), size)
		}
		*dst = T(v.TOML.Int)
		return nil
	}
	n, err := strconv.ParseInt(v.Text, 0, size)
	if err != nil {
		return numberErr(err, v.Text, size, "an integer")
	}
	*dst = T(n)
	return nil
}

// Uint reads an unsigned integer of any width, or any type whose underlying
// type is one. A negative number is an error rather than a very large one.
func Uint[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr](dst *T, v Value) error {
	size := bits(dst)
	if v.TOML != nil {
		if v.TOML.Kind != toml.KindInt {
			return wantErr("an integer", v)
		}
		if v.TOML.Int < 0 {
			return fmt.Errorf("want a number that is not negative, got %d", v.TOML.Int)
		}
		if !fitsUnsigned(uint64(v.TOML.Int), size) {
			return tooBig(strconv.FormatInt(v.TOML.Int, 10), size)
		}
		*dst = T(v.TOML.Int)
		return nil
	}
	n, err := strconv.ParseUint(v.Text, 0, size)
	if err != nil {
		return numberErr(err, v.Text, size, "a number that is not negative")
	}
	*dst = T(n)
	return nil
}

// Float reads a float32 or a float64, or any type whose underlying type is
// one. An integer in a file is accepted, because 1 and 1.0 are the same number
// to everyone except a parser.
func Float[T ~float32 | ~float64](dst *T, v Value) error {
	var f float64
	if v.TOML != nil {
		switch v.TOML.Kind {
		case toml.KindFloat:
			f = v.TOML.Float
		case toml.KindInt:
			f = float64(v.TOML.Int)
		default:
			return wantErr("a number", v)
		}
	} else {
		var err error
		if f, err = strconv.ParseFloat(v.Text, 64); err != nil {
			if errors.Is(err, strconv.ErrRange) {
				return fmt.Errorf("%s is too large a number to hold", v.Text)
			}
			return fmt.Errorf("want a number, got %s", strconv.Quote(v.Text))
		}
	}
	// A float32 that came from a number a float32 cannot hold turns into an
	// infinity, and a number the file did not say was infinite should not.
	*dst = T(f)
	if math.IsInf(float64(*dst), 0) && !math.IsInf(f, 0) {
		return fmt.Errorf("%v is too large a number to hold", f)
	}
	return nil
}

// Duration reads a length of time, written the way [time.ParseDuration] wants
// it: 30s, 2h45m, 150ms. It is a string in a file too, because a bare number
// leaves the reader guessing at the unit.
func Duration[T ~int64](dst *T, v Value) error {
	s, err := text(v)
	if err != nil {
		return err
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("want a length of time such as 30s or 2h45m, got %s", strconv.Quote(s))
	}
	*dst = T(d)
	return nil
}

// Time reads a moment. A file may write it as a date and time of its own, and
// anywhere else it is RFC 3339.
func Time(dst *time.Time, v Value) error {
	s := v.Text
	if v.TOML != nil {
		switch v.TOML.Kind {
		case toml.KindOffsetDateTime, toml.KindLocalDateTime, toml.KindLocalDate:
			*dst = v.TOML.Time
			return nil
		case toml.KindString:
			s = v.TOML.Str // a string in a file is the same syntax as anywhere else
		default:
			return wantErr("a date and time", v)
		}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fmt.Errorf("want a date and time such as 2026-01-01T00:00:00Z, got %s", strconv.Quote(s))
	}
	*dst = t
	return nil
}

// Bytes reads binary data, written as standard or URL safe base64, with or
// without padding. A base64: prefix is allowed and ignored, because a key
// written that way says what it is.
func Bytes(dst *[]byte, v Value) error {
	s, err := text(v)
	if err != nil {
		return err
	}
	s = strings.TrimPrefix(s, "base64:")
	b, err := decodeBase64(s)
	if err != nil {
		return errors.New("want base64, and this is not")
	}
	*dst = b
	return nil
}

func decodeBase64(s string) ([]byte, error) {
	encodings := []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	}
	var err error
	for _, enc := range encodings {
		var b []byte
		if b, err = enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, err
}

// Addr reads an IP address, such as 10.0.0.1 or ::1.
func Addr(dst *netip.Addr, v Value) error {
	s, err := text(v)
	if err != nil {
		return err
	}
	a, err := netip.ParseAddr(s)
	if err != nil {
		return fmt.Errorf("want an IP address, got %s", strconv.Quote(s))
	}
	*dst = a
	return nil
}

// Prefix reads a network, such as 10.0.0.0/8.
func Prefix(dst *netip.Prefix, v Value) error {
	s, err := text(v)
	if err != nil {
		return err
	}
	p, err := netip.ParsePrefix(s)
	if err != nil {
		return fmt.Errorf("want a network such as 10.0.0.0/8, got %s", strconv.Quote(s))
	}
	*dst = p
	return nil
}

// AddrPort reads an address and a port together, such as 127.0.0.1:8080.
func AddrPort(dst *netip.AddrPort, v Value) error {
	s, err := text(v)
	if err != nil {
		return err
	}
	a, err := netip.ParseAddrPort(s)
	if err != nil {
		return fmt.Errorf("want an address and a port such as 127.0.0.1:8080, got %s", strconv.Quote(s))
	}
	*dst = a
	return nil
}

// Level reads a logging level: debug, info, warn, error, or one of those with
// an offset, such as debug+2.
func Level(dst *slog.Level, v Value) error {
	s, err := text(v)
	if err != nil {
		return err
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(s)); err != nil {
		return fmt.Errorf("want a level such as debug, info, warn or error, got %s", strconv.Quote(s))
	}
	*dst = level
	return nil
}

// Text reads any type that unmarshals itself from text.
//
// The type parameter is the type of the field, and the pointer to it is what
// has to have the UnmarshalText method, which is where such a method always
// is.
//
//	config.Get(l, &c.App.URL, field, config.Text)
func Text[T any, P interface {
	*T
	encoding.TextUnmarshaler
}](dst *T, v Value) error {
	s, err := text(v)
	if err != nil {
		return err
	}
	if err := P(dst).UnmarshalText([]byte(s)); err != nil {
		return err
	}
	return nil
}

// Config reads any type that reads itself, which is the way in for a field
// that wants the whole of a table rather than a line of text.
func Config[T any, P interface {
	*T
	Parser
}](dst *T, v Value) error {
	return P(dst).ParseConfig(v)
}

// Slice makes a parser for a list out of a parser for one of its elements.
//
// A file writes a list as an array. Everywhere else it is separated by commas,
// and an element with a comma in it can be put in double quotes, so that
// a,b,"c,d" is three of them. Spaces around an element are not part of it.
//
// An element parser that is itself generic has to say which type it is for,
// because Go works out the type arguments of this call before it looks at what
// the result is used as:
//
//	config.Slice(config.Prefix)          // Prefix is not generic
//	config.Slice(config.String[string])  // String is, so name the type
func Slice[T any](one Parse[T]) Parse[[]T] {
	return func(dst *[]T, v Value) error {
		if v.TOML != nil {
			if v.TOML.Kind != toml.KindArray {
				return wantErr("a list", v)
			}
			out := make([]T, len(v.TOML.Array))
			for i, e := range v.TOML.Array {
				if err := one(&out[i], Value{Source: v.Source, TOML: e}); err != nil {
					return fmt.Errorf("item %d: %w", i+1, err)
				}
			}
			*dst = out
			return nil
		}
		parts := splitList(v.Text)
		out := make([]T, len(parts))
		for i, p := range parts {
			if err := one(&out[i], Value{Source: v.Source, Text: p}); err != nil {
				return fmt.Errorf("item %d: %w", i+1, err)
			}
		}
		*dst = out
		return nil
	}
}

// Map makes a parser for a set of named values out of a parser for one of
// them. It has to be written in a file, because there is no good way to write
// a table in an environment variable and every attempt at one is worse than
// putting it in the file where it belongs.
//
// As with [Slice], a generic element parser has to name its type, as
// config.Map(config.String[string]).
func Map[T any](one Parse[T]) Parse[map[string]T] {
	return func(dst *map[string]T, v Value) error {
		if v.TOML == nil {
			return errors.New("has to be written in a configuration file, not as text")
		}
		if v.TOML.Kind != toml.KindTable {
			return wantErr("a table", v)
		}
		out := make(map[string]T, v.TOML.Table.Len())
		for key, e := range v.TOML.Table.All() {
			var item T
			if err := one(&item, Value{Source: v.Source, TOML: e}); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
			out[key] = item
		}
		*dst = out
		return nil
	}
}

// splitList cuts a list written as text, respecting double quotes so that an
// element can have a comma in it.
func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	var item strings.Builder
	quoted := false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '"':
			quoted = !quoted
		case c == ',' && !quoted:
			out = append(out, strings.TrimSpace(item.String()))
			item.Reset()
		default:
			item.WriteByte(c)
		}
	}
	return append(out, strings.TrimSpace(item.String()))
}

// text is the value as a string, and an error when it came from a file and was
// not written as one.
func text(v Value) (string, error) {
	if v.TOML == nil {
		return v.Text, nil
	}
	if v.TOML.Kind != toml.KindString {
		return "", wantErr("a string", v)
	}
	return v.TOML.Str, nil
}

// fits is whether n survives being put in a signed number of that width.
func fits(n int64, size int) bool {
	if size == 64 {
		return true
	}
	limit := int64(1) << (size - 1)
	return n >= -limit && n < limit
}

// fitsUnsigned is the same for a number that has no sign.
func fitsUnsigned(n uint64, size int) bool {
	return size == 64 || n < uint64(1)<<size
}

// bits is how wide a number type is, which is what says whether a value fits
// in it. Shifting a one along until it falls off the end counts the width of
// a type without naming it, and works the same for a signed type and an
// unsigned one.
func bits[T integer](*T) int {
	n := 0
	for x := T(1); x != 0; x <<= 1 {
		n++
	}
	return n
}

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func wantErr(want string, v Value) error {
	return fmt.Errorf("want %s, got %s", want, v.TOML.Kind)
}

func tooBig(text string, size int) error {
	return fmt.Errorf("%s does not fit in %d bits", text, size)
}

func numberErr(err error, text string, size int, want string) error {
	if errors.Is(err, strconv.ErrRange) {
		return tooBig(text, size)
	}
	return fmt.Errorf("want %s, got %s", want, strconv.Quote(text))
}
