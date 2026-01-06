// Package formula provides formula parsing and evaluation.
package formula

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType represents a token type.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenNumber
	TokenString
	TokenBool
	TokenError
	TokenReference  // A1, $A$1
	TokenRange      // A1:B10
	TokenName       // Named range
	TokenFunction   // SUM, IF, etc.
	TokenOperator   // +, -, *, /, ^, %, &, =, <>, <, >, <=, >=
	TokenLParen     // (
	TokenRParen     // )
	TokenComma      // ,
	TokenColon      // :
	TokenSemicolon  // ;
	TokenSheet      // Sheet1!
	TokenLBrace     // {
	TokenRBrace     // }
)

// Token represents a lexical token.
type Token struct {
	Type  TokenType
	Value string
	Pos   int
}

// Lexer tokenizes a formula string.
type Lexer struct {
	input  string
	pos    int
	tokens []Token
}

// NewLexer creates a new lexer.
func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

// Tokenize tokenizes the input and returns tokens.
func (l *Lexer) Tokenize() ([]Token, error) {
	l.tokens = []Token{}
	l.pos = 0

	// Skip leading '=' if present
	if len(l.input) > 0 && l.input[0] == '=' {
		l.pos = 1
	}

	for l.pos < len(l.input) {
		if err := l.scanToken(); err != nil {
			return nil, err
		}
	}

	l.tokens = append(l.tokens, Token{Type: TokenEOF, Pos: l.pos})
	return l.tokens, nil
}

func (l *Lexer) scanToken() error {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return nil
	}

	startPos := l.pos
	ch := l.input[l.pos]

	// String literal
	if ch == '"' {
		return l.scanString()
	}

	// Number
	if unicode.IsDigit(rune(ch)) || (ch == '.' && l.pos+1 < len(l.input) && unicode.IsDigit(rune(l.input[l.pos+1]))) {
		return l.scanNumber()
	}

	// Operators and punctuation
	switch ch {
	case '+', '-', '*', '/', '^', '%', '&':
		l.tokens = append(l.tokens, Token{Type: TokenOperator, Value: string(ch), Pos: startPos})
		l.pos++
		return nil
	case '(':
		l.tokens = append(l.tokens, Token{Type: TokenLParen, Value: "(", Pos: startPos})
		l.pos++
		return nil
	case ')':
		l.tokens = append(l.tokens, Token{Type: TokenRParen, Value: ")", Pos: startPos})
		l.pos++
		return nil
	case ',':
		l.tokens = append(l.tokens, Token{Type: TokenComma, Value: ",", Pos: startPos})
		l.pos++
		return nil
	case ':':
		l.tokens = append(l.tokens, Token{Type: TokenColon, Value: ":", Pos: startPos})
		l.pos++
		return nil
	case ';':
		l.tokens = append(l.tokens, Token{Type: TokenSemicolon, Value: ";", Pos: startPos})
		l.pos++
		return nil
	case '{':
		l.tokens = append(l.tokens, Token{Type: TokenLBrace, Value: "{", Pos: startPos})
		l.pos++
		return nil
	case '}':
		l.tokens = append(l.tokens, Token{Type: TokenRBrace, Value: "}", Pos: startPos})
		l.pos++
		return nil
	case '=':
		l.tokens = append(l.tokens, Token{Type: TokenOperator, Value: "=", Pos: startPos})
		l.pos++
		return nil
	case '<':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '>' {
			l.tokens = append(l.tokens, Token{Type: TokenOperator, Value: "<>", Pos: startPos})
			l.pos += 2
		} else if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.tokens = append(l.tokens, Token{Type: TokenOperator, Value: "<=", Pos: startPos})
			l.pos += 2
		} else {
			l.tokens = append(l.tokens, Token{Type: TokenOperator, Value: "<", Pos: startPos})
			l.pos++
		}
		return nil
	case '>':
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '=' {
			l.tokens = append(l.tokens, Token{Type: TokenOperator, Value: ">=", Pos: startPos})
			l.pos += 2
		} else {
			l.tokens = append(l.tokens, Token{Type: TokenOperator, Value: ">", Pos: startPos})
			l.pos++
		}
		return nil
	case '!':
		// Sheet reference separator
		l.tokens = append(l.tokens, Token{Type: TokenSheet, Value: "!", Pos: startPos})
		l.pos++
		return nil
	case '$':
		// Absolute reference - handled in scanIdentifier
		return l.scanIdentifier()
	case '\'':
		// Quoted sheet name
		return l.scanQuotedSheet()
	}

	// Identifier (function, cell reference, or named range)
	if unicode.IsLetter(rune(ch)) || ch == '_' {
		return l.scanIdentifier()
	}

	return fmt.Errorf("unexpected character '%c' at position %d", ch, l.pos)
}

func (l *Lexer) scanString() error {
	startPos := l.pos
	l.pos++ // Skip opening quote

	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '"' {
			// Check for escaped quote
			if l.pos+1 < len(l.input) && l.input[l.pos+1] == '"' {
				sb.WriteByte('"')
				l.pos += 2
			} else {
				l.pos++ // Skip closing quote
				l.tokens = append(l.tokens, Token{Type: TokenString, Value: sb.String(), Pos: startPos})
				return nil
			}
		} else {
			sb.WriteByte(ch)
			l.pos++
		}
	}

	return fmt.Errorf("unterminated string at position %d", startPos)
}

func (l *Lexer) scanNumber() error {
	startPos := l.pos
	var sb strings.Builder

	// Integer part
	for l.pos < len(l.input) && unicode.IsDigit(rune(l.input[l.pos])) {
		sb.WriteByte(l.input[l.pos])
		l.pos++
	}

	// Decimal part
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		sb.WriteByte('.')
		l.pos++
		for l.pos < len(l.input) && unicode.IsDigit(rune(l.input[l.pos])) {
			sb.WriteByte(l.input[l.pos])
			l.pos++
		}
	}

	// Exponent part
	if l.pos < len(l.input) && (l.input[l.pos] == 'e' || l.input[l.pos] == 'E') {
		sb.WriteByte(l.input[l.pos])
		l.pos++
		if l.pos < len(l.input) && (l.input[l.pos] == '+' || l.input[l.pos] == '-') {
			sb.WriteByte(l.input[l.pos])
			l.pos++
		}
		for l.pos < len(l.input) && unicode.IsDigit(rune(l.input[l.pos])) {
			sb.WriteByte(l.input[l.pos])
			l.pos++
		}
	}

	l.tokens = append(l.tokens, Token{Type: TokenNumber, Value: sb.String(), Pos: startPos})
	return nil
}

func (l *Lexer) scanIdentifier() error {
	startPos := l.pos
	var sb strings.Builder

	// Handle $ for absolute references
	for l.pos < len(l.input) && l.input[l.pos] == '$' {
		sb.WriteByte('$')
		l.pos++
	}

	// Scan letters and digits
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if unicode.IsLetter(rune(ch)) || unicode.IsDigit(rune(ch)) || ch == '_' || ch == '$' {
			sb.WriteByte(ch)
			l.pos++
		} else {
			break
		}
	}

	value := sb.String()
	upper := strings.ToUpper(value)

	// Check for boolean literals
	if upper == "TRUE" || upper == "FALSE" {
		l.tokens = append(l.tokens, Token{Type: TokenBool, Value: upper, Pos: startPos})
		return nil
	}

	// Check for error values
	if strings.HasPrefix(upper, "#") {
		l.tokens = append(l.tokens, Token{Type: TokenError, Value: upper, Pos: startPos})
		return nil
	}

	// Check if it's a function (followed by '(')
	l.skipWhitespace()
	if l.pos < len(l.input) && l.input[l.pos] == '(' {
		l.tokens = append(l.tokens, Token{Type: TokenFunction, Value: upper, Pos: startPos})
		return nil
	}

	// Check if it's a cell reference (letter(s) followed by number(s))
	if isCellReference(value) {
		l.tokens = append(l.tokens, Token{Type: TokenReference, Value: strings.ToUpper(value), Pos: startPos})
		return nil
	}

	// Otherwise, it's a named range
	l.tokens = append(l.tokens, Token{Type: TokenName, Value: value, Pos: startPos})
	return nil
}

func (l *Lexer) scanQuotedSheet() error {
	startPos := l.pos
	l.pos++ // Skip opening quote

	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\'' {
			// Check for escaped quote
			if l.pos+1 < len(l.input) && l.input[l.pos+1] == '\'' {
				sb.WriteByte('\'')
				l.pos += 2
			} else {
				l.pos++ // Skip closing quote
				// Expect '!' after sheet name
				if l.pos < len(l.input) && l.input[l.pos] == '!' {
					l.tokens = append(l.tokens, Token{Type: TokenSheet, Value: sb.String(), Pos: startPos})
					l.pos++ // Skip '!'
					return nil
				}
				return fmt.Errorf("expected '!' after sheet name at position %d", l.pos)
			}
		} else {
			sb.WriteByte(ch)
			l.pos++
		}
	}

	return fmt.Errorf("unterminated sheet name at position %d", startPos)
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && unicode.IsSpace(rune(l.input[l.pos])) {
		l.pos++
	}
}

// isCellReference checks if a string is a cell reference (e.g., A1, $A$1, AA100).
func isCellReference(s string) bool {
	s = strings.ReplaceAll(s, "$", "")
	if len(s) < 2 {
		return false
	}

	// Find where letters end and numbers begin
	i := 0
	for i < len(s) && unicode.IsLetter(rune(s[i])) {
		i++
	}

	if i == 0 || i == len(s) {
		return false
	}

	// Check remaining characters are digits
	for j := i; j < len(s); j++ {
		if !unicode.IsDigit(rune(s[j])) {
			return false
		}
	}

	return true
}
