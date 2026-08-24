package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// A variable is one setting out of a .env file.
type variable struct {
	name  string
	value string
	line  int
}

// parseDotEnv reads a .env file.
//
// A line is NAME=value, with an optional export in front of it. Blank lines
// and lines starting with # are skipped, and so is a # comment after an
// unquoted value. A value in single quotes is taken as written, a value in
// double quotes has the usual backslash escapes, and either may run over more
// than one line, which is how a private key gets into a .env file.
//
// ${NAME} in an unquoted or double quoted value is replaced by the value of
// NAME, and ${NAME:-text} uses text when there is no such value. A name set
// earlier in the same file wins, and resolve answers for everything else. A
// lone $ is left alone, because a $ in the middle of a password is more common
// than a variable is.
func parseDotEnv(file string, data []byte, resolve func(string) (string, bool)) ([]variable, error) {
	p := &dotenv{file: file, src: data, line: 1}
	seen := map[string]string{}
	here := func(name string) (string, bool) {
		if v, ok := seen[name]; ok {
			return v, true
		}
		return resolve(name)
	}

	var out []variable
	for {
		p.skipBlank()
		if p.done() {
			return out, nil
		}
		v, err := p.entry(here)
		if err != nil {
			return nil, err
		}
		seen[v.name] = v.value
		out = append(out, v)
	}
}

type dotenv struct {
	file string
	src  []byte
	off  int
	line int
}

func (p *dotenv) done() bool { return p.off >= len(p.src) }

func (p *dotenv) peek() byte {
	if p.done() {
		return 0
	}
	return p.src[p.off]
}

// advance moves past one byte and keeps the line count right, so everything
// that consumes input has to go through it.
func (p *dotenv) advance() byte {
	c := p.src[p.off]
	p.off++
	if c == '\n' {
		p.line++
	}
	return c
}

func (p *dotenv) errf(line int, format string, a ...any) error {
	return &Error{File: p.file, Line: line, Msg: fmt.Sprintf(format, a...)}
}

// skipBlank moves to the start of the next line that has something on it.
func (p *dotenv) skipBlank() {
	for !p.done() {
		switch c := p.peek(); {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			p.advance()
		case c == '#':
			p.skipLine()
		default:
			return
		}
	}
}

func (p *dotenv) skipLine() {
	for !p.done() && p.peek() != '\n' {
		p.advance()
	}
}

func (p *dotenv) skipSpace() {
	for !p.done() && (p.peek() == ' ' || p.peek() == '\t') {
		p.advance()
	}
}

// entry reads one NAME=value.
func (p *dotenv) entry(resolve func(string) (string, bool)) (variable, error) {
	line := p.line
	if p.hasWord("export") {
		p.skipSpace()
	}

	start := p.off
	for !p.done() && p.peek() != '=' && p.peek() != '\n' {
		p.advance()
	}
	name := strings.TrimRight(string(p.src[start:p.off]), " \t\r")
	if p.peek() != '=' {
		// Nothing reaches here with an empty name, because a line with only
		// spaces on it was skipped before this ever started.
		return variable{}, p.errf(line, "%s has no value; a line is NAME=value", strconv.Quote(name))
	}
	p.advance() // the =
	if err := checkName(name); err != nil {
		return variable{}, p.errf(line, "%s", err)
	}

	value, err := p.value(name, resolve)
	if err != nil {
		return variable{}, err
	}
	return variable{name: name, value: value, line: line}, nil
}

// hasWord consumes a word at the start of a line, such as export, when it is
// there and is followed by a space.
func (p *dotenv) hasWord(word string) bool {
	rest := p.src[p.off:]
	if len(rest) <= len(word) || string(rest[:len(word)]) != word {
		return false
	}
	if c := rest[len(word)]; c != ' ' && c != '\t' {
		return false
	}
	for range len(word) {
		p.advance()
	}
	return true
}

func (p *dotenv) value(name string, resolve func(string) (string, bool)) (string, error) {
	p.skipSpace()
	switch p.peek() {
	case '\'':
		return p.quoted(name, '\'')
	case '"':
		s, err := p.quoted(name, '"')
		if err != nil {
			return "", err
		}
		return p.expand(name, unescape(s), resolve)
	}

	start := p.off
	p.skipLine()
	raw := string(p.src[start:p.off])
	if i := commentAt(raw); i >= 0 {
		raw = raw[:i]
	}
	return p.expand(name, strings.TrimRight(raw, " \t\r"), resolve)
}

// quoted reads a value between quotes, which may run over more than one line.
func (p *dotenv) quoted(name string, quote byte) (string, error) {
	line := p.line
	p.advance() // the opening quote
	start := p.off
	for {
		if p.done() {
			return "", p.errf(line, "the value of %s has no closing %c", name, quote)
		}
		if p.peek() == quote {
			s := string(p.src[start:p.off])
			p.advance()
			p.skipLine() // anything after the closing quote is a comment
			return s, nil
		}
		if quote == '"' && p.peek() == '\\' && p.off+1 < len(p.src) {
			p.advance()
		}
		p.advance()
	}
}

// commentAt finds where a # comment starts in an unquoted value, which is at a
// # that has a space in front of it. A # with no space is part of the value,
// because a URL fragment and a colour both look like that.
func commentAt(s string) int {
	for i := range len(s) {
		if s[i] == '#' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
			return i
		}
	}
	return -1
}

// unescape applies the backslash escapes inside a double quoted value. An
// escape that means nothing keeps its backslash, so a Windows path written
// without doubling up survives.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i == len(s)-1 {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case '\\', '"', '\'', '$':
			b.WriteByte(s[i])
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// expand replaces ${NAME} and ${NAME:-text} in a value.
func (p *dotenv) expand(name, s string, resolve func(string) (string, bool)) (string, error) {
	if !strings.Contains(s, "${") {
		return s, nil
	}
	var b strings.Builder
	b.Grow(len(s))
	for {
		i := strings.Index(s, "${")
		if i < 0 {
			b.WriteString(s)
			return b.String(), nil
		}
		b.WriteString(s[:i])
		ref, rest, ok := strings.Cut(s[i+2:], "}")
		if !ok {
			return "", p.errf(p.line, "the value of %s has a ${ with no closing }", name)
		}
		key, fallback, hasFallback := strings.Cut(ref, ":-")
		if v, ok := resolve(key); ok && v != "" {
			b.WriteString(v)
		} else if hasFallback {
			b.WriteString(fallback)
		}
		s = rest
	}
}

// checkName rejects what would not survive being put in the environment.
func checkName(name string) error {
	if name == "" {
		return errors.New("a line starts with = and so names nothing")
	}
	for i := range len(name) {
		c := name[i]
		ok := c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' ||
			(i > 0 && c >= '0' && c <= '9')
		if !ok {
			return errors.New(strconv.Quote(name) + " is not a variable name; they are letters, digits and underscores, and do not start with a digit")
		}
	}
	return nil
}
