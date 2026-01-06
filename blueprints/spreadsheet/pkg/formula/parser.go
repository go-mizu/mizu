package formula

import (
	"fmt"
)

// NodeType represents the type of AST node.
type NodeType int

const (
	NodeNumber NodeType = iota
	NodeString
	NodeBool
	NodeError
	NodeReference
	NodeRange
	NodeName
	NodeFunction
	NodeBinaryOp
	NodeUnaryOp
	NodeArray
)

// ASTNode represents a node in the abstract syntax tree.
type ASTNode struct {
	Type     NodeType
	Value    interface{}
	Children []*ASTNode
	Token    Token
}

// Parser parses tokens into an AST.
type Parser struct {
	tokens []Token
	pos    int
}

// NewParser creates a new parser.
func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens}
}

// Parse parses the tokens and returns the AST.
func (p *Parser) Parse() (*ASTNode, error) {
	if len(p.tokens) == 0 {
		return nil, fmt.Errorf("empty token list")
	}

	node, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if p.current().Type != TokenEOF {
		return nil, fmt.Errorf("unexpected token: %v", p.current())
	}

	return node, nil
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() Token {
	token := p.current()
	p.pos++
	return token
}

func (p *Parser) expect(tokenType TokenType) (Token, error) {
	token := p.current()
	if token.Type != tokenType {
		return token, fmt.Errorf("expected %v, got %v at position %d", tokenType, token.Type, token.Pos)
	}
	p.advance()
	return token, nil
}

// parseExpression parses an expression.
func (p *Parser) parseExpression() (*ASTNode, error) {
	return p.parseComparison()
}

// parseComparison parses comparison operators (=, <>, <, >, <=, >=).
func (p *Parser) parseComparison() (*ASTNode, error) {
	left, err := p.parseConcatenation()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenOperator {
		op := p.current().Value
		if op != "=" && op != "<>" && op != "<" && op != ">" && op != "<=" && op != ">=" {
			break
		}
		token := p.advance()
		right, err := p.parseConcatenation()
		if err != nil {
			return nil, err
		}
		left = &ASTNode{
			Type:     NodeBinaryOp,
			Value:    op,
			Children: []*ASTNode{left, right},
			Token:    token,
		}
	}

	return left, nil
}

// parseConcatenation parses string concatenation (&).
func (p *Parser) parseConcatenation() (*ASTNode, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenOperator && p.current().Value == "&" {
		token := p.advance()
		right, err := p.parseAdditive()
		if err != nil {
			return nil, err
		}
		left = &ASTNode{
			Type:     NodeBinaryOp,
			Value:    "&",
			Children: []*ASTNode{left, right},
			Token:    token,
		}
	}

	return left, nil
}

// parseAdditive parses addition and subtraction (+, -).
func (p *Parser) parseAdditive() (*ASTNode, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenOperator {
		op := p.current().Value
		if op != "+" && op != "-" {
			break
		}
		token := p.advance()
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &ASTNode{
			Type:     NodeBinaryOp,
			Value:    op,
			Children: []*ASTNode{left, right},
			Token:    token,
		}
	}

	return left, nil
}

// parseMultiplicative parses multiplication and division (*, /).
func (p *Parser) parseMultiplicative() (*ASTNode, error) {
	left, err := p.parseExponent()
	if err != nil {
		return nil, err
	}

	for p.current().Type == TokenOperator {
		op := p.current().Value
		if op != "*" && op != "/" {
			break
		}
		token := p.advance()
		right, err := p.parseExponent()
		if err != nil {
			return nil, err
		}
		left = &ASTNode{
			Type:     NodeBinaryOp,
			Value:    op,
			Children: []*ASTNode{left, right},
			Token:    token,
		}
	}

	return left, nil
}

// parseExponent parses exponentiation (^).
func (p *Parser) parseExponent() (*ASTNode, error) {
	left, err := p.parsePercent()
	if err != nil {
		return nil, err
	}

	if p.current().Type == TokenOperator && p.current().Value == "^" {
		token := p.advance()
		right, err := p.parseExponent() // Right associative
		if err != nil {
			return nil, err
		}
		left = &ASTNode{
			Type:     NodeBinaryOp,
			Value:    "^",
			Children: []*ASTNode{left, right},
			Token:    token,
		}
	}

	return left, nil
}

// parsePercent parses percentage (%).
func (p *Parser) parsePercent() (*ASTNode, error) {
	node, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	if p.current().Type == TokenOperator && p.current().Value == "%" {
		token := p.advance()
		node = &ASTNode{
			Type:     NodeUnaryOp,
			Value:    "%",
			Children: []*ASTNode{node},
			Token:    token,
		}
	}

	return node, nil
}

// parseUnary parses unary operators (+, -).
func (p *Parser) parseUnary() (*ASTNode, error) {
	if p.current().Type == TokenOperator {
		op := p.current().Value
		if op == "+" || op == "-" {
			token := p.advance()
			operand, err := p.parseUnary()
			if err != nil {
				return nil, err
			}
			return &ASTNode{
				Type:     NodeUnaryOp,
				Value:    op,
				Children: []*ASTNode{operand},
				Token:    token,
			}, nil
		}
	}

	return p.parsePrimary()
}

// parsePrimary parses primary expressions.
func (p *Parser) parsePrimary() (*ASTNode, error) {
	token := p.current()

	switch token.Type {
	case TokenNumber:
		p.advance()
		return &ASTNode{Type: NodeNumber, Value: token.Value, Token: token}, nil

	case TokenString:
		p.advance()
		return &ASTNode{Type: NodeString, Value: token.Value, Token: token}, nil

	case TokenBool:
		p.advance()
		return &ASTNode{Type: NodeBool, Value: token.Value, Token: token}, nil

	case TokenError:
		p.advance()
		return &ASTNode{Type: NodeError, Value: token.Value, Token: token}, nil

	case TokenReference:
		return p.parseReference()

	case TokenName:
		p.advance()
		return &ASTNode{Type: NodeName, Value: token.Value, Token: token}, nil

	case TokenSheet:
		return p.parseSheetReference()

	case TokenFunction:
		return p.parseFunction()

	case TokenLParen:
		p.advance()
		node, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return node, nil

	case TokenLBrace:
		return p.parseArray()

	default:
		return nil, fmt.Errorf("unexpected token: %v at position %d", token.Type, token.Pos)
	}
}

// parseReference parses a cell reference, potentially a range.
func (p *Parser) parseReference() (*ASTNode, error) {
	token := p.advance()
	node := &ASTNode{Type: NodeReference, Value: token.Value, Token: token}

	// Check for range
	if p.current().Type == TokenColon {
		p.advance()
		if p.current().Type != TokenReference {
			return nil, fmt.Errorf("expected cell reference after ':' at position %d", p.current().Pos)
		}
		endToken := p.advance()
		return &ASTNode{
			Type:     NodeRange,
			Value:    token.Value + ":" + endToken.Value,
			Children: []*ASTNode{node, {Type: NodeReference, Value: endToken.Value, Token: endToken}},
			Token:    token,
		}, nil
	}

	return node, nil
}

// parseSheetReference parses a sheet reference (Sheet1!A1 or 'Sheet Name'!A1).
func (p *Parser) parseSheetReference() (*ASTNode, error) {
	sheetToken := p.advance()
	sheetName := sheetToken.Value

	// Parse the cell reference after '!'
	if p.current().Type != TokenReference {
		return nil, fmt.Errorf("expected cell reference after sheet name at position %d", p.current().Pos)
	}

	refNode, err := p.parseReference()
	if err != nil {
		return nil, err
	}

	// Add sheet name to the reference
	if refNode.Type == NodeRange {
		refNode.Value = sheetName + "!" + refNode.Value.(string)
	} else {
		refNode.Value = sheetName + "!" + refNode.Value.(string)
	}

	return refNode, nil
}

// parseFunction parses a function call.
func (p *Parser) parseFunction() (*ASTNode, error) {
	token := p.advance()
	funcName := token.Value

	if _, err := p.expect(TokenLParen); err != nil {
		return nil, err
	}

	args := []*ASTNode{}

	// Parse arguments
	if p.current().Type != TokenRParen {
		for {
			arg, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if p.current().Type == TokenComma {
				p.advance()
			} else {
				break
			}
		}
	}

	if _, err := p.expect(TokenRParen); err != nil {
		return nil, err
	}

	return &ASTNode{
		Type:     NodeFunction,
		Value:    funcName,
		Children: args,
		Token:    token,
	}, nil
}

// parseArray parses an array literal {1,2,3;4,5,6}.
func (p *Parser) parseArray() (*ASTNode, error) {
	token := p.advance() // Skip '{'

	rows := []*ASTNode{}
	currentRow := []*ASTNode{}

	for p.current().Type != TokenRBrace {
		if p.current().Type == TokenSemicolon {
			p.advance()
			if len(currentRow) > 0 {
				rows = append(rows, &ASTNode{Type: NodeArray, Children: currentRow})
				currentRow = []*ASTNode{}
			}
			continue
		}

		if p.current().Type == TokenComma {
			p.advance()
			continue
		}

		elem, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		currentRow = append(currentRow, elem)
	}

	if len(currentRow) > 0 {
		rows = append(rows, &ASTNode{Type: NodeArray, Children: currentRow})
	}

	p.advance() // Skip '}'

	return &ASTNode{
		Type:     NodeArray,
		Children: rows,
		Token:    token,
	}, nil
}
