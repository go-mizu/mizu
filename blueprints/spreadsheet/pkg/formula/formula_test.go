package formula

import (
	"context"
	"testing"
)

// TestLexer tests the formula lexer.
func TestLexer(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []TokenType
		wantErr bool
	}{
		{
			name:  "simple number",
			input: "123",
			want:  []TokenType{TokenNumber, TokenEOF},
		},
		{
			name:  "decimal number",
			input: "123.45",
			want:  []TokenType{TokenNumber, TokenEOF},
		},
		{
			name:  "scientific notation",
			input: "1.5e10",
			want:  []TokenType{TokenNumber, TokenEOF},
		},
		{
			name:  "string literal",
			input: `"hello"`,
			want:  []TokenType{TokenString, TokenEOF},
		},
		{
			name:  "escaped string",
			input: `"say ""hello"""`,
			want:  []TokenType{TokenString, TokenEOF},
		},
		{
			name:  "cell reference",
			input: "A1",
			want:  []TokenType{TokenReference, TokenEOF},
		},
		{
			name:  "absolute reference",
			input: "$A$1",
			want:  []TokenType{TokenReference, TokenEOF},
		},
		{
			name:  "mixed reference",
			input: "$A1",
			want:  []TokenType{TokenReference, TokenEOF},
		},
		{
			name:  "multi-letter column",
			input: "AA100",
			want:  []TokenType{TokenReference, TokenEOF},
		},
		{
			name:  "function call",
			input: "SUM(A1,B2)",
			want:  []TokenType{TokenFunction, TokenLParen, TokenReference, TokenComma, TokenReference, TokenRParen, TokenEOF},
		},
		{
			name:  "boolean TRUE",
			input: "TRUE",
			want:  []TokenType{TokenBool, TokenEOF},
		},
		{
			name:  "boolean FALSE",
			input: "FALSE",
			want:  []TokenType{TokenBool, TokenEOF},
		},
		{
			name:  "arithmetic operators",
			input: "1+2-3*4/5^6",
			want:  []TokenType{TokenNumber, TokenOperator, TokenNumber, TokenOperator, TokenNumber, TokenOperator, TokenNumber, TokenOperator, TokenNumber, TokenOperator, TokenNumber, TokenEOF},
		},
		{
			name:  "comparison operators",
			input: "A1<>B1",
			want:  []TokenType{TokenReference, TokenOperator, TokenReference, TokenEOF},
		},
		{
			name:  "less than equal",
			input: "A1<=B1",
			want:  []TokenType{TokenReference, TokenOperator, TokenReference, TokenEOF},
		},
		{
			name:  "greater than equal",
			input: "A1>=B1",
			want:  []TokenType{TokenReference, TokenOperator, TokenReference, TokenEOF},
		},
		{
			name:  "concatenation",
			input: `"hello"&"world"`,
			want:  []TokenType{TokenString, TokenOperator, TokenString, TokenEOF},
		},
		{
			name:  "percentage",
			input: "50%",
			want:  []TokenType{TokenNumber, TokenOperator, TokenEOF},
		},
		{
			name:  "range reference",
			input: "A1:B10",
			want:  []TokenType{TokenReference, TokenColon, TokenReference, TokenEOF},
		},
		{
			name:  "formula with leading equals",
			input: "=A1+B1",
			want:  []TokenType{TokenReference, TokenOperator, TokenReference, TokenEOF},
		},
		{
			name:  "nested function",
			input: "SUM(A1,MAX(B1,C1))",
			want:  []TokenType{TokenFunction, TokenLParen, TokenReference, TokenComma, TokenFunction, TokenLParen, TokenReference, TokenComma, TokenReference, TokenRParen, TokenRParen, TokenEOF},
		},
		{
			name:  "array literal",
			input: "{1,2,3}",
			want:  []TokenType{TokenLBrace, TokenNumber, TokenComma, TokenNumber, TokenComma, TokenNumber, TokenRBrace, TokenEOF},
		},
		{
			name:  "named range",
			input: "Sales",
			want:  []TokenType{TokenName, TokenEOF},
		},
		{
			name:    "unterminated string",
			input:   `"hello`,
			wantErr: true,
		},
		{
			name:  "whitespace handling",
			input: "  A1  +  B1  ",
			want:  []TokenType{TokenReference, TokenOperator, TokenReference, TokenEOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(tokens) != len(tt.want) {
				t.Errorf("got %d tokens, want %d", len(tokens), len(tt.want))
				for i, tok := range tokens {
					t.Logf("token[%d]: type=%v value=%q", i, tok.Type, tok.Value)
				}
				return
			}

			for i, wantType := range tt.want {
				if tokens[i].Type != wantType {
					t.Errorf("token[%d] type = %v, want %v", i, tokens[i].Type, wantType)
				}
			}
		})
	}
}

// TestParser tests the formula parser.
func TestParser(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "simple number",
			input: "42",
		},
		{
			name:  "string literal",
			input: `"hello"`,
		},
		{
			name:  "cell reference",
			input: "A1",
		},
		{
			name:  "simple addition",
			input: "1+2",
		},
		{
			name:  "operator precedence",
			input: "1+2*3",
		},
		{
			name:  "parentheses",
			input: "(1+2)*3",
		},
		{
			name:  "function call",
			input: "SUM(1,2,3)",
		},
		{
			name:  "nested function",
			input: "SUM(1,MAX(2,3),4)",
		},
		{
			name:  "range reference",
			input: "SUM(A1:B10)",
		},
		{
			name:  "comparison",
			input: "A1>B1",
		},
		{
			name:  "logical expression",
			input: "IF(A1>0,1,0)",
		},
		{
			name:  "unary minus",
			input: "-A1",
		},
		{
			name:  "unary plus",
			input: "+A1",
		},
		{
			name:  "exponentiation",
			input: "2^3^4",
		},
		{
			name:  "concatenation",
			input: `"a"&"b"`,
		},
		{
			name:  "complex expression",
			input: "IF(AND(A1>0,B1<100),SUM(C1:C10)*2,0)",
		},
		{
			name:  "array literal",
			input: "{1,2;3,4}",
		},
		{
			name:    "unmatched paren",
			input:   "(1+2",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				if !tt.wantErr {
					t.Errorf("lexer error: %v", err)
				}
				return
			}

			parser := NewParser(tokens)
			_, err = parser.Parse()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// mockCellGetter implements CellGetter for testing.
type mockCellGetter struct {
	cells map[string]map[string]interface{} // sheetID -> "row,col" -> value
}

func newMockCellGetter() *mockCellGetter {
	return &mockCellGetter{
		cells: make(map[string]map[string]interface{}),
	}
}

func (m *mockCellGetter) setCell(sheetID string, row, col int, value interface{}) {
	if m.cells[sheetID] == nil {
		m.cells[sheetID] = make(map[string]interface{})
	}
	key := CellRefString(row, col)
	m.cells[sheetID][key] = value
}

func (m *mockCellGetter) GetCellValue(ctx context.Context, sheetID string, row, col int) (interface{}, error) {
	if m.cells[sheetID] == nil {
		return nil, nil
	}
	key := CellRefString(row, col)
	return m.cells[sheetID][key], nil
}

func (m *mockCellGetter) GetRangeValues(ctx context.Context, sheetID string, startRow, startCol, endRow, endCol int) ([][]interface{}, error) {
	rows := endRow - startRow + 1
	cols := endCol - startCol + 1
	result := make([][]interface{}, rows)
	for i := range result {
		result[i] = make([]interface{}, cols)
		for j := range result[i] {
			key := CellRefString(startRow+i, startCol+j)
			if m.cells[sheetID] != nil {
				result[i][j] = m.cells[sheetID][key]
			}
		}
	}
	return result, nil
}

func (m *mockCellGetter) GetNamedRange(ctx context.Context, name string) (sheetID string, startRow, startCol, endRow, endCol int, err error) {
	return "", 0, 0, 0, 0, nil
}

func (m *mockCellGetter) ResolveSheetName(ctx context.Context, workbookID, sheetName string) (string, error) {
	// For testing, just return the sheet name as the ID
	return sheetName, nil
}

// TestEvaluator tests the formula evaluator.
func TestEvaluator(t *testing.T) {
	ctx := context.Background()
	mock := newMockCellGetter()

	// Set up test cells
	mock.setCell("sheet1", 0, 0, 10.0)  // A1 = 10
	mock.setCell("sheet1", 0, 1, 20.0)  // B1 = 20
	mock.setCell("sheet1", 0, 2, 30.0)  // C1 = 30
	mock.setCell("sheet1", 1, 0, 1.0)   // A2 = 1
	mock.setCell("sheet1", 1, 1, 2.0)   // B2 = 2
	mock.setCell("sheet1", 1, 2, 3.0)   // C2 = 3
	mock.setCell("sheet1", 2, 0, "abc") // A3 = "abc"

	tests := []struct {
		name    string
		input   string
		want    interface{}
		wantErr bool
	}{
		// Basic arithmetic
		{name: "addition", input: "1+2", want: 3.0},
		{name: "subtraction", input: "5-3", want: 2.0},
		{name: "multiplication", input: "4*5", want: 20.0},
		{name: "division", input: "10/2", want: 5.0},
		{name: "exponent", input: "2^3", want: 8.0},
		{name: "percentage", input: "50%", want: 0.5},
		{name: "negative", input: "-5", want: -5.0},

		// Operator precedence
		{name: "precedence mul over add", input: "1+2*3", want: 7.0},
		{name: "parentheses", input: "(1+2)*3", want: 9.0},

		// Cell references
		{name: "cell reference A1", input: "A1", want: 10.0},
		{name: "cell reference B1", input: "B1", want: 20.0},
		{name: "cell expression", input: "A1+B1", want: 30.0},

		// Comparison operators
		{name: "equals true", input: "1=1", want: true},
		{name: "equals false", input: "1=2", want: false},
		{name: "not equals", input: "1<>2", want: true},
		{name: "less than", input: "1<2", want: true},
		{name: "greater than", input: "2>1", want: true},
		{name: "less or equal", input: "1<=1", want: true},
		{name: "greater or equal", input: "1>=1", want: true},

		// String concatenation
		{name: "concat", input: `"hello"&" "&"world"`, want: "hello world"},

		// Boolean literals
		{name: "true", input: "TRUE", want: true},
		{name: "false", input: "FALSE", want: false},

		// Division by zero
		{name: "div by zero", input: "1/0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := NewLexer(tt.input)
			tokens, err := lexer.Tokenize()
			if err != nil {
				t.Fatalf("lexer error: %v", err)
			}

			parser := NewParser(tokens)
			ast, err := parser.Parse()
			if err != nil {
				t.Fatalf("parser error: %v", err)
			}

			evalCtx := &EvalContext{
				SheetID:    "sheet1",
				CellGetter: mock,
				Circular:   make(map[string]bool),
			}
			evaluator := NewEvaluator(evalCtx)
			result, err := evaluator.Evaluate(ctx, ast)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got result: %v", result)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Compare results
			switch want := tt.want.(type) {
			case float64:
				got, ok := result.(float64)
				if !ok {
					t.Errorf("expected float64, got %T", result)
				} else if got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			case bool:
				got, ok := result.(bool)
				if !ok {
					t.Errorf("expected bool, got %T", result)
				} else if got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			case string:
				got, ok := result.(string)
				if !ok {
					t.Errorf("expected string, got %T", result)
				} else if got != want {
					t.Errorf("got %q, want %q", got, want)
				}
			}
		})
	}
}

// TestFunctions tests built-in functions.
func TestFunctions(t *testing.T) {
	tests := []struct {
		name    string
		fn      string
		args    []interface{}
		want    interface{}
		wantErr bool
	}{
		// Math functions
		{name: "SUM", fn: "SUM", args: []interface{}{1.0, 2.0, 3.0}, want: 6.0},
		{name: "SUM with array", fn: "SUM", args: []interface{}{[][]interface{}{{1.0, 2.0}, {3.0, 4.0}}}, want: 10.0},
		{name: "AVERAGE", fn: "AVERAGE", args: []interface{}{2.0, 4.0, 6.0}, want: 4.0},
		{name: "MIN", fn: "MIN", args: []interface{}{5.0, 2.0, 8.0}, want: 2.0},
		{name: "MAX", fn: "MAX", args: []interface{}{5.0, 2.0, 8.0}, want: 8.0},
		{name: "COUNT", fn: "COUNT", args: []interface{}{1.0, 2.0, "text", nil}, want: 2.0},
		{name: "COUNTA", fn: "COUNTA", args: []interface{}{1.0, 2.0, "text", nil}, want: 3.0},
		{name: "ABS positive", fn: "ABS", args: []interface{}{-5.0}, want: 5.0},
		{name: "ABS negative", fn: "ABS", args: []interface{}{5.0}, want: 5.0},
		{name: "ROUND", fn: "ROUND", args: []interface{}{2.567, 2.0}, want: 2.57},
		{name: "ROUND no decimals", fn: "ROUND", args: []interface{}{2.567}, want: 3.0},
		{name: "INT", fn: "INT", args: []interface{}{2.9}, want: 2.0},
		{name: "MOD", fn: "MOD", args: []interface{}{7.0, 3.0}, want: 1.0},
		{name: "POWER", fn: "POWER", args: []interface{}{2.0, 3.0}, want: 8.0},
		{name: "SQRT", fn: "SQRT", args: []interface{}{16.0}, want: 4.0},
		{name: "SQRT negative", fn: "SQRT", args: []interface{}{-1.0}, wantErr: true},
		{name: "LN", fn: "LN", args: []interface{}{2.718281828}, want: 0.9999999998311266},
		{name: "LOG10", fn: "LOG10", args: []interface{}{100.0}, want: 2.0},
		{name: "SIGN positive", fn: "SIGN", args: []interface{}{5.0}, want: 1.0},
		{name: "SIGN negative", fn: "SIGN", args: []interface{}{-5.0}, want: -1.0},
		{name: "SIGN zero", fn: "SIGN", args: []interface{}{0.0}, want: 0.0},

		// Text functions
		{name: "CONCATENATE", fn: "CONCATENATE", args: []interface{}{"a", "b", "c"}, want: "abc"},
		{name: "LEFT", fn: "LEFT", args: []interface{}{"hello", 3.0}, want: "hel"},
		{name: "RIGHT", fn: "RIGHT", args: []interface{}{"hello", 3.0}, want: "llo"},
		{name: "MID", fn: "MID", args: []interface{}{"hello", 2.0, 3.0}, want: "ell"},
		{name: "LEN", fn: "LEN", args: []interface{}{"hello"}, want: 5.0},
		{name: "LOWER", fn: "LOWER", args: []interface{}{"HELLO"}, want: "hello"},
		{name: "UPPER", fn: "UPPER", args: []interface{}{"hello"}, want: "HELLO"},
		{name: "TRIM", fn: "TRIM", args: []interface{}{"  hello  "}, want: "hello"},
		{name: "SUBSTITUTE", fn: "SUBSTITUTE", args: []interface{}{"hello world", "world", "there"}, want: "hello there"},
		{name: "REPT", fn: "REPT", args: []interface{}{"ab", 3.0}, want: "ababab"},

		// Logical functions
		{name: "IF true", fn: "IF", args: []interface{}{true, "yes", "no"}, want: "yes"},
		{name: "IF false", fn: "IF", args: []interface{}{false, "yes", "no"}, want: "no"},
		{name: "AND all true", fn: "AND", args: []interface{}{true, true, true}, want: true},
		{name: "AND one false", fn: "AND", args: []interface{}{true, false, true}, want: false},
		{name: "OR all false", fn: "OR", args: []interface{}{false, false, false}, want: false},
		{name: "OR one true", fn: "OR", args: []interface{}{false, true, false}, want: true},
		{name: "NOT true", fn: "NOT", args: []interface{}{true}, want: false},
		{name: "NOT false", fn: "NOT", args: []interface{}{false}, want: true},
		{name: "XOR odd", fn: "XOR", args: []interface{}{true, false, true, false}, want: false},
		{name: "CHOOSE", fn: "CHOOSE", args: []interface{}{2.0, "a", "b", "c"}, want: "b"},

		// Statistical functions
		{name: "MEDIAN odd", fn: "MEDIAN", args: []interface{}{1.0, 3.0, 5.0}, want: 3.0},
		{name: "MEDIAN even", fn: "MEDIAN", args: []interface{}{1.0, 2.0, 3.0, 4.0}, want: 2.5},
		{name: "LARGE", fn: "LARGE", args: []interface{}{[][]interface{}{{1.0, 5.0, 3.0}}, 2.0}, want: 3.0},
		{name: "SMALL", fn: "SMALL", args: []interface{}{[][]interface{}{{1.0, 5.0, 3.0}}, 2.0}, want: 3.0},

		// Information functions
		{name: "ISBLANK nil", fn: "ISBLANK", args: []interface{}{nil}, want: true},
		{name: "ISBLANK empty", fn: "ISBLANK", args: []interface{}{""}, want: true},
		{name: "ISBLANK value", fn: "ISBLANK", args: []interface{}{1.0}, want: false},
		{name: "ISNUMBER true", fn: "ISNUMBER", args: []interface{}{42.0}, want: true},
		{name: "ISNUMBER false", fn: "ISNUMBER", args: []interface{}{"text"}, want: false},
		{name: "ISTEXT true", fn: "ISTEXT", args: []interface{}{"hello"}, want: true},
		{name: "ISTEXT false", fn: "ISTEXT", args: []interface{}{42.0}, want: false},

		// New math functions
		{name: "TRUNC", fn: "TRUNC", args: []interface{}{3.567, 2.0}, want: 3.56},
		{name: "TRUNC no decimals", fn: "TRUNC", args: []interface{}{3.567}, want: 3.0},
		{name: "GCD", fn: "GCD", args: []interface{}{12.0, 18.0}, want: 6.0},
		{name: "GCD three", fn: "GCD", args: []interface{}{24.0, 36.0, 48.0}, want: 12.0},
		{name: "LCM", fn: "LCM", args: []interface{}{4.0, 6.0}, want: 12.0},
		{name: "FACT", fn: "FACT", args: []interface{}{5.0}, want: 120.0},
		{name: "FACT zero", fn: "FACT", args: []interface{}{0.0}, want: 1.0},
		{name: "COMBIN", fn: "COMBIN", args: []interface{}{5.0, 2.0}, want: 10.0},
		{name: "PERMUT", fn: "PERMUT", args: []interface{}{5.0, 2.0}, want: 20.0},
		{name: "QUOTIENT", fn: "QUOTIENT", args: []interface{}{10.0, 3.0}, want: 3.0},
		{name: "MROUND", fn: "MROUND", args: []interface{}{7.0, 3.0}, want: 6.0},
		{name: "ODD", fn: "ODD", args: []interface{}{2.5}, want: 3.0},
		{name: "EVEN", fn: "EVEN", args: []interface{}{3.0}, want: 4.0},

		// Hyperbolic functions
		{name: "SINH", fn: "SINH", args: []interface{}{0.0}, want: 0.0},
		{name: "COSH", fn: "COSH", args: []interface{}{0.0}, want: 1.0},
		{name: "TANH", fn: "TANH", args: []interface{}{0.0}, want: 0.0},

		// New text functions
		{name: "EXACT match", fn: "EXACT", args: []interface{}{"hello", "hello"}, want: true},
		{name: "EXACT different", fn: "EXACT", args: []interface{}{"hello", "Hello"}, want: false},
		{name: "DOLLAR", fn: "DOLLAR", args: []interface{}{1234.567, 2.0}, want: "$1,234.57"},
		{name: "REGEXMATCH true", fn: "REGEXMATCH", args: []interface{}{"hello world", "wo.*d"}, want: true},
		{name: "REGEXMATCH false", fn: "REGEXMATCH", args: []interface{}{"hello", "wo.*d"}, want: false},
		{name: "REGEXEXTRACT", fn: "REGEXEXTRACT", args: []interface{}{"hello123world", "[0-9]+"}, want: "123"},
		{name: "REGEXREPLACE", fn: "REGEXREPLACE", args: []interface{}{"hello123", "[0-9]+", "XXX"}, want: "helloXXX"},

		// Date/Time functions
		{name: "TIME", fn: "TIME", args: []interface{}{12.0, 30.0, 45.0}, want: 0.5213541666666667},

		// New information functions
		{name: "ISEVEN true", fn: "ISEVEN", args: []interface{}{4.0}, want: true},
		{name: "ISEVEN false", fn: "ISEVEN", args: []interface{}{5.0}, want: false},
		{name: "ISODD true", fn: "ISODD", args: []interface{}{5.0}, want: true},
		{name: "ISODD false", fn: "ISODD", args: []interface{}{4.0}, want: false},
		{name: "ADDRESS", fn: "ADDRESS", args: []interface{}{1.0, 1.0, 1.0}, want: "$A$1"},
		{name: "ADDRESS relative", fn: "ADDRESS", args: []interface{}{1.0, 1.0, 4.0}, want: "A1"},

		// Conditional aggregates with criteria
		{name: "COUNTIF", fn: "COUNTIF", args: []interface{}{[][]interface{}{{1.0, 2.0, 3.0, 4.0, 5.0}}, ">2"}, want: 3.0},
		{name: "SUMIF", fn: "SUMIF", args: []interface{}{[][]interface{}{{1.0, 2.0, 3.0, 4.0, 5.0}}, ">2"}, want: 12.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn, ok := Functions[tt.fn]
			if !ok {
				t.Fatalf("function %s not found", tt.fn)
			}

			result, err := fn(tt.args...)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got result: %v", result)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			switch want := tt.want.(type) {
			case float64:
				got, ok := result.(float64)
				if !ok {
					t.Errorf("expected float64, got %T (%v)", result, result)
				} else if got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			case bool:
				got, ok := result.(bool)
				if !ok {
					t.Errorf("expected bool, got %T (%v)", result, result)
				} else if got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			case string:
				got, ok := result.(string)
				if !ok {
					t.Errorf("expected string, got %T (%v)", result, result)
				} else if got != want {
					t.Errorf("got %q, want %q", got, want)
				}
			}
		})
	}
}

// TestParseCellRef tests cell reference parsing.
func TestParseCellRef(t *testing.T) {
	tests := []struct {
		input   string
		wantRow int
		wantCol int
		wantErr bool
	}{
		{input: "A1", wantRow: 0, wantCol: 0},
		{input: "B1", wantRow: 0, wantCol: 1},
		{input: "A2", wantRow: 1, wantCol: 0},
		{input: "Z1", wantRow: 0, wantCol: 25},
		{input: "AA1", wantRow: 0, wantCol: 26},
		{input: "AB1", wantRow: 0, wantCol: 27},
		{input: "$A$1", wantRow: 0, wantCol: 0},
		{input: "$A1", wantRow: 0, wantCol: 0},
		{input: "A$1", wantRow: 0, wantCol: 0},
		{input: "A100", wantRow: 99, wantCol: 0},
		{input: "XFD1048576", wantRow: 1048575, wantCol: 16383}, // Excel max
		{input: "", wantErr: true},
		{input: "A", wantErr: true},
		{input: "1", wantErr: true},
		{input: "1A", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			row, col, err := ParseCellRef(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got row=%d, col=%d", row, col)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if row != tt.wantRow || col != tt.wantCol {
				t.Errorf("got row=%d, col=%d, want row=%d, col=%d", row, col, tt.wantRow, tt.wantCol)
			}
		})
	}
}

// TestColToLetter tests column number to letter conversion.
func TestColToLetter(t *testing.T) {
	tests := []struct {
		col  int
		want string
	}{
		{col: 0, want: "A"},
		{col: 1, want: "B"},
		{col: 25, want: "Z"},
		{col: 26, want: "AA"},
		{col: 27, want: "AB"},
		{col: 51, want: "AZ"},
		{col: 52, want: "BA"},
		{col: 701, want: "ZZ"},
		{col: 702, want: "AAA"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := ColToLetter(tt.col)
			if got != tt.want {
				t.Errorf("ColToLetter(%d) = %q, want %q", tt.col, got, tt.want)
			}
		})
	}
}

// TestParseRangeRef tests range reference parsing.
func TestParseRangeRef(t *testing.T) {
	tests := []struct {
		input                                    string
		wantStartRow, wantStartCol               int
		wantEndRow, wantEndCol                   int
		wantErr                                  bool
	}{
		{input: "A1:B2", wantStartRow: 0, wantStartCol: 0, wantEndRow: 1, wantEndCol: 1},
		{input: "A1:A10", wantStartRow: 0, wantStartCol: 0, wantEndRow: 9, wantEndCol: 0},
		{input: "B2:A1", wantStartRow: 0, wantStartCol: 0, wantEndRow: 1, wantEndCol: 1}, // Normalized
		{input: "$A$1:$B$2", wantStartRow: 0, wantStartCol: 0, wantEndRow: 1, wantEndCol: 1},
		{input: "A1", wantErr: true},
		{input: "A1:B", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			startRow, startCol, endRow, endCol, err := ParseRangeRef(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if startRow != tt.wantStartRow || startCol != tt.wantStartCol ||
				endRow != tt.wantEndRow || endCol != tt.wantEndCol {
				t.Errorf("got (%d,%d):(%d,%d), want (%d,%d):(%d,%d)",
					startRow, startCol, endRow, endCol,
					tt.wantStartRow, tt.wantStartCol, tt.wantEndRow, tt.wantEndCol)
			}
		})
	}
}
