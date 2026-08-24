package toml

import (
	"fmt"
	"iter"
	"slices"
	"strconv"
	"time"
)

// A Kind says what a [Value] holds.
type Kind int

const (
	KindInvalid Kind = iota
	KindString
	KindInt
	KindFloat
	KindBool
	KindOffsetDateTime
	KindLocalDateTime
	KindLocalDate
	KindLocalTime
	KindArray
	KindTable
)

func (k Kind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindInt:
		return "integer"
	case KindFloat:
		return "float"
	case KindBool:
		return "boolean"
	case KindOffsetDateTime:
		return "offset date-time"
	case KindLocalDateTime:
		return "local date-time"
	case KindLocalDate:
		return "local date"
	case KindLocalTime:
		return "local time"
	case KindArray:
		return "array"
	case KindTable:
		return "table"
	}
	return "invalid"
}

// A Position is where something was written.
type Position struct {
	File string
	Line int
	Col  int // in bytes, counting from one
}

func (p Position) String() string {
	if p.File == "" {
		return fmt.Sprintf("%d:%d", p.Line, p.Col)
	}
	return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Col)
}

// A Value is one value from a document, with the position it was written at.
//
// Kind says which of the remaining fields holds the value, and the rest are
// zero. A record with a tag beats an interface here because of what reads it:
// a generated decoder checks the kind and takes the field, with no type
// assertion, no allocation, and nowhere for a nil to come from.
//
// The four date and time kinds all use Time. A local date has a zero clock, a
// local time has a zero date, and neither has an offset, so both sit in UTC
// and the kind is what says the offset was never written down.
type Value struct {
	Kind Kind
	Pos  Position

	Str   string
	Int   int64
	Float float64
	Bool  bool
	Time  time.Time
	Array []*Value
	Table *Table

	// arrayOfTables marks an array written as [[name]] rather than as an
	// array literal. Only the first can be appended to by a later header.
	arrayOfTables bool
}

// Errorf returns an error about this value, with its position on the front.
//
//	config/local.toml:4:7: db.dsn: want a string, got an integer
//
// It is here because every caller reading a value has the same thing to say
// when the value is not what it wanted, and the position is the part that
// makes it worth reading.
func (v *Value) Errorf(format string, a ...any) error {
	return &Error{Pos: v.Pos, Msg: fmt.Sprintf(format, a...)}
}

// A Table is a set of keys and values, in the order they were written.
//
// Document order is kept because output made from a table has to come out the
// same way every time, and because an error message naming the third key is
// easier to find when the third key is where it was written.
type Table struct {
	Pos Position

	keys []string
	vals map[string]*Value

	// How the table came about, which is what the rules about redefining one
	// are written in terms of.
	explicit bool // written as [name], rather than implied by [name.sub]
	inline   bool // written as {}, which closes it for good
	dotted   bool // implied by a dotted key, which also closes it
}

func newTable(pos Position) *Table {
	return &Table{Pos: pos, vals: map[string]*Value{}}
}

// Len is how many keys the table has.
//
// The methods that read a table work on a nil one, because a table missing
// from a document is a table with nothing in it, and a caller reading optional
// configuration should not have to check twice.
func (t *Table) Len() int {
	if t == nil {
		return 0
	}
	return len(t.keys)
}

// Keys are the table's keys in the order they were written.
func (t *Table) Keys() []string {
	if t == nil {
		return nil
	}
	return slices.Clone(t.keys)
}

// Get returns the value for a key, or nil when there is not one.
func (t *Table) Get(key string) *Value {
	if t == nil {
		return nil
	}
	return t.vals[key]
}

// Lookup follows a path of keys down through nested tables, and returns nil if
// any of them is missing or is not a table.
//
//	if v := doc.Lookup("db", "dsn"); v != nil {
//		...
//	}
func (t *Table) Lookup(path ...string) *Value {
	cur := t
	for i, key := range path {
		v := cur.Get(key)
		if v == nil {
			return nil
		}
		if i == len(path)-1 {
			return v
		}
		if v.Kind != KindTable {
			return nil
		}
		cur = v.Table
	}
	return nil
}

// All iterates the table's keys and values in the order they were written.
func (t *Table) All() iter.Seq2[string, *Value] {
	return func(yield func(string, *Value) bool) {
		if t == nil {
			return
		}
		for _, k := range t.keys {
			if !yield(k, t.vals[k]) {
				return
			}
		}
	}
}

func (t *Table) set(key string, v *Value) {
	if _, ok := t.vals[key]; !ok {
		t.keys = append(t.keys, key)
	}
	t.vals[key] = v
}

// An Error is something wrong with a document, at the place it went wrong.
type Error struct {
	Pos Position
	Msg string
}

func (e *Error) Error() string { return e.Pos.String() + ": " + e.Msg }

// quoteKey renders a key the way it would be written, for error messages
// about keys that are not bare words.
func quoteKey(key string) string {
	if isBareKey(key) {
		return key
	}
	return strconv.Quote(key)
}

func isBareKey(key string) bool {
	if key == "" {
		return false
	}
	for i := range len(key) {
		if !isBareKeyByte(key[i]) {
			return false
		}
	}
	return true
}

func isBareKeyByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-'
}
