package dsl

import (
	"fmt"
	"strconv"
)

// Precedence levels for Pratt parsing.
const (
	precNone       = 0
	precOr         = 1
	precAnd        = 2
	precComparison = 3
	precAddSub     = 4
	precMulDiv     = 5
	precUnary      = 6
	precCall       = 7
)

// parser holds the state for Pratt parsing.
type parser struct {
	tokens  []Token
	pos     int
	current Token
}

// Parse takes a slice of tokens and returns the root AST node.
func Parse(tokens []Token) (Node, error) {
	// Filter out newlines — they are not semantically meaningful.
	var filtered []Token
	for _, t := range tokens {
		if t.Type != TOKEN_NEWLINE {
			filtered = append(filtered, t)
		}
	}

	p := &parser{tokens: filtered, pos: 0}
	if len(p.tokens) == 0 {
		return nil, fmt.Errorf("empty token list")
	}
	p.current = p.tokens[0]

	node, err := p.parseTopLevel()
	if err != nil {
		return nil, err
	}

	if p.current.Type != TOKEN_EOF {
		return nil, fmt.Errorf("unexpected token '%s' at line %d, column %d", p.current.Value, p.current.Line, p.current.Column)
	}

	return node, nil
}

func (p *parser) advance() {
	p.pos++
	if p.pos < len(p.tokens) {
		p.current = p.tokens[p.pos]
	}
}

func (p *parser) peek() Token {
	if p.pos+1 < len(p.tokens) {
		return p.tokens[p.pos+1]
	}
	return Token{Type: TOKEN_EOF}
}

func (p *parser) expect(tt TokenType) (Token, error) {
	if p.current.Type != tt {
		return Token{}, fmt.Errorf("expected token type %d but got '%s' at line %d, column %d", tt, p.current.Value, p.current.Line, p.current.Column)
	}
	tok := p.current
	p.advance()
	return tok, nil
}

func (p *parser) parseTopLevel() (Node, error) {
	// scan query
	if p.current.Type == TOKEN_SCAN {
		return p.parseScanQuery()
	}

	// assignment: identifier = expr
	if p.current.Type == TOKEN_IDENTIFIER && p.peek().Type == TOKEN_ASSIGN {
		return p.parseAssignment()
	}

	// standalone expression
	return p.parseExpression(precNone)
}

func (p *parser) parseScanQuery() (Node, error) {
	// consume 'scan'
	p.advance()

	query := &ScanQuery{}

	// expect 'where'
	if p.current.Type != TOKEN_WHERE {
		return nil, fmt.Errorf("expected 'where' after 'scan' at line %d, column %d", p.current.Line, p.current.Column)
	}
	p.advance()

	// parse where expression
	expr, err := p.parseExpression(precNone)
	if err != nil {
		return nil, err
	}
	query.Where = expr

	// optional 'sort by field (asc|desc)'
	if p.current.Type == TOKEN_SORT {
		p.advance()
		if _, err := p.expect(TOKEN_BY); err != nil {
			return nil, fmt.Errorf("expected 'by' after 'sort' at line %d, column %d", p.current.Line, p.current.Column)
		}
		if p.current.Type != TOKEN_IDENTIFIER {
			return nil, fmt.Errorf("expected field name after 'sort by' at line %d, column %d", p.current.Line, p.current.Column)
		}
		field := p.current.Value
		p.advance()

		direction := "asc"
		if p.current.Type == TOKEN_ASC {
			direction = "asc"
			p.advance()
		} else if p.current.Type == TOKEN_DESC {
			direction = "desc"
			p.advance()
		}

		query.SortBy = &SortClause{Field: field, Direction: direction}
	}

	// optional 'limit number'
	if p.current.Type == TOKEN_LIMIT {
		p.advance()
		if p.current.Type != TOKEN_NUMBER {
			return nil, fmt.Errorf("expected number after 'limit' at line %d, column %d", p.current.Line, p.current.Column)
		}
		val, err := strconv.Atoi(p.current.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid limit value '%s'", p.current.Value)
		}
		query.Limit = &val
		p.advance()
	}

	return query, nil
}

func (p *parser) parseAssignment() (Node, error) {
	name := p.current.Value
	p.advance() // consume identifier
	p.advance() // consume '='

	value, err := p.parseExpression(precNone)
	if err != nil {
		return nil, err
	}

	return &AssignmentExpr{Name: name, Value: value}, nil
}

// parseExpression implements Pratt parsing.
func (p *parser) parseExpression(minPrec int) (Node, error) {
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}

	for {
		prec := p.infixPrecedence()
		if prec <= minPrec {
			break
		}

		left, err = p.parseInfix(left, prec)
		if err != nil {
			return nil, err
		}
	}

	return left, nil
}

func (p *parser) parsePrefix() (Node, error) {
	switch p.current.Type {
	case TOKEN_NUMBER:
		val, err := strconv.ParseFloat(p.current.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number '%s'", p.current.Value)
		}
		p.advance()
		return &NumberLiteral{Value: val}, nil

	case TOKEN_STRING:
		val := p.current.Value
		p.advance()
		return &StringLiteral{Value: val}, nil

	case TOKEN_TRUE:
		p.advance()
		return &BooleanLiteral{Value: true}, nil

	case TOKEN_FALSE:
		p.advance()
		return &BooleanLiteral{Value: false}, nil

	case TOKEN_IDENTIFIER:
		name := p.current.Value
		p.advance()
		// Check for function call
		if p.current.Type == TOKEN_LPAREN {
			return p.parseFunctionCall(name)
		}
		return &Identifier{Name: name}, nil

	case TOKEN_LPAREN:
		p.advance() // consume '('
		expr, err := p.parseExpression(precNone)
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TOKEN_RPAREN); err != nil {
			return nil, fmt.Errorf("expected ')' at line %d, column %d", p.current.Line, p.current.Column)
		}
		return expr, nil

	case TOKEN_MINUS:
		p.advance()
		operand, err := p.parseExpression(precUnary)
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Operator: "-", Operand: operand}, nil

	case TOKEN_NOT:
		p.advance()
		operand, err := p.parseExpression(precUnary)
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Operator: "not", Operand: operand}, nil

	default:
		return nil, fmt.Errorf("unexpected token '%s' at line %d, column %d", p.current.Value, p.current.Line, p.current.Column)
	}
}

func (p *parser) parseFunctionCall(name string) (Node, error) {
	p.advance() // consume '('
	var args []Node

	if p.current.Type != TOKEN_RPAREN {
		arg, err := p.parseExpression(precNone)
		if err != nil {
			return nil, err
		}
		args = append(args, arg)

		for p.current.Type == TOKEN_COMMA {
			p.advance() // consume ','
			arg, err := p.parseExpression(precNone)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
	}

	if _, err := p.expect(TOKEN_RPAREN); err != nil {
		return nil, fmt.Errorf("expected ')' after function arguments at line %d, column %d", p.current.Line, p.current.Column)
	}

	return &FunctionCall{Name: name, Args: args}, nil
}

func (p *parser) infixPrecedence() int {
	switch p.current.Type {
	case TOKEN_OR:
		return precOr
	case TOKEN_AND:
		return precAnd
	case TOKEN_EQ, TOKEN_NEQ, TOKEN_LT, TOKEN_GT, TOKEN_LTE, TOKEN_GTE:
		return precComparison
	case TOKEN_PLUS, TOKEN_MINUS:
		return precAddSub
	case TOKEN_STAR, TOKEN_SLASH:
		return precMulDiv
	default:
		return precNone
	}
}

func (p *parser) parseInfix(left Node, prec int) (Node, error) {
	op := p.current.Value
	p.advance()

	right, err := p.parseExpression(prec)
	if err != nil {
		return nil, err
	}

	return &BinaryExpr{Operator: op, Left: left, Right: right}, nil
}
