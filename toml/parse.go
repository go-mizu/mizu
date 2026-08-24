package toml

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Parse reads a TOML document. The name is what error messages call it, and is
// usually the path the data was read from.
func Parse(name string, data []byte) (*Table, error) {
	p := &parser{name: name, src: data, line: 1}
	if err := p.parse(); err != nil {
		return nil, err
	}
	return p.root, nil
}

// ParseFile reads a file and parses it.
func ParseFile(path string) (*Table, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(path, data)
}

type parser struct {
	name string
	src  []byte
	off  int // where we are in src
	line int // the line off is on, counting from one
	bol  int // where that line starts, for working out the column

	root *Table
	cur  *Table // where a bare key goes, which the last header set
}

func (p *parser) parse() error {
	if err := p.checkUTF8(); err != nil {
		return err
	}
	p.src, _ = bytes.CutPrefix(p.src, []byte{0xEF, 0xBB, 0xBF}) // a byte order mark

	p.root = newTable(Position{File: p.name, Line: 1, Col: 1})
	p.root.explicit = true
	p.cur = p.root

	for {
		p.skipSpace()
		if p.done() {
			return nil
		}
		switch c := p.peek(); {
		case c == '\n' || c == '\r':
			if _, err := p.newline(); err != nil {
				return err
			}
			continue
		case c == '#':
			if err := p.comment(); err != nil {
				return err
			}
		case c == '[':
			if err := p.header(); err != nil {
				return err
			}
		default:
			if err := p.keyval(p.cur); err != nil {
				return err
			}
		}

		// Whatever it was, the line has to end after it.
		p.skipSpace()
		if p.peek() == '#' {
			if err := p.comment(); err != nil {
				return err
			}
		}
		ok, err := p.newline()
		if err != nil {
			return err
		}
		if !ok {
			return p.errHere("want the end of the line, got %s", p.describe())
		}
	}
}

// checkUTF8 reports where the file stops being UTF-8, if it does. TOML says a
// document is UTF-8, and checking once here means nothing below has to.
func (p *parser) checkUTF8() error {
	if utf8.Valid(p.src) {
		return nil
	}
	line, col := 1, 1
	for i := 0; i < len(p.src); {
		r, size := utf8.DecodeRune(p.src[i:])
		if r == utf8.RuneError && size <= 1 {
			return &Error{Pos: Position{File: p.name, Line: line, Col: col}, Msg: "this file is not valid UTF-8"}
		}
		if r == '\n' {
			line, col = line+1, 1
		} else {
			col += size
		}
		i += size
	}
	return nil
}

// Reading.

func (p *parser) done() bool { return p.off >= len(p.src) }

func (p *parser) peek() byte { return p.at(0) }

func (p *parser) at(i int) byte {
	if p.off+i >= len(p.src) {
		return 0
	}
	return p.src[p.off+i]
}

func (p *parser) advance() byte {
	c := p.src[p.off]
	p.off++
	if c == '\n' {
		p.line++
		p.bol = p.off
	}
	return c
}

func (p *parser) hasPrefix(s string) bool {
	return p.off+len(s) <= len(p.src) && string(p.src[p.off:p.off+len(s)]) == s
}

func (p *parser) pos() Position {
	return Position{File: p.name, Line: p.line, Col: p.off - p.bol + 1}
}

// describe names what is under the cursor, for an error message.
func (p *parser) describe() string {
	if p.done() {
		return "the end of the file"
	}
	r, _ := utf8.DecodeRune(p.src[p.off:])
	switch {
	case r == '\n' || r == '\r':
		return "the end of the line"
	case unicode.IsPrint(r):
		return strconv.QuoteRune(r)
	}
	return fmt.Sprintf("U+%04X", r)
}

func (p *parser) errf(pos Position, format string, a ...any) error {
	return &Error{Pos: pos, Msg: fmt.Sprintf(format, a...)}
}

func (p *parser) errHere(format string, a ...any) error {
	return p.errf(p.pos(), format, a...)
}

func (p *parser) skipSpace() {
	for !p.done() && (p.peek() == ' ' || p.peek() == '\t') {
		p.advance()
	}
}

// newline consumes one line ending and reports whether it found one. The end
// of the file counts, since it ends the last line.
func (p *parser) newline() (bool, error) {
	switch {
	case p.done():
		return true, nil
	case p.peek() == '\n':
		p.advance()
		return true, nil
	case p.peek() == '\r':
		if p.at(1) != '\n' {
			return false, p.errHere("a carriage return has to be followed by a newline")
		}
		p.advance()
		p.advance()
		return true, nil
	}
	return false, nil
}

func (p *parser) comment() error {
	p.advance() // #
	for !p.done() {
		c := p.peek()
		if c == '\n' || c == '\r' {
			return nil
		}
		if isControl(c) {
			return p.errHere("a comment cannot contain %s", p.describe())
		}
		p.advance()
	}
	return nil
}

// skipBlank skips whitespace, newlines and comments, which is what is allowed
// between the elements of an array.
func (p *parser) skipBlank() error {
	for {
		p.skipSpace()
		switch {
		case p.peek() == '#':
			if err := p.comment(); err != nil {
				return err
			}
		case p.peek() == '\n' || p.peek() == '\r':
			if _, err := p.newline(); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

// Keys.

// key parses a key, which may be dotted, and returns its parts.
func (p *parser) key() ([]string, Position, error) {
	p.skipSpace()
	pos := p.pos()
	var parts []string
	for {
		part, err := p.keyPart()
		if err != nil {
			return nil, pos, err
		}
		parts = append(parts, part)

		p.skipSpace()
		if p.peek() != '.' {
			return parts, pos, nil
		}
		p.advance()
		p.skipSpace()
	}
}

func (p *parser) keyPart() (string, error) {
	switch c := p.peek(); {
	case c == '"' || c == '\'':
		if p.hasPrefix(`"""`) || p.hasPrefix(`'''`) {
			return "", p.errHere("a key cannot be a multi-line string")
		}
		return p.stringValue()
	case isBareKeyByte(c):
		start := p.off
		for !p.done() && isBareKeyByte(p.peek()) {
			p.advance()
		}
		return string(p.src[start:p.off]), nil
	}
	return "", p.errHere("want a key, got %s", p.describe())
}

func dottedName(parts []string) string {
	quoted := make([]string, len(parts))
	for i, part := range parts {
		quoted[i] = quoteKey(part)
	}
	return strings.Join(quoted, ".")
}

// keyval parses one key and value into a table.
func (p *parser) keyval(t *Table) error {
	parts, pos, err := p.key()
	if err != nil {
		return err
	}
	p.skipSpace()
	if p.peek() != '=' {
		return p.errHere("want = after the key %s, got %s", dottedName(parts), p.describe())
	}
	p.advance()
	p.skipSpace()

	v, err := p.value()
	if err != nil {
		return err
	}

	target, err := p.descend(t, parts, pos)
	if err != nil {
		return err
	}
	last := parts[len(parts)-1]
	if old := target.Get(last); old != nil {
		return p.errf(pos, "%s is defined twice, and was already set at %s", dottedName(parts), old.Pos)
	}
	target.set(last, v)
	return nil
}

// descend walks the tables a dotted key names, making the ones that are not
// there yet. A table made this way is closed to headers afterwards, which is
// what dotted says.
func (p *parser) descend(t *Table, parts []string, pos Position) (*Table, error) {
	for i, k := range parts[:len(parts)-1] {
		v := t.Get(k)
		if v == nil {
			sub := newTable(pos)
			sub.dotted = true
			t.set(k, &Value{Kind: KindTable, Pos: pos, Table: sub})
			t = sub
			continue
		}
		if v.Kind != KindTable {
			return nil, p.errf(pos, "%s is a %s, so %s cannot go inside it", dottedName(parts[:i+1]), v.Kind, dottedName(parts))
		}
		if v.Table.inline {
			return nil, p.errf(pos, "%s was written as an inline table, which is closed once it ends", dottedName(parts[:i+1]))
		}
		t = v.Table
	}
	return t, nil
}

// Table headers.

func (p *parser) header() error {
	pos := p.pos()
	p.advance() // [
	array := p.peek() == '['
	if array {
		p.advance()
	}

	parts, _, err := p.key()
	if err != nil {
		return err
	}
	p.skipSpace()
	if p.peek() != ']' {
		return p.errHere("want ] to close the header for %s, got %s", dottedName(parts), p.describe())
	}
	p.advance()
	if array {
		if p.peek() != ']' {
			return p.errHere("want ]] to close the header for %s, got %s", dottedName(parts), p.describe())
		}
		p.advance()
	}

	t, err := p.open(parts, pos, array)
	if err != nil {
		return err
	}
	p.cur = t
	return nil
}

// open finds or makes the table a header names, and reports the ways that can
// go wrong. Most of the rules about what a document may not say twice live
// here.
func (p *parser) open(parts []string, pos Position, array bool) (*Table, error) {
	t := p.root
	for i, k := range parts[:len(parts)-1] {
		v := t.Get(k)
		switch {
		case v == nil:
			sub := newTable(pos)
			t.set(k, &Value{Kind: KindTable, Pos: pos, Table: sub})
			t = sub
		case v.Kind == KindTable:
			switch {
			case v.Table.inline:
				return nil, p.errf(pos, "%s was written as an inline table, which is closed once it ends", dottedName(parts[:i+1]))
			case v.Table.dotted:
				return nil, p.errf(pos, "%s was made by a dotted key, which cannot be reopened as a table", dottedName(parts[:i+1]))
			}
			t = v.Table
		case v.Kind == KindArray && v.arrayOfTables:
			t = v.Array[len(v.Array)-1].Table
		default:
			return nil, p.errf(pos, "%s is a %s, so it cannot hold a table", dottedName(parts[:i+1]), v.Kind)
		}
	}

	last := parts[len(parts)-1]
	v := t.Get(last)
	name := dottedName(parts)

	if array {
		switch {
		case v == nil:
			sub := newTable(pos)
			sub.explicit = true
			t.set(last, &Value{
				Kind:          KindArray,
				Pos:           pos,
				Array:         []*Value{{Kind: KindTable, Pos: pos, Table: sub}},
				arrayOfTables: true,
			})
			return sub, nil
		case v.Kind == KindArray && v.arrayOfTables:
			sub := newTable(pos)
			sub.explicit = true
			v.Array = append(v.Array, &Value{Kind: KindTable, Pos: pos, Table: sub})
			return sub, nil
		case v.Kind == KindArray:
			return nil, p.errf(pos, "%s is an array of values, written at %s, so [[%s]] cannot add to it", name, v.Pos, name)
		}
		return nil, p.errf(pos, "%s is a %s, written at %s, so [[%s]] cannot add to it", name, v.Kind, v.Pos, name)
	}

	switch {
	case v == nil:
		sub := newTable(pos)
		sub.explicit = true
		t.set(last, &Value{Kind: KindTable, Pos: pos, Table: sub})
		return sub, nil
	case v.Kind == KindTable:
		switch tab := v.Table; {
		case tab.explicit:
			return nil, p.errf(pos, "table %s is defined twice, and was already defined at %s", name, tab.Pos)
		case tab.inline:
			return nil, p.errf(pos, "%s was written as an inline table at %s, which is closed once it ends", name, tab.Pos)
		case tab.dotted:
			return nil, p.errf(pos, "%s was made by a dotted key at %s, which cannot be reopened as a table", name, tab.Pos)
		default:
			tab.explicit = true
			tab.Pos = pos
			return tab, nil
		}
	}
	return nil, p.errf(pos, "%s is a %s, written at %s, so it cannot be a table", name, v.Kind, v.Pos)
}

// Values.

func (p *parser) value() (*Value, error) {
	pos := p.pos()
	switch c := p.peek(); {
	case c == '"' || c == '\'':
		s, err := p.stringValue()
		if err != nil {
			return nil, err
		}
		return &Value{Kind: KindString, Pos: pos, Str: s}, nil
	case c == '[':
		return p.array()
	case c == '{':
		return p.inlineTable()
	case c == 't' || c == 'f':
		return p.boolean()
	}
	return p.numberOrTime()
}

func (p *parser) boolean() (*Value, error) {
	pos := p.pos()
	for _, word := range [...]string{"true", "false"} {
		if p.hasPrefix(word) && !isValueByte(p.at(len(word))) {
			for range len(word) {
				p.advance()
			}
			return &Value{Kind: KindBool, Pos: pos, Bool: word == "true"}, nil
		}
	}
	return nil, p.errHere("want a value, got %s", p.describe())
}

func (p *parser) array() (*Value, error) {
	pos := p.pos()
	p.advance() // [
	v := &Value{Kind: KindArray, Pos: pos}
	for {
		if err := p.skipBlank(); err != nil {
			return nil, err
		}
		if p.done() {
			return nil, p.errf(pos, "this array is missing its closing ]")
		}
		if p.peek() == ']' {
			p.advance()
			return v, nil
		}

		el, err := p.value()
		if err != nil {
			return nil, err
		}
		v.Array = append(v.Array, el)

		if err := p.skipBlank(); err != nil {
			return nil, err
		}
		switch p.peek() {
		case ',':
			p.advance()
		case ']':
			p.advance()
			return v, nil
		default:
			if p.done() {
				return nil, p.errf(pos, "this array is missing its closing ]")
			}
			return nil, p.errHere("want a comma or ] in this array, got %s", p.describe())
		}
	}
}

func (p *parser) inlineTable() (*Value, error) {
	pos := p.pos()
	p.advance() // {
	t := newTable(pos)
	t.inline = true
	v := &Value{Kind: KindTable, Pos: pos, Table: t}

	p.skipSpace()
	if p.peek() == '}' {
		p.advance()
		return v, nil
	}
	for {
		p.skipSpace()
		if err := p.keyval(t); err != nil {
			return nil, err
		}
		p.skipSpace()
		switch p.peek() {
		case ',':
			p.advance()
			p.skipSpace()
			switch p.peek() {
			case '}':
				return nil, p.errHere("an inline table cannot end with a comma")
			case '\n', '\r':
				return nil, p.errHere("an inline table has to be written on one line")
			}
		case '}':
			p.advance()
			return v, nil
		case '\n', '\r':
			return nil, p.errHere("an inline table has to be written on one line")
		default:
			if p.done() {
				return nil, p.errf(pos, "this inline table is missing its closing }")
			}
			return nil, p.errHere("want a comma or } in this inline table, got %s", p.describe())
		}
	}
}
