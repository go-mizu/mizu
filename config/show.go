package config

import (
	"encoding/base64"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Redacted is what a secret looks like anywhere it is printed.
const Redacted = "***"

// Show is one value written out for a person to read.
//
// Generated code calls it once per field, and the result is what config:show
// puts in the value column and what config:diff compares. It is not a format
// anything reads back: a value goes into a field through a [Parse], and comes
// out of one through this.
//
// A type that knows how to write itself is asked, which covers [time.Duration],
// [log/slog.Level], the netip types and anything else with a String method.
func Show[T any](v T) string {
	switch x := any(v).(type) {
	case string:
		return x
	case []byte:
		return base64.StdEncoding.EncodeToString(x)
	case bool:
		return strconv.FormatBool(x)
	case time.Time:
		// Ahead of the Stringer case, because a configuration file writes a
		// moment as RFC 3339 and time.Time.String does not.
		return x.Format(time.RFC3339)
	case fmt.Stringer:
		return x.String()
	}
	return fmt.Sprint(v)
}

// ShowSlice is a list written the way [Value.Display] writes one, so a value
// that came from a file and a value that is in the struct look the same in
// config:show.
func ShowSlice[T any](vs []T) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = Show(v)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ShowMap is a table, in key order, so two runs of config:show agree.
func ShowMap[T any](m map[string]T) string {
	var b strings.Builder
	b.WriteByte('{')
	for _, k := range slices.Sorted(maps.Keys(m)) {
		if b.Len() > 1 {
			b.WriteString(", ")
		}
		b.WriteString(k)
		b.WriteString(" = ")
		b.WriteString(Show(m[k]))
	}
	b.WriteByte('}')
	return b.String()
}

// A FieldDoc is one setting and what a person needs to know about it.
//
// Generated code holds the part that comes from the struct, and [Describe]
// fills in the part that comes from the configuration in hand. config:doc
// prints the first part and config:show prints both.
type FieldDoc struct {
	Field

	// Type is the Go type of the field, written the way the struct writes it.
	Type string

	// Doc is the field's doc comment as one line, which is where the
	// explanation of a setting belongs: next to the field it is about.
	Doc string

	// Value is what the field holds, or [Redacted] for a secret that is set.
	// It is empty until Describe fills it in.
	Value string
}

// Describe pairs the settings a configuration has with the values it holds.
//
// The two slices line up, entry for entry, because generated code builds both
// from the same walk of the same struct.
func Describe(fields []FieldDoc, values []string) []FieldDoc {
	out := slices.Clone(fields)
	for i := range out {
		if i < len(values) {
			out[i].Value = hide(out[i].Secret, values[i])
		}
	}
	return out
}

// A Change is one setting that two configurations disagree about.
type Change struct {
	Field
	From string
	To   string
}

func (c Change) String() string {
	return c.Path + ": " + c.From + " -> " + c.To
}

// Diff is every setting the two sets of values disagree about, in the order
// the fields are declared.
//
// A secret that changed is reported as changed without either value being
// printed, because the point of config:diff is to find out that something
// moved and not to read what it moved to.
func Diff(fields []FieldDoc, from, to []string) []Change {
	var out []Change
	for i, f := range fields {
		if i >= len(from) || i >= len(to) || from[i] == to[i] {
			continue
		}
		out = append(out, Change{
			Field: f.Field,
			From:  hide(f.Secret, from[i]),
			To:    hide(f.Secret, to[i]),
		})
	}
	return out
}

// hide is a value as it may be printed. A secret that is set is never printed,
// and a secret that is not set says so, since knowing a secret is missing is
// the whole reason to look.
func hide(secret bool, value string) string {
	if secret && value != "" {
		return Redacted
	}
	return value
}
