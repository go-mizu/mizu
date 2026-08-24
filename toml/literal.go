package toml

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// The literals: strings, numbers, dates and times. Every function here reads
// one value out of the source and leaves the cursor just after it.

// stringValue reads any of the four string forms. Which one it is comes down
// to the first three bytes.
func (p *parser) stringValue() (string, error) {
	switch {
	case p.hasPrefix(`"""`):
		return p.multiline('"')
	case p.hasPrefix(`'''`):
		return p.multiline('\'')
	case p.peek() == '"':
		return p.basicString()
	}
	return p.literalString()
}

func (p *parser) basicString() (string, error) {
	open := p.pos()
	p.advance() // "

	// Most strings have no escapes in them, and one that does not is already
	// sitting in the source, so it is copied out in one piece. b stays nil
	// until the first escape, and start is where the run of plain bytes began.
	var b *strings.Builder
	start := p.off

	for {
		switch c := p.peek(); {
		case p.done(), c == '\n', c == '\r':
			return "", p.errf(open, "this string is missing its closing quote")

		case c == '"':
			plain := p.src[start:p.off]
			p.advance()
			if b == nil {
				return string(plain), nil
			}
			b.Write(plain)
			return b.String(), nil

		case c == '\\':
			if b == nil {
				b = new(strings.Builder)
			}
			b.Write(p.src[start:p.off])
			s, err := p.escape()
			if err != nil {
				return "", err
			}
			b.WriteString(s)
			start = p.off

		case isControl(c):
			return "", p.errHere("a string cannot contain %s; write it as an escape", p.describe())

		default:
			p.advance()
		}
	}
}

func (p *parser) literalString() (string, error) {
	open := p.pos()
	p.advance() // '
	start := p.off
	for {
		switch c := p.peek(); {
		case p.done(), c == '\n', c == '\r':
			return "", p.errf(open, "this string is missing its closing quote")
		case c == '\'':
			s := string(p.src[start:p.off])
			p.advance()
			return s, nil
		case isControl(c):
			return "", p.errHere("a string cannot contain %s, and a literal string has no escapes", p.describe())
		default:
			p.advance()
		}
	}
}

// multiline reads a string in triple quotes. quote is '"' for the kind with
// escapes in it and '\” for the kind without.
func (p *parser) multiline(quote byte) (string, error) {
	open := p.pos()
	for range 3 {
		p.advance()
	}
	// A newline straight after the opening quotes is not part of the string.
	if p.peek() == '\r' && p.at(1) == '\n' {
		p.advance()
	}
	if p.peek() == '\n' {
		p.advance()
	}

	var b strings.Builder
	for {
		c := p.peek()
		switch {
		case p.done():
			return "", p.errf(open, "this string is missing its closing %s", strings.Repeat(string(quote), 3))

		case c == quote:
			// The closing quotes are the last three of the run, so a run of
			// four ends a string that finishes with a quote.
			n := 0
			for p.at(n) == quote {
				n++
			}
			if n < 3 {
				for range n {
					b.WriteByte(p.advance())
				}
				continue
			}
			if n > 5 {
				return "", p.errHere("a multi-line string cannot contain %s in a row", strings.Repeat(string(quote), 3))
			}
			for range n - 3 {
				b.WriteByte(p.advance())
			}
			for range 3 {
				p.advance()
			}
			return b.String(), nil

		case quote == '"' && c == '\\':
			// A backslash at the end of a line eats the newline and the
			// whitespace after it, which is how a long line gets folded.
			if p.foldsLine() {
				p.advance()
				for !p.done() && (p.peek() == ' ' || p.peek() == '\t' || p.peek() == '\n' || p.peek() == '\r') {
					p.advance()
				}
				continue
			}
			s, err := p.escape()
			if err != nil {
				return "", err
			}
			b.WriteString(s)

		case c == '\n' || c == '\t':
			b.WriteByte(p.advance())

		case c == '\r':
			if p.at(1) != '\n' {
				return "", p.errHere("a carriage return has to be followed by a newline")
			}
			b.WriteByte(p.advance())
			b.WriteByte(p.advance())

		case isControl(c):
			return "", p.errHere("a string cannot contain %s", p.describe())

		default:
			b.WriteByte(p.advance())
		}
	}
}

// foldsLine reports whether the backslash under the cursor is the last thing
// on its line.
func (p *parser) foldsLine() bool {
	for i := 1; ; i++ {
		switch p.at(i) {
		case ' ', '\t':
		case '\n':
			return true
		case '\r':
			return p.at(i+1) == '\n'
		default:
			return false
		}
	}
}

func (p *parser) escape() (string, error) {
	pos := p.pos()
	p.advance() // backslash
	if p.done() {
		return "", p.errf(pos, "the file ends in the middle of an escape")
	}
	var s string
	switch c := p.peek(); c {
	case 'b':
		s = "\b"
	case 't':
		s = "\t"
	case 'n':
		s = "\n"
	case 'f':
		s = "\f"
	case 'r':
		s = "\r"
	case '"':
		s = `"`
	case '\\':
		s = `\`
	case 'u', 'U':
		p.advance()
		if c == 'u' {
			return p.hexEscape(pos, 4)
		}
		return p.hexEscape(pos, 8)
	default:
		return "", p.errf(pos, `a backslash followed by %s is not an escape; they are \b \t \n \f \r \" \\ \uXXXX and \UXXXXXXXX`, p.describe())
	}
	p.advance()
	return s, nil
}

func (p *parser) hexEscape(pos Position, n int) (string, error) {
	if p.off+n > len(p.src) {
		return "", p.errf(pos, "the file ends in the middle of an escape")
	}
	digits := string(p.src[p.off : p.off+n])
	code, err := strconv.ParseUint(digits, 16, 32)
	if err != nil {
		return "", p.errf(pos, "%q is not %d hexadecimal digits", digits, n)
	}
	for range n {
		p.advance()
	}
	if r := rune(code); utf8.ValidRune(r) {
		return string(r), nil
	}
	return "", p.errf(pos, "U+%04X is not a character", code)
}

// Numbers, dates and times.

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isValueByte(c byte) bool {
	return isDigit(c) || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' ||
		c == '_' || c == '+' || c == '-' || c == '.' || c == ':'
}

func isControl(c byte) bool { return c < 0x20 && c != '\t' || c == 0x7f }

func (p *parser) numberOrTime() (*Value, error) {
	pos := p.pos()
	start := p.off
	for !p.done() && isValueByte(p.peek()) {
		p.advance()
	}
	// A date and a time can be separated by a space as well as by a T, and the
	// space would otherwise end the value.
	if looksLikeDate(p.src[start:p.off]) && p.peek() == ' ' && isDigit(p.at(1)) && isDigit(p.at(2)) && p.at(3) == ':' {
		p.advance()
		for !p.done() && isValueByte(p.peek()) {
			p.advance()
		}
	}

	tok := string(p.src[start:p.off])
	if tok == "" {
		return nil, p.errHere("want a value, got %s", p.describe())
	}
	if looksLikeDate(p.src[start:p.off]) || looksLikeTime(p.src[start:p.off]) {
		return p.datetime(tok, pos)
	}
	return p.number(tok, pos)
}

func looksLikeDate(b []byte) bool {
	return len(b) >= 10 && isDigit(b[0]) && isDigit(b[1]) && isDigit(b[2]) && isDigit(b[3]) &&
		b[4] == '-' && isDigit(b[5]) && isDigit(b[6]) && b[7] == '-' && isDigit(b[8]) && isDigit(b[9])
}

func looksLikeTime(b []byte) bool {
	return len(b) >= 3 && isDigit(b[0]) && isDigit(b[1]) && b[2] == ':'
}

func (p *parser) datetime(tok string, pos Position) (*Value, error) {
	v := &Value{Pos: pos}
	text, layout := tok, ""

	switch {
	case looksLikeTime([]byte(tok)):
		v.Kind, layout = KindLocalTime, "15:04:05"
	case len(tok) == 10:
		v.Kind, layout = KindLocalDate, "2006-01-02"
	default:
		switch tok[10] {
		case 'T', 't', ' ':
			text = tok[:10] + "T" + tok[11:]
		default:
			return nil, p.errf(pos, "%s needs a T or a space between the date and the time", tok)
		}
		rest := text[11:]
		switch {
		case strings.HasSuffix(rest, "Z"), strings.HasSuffix(rest, "z"):
			text = text[:len(text)-1] + "Z" // time.Parse wants the capital
			v.Kind, layout = KindOffsetDateTime, time.RFC3339
		case strings.ContainsAny(rest, "+-"):
			v.Kind, layout = KindOffsetDateTime, time.RFC3339
		default:
			v.Kind, layout = KindLocalDateTime, "2006-01-02T15:04:05"
		}
	}

	t, err := time.Parse(layout, text)
	if err != nil {
		var perr *time.ParseError
		if errors.As(err, &perr) && perr.Message != "" {
			return nil, p.errf(pos, "%s is not a valid %s%s", tok, v.Kind, perr.Message)
		}
		return nil, p.errf(pos, "%s is not a valid %s", tok, v.Kind)
	}
	v.Time = t
	return v, nil
}

func (p *parser) number(tok string, pos Position) (*Value, error) {
	switch tok {
	case "inf", "+inf":
		return &Value{Kind: KindFloat, Pos: pos, Float: math.Inf(1)}, nil
	case "-inf":
		return &Value{Kind: KindFloat, Pos: pos, Float: math.Inf(-1)}, nil
	case "nan", "+nan", "-nan":
		return &Value{Kind: KindFloat, Pos: pos, Float: math.NaN()}, nil
	}

	body := tok
	signed := false
	if body[0] == '+' || body[0] == '-' {
		signed, body = true, body[1:]
	}
	if body == "" {
		return nil, p.errf(pos, "%s is not a number", tok)
	}

	// Hexadecimal, octal and binary integers, which have no sign and no
	// fractional part.
	if len(body) > 1 && body[0] == '0' {
		base, digits := 0, ""
		switch body[1] {
		case 'x':
			base, digits = 16, body[2:]
		case 'o':
			base, digits = 8, body[2:]
		case 'b':
			base, digits = 2, body[2:]
		}
		if base != 0 {
			if signed {
				return nil, p.errf(pos, "%s cannot have a sign, because it is written in base %d", tok, base)
			}
			clean, err := digitsOf(digits, base)
			if err != nil {
				return nil, p.errf(pos, "%s is not a number: %s", tok, err)
			}
			n, err := strconv.ParseInt(clean, base, 64)
			if err != nil {
				return nil, p.errf(pos, "%s does not fit in a 64 bit integer", tok)
			}
			return &Value{Kind: KindInt, Pos: pos, Int: n}, nil
		}
	}

	if strings.ContainsAny(body, ".eE") {
		clean, err := checkFloat(body)
		if err != nil {
			return nil, p.errf(pos, "%s is not a number: %s", tok, err)
		}
		f, err := strconv.ParseFloat(tok[:len(tok)-len(body)]+clean, 64)
		if err != nil {
			return nil, p.errf(pos, "%s is not a number", tok)
		}
		return &Value{Kind: KindFloat, Pos: pos, Float: f}, nil
	}

	clean, err := checkInteger(body)
	if err != nil {
		return nil, p.errf(pos, "%s is not a number: %s", tok, err)
	}
	n, err := strconv.ParseInt(tok[:len(tok)-len(body)]+clean, 10, 64)
	if err != nil {
		return nil, p.errf(pos, "%s does not fit in a 64 bit integer", tok)
	}
	return &Value{Kind: KindInt, Pos: pos, Int: n}, nil
}

// digitsOf checks a run of digits in the given base and returns it without the
// underscores. TOML allows an underscore between two digits and nowhere else,
// which is stricter than Go and than strconv.
func digitsOf(s string, base int) (string, error) {
	if s == "" {
		return "", errors.New("it has no digits")
	}
	var b strings.Builder
	for i := range len(s) {
		c := s[i]
		if c == '_' {
			if i == 0 || i == len(s)-1 || s[i-1] == '_' {
				return "", errors.New("an underscore has to sit between two digits")
			}
			continue
		}
		if !isDigitIn(c, base) {
			return "", fmt.Errorf("%q is not a digit in base %d", string(c), base)
		}
		b.WriteByte(c)
	}
	return b.String(), nil
}

func isDigitIn(c byte, base int) bool {
	switch {
	case base == 16:
		return isDigit(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
	case base == 8:
		return c >= '0' && c <= '7'
	case base == 2:
		return c == '0' || c == '1'
	}
	return isDigit(c)
}

func checkInteger(s string) (string, error) {
	clean, err := digitsOf(s, 10)
	if err != nil {
		return "", err
	}
	if len(clean) > 1 && clean[0] == '0' {
		return "", errors.New("a number cannot start with a zero")
	}
	return clean, nil
}

func checkFloat(s string) (string, error) {
	mantissa, exponent, hasExponent := cutAny(s, "eE")

	whole, fraction, hasFraction := strings.Cut(mantissa, ".")
	clean, err := checkInteger(whole)
	if err != nil {
		return "", err
	}
	if hasFraction {
		digits, err := digitsOf(fraction, 10)
		if err != nil {
			return "", err
		}
		clean += "." + digits
	}
	if hasExponent {
		sign := ""
		if exponent != "" && (exponent[0] == '+' || exponent[0] == '-') {
			sign, exponent = exponent[:1], exponent[1:]
		}
		digits, err := digitsOf(exponent, 10)
		if err != nil {
			return "", err
		}
		clean += "e" + sign + digits
	}
	return clean, nil
}

func cutAny(s, chars string) (before, after string, found bool) {
	if i := strings.IndexAny(s, chars); i >= 0 {
		return s[:i], s[i+1:], true
	}
	return s, "", false
}
